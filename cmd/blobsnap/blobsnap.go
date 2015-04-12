package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/codegangsta/cli"

	"github.com/tsileo/blobsnap/fs"
	"github.com/tsileo/blobsnap/scheduler"
	"github.com/tsileo/blobsnap/snapshot"
)

var nCPU = runtime.NumCPU()

func init() {
	runtime.GOMAXPROCS(nCPU)
}

//func loadConf(config_path string) *simplejson.Json {
//	if config_path == "" {
//		config_path = filepath.Join(pathutil.ConfigDir(), "client-config.json")
//	}
//	dat, err := ioutil.ReadFile(config_path)
//	if err != nil {
//		panic(fmt.Errorf("failed to read config file: %v", err))
//	}
//	conf, err := simplejson.NewJson(dat)
//	if err != nil {
//		panic(fmt.Errorf("failed decode config file (invalid json): %v", err))
//	}
//	return conf
//}

func main() {
	app := cli.NewApp()
	commonFlags := []cli.Flag{
		cli.StringFlag{"host", "", "override the real hostname"},
		cli.StringFlag{"config", "", "config file"},
	}

	//ignoredFiles, _ := conf.Get("ignored-files").StringArray()

	app.Name = "blobsnap"
	app.Usage = "BlobSnap command-line tool"
	app.Version = "0.1.0"
	//  app.Action = func(c *cli.Context) {
	//    println("Hello friend!")
	//  }
	app.Commands = []cli.Command{
		{
			Name:  "put",
			Usage: "put a file/directory",
			Flags: commonFlags,
			Action: func(c *cli.Context) {
				//defaultHost := loadConf(c.String("config")).Get("server").MustString("localhost:9735")
				up, _ := snapshot.NewUploader("")
				//if c.String("host") != "" {
				// Override the hostname if needed
				//	up.Client.Hostname = c.String("host")
				//}
				//cl.SetIgnoredFiles(ignoredFiles)
				//defer up.Close()
				meta, err := up.Put(c.Args().First())
				if err != nil {
					fmt.Printf("snapshot failed: %v", err)
				}
				fmt.Printf("%v", meta.Hash)
				//b, m, wr, err := cl.Put(&client.Ctx{Namespace: cl.Hostname, Archive: c.Bool("archive")}, c.Args().First())
				//fmt.Printf("b:%+v,m:%+v,wr:%+v,err:%v\n", b, m, wr, err)
			},
		},
		{
			Name:  "mount",
			Usage: "Mount the read-only filesystem to the given path",
			Action: func(c *cli.Context) {
				//defaultHost := loadConf(c.String("config")).Get("server").MustString("localhost:9735")
				stop := make(chan bool, 1)
				stopped := make(chan bool, 1)
				fs.Mount("http://192.168.1.10:8050", c.Args().First(), stop, stopped)
			},
		},
		{
			Name:      "scheduler",
			ShortName: "sched",
			Usage:     "Start the backup scheduler",
			Flags:     commonFlags,
			Action: func(c *cli.Context) {
				//defaultHost := loadConf(c.String("config")).Get("server").MustString("localhost:9735")
				up, _ := snapshot.NewUploader("")
				//if c.String("host") != "" {
				// Override the hostname if needed
				//	up.Client.Hostname = c.String("host")
				//	}
				//cl.SetIgnoredFiles(ignoredFiles)
				defer up.Close()
				d := scheduler.New(up)
				d.Run()
			},
		},
	}
	app.Run(os.Args)
}
