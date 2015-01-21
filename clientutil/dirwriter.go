package clientutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// node represents either a file or directory in the directory tree
type node struct {
	// root of the snapshot
	root    bool
	skipped bool

	done bool

	// File path/FileInfo
	path string
	fi   os.FileInfo

	// Children (if the node is a directory)
	children []*node
	parent   *node

	// Upload result is stored in the node
	wr   *WriteResult
	meta *Meta
	err  error

	// Used to sync access to the WriteResult/Meta
	mu   sync.Mutex
	cond sync.Cond
}

func (node *node) String() string {
	return fmt.Sprintf("[node %v done=%v, meta=%+v, err=%v]", node.path, node.done, node.meta, node.err)
}

// excluded returns true if the base path match one of the defined shell pattern
func (up *Uploader) excluded(path string) bool {
	//for _, ignoredFile := range client.ignoredFiles {
	//	matched, _ := filepath.Match(ignoredFile, filepath.Base(path))
	//	if matched {
	//		return true
	//	}
	//}
	return false
}

// Recursively read the directory and
// send/route the files/directories to the according channel for processing
func (up *Uploader) DirExplorer(path string, pnode *node, nodes chan<- *node) {
	pnode.mu.Lock()
	defer pnode.mu.Unlock()
	dirdata, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}
	for _, fi := range dirdata {
		abspath := filepath.Join(path, fi.Name())
		n := &node{path: abspath, fi: fi, parent: pnode}
		n.cond.L = &n.mu
		if fi.IsDir() {
			up.DirExplorer(abspath, n, nodes)
			nodes <- n
			pnode.children = append(pnode.children, n)
		} else {
			if fi.Mode()&os.ModeSymlink == 0 {
				if !up.excluded(abspath) {
					nodes <- n
					pnode.children = append(pnode.children, n)
				}
				// else {
				//	log.Printf("DirExplorer: file %v excluded", abspath)
				//}
			}
		}
	}
	pnode.cond.Broadcast()
	return
}

// DirWriter reads the directory and upload it.
func (up *Uploader) DirWriterNode(node *node) {
	node.mu.Lock()
	defer node.mu.Unlock()
	//log.Printf("DirWriterNode %v star", node)
	node.wr = NewWriteResult()
	hashes := []string{}

	// Wait for all children node to finish
	node.skipped = true
	for _, cnode := range node.children {
		cnode.mu.Lock()
		for !cnode.done {
			cnode.cond.Wait()
		}
		if cnode.err != nil {
			panic(cnode.err)
			node.err = cnode.err
			return
		}
		node.skipped = node.skipped && cnode.skipped
		node.wr.Add(cnode.wr)
		if up.Wr != nil {
			up.Wr.Add(cnode.wr)
		}
		cnode.wr.free()
		cnode.wr = nil
		hashes = append(hashes, cnode.meta.Hash)
		cnode.meta.free()
		cnode.meta = nil
		cnode.mu.Unlock()
	}
	up.StartDirUpload()
	defer up.DirUploadDone()

	sort.Strings(hashes)
	mc := NewMetaContent()
	for _, hash := range hashes {
		mc.AddHash(hash)
	}
	mhash, mjs := mc.Json()
	//cnt, err := up.client.Scard(con, node.wr.Hash)
	if err := up.bs.Put(mhash, mjs); err != nil {
		node.err = err
		return
	}
	node.wr.Hash = mhash
	if node.skipped {
		node.wr.DirsSkipped++
	} else {
		node.wr.DirsUploaded++
	}
	node.wr.DirsCount++
	if up.Wr != nil {
		twr := NewWriteResult()
		if node.skipped {
			twr.DirsSkipped++
		} else {
			twr.DirsUploaded++
		}
		twr.DirsCount++
		up.Wr.Add(twr)
		twr.free()
	}
	// TODO WriteResult exisiting handling
	node.meta = NewMeta()
	node.meta.Name = filepath.Base(node.path)
	node.meta.Type = "dir"
	node.meta.Ref = mhash
	node.meta.Size = node.wr.Size
	node.meta.Mode = uint32(node.fi.Mode())
	node.meta.ModTime = node.fi.ModTime().Format(time.RFC3339)
	mhash, mjs = node.meta.Json()
	node.meta.Hash = mhash
	if err := up.bs.Put(mhash, mjs); err != nil {
		node.err = err
		return
	}
	node.done = true
	node.cond.Broadcast()
	return
}

// PutDir upload a directory, it returns the saved Meta,
// a WriteResult containing infos about uploaded blobs.
func (up *Uploader) PutDir(path string) (*Meta, *WriteResult, error) {
	//log.Printf("PutDir %v\n", path)
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}
	nodes := make(chan *node)
	fi, _ := os.Stat(abspath)
	n := &node{root: true, path: abspath, fi: fi}
	n.cond.L = &n.mu

	var wg sync.WaitGroup
	// Iterate the directory tree in a goroutine
	// and dispatch node accordingly in the files/result channels.
	wg.Add(1)
	go func() {
		defer wg.Done()
		up.DirExplorer(path, n, nodes)
		defer close(nodes)
	}()
	// Upload discovered files (100 file descriptor at the same time max).
	wg.Add(1)
	l := make(chan struct{}, 25)
	go func() {
		defer wg.Done()
		for f := range nodes {
			wg.Add(1)
			l <- struct{}{}
			go func(node *node) {
				defer func() {
					<-l
				}()
				defer wg.Done()
				if node.fi.IsDir() {
					up.DirWriterNode(node)
					if node.err != nil {
						n.err = fmt.Errorf("error DirWriterNode with node %v", node)
					}
				} else {
					node.mu.Lock()
					defer node.mu.Unlock()
					node.meta, node.wr, node.err = up.PutFile(node.path)
					if node.err != nil {
						n.err = fmt.Errorf("error PutFile with node %v", node)
					}
					if node.wr.FilesSkipped == 1 {
						node.skipped = true
					}
					node.done = true
					node.cond.Broadcast()
				}
			}(f)
		}
	}()
	wg.Wait()
	// Upload the root directory
	up.DirWriterNode(n)
	return n.meta, n.wr, n.err
}
