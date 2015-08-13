package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"

	"github.com/tsileo/blobsnap/fs"
	"github.com/tsileo/blobsnap/scheduler"
	"github.com/tsileo/blobsnap/snapshot"
)

var version = "dev"

func main() {
	app := cli.NewApp()
	commonFlags := []cli.Flag{
		cli.StringFlag{"host", "", "override the real hostname"},
		cli.StringFlag{"config", "", "config file"},
	}
	app.Name = "blobsnap"
	app.Usage = "BlobSnap command-line tool"
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:  "put",
			Usage: "Upload a file/directory",
			Flags: commonFlags,
			Action: func(c *cli.Context) {
				up, err := snapshot.NewUploader(c.String("host"))
				defer up.Close()
				if err != nil {
					log.Fatalf("failed to initialize uploader: %v", err)
				}
				meta, err := up.Put(c.Args().First())
				if err != nil {
					log.Fatalf("snapshot failed: %v", err)
				}
				fmt.Printf("%v", meta.Hash)
			},
		},
		{
			Name:  "mount",
			Usage: "Mount the read-only filesystem to the given path",
			Flags: commonFlags,
			Action: func(c *cli.Context) {
				stop := make(chan bool, 1)
				stopped := make(chan bool, 1)
				fs.Mount(c.String("host"), c.Args().First(), stop, stopped)
			},
		},
		{
			Name:      "scheduler",
			ShortName: "sched",
			Usage:     "Start the backup scheduler",
			Flags:     commonFlags,
			Action: func(c *cli.Context) {
				up, _ := snapshot.NewUploader(c.Args().First())
				defer up.Close()
				d := scheduler.New(up)
				d.Run()
			},
		},
	}
	app.Run(os.Args)
}
