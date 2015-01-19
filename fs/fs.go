/*

Implements a FUSE filesystem, with a focus on snapshots.

The root directory contains two specials directory:

- **latest**, it contains the latest version of each files/directories (e.g. /datadb/mnt/latest/writing).
- **snapshots**, it contains a list of directory with the file/dir name, and inside this directory,
a list of directory: one directory per snapshots, and finally inside this dir,
the file/dir (e.g /datadb/mnt/snapshots/writing/2014-05-04T17:42:48+02:00/writing).

*/
package fs

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/jinzhu/now"

	"github.com/tsileo/blobsnap/clientutil"
	"github.com/tsileo/blobsnap/snapshot"
	"github.com/tsileo/blobstash/client2"
)

type DirType int

const (
	BasicDir DirType = iota
	Root
	HostRoot
	HostLatest
	HostSnapshots
	SnapshotDir
	SnapshotsDir
)

func (dt DirType) String() string {
	switch dt {
	case Root:
		return "Root"
	case BasicDir:
		return "BasicDir"
	case HostRoot:
		return "HostRoot"
	case HostLatest:
		return "HostLatest"
	case HostSnapshots:
		return "HostSnapshots"
	case SnapshotsDir:
		return "SnapshotsDir"
	case SnapshotDir:
		return "SnapshotDir"
	}
	return ""
}

// Mount the filesystem to the given mountpoint
func Mount(server string, mountpoint string, stop <-chan bool, stopped chan<- bool) {
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	log.Printf("Mounting read-only filesystem on %v\nCtrl+C to unmount.", mountpoint)

	cs := make(chan os.Signal, 1)
	signal.Notify(cs, os.Interrupt)
	go func() {
		select {
		case <-cs:
			log.Printf("got signal")
			break
		case <-stop:
			log.Printf("got stop")
			break
		}
		log.Println("Closing client...")
		//client.Blobs.Close()
		log.Printf("Unmounting %v...\n", mountpoint)
		err := fuse.Unmount(mountpoint)
		if err != nil {
			log.Printf("Error unmounting: %v", err)
		} else {
			stopped <- true
		}
	}()

	err = fs.Serve(c, NewFS(server))
	if err != nil {
		log.Fatal(err)
	}
	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	Hosts    []string
	SnapSets map[string][]*snapshot.Snapshot

	RootDir *Dir
	bs      *client2.BlobStore
	kvs     *client2.KvStore
}

// NewFS initialize a new file system.
func NewFS(server string) (fs *FS) {
	// Override supported time format
	now.TimeFormats = []string{"2006-1-2T15:4:5", "2006-1-2T15:4", "2006-1-2T15", "2006-1-2", "2006-1", "2006"}
	bs := client2.NewBlobStore(server)
	kvs := client2.NewKvStore(server)
	fs = &FS{
		bs:       bs,
		kvs:      kvs,
		Hosts:    []string{},
		SnapSets: map[string][]*snapshot.Snapshot{},
	}
	if err := fs.Reload(); err != nil {
		panic(err)
	}
	return
}
func (fs *FS) Reload() error {
	fs.Hosts = []string{}
	fs.SnapSets = map[string][]*snapshot.Snapshot{}
	keys, err := fs.kvs.Keys("blobsnap:snapset:", "blobsnap:snapset:\xff", 0)
	if err != nil {
		return fmt.Errorf("failed kvs.Keys: %v", err)
	}
	for _, kv := range keys {
		log.Printf("%+v", kv)
		snapshot := &snapshot.Snapshot{}
		if err := json.Unmarshal([]byte(kv.Value), snapshot); err != nil {
			return fmt.Errorf("failed to unmarshal: %v", err)
		}
		_, ok := fs.SnapSets[snapshot.Hostname]
		if !ok {
			fs.Hosts = append(fs.Hosts, snapshot.Hostname)
		}
		fs.SnapSets[snapshot.Hostname] = append(fs.SnapSets[snapshot.Hostname], snapshot)
	}
	return nil
}

