package snapshot

import (
	"os"
	"fmt"
	"time"

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

func (up *Uploader) Put(path string) (*clientutil.Meta, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, err
	}
	var meta *clientutil.Meta
	tx := client.NewTransaction()
	//var wr *clientutil.WriteResult
	if info.IsDir() {
		meta, _, err = up.Uploader.PutDir(up.Ctx, path)
	} else {
		// If we upload a single, bundle the meta in a single Transaction
		meta, _, err = up.Uploader.PutFile(up.Ctx, tx, path)
	}
	if err != nil {
		return meta, err
	}
	setKey := SetKey(path, up.Client.Hostname)
	snapSet := &SnapSet{
		Path: path,
		Hostname: up.Client.Hostname,
		Hash: setKey,
	}
	//tx.Hmset(fmt.Sprintf("blobsnap:snapshot:%v", snap.Hash), client.FormatStruct(snap)...)
	tx.Hmset(fmt.Sprintf("blobsnap:snapset:%v", setKey), client.FormatStruct(snapSet)...)
	tx.Sadd("blobsnap:hostnames", up.Client.Hostname)
	tx.Sadd(fmt.Sprintf("blobsnap:host:%v", up.Client.Hostname), setKey)
	tx.Ladd(fmt.Sprintf("blobsnap:snapset:%v:history", setKey), int(time.Now().UTC().Unix()), meta.Hash)
	if up.Client.Commit(up.Ctx, tx); err != nil {
		return meta, err
	}
	return meta, nil
}
