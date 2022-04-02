package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/antonovvk/blobsnap/fs"
	"github.com/antonovvk/blobsnap/scheduler"
	"github.com/antonovvk/blobsnap/snapshot"
	"github.com/antonovvk/blobsnap/store"
)

var (
	version = "dev"

	//~ host   = flag.String("host", "", "override the real hostname")
	//~ config = flag.String("config", "", "config file")
	localBs = flag.String("local_bs", "", "Use local blob store")
	localKv = flag.String("local_kv", "", "Use local KV store")
)

func main() {
	flag.Parse()

	var blobStore store.BlobStore
	blobStore = store.FakeBlobStore{}
	if *localBs != "" {
		blobStore = store.NewFileBlobStore(*localBs)
	}

	var kvStore store.KvStore
	kvStore = store.FakeKvStore{}
	if *localKv != "" {
		var err error
		if kvStore, err = store.NewBoltKbStore(*localKv); err != nil {
			log.Fatalf("failed to initialize BoltDB KV store: %v", err)
		}
	}

	switch flag.Arg(1) {
	case "put":
		up, err := snapshot.NewUploader(blobStore, kvStore)
		defer up.Close()
		if err != nil {
			log.Fatalf("failed to initialize uploader: %v", err)
		}
		meta, err := up.Put(flag.Arg(2))
		if err != nil {
			log.Fatalf("snapshot failed: %v", err)
		}
		fmt.Printf("%v", meta.Hash)
	case "mount":
		stop := make(chan bool, 1)
		stopped := make(chan bool, 1)
		fs.Mount(blobStore, kvStore, flag.Arg(2), stop, stopped)
	case "scheduler":
		up, _ := snapshot.NewUploader(blobStore, kvStore)
		defer up.Close()
		d := scheduler.New(up)
		d.Run()
	}
}
