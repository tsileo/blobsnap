package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/blake2b"
	"github.com/tsileo/blobsnap/clientutil"
	"github.com/tsileo/blobstash/client"
)

type Uploader struct {
	bs       *client.BlobStore
	kvs      *client.KvStore
	Uploader *clientutil.Uploader
}

func NewUploader(serverAddr string) (*Uploader, error) {
	bs := client.NewBlobStore(serverAddr)
	bs.ProcessBlobs()
	kvs := client.NewKvStore(serverAddr)
	return &Uploader{
		bs:       bs,
		kvs:      kvs,
		Uploader: clientutil.NewUploader(bs, kvs),
	}, nil
}

func (up *Uploader) Close() error {
	up.bs.WaitBlobs()
	return nil
}

type Snapshot struct {
	Path        string                  `json:"path"`
	Hostname    string                  `json:"hostname"`
	Ref         string                  `json:"ref"`
	Time        int                     `json:"time"`
	SnapSetKey  string                  `json:"key"`
	Comment     string                  `json:"comment,omitempty"`
	WriteResult *clientutil.WriteResult `json:"wr"`
}

func (s *Snapshot) ComputeSnapSetKey() string {
	hash := blake2b.New256()
	hash.Write([]byte(s.Path))
	hash.Write([]byte(s.Hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (s *Snapshot) FetchMeta(bs *client.BlobStore) (*clientutil.Meta, error) {
	blob, err := bs.Get(s.Ref)
	if err != nil {
		return nil, err
	}
	m := clientutil.NewMeta()
	if err := json.Unmarshal(blob, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (up *Uploader) Put(path string) (*clientutil.Meta, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, err
	}
	var meta *clientutil.Meta
	var wr *clientutil.WriteResult
	if info.IsDir() {
		meta, wr, err = up.Uploader.PutDir(path)
	} else {
		meta, wr, err = up.Uploader.PutFile(path)
	}
	if err != nil {
		return meta, err
	}
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
		Path:        filepath.Clean(path),
		Hostname:    hostname,
		Ref:         meta.Hash,
		Time:        int(t.Unix()),
		WriteResult: wr,
	}
	snap.SnapSetKey = snap.ComputeSnapSetKey()
	snapjs, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	_, err = up.kvs.Put(fmt.Sprintf("blobsnap:snapset:%v", snap.SnapSetKey), string(snapjs), int(t.UnixNano()))
	if err != nil {
		return nil, err
	}
	return meta, nil
}
