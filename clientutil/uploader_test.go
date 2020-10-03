package clientutil

import (
	"os"
	"testing"
	"time"

	"github.com/antonovvk/blobsnap/store"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func TestUploader(t *testing.T) {
	bs := store.NewBlobStore("")
	// setup the blobstore pipeline for async upload
	bs.ProcessBlobs()
	kvs := store.NewKvStore("")
	up := NewUploader(bs, kvs)

	t.Logf("Testing with a random file...")

	fname := test.NewRandomFile(".")
	defer os.Remove(fname)
	meta, wr, err := up.PutFile(fname)
	check(err)

	time.Sleep(2 * time.Second)

	rr, err := GetFile(bs, meta.Hash, fname+"restored")
	defer os.Remove(fname + "restored")
	check(err)
	t.Logf("%v %v %v %v", up, meta, wr, rr)

	t.Logf("Testing with a random directory tree")
	path, _ := test.CreateRandomTree(t, ".", 0, 1)
	defer os.RemoveAll(path)
	meta, wr, err = up.PutDir(path)
	check(err)
	t.Logf("%v %v %v %v", up, meta, wr, rr)

	time.Sleep(4 * time.Second)
	rr, err = GetDir(bs, meta.Hash, path+"restored")
	defer os.RemoveAll(path + "restored")
	check(err)
}
