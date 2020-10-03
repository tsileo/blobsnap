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
	fakeBs  = store.FakeBlobStore{}
	fakeKvs = store.FakeKvStore{}
	version = "dev"

	//~ host   = flag.String("host", "", "override the real hostname")
	//~ config = flag.String("config", "", "config file")
)

func main() {
	flag.Parse()

	switch flag.Arg(1) {
	case "put":
		up, err := snapshot.NewUploader(fakeBs, fakeKvs)
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
		fs.Mount(fakeBs, fakeKvs, flag.Arg(2), stop, stopped)
	case "scheduler":
		up, _ := snapshot.NewUploader(fakeBs, fakeKvs)
		defer up.Close()
		d := scheduler.New(up)
		d.Run()
	}
}
