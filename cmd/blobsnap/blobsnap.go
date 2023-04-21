package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/antonovvk/blobsnap/fs"
	//~ "github.com/antonovvk/blobsnap/scheduler"
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

	fmt.Println(flag.Args())

	switch flag.Arg(0) {
	case "put":
		up, err := snapshot.NewUploader(blobStore, kvStore)
		defer up.Close()
		if err != nil {
			log.Fatalf("failed to initialize uploader: %v", err)
		}
		fmt.Println("PUT")
		meta, err := up.Put(flag.Arg(1))
		if err != nil {
			log.Fatalf("snapshot failed: %v", err)
		}
		fmt.Printf("META %v\n", meta.Hash)
	case "mount":
		stop := make(chan bool, 1)
		stopped := make(chan bool, 1)
		fs.Mount(blobStore, kvStore, flag.Arg(1), stop, stopped)
	case "dump_kv":
		res, err := kvStore.Dump()
		if err != nil {
			log.Fatalf("kv_entries failed: %v", err)
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	//~ case "scheduler":
		//~ up, _ := snapshot.NewUploader(blobStore, kvStore)
		//~ defer up.Close()
		//~ d := scheduler.New(up)
		//~ d.Run()
	}
}