func (fs *FS) Root() (fs.Node, fuse.Error) {
	return NewRootDir(fs), nil
}

func NewRootDir(fs *FS) (d *Dir) {
	d = NewDir(fs, Root, "root", "", "", os.ModeDir, "")
	return d
}

type Node struct {
	Name    string
	Mode    os.FileMode
	Ref     string
	Size    uint64
	ModTime string
	Extra   string
	fs      *FS
}

func (n *Node) Attr() fuse.Attr {
	attr := fuse.Attr{
		Mode: n.Mode,
		Size: n.Size,
	}
	if n.ModTime != "" {
		t, err := time.Parse(time.RFC3339, n.ModTime)
		if err != nil {
			panic(fmt.Errorf("error parsing mtime for %v: %v", n, err))
		}
		attr.Mtime = t
	}
	return attr
}

func (n *Node) Setattr(req *fuse.SetattrRequest, resp *fuse.SetattrResponse, intr fs.Intr) fuse.Error {
	n.Mode = req.Mode
	return nil
}

type Dir struct {
	Node
	Type     DirType
	Children map[string]fs.Node
	Meta     *clientutil.Meta
}

func NewDir(cfs *FS, dtype DirType, name string, ref string, modTime string, mode os.FileMode, extra string) (d *Dir) {
	d = &Dir{}
	d.Type = dtype
	d.Node = Node{}
	d.Mode = os.ModeDir
	d.fs = cfs
	d.Ref = ref
	d.Name = name
	d.ModTime = modTime
	d.Mode = os.FileMode(mode)
	d.Children = make(map[string]fs.Node)
	d.Extra = extra
	return
}

func (d *Dir) readDir() (out []fuse.Dirent, ferr fuse.Error) {
	meta, err := clientutil.NewMetaFromBlobStore(d.fs.bs, d.Ref)
	if err != nil {
		panic(err)
	}
	if meta.Size > 0 {
		metacontent, err := meta.FetchMetaContent(d.fs.bs)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch meta content: %v", err)
		}
		for _, hash := range metacontent.Mapping {
			meta, err := clientutil.NewMetaFromBlobStore(d.fs.bs, hash.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to fetch meta: %v", err)
			}
			var dirent fuse.Dirent
			if meta.Type == "file" {
				dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
				d.Children[meta.Name] = NewFile(d.fs, meta.Name, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
			} else {
				dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
				d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, meta.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
			}
			out = append(out, dirent)
		}
	}
	return
}

func (d *Dir) Lookup(name string, intr fs.Intr) (fs fs.Node, err fuse.Error) {
	log.Printf("OP Lookup %v", name)
	if len(d.Children) == 0 {
		d.loadDir()
	}
	var ok bool
	fs, ok = d.Children[name]
	if ok {
		return
	}
	return
}

func (d *Dir) ReadDir(intr fs.Intr) (out []fuse.Dirent, err fuse.Error) {
	log.Printf("OP ReadDir %v", d)
	return d.loadDir()
}

