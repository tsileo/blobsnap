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
	"io"
	"log"
	"os"
	"os/signal"
	"time"
	_ "fmt"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/jinzhu/now"

	"github.com/tsileo/blobstash/client"

	"github.com/tsileo/blobstash/client/clientutil"
	"github.com/tsileo/blobstash/client/ctx"
)

type DirType int

const (
	Root DirType = iota
	HostRoot
	HostLatest
	FakeDir
)

func (dt DirType) String() string {
	switch dt {
	case Root:
		return "Root"
	case FakeDir:
		return "FakeDir"
	}
	return ""
}

// Mount the filesystem to the given mountpoint
func Mount(client *client.Client, mountpoint string, stop <-chan bool, stopped chan<- bool) {
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
			stopped <-true
		}
	}()

	err = fs.Serve(c, NewFS(client))
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
	RootDir *Dir
	Client  *client.Client
}

// NewFS initialize a new file system.
func NewFS(client *client.Client) (fs *FS) {
	// Override supported time format
	now.TimeFormats = []string{"2006-1-2T15:4:5", "2006-1-2T15:4", "2006-1-2T15", "2006-1-2", "2006-1", "2006"}
	fs = &FS{Client: client}
	return
}

func (fs *FS) Root() (fs.Node, fuse.Error) {
	return NewRootDir(fs), nil
}

func NewRootDir(fs *FS) (d *Dir) {
	d = NewDir(fs, Root, "root", &ctx.Ctx{}, "", "", os.ModeDir)
	return d
}
type Node struct {
	Name string
	Mode os.FileMode
	Ref  string
	Size uint64
	ModTime string
	//Mode uint32
	fs   *FS
}

func (n *Node) Attr() fuse.Attr {
	t, _ := time.Parse(time.RFC3339, n.ModTime)
	return fuse.Attr{Mode: n.Mode, Size: n.Size, Mtime: t}
}

func (n *Node) Setattr(req *fuse.SetattrRequest, resp *fuse.SetattrResponse, intr fs.Intr) fuse.Error {
	n.Mode = req.Mode
	return nil
}

type Dir struct {
	Node
	Type DirType
	Root           bool
	RootHost       bool
	RootArchives   bool
	Latest         bool
	Snapshots      bool
	SnapshotDir    bool
	FakeDir        bool
	AtRoot         bool
	AtDir          bool
	FakeDirContent []fuse.Dirent
	Children       map[string]fs.Node
	SnapKey        string
	Ctx            *ctx.Ctx

}

func NewDir(cfs *FS, dtype DirType, name string, cctx *ctx.Ctx, ref string, modTime string, mode os.FileMode) (d *Dir) {
	d = &Dir{}
	d.Type = dtype
	d.Ctx = cctx
	d.Node = Node{}
	d.Mode = os.ModeDir
	d.fs = cfs
	d.Ref = ref
	d.Name = name
	d.ModTime = modTime
	d.Mode = os.FileMode(mode)
	d.Children = make(map[string]fs.Node)
	return
}

func (d *Dir) readDir() (out []fuse.Dirent, ferr fuse.Error) {
	con := d.fs.Client.ConnWithCtx(d.Ctx)
	defer con.Close()
	//for _, meta := range d.fs.Client.Dirs.Get(con, d.Ref).() {
	for _, meta := range []*clientutil.Meta{} {
		var dirent fuse.Dirent
		if meta.Type == "file" {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, d.Ctx, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
		} else {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, FakeDir, meta.Name, d.Ctx, meta.Ref, meta.ModTime, os.FileMode(meta.Mode))
		}
		out = append(out, dirent)
	}
	return
}

func NewFakeDir(cfs *FS, name string, cctx *ctx.Ctx, ref string) (d *Dir) {
	d = NewDir(cfs, FakeDir, name, cctx, ref, "", os.ModeDir)
	d.Children = make(map[string]fs.Node)
	d.FakeDir = true
	return
}

func (d *Dir) Lookup(name string, intr fs.Intr) (fs fs.Node, err fuse.Error) {
	var ok bool
	fs, ok = d.Children[name]
	if ok {
		return
	}
	return
}

func (d *Dir) ReadDir(intr fs.Intr) (out []fuse.Dirent, err fuse.Error) {
	switch d.Type{
	case Root:
		d.Children = make(map[string]fs.Node)
		con := d.fs.Client.ConnWithCtx(d.Ctx)
		defer con.Close()
		hosts, err := d.fs.Client.Smembers(con, "blobsnap:hostnames")
		if err != nil {
			panic("failed to fetch hosts")
		}
		for _, host := range hosts {
			dirent := fuse.Dirent{Name: host, Type: fuse.DT_Dir}
			out = append(out, dirent)
			d.Children[host] = NewDir(d.fs, HostRoot, host, d.Ctx, "", "", os.ModeDir)
		}
		return out, err
	case HostRoot:
		d.Children = make(map[string]fs.Node)
		dirent := fuse.Dirent{Name: "latest", Type: fuse.DT_Dir}
		out = append(out, dirent)
		d.Children["latest"] = NewDir(d.fs, HostLatest, "latest", d.Ctx, "", "", os.ModeDir)
		return out, err
	case Latest:
		// TODO a Lua script to gather the LLAST of SMEMBERS
		return out, err
	case FakeDir:
		d.Children = make(map[string]fs.Node)
		//meta, _ := client.NewMetaFromDB(d.fs.Client.Pool, d.Ref)
		log.Printf("FakeDir/ctx:%v", d.Ctx)
		con := d.fs.Client.ConnWithCtx(d.Ctx)
		defer con.Close()
		//meta := d.fs.Client.Metas.Get(con, d.Ref).(*client.Meta)
		meta := clientutil.NewMeta()
		if meta.Type == "file" {
			dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, d.Ctx, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
			out = append(out, dirent)
		} else {
			dirent := fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, FakeDir, meta.Name, d.Ctx, meta.Ref, meta.ModTime, os.FileMode(meta.Mode))
			out = append(out, dirent)
		}
		return out, nil
	}
	return d.readDir()
}

type File struct {
	Node
	Ctx *ctx.Ctx
	FakeFile *clientutil.FakeFile
}

func NewFile(fs *FS, name string, cctx *ctx.Ctx, ref string, size int, modTime string, mode os.FileMode) *File {
	f := &File{}
	f.Ctx = cctx
	f.Name = name
	f.Ref = ref
	f.Size = uint64(size)
	f.ModTime = modTime
	f.Mode = mode
	f.fs = fs
	return f
}

// TODO(tsileo) handle release request and close FakeFile if needed?

func (f *File) Attr() fuse.Attr {
	return fuse.Attr{Inode: 2, Mode: 0444, Size: f.Size}
}
func (f *File) Open(req *fuse.OpenRequest, res *fuse.OpenResponse, intr fs.Intr) (fs.Handle, fuse.Error) {
	f.FakeFile = clientutil.NewFakeFile(f.fs.Client, f.Ctx, f.Ref, int(f.Size))
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
