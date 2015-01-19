package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dchest/blake2b"
	"github.com/tsileo/blobsnap/clientutil"
	"github.com/tsileo/blobstash/client"
	"github.com/tsileo/blobstash/client2"
)

type Uploader struct {
	bs       *client2.BlobStore
	kvs      *client2.KvStore
	Uploader *clientutil.Uploader
}

func NewUploader(serverAddr string) (*Uploader, error) {
	cl, err := client.New(serverAddr)
	if err != nil {
		return nil, err
	}
	bs := client2.NewBlobStore(serverAddr)
	bs.ProcessBlobs()
	kvs := client2.NewKvStore(serverAddr)
	return &Uploader{
		bs:       bs,
		kvs:      kvs,
		Uploader: clientutil.NewUploader(cl),
	}, nil
}

func (up *Uploader) Close() error {
	bs.WaitBlobs()
	return nil
}

func (s *Snapshot) SnapSetKey() string {
	hash := blake2b.New256()
	hash.Write([]byte(s.Path))
	hash.Write([]byte(s.Hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

type Snaphot struct {
	Path       string `json:"path"`
	Hostname   string `json:"hostname"`
	Ref        string `json:"ref"`
	Time       int    `json:"time"`
	SnapSetKey string `json:"key"`
}

func (up *Uploader) Put(path string) (*clientutil.Meta, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, err
	}
	var meta *clientutil.Meta
	var wr *clientutil.WriteResult
	//var wr *clientutil.WriteResult
	if info.IsDir() {
		meta, wr, err = up.Uploader.PutDir(path)
	} else {
		meta, wr, err = up.Uploader.PutFile(path)
	}
	if err != nil {
		return meta, err
	}
	setKey := SetKey(path, up.Client.Hostname)
	if wr.SizeUploaded == 0 {
		log.Println("Nothing has been uploaded, no snapshot will be created.")
		return meta, nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	t := time.Now().UTC()
	snap := &Snapshot{
		Path:     path,
		Hostname: hostname,
		Ref:      meta.Hash,
		Time:     int(t.Unix()),
	}
	snap.SnapSetKey = snap.SnapSetKey()
	snapjs, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	_, err := kvs.Put(fmt.Sprintf("blobsnap:snapset:%v", snap.SnapSetKey), string(snapjs), int(t.UnixNano()))
	if err != nil {
		return nil, err
	}
	return meta, nil
}
