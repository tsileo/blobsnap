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
	"context"
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

	"github.com/antonovvk/blobsnap/clientutil"
	"github.com/antonovvk/blobsnap/snapshot"
	"github.com/antonovvk/blobsnap/store"
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
func Mount(bs store.BlobStore, kvs store.KvStore, mountpoint string, stop <-chan bool, stopped chan<- bool) {
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
		log.Printf("Unmounting %v...\n", mountpoint)
		err := fuse.Unmount(mountpoint)
		if err != nil {
			log.Printf("Error unmounting: %v", err)
		} else {
			stopped <- true
		}
	}()

	err = fs.Serve(c, NewFS(bs, kvs))
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
	bs      store.BlobStore
	kvs     store.KvStore
}

// NewFS initialize a new file system.
func NewFS(bs store.BlobStore, kvs store.KvStore) (fs *FS) {
	// Override supported time format
	now.TimeFormats = []string{"2006-1-2T15:4:5", "2006-1-2T15:4", "2006-1-2T15", "2006-1-2", "2006-1", "2006"}
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
	entries, err := fs.kvs.Entries("blobsnap:snapset:", "blobsnap:snapset:\xff", 0)
	if err != nil {
		return fmt.Errorf("failed kvs.Keys: %v", err)
	}
	for _, e := range entries {
		log.Printf("entry: %s", string(e.Data))
		snapshot := &snapshot.Snapshot{}
		if err := json.Unmarshal(e.Data, snapshot); err != nil {
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

func (fs *FS) Root() (fs.Node, error) {
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

func (d Dir) String() string {
	return fmt.Sprintf("DIR %s mode=%d type=%s fs=%v", d.Name, d.Mode, d.Type, *d.fs)
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = d.Mode
	if d.ModTime != "" {
		t, err := time.Parse(time.RFC3339, d.ModTime)
		if err != nil {
			return fmt.Errorf("error parsing mtime for %v: %v", d, err)
		}
		a.Mtime = t
	}
	return nil
}

func (d *Dir) readDir() (out []fuse.Dirent, ferr error) {
	meta, err := clientutil.NewMetaFromBlobStore(d.fs.bs, d.Ref)
	if err != nil {
		panic(err)
	}
	if meta.Size > 0 {
		for _, hash := range meta.Refs {
			meta, err := clientutil.NewMetaFromBlobStore(d.fs.bs, hash.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to fetch meta: %v", err)
			}
			log.Printf("meta: %v", meta)
			var dirent fuse.Dirent
			if meta.Type == "file" {
				dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
				d.Children[meta.Name] = NewFile(d.fs, meta.Name, hash.(string), meta.Size, meta.ModTime, os.FileMode(meta.Mode))
			} else {
				dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
				d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, hash.(string), meta.ModTime, os.FileMode(meta.Mode), "")
			}
			out = append(out, dirent)
		}
	}
	return
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	log.Printf("OP Lookup %v", name)
	log.Printf("DEBUG %+s", d)
	if len(d.Children) == 0 {
		d.loadDir()
	}
	log.Printf("DEBUG %+s", d)
	fs, ok := d.Children[name]
	if ok {
		return fs, nil
	}
	return nil, fuse.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) (out []fuse.Dirent, err error) {
	log.Printf("OP ReadDirAll %s", d)
	return d.loadDir()
}

func (d *Dir) loadDir() (out []fuse.Dirent, err error) {
	log.Printf("OP loadDir %s", d)
	// TODO only reload when needed
	switch d.Type {
	case Root:
		d.fs.Reload()
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
		d.fs.Reload()
		for _, snap := range d.fs.SnapSets[d.Ref] {
			meta, err := snap.FetchMeta(d.fs.bs)
			if err != nil {
				panic(err)
			}
			if meta.IsFile() {
				dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
				d.Children[meta.Name] = NewFile(d.fs, meta.Name, snap.Ref, meta.Size, meta.ModTime, os.FileMode(uint32(meta.Mode)))
				out = append(out, dirent)
			} else {
				dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
				d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, snap.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
				out = append(out, dirent)
			}
		}
		return out, err
	case HostSnapshots:
		d.fs.Reload()
		for _, snap := range d.fs.SnapSets[d.Ref] {
			snapName := filepath.Base(snap.Path)
			snapHash := snap.SnapSetKey
			dirent := fuse.Dirent{Name: snapName, Type: fuse.DT_Dir}
			d.Children[snapName] = NewDir(d.fs, SnapshotsDir, snapName, snapHash, "", os.ModeDir, "")
			out = append(out, dirent)
		}
		return out, err
	case SnapshotsDir:
		versions, err := d.fs.kvs.Versions(fmt.Sprintf("blobsnap:snapset:%v", d.Ref), 0, time.Now().UTC().UnixNano(), 0)
		if err != nil {
			panic(err)
		}
		for _, e := range versions.Versions {
			snap := &snapshot.Snapshot{}
			if err := json.Unmarshal(e.Data, snap); err != nil {
				panic(err)
			}
			stime := time.Unix(snap.Time, 0)
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
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, d.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
		} else {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, d.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
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

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 2
	a.Mode = f.Mode
	a.Size = f.Size
	if f.ModTime != "" {
		t, err := time.Parse(time.RFC3339, f.ModTime)
		if err != nil {
			return fmt.Errorf("error parsing mtime for %v: %v", f, err)
		}
		a.Mtime = t
	}
	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, res *fuse.OpenResponse) (fs.Handle, error) {
	f.FakeFile = clientutil.NewFakeFile(f.fs.bs, f.Meta)
	return f, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	f.FakeFile.Close()
	f.FakeFile = nil
	return nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, res *fuse.ReadResponse) error {
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
