package snapshot

import (
	"os"
	"fmt"

	"github.com/tsileo/blobstash/client"
	"github.com/tsileo/blobstash/client/ctx"
	"github.com/tsileo/blobstash/client/clientutil"
)

type Uploader struct {
	Client *client.Client
	Uploader *clientutil.Uploader
	Ctx *ctx.Ctx
}

func NewUploader(serverAddr string) (*Uploader, error) {
	cl, err := client.New(serverAddr)
	if err != nil {
		return nil, err
	}
	return &Uploader{
		Client: cl,
		Uploader: clientutil.NewUploader(cl),
		Ctx: &ctx.Ctx{Namespace: "blobsnap"},
	}, nil
}

func (up *Uploader) Close() error {
	return up.Client.Close()
}

func (up *Uploader) Put(path string) (error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return err
	}
	var meta *clientutil.Meta
	//var wr *clientutil.WriteResult
	if info.IsDir() {
		meta, _, err = up.Uploader.PutDir(up.Ctx, path)
	} else {
		meta, _, err = up.Uploader.PutFile(up.Ctx, nil, path)
	}
	if err != nil {
		return err
	}
	tx := client.NewTransaction()
	snap := NewSnapshot(path, up.Client.Hostname, meta.Hash)
	snap.ComputeHash()
	snapSet := &SnapSet{
		Path: path,
		Hostname: up.Client.Hostname,
		Hash: snap.SetKey(),
	}
	tx.Hmset(fmt.Sprintf("blobsnap:snapshot:%v", snap.Hash), client.FormatStruct(snap)...)
	tx.Hmset(fmt.Sprintf("blobsnap:snapset:%v", snap.SetKey()), client.FormatStruct(snapSet)...)
	tx.Sadd("blobsnap:hostnames", up.Client.Hostname)
	tx.Sadd(fmt.Sprintf("blobsnap:host:%v", up.Client.Hostname), snap.SetKey())
	tx.Ladd(fmt.Sprintf("blobsnap:snapset:%v:history", snap.SetKey()), int(snap.Time), snap.Hash)
	if up.Client.Commit(up.Ctx, tx); err != nil {
		return err
	}
	return nil
}
