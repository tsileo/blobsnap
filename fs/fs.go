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
	"path/filepath"
	"os"
	"os/signal"
	"time"
	_ "fmt"
	"strconv"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/jinzhu/now"

	"github.com/tsileo/blobsnap/snapshot"
	"github.com/tsileo/blobstash/client"

	"github.com/tsileo/blobstash/client/clientutil"
	"github.com/tsileo/blobstash/client/ctx"
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
	d = NewDir(fs, Root, "root", &ctx.Ctx{}, "", "", os.ModeDir, "")
	return d
}
type Node struct {
	Name string
	Mode os.FileMode
	Ref  string
	Size uint64
	ModTime string
	Extra string
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
	Children       map[string]fs.Node
	Ctx            *ctx.Ctx
}

func NewDir(cfs *FS, dtype DirType, name string, cctx *ctx.Ctx, ref string, modTime string, mode os.FileMode, extra string) (d *Dir) {
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
	d.Extra = extra
	return
}

func (d *Dir) readDir() (out []fuse.Dirent, ferr fuse.Error) {
	con := d.fs.Client.ConnWithCtx(d.Ctx)
	defer con.Close()
	metaHashes, err := d.fs.Client.Smembers(con, d.Ref)
	if err != nil {
		panic(err)
	}
	for _, hash := range metaHashes {
		meta := clientutil.NewMeta()
		if err := d.fs.Client.HscanStruct(con, hash, meta); err != nil {
			panic(err)
		}
		var dirent fuse.Dirent
		if meta.Type == "file" {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, d.Ctx, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
		} else {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, d.Ctx, meta.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
		}
		out = append(out, dirent)
	}
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
	log.Printf("ReadDir %v", d)
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
			out = append(out, fuse.Dirent{Name: host, Type: fuse.DT_Dir})
			d.Children[host] = NewDir(d.fs, HostRoot, host, d.Ctx, host, "", os.ModeDir, "")
		}
		return out, err
	case HostRoot:
		d.Children = make(map[string]fs.Node)
		out = append(out, fuse.Dirent{Name: "latest", Type: fuse.DT_Dir})
		d.Children["latest"] = NewDir(d.fs, HostLatest, "latest", d.Ctx, d.Ref, "", os.ModeDir, "")
		out = append(out, fuse.Dirent{Name: "snapshots", Type: fuse.DT_Dir})
		d.Children["snapshots"] = NewDir(d.fs, HostSnapshots, "snapshots", d.Ctx, d.Ref, "", os.ModeDir, "")
		return out, err
	case HostLatest:
		log.Printf("HostLatest")
		snapshots, herr := snapshot.HostLatest(d.Ref)
		if herr != nil {
			panic(herr)
		}
		// TODO make snapshot.HostLatest return a []*clientutil.Meta ?
		for _, data := range snapshots["snapshots"].([]interface{}) {
			meta := data.(map[string]interface{})
			//metaHash := meta["hash"].(string)
			metaMode, _ := strconv.Atoi(meta["mode"].(string))
			metaName := meta["name"].(string)
			metaRef := meta["ref"].(string)
			metaMtime := meta["mtime"].(string)
			if meta["type"] == "file" {
				dirent := fuse.Dirent{Name: metaName, Type: fuse.DT_File}
				metaSize, _ := strconv.Atoi(meta["size"].(string))
				d.Children[metaName] = NewFile(d.fs, metaName, d.Ctx, metaRef, metaSize, metaMtime, os.FileMode(uint32(metaMode)))
				out = append(out, dirent)
			} else {
				dirent := fuse.Dirent{Name: metaName, Type: fuse.DT_Dir}
				d.Children[metaName] = NewDir(d.fs, BasicDir, metaName, d.Ctx, metaRef, metaMtime, os.FileMode(metaMode), "")
				out = append(out, dirent)
			}
		}
		return out, err
	case HostSnapshots:
		snapshots, herr := snapshot.HostSnapSet(d.Ref)
		if herr != nil {
			panic(err)
		}
		for _, data := range snapshots["snapshots"].([]interface{}) {
			snap := data.(map[string]interface{})
			snapName := filepath.Base(snap["path"].(string))
			snapHash := snap["hash"].(string)
			dirent := fuse.Dirent{Name: snapName, Type: fuse.DT_Dir}
			d.Children[snapName] = NewDir(d.fs, SnapshotsDir, snapName, d.Ctx, snapHash, "", os.ModeDir, "")
			out = append(out, dirent)
		}
		return out, err
	case SnapshotsDir:
		snapshots, herr := snapshot.Snapshots(d.Ref)
		if herr != nil {
			panic(err)
		}
		for _, data := range snapshots["snapshots"].([]interface{}) {
			iv := data.(map[string]interface{})
			stime := time.Unix(int64(iv["index"].(float64)), 0)
			sname := stime.Format(time.RFC3339)
			dirent := fuse.Dirent{Name: sname, Type: fuse.DT_Dir}
			d.Children[sname] = NewDir(d.fs, SnapshotDir, sname, d.Ctx, iv["value"].(string), "", os.ModeDir, d.Name)
			out = append(out, dirent)
		}
		return out, err
	case SnapshotDir:
		con := d.fs.Client.ConnWithCtx(d.Ctx)
		defer con.Close()
		meta := clientutil.NewMeta()
		if err := d.fs.Client.HscanStruct(con, d.Ref, meta); err != nil {
			panic(err)
		}
		var dirent fuse.Dirent
		if meta.Type == "file" {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_File}
			d.Children[meta.Name] = NewFile(d.fs, meta.Name, d.Ctx, meta.Ref, meta.Size, meta.ModTime, os.FileMode(meta.Mode))
		} else {
			dirent = fuse.Dirent{Name: meta.Name, Type: fuse.DT_Dir}
			d.Children[meta.Name] = NewDir(d.fs, BasicDir, meta.Name, d.Ctx, meta.Ref, meta.ModTime, os.FileMode(meta.Mode), "")
		}
		out = append(out, dirent)
		return out, err
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