func (d *Dir) loadDir() (out []fuse.Dirent, err fuse.Error) {
	log.Printf("OP loadDir %v", d)
	d.fs.Reload()
	// TODO only reload when needed
	switch d.Type {
	case Root:
		d.Children = make(map[string]fs.Node)
		for _, host := range d.fs.Hosts {
			out = append(out, fuse.Dirent{Name: host, Type: fuse.DT_Dir})
			d.Children[host] = NewDir(d.fs, HostRoot, host, host, "", os.ModeDir, "")
		}
		return out, err
	case HostRoot:
		d.Children = make(map[string]fs.Node)
		out = append(out, fuse.Dirent{Name: "latest", Type: fuse.DT_Dir})
		d.Children["latest"] = NewDir(d.fs, HostLatest, "latest", d.Ref, "", os.ModeDir, "")
		out = append(out, fuse.Dirent{Name: "snapshots", Type: fuse.DT_Dir})
		d.Children["snapshots"] = NewDir(d.fs, HostSnapshots, "snapshots", d.Ref, "", os.ModeDir, "")
		return out, err
	case HostLatest:
		log.Printf("HostLatest")
		for _, snap := range d.fs.SnapSets[d.Ref] {
			meta, err := snap.FetchMeta(d.fs.bs)
			if err != nil {
				panic(err)
			}
			if meta.IsFile() {
				dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
				d.Children[meta.Name] = NewFile(d.fs, meta.Name, meta.Ref, meta.Size, meta.ModTime, os.FileMode(uint32(meta.Mode)))
				out = append(out, dirent)
			} else {
				dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
				d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, meta.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
				out = append(out, dirent)
			}
		}
		return out, err
	case HostSnapshots:
		for _, snap := range d.fs.SnapSets[d.Ref] {
			snapName := filepath.Base(snap.Path)
			snapHash := snap.SnapSetKey
			dirent := fuse.Dirent{Name: snapName, Type: fuse.DT_Dir}
			d.Children[snapName] = NewDir(d.fs, SnapshotsDir, snapName, snapHash, "", os.ModeDir, "")
			out = append(out, dirent)
		}
		return out, err
	case SnapshotsDir:
		versions, err := d.fs.kvs.Versions(fmt.Sprintf("blobsnap:snapset:%v", d.Ref), 0, int(time.Now().UTC().UnixNano()), 0)
		if err != nil {
			panic(err)
		}
		for _, kv := range versions.Versions {
			snap := &snapshot.Snapshot{}
			if err := json.Unmarshal([]byte(kv.Value), snap); err != nil {
				panic(err)
			}
			stime := time.Unix(0, int64(kv.Version))
			sname := stime.Format(time.RFC3339)
			dirent := fuse.Dirent{Name: sname, Type: fuse.DT_Dir}
			d.Children[sname] = NewDir(d.fs, SnapshotDir, sname, snap.Ref, "", os.ModeDir, d.Name)
			out = append(out, dirent)

		}
		return out, err
	case SnapshotDir:
		meta, err := clientutil.NewMetaFromBlobStore(d.fs.bs, d.Ref)
		if err != nil {
			panic(err)
		}
		var dirent fuse.Dirent
		if meta.IsFile() {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
		} else {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, meta.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
		}
		out = append(out, dirent)
		return out, err
	}
	return d.readDir()
}

type File struct {
	Node
	Meta     *clientutil.Meta
	FakeFile *clientutil.FakeFile
}

func NewFile(fs *FS, name string, ref string, size int, modTime string, mode os.FileMode) *File {
	f := &File{}
	f.Name = name
	f.Ref = ref
	f.Size = uint64(size)
	f.ModTime = modTime
	f.Mode = mode
	f.fs = fs
	meta, err := clientutil.NewMetaFromBlobStore(fs.bs, ref)
	if err != nil {
		panic(err)
	}
	f.Meta = meta
	return f
}

func (f *File) Attr() fuse.Attr {
	return fuse.Attr{Inode: 2, Mode: 0444, Size: f.Size}
}
func (f *File) Open(req *fuse.OpenRequest, res *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	f.FakeFile = clientutil.NewFakeFile(f.fs.bs, f.Meta)
	return f, nil
}
func (f *File) Release(req *fuse.ReleaseRequest, intr fs.Intr) fuse.Error {
	f.FakeFile.Close()
	f.FakeFile = nil
	return nil
}
func (f *File) Read(req *fuse.ReadRequest, res *fuse.ReadResponse, intr fs.Intr) fuse.Error {
	//log.Printf("Read %+v", f)
	if req.Offset >= int64(f.Size) {
		return nil
	}
	buf := make([]byte, req.Size)
	n, err := f.FakeFile.ReadAt(buf, req.Offset)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Printf("Error reading FakeFile %+v on %v at %d: %v", f, f.Ref, req.Offset, err)
		return fuse.EIO
	}
	res.Data = buf[:n]
	return nil
}
