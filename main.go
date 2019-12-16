package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/schollz/duct/src/server"
	log "github.com/schollz/logger"
	"github.com/urfave/cli"
)

func main() {
	err := Run()
	if err != nil {
		log.Debug(err)
	}

}

// Version specifies the version
var Version string

// Run will run the command line proram
func Run() (err error) {
	// use all of the processors
	runtime.GOMAXPROCS(runtime.NumCPU())

	app := cli.NewApp()
	app.Name = "duct"
	if Version == "" {
		Version = "v1.0.0"
	}
	app.Version = Version
	app.Compiled = time.Now()
	app.Usage = "duct provides simple endpoints for managing data between applications"
	app.UsageText = ``
	app.Commands = []cli.Command{
		{
			Name:        "serve",
			Description: "start server for relaying data",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "port", Value: "9002", Usage: "port to use"},
			},
			HelpName: "duct serve",
			Action: func(c *cli.Context) error {
				setDebug(c)
				return server.Serve(c.String("port"))
			},
		},
		{
			Name:        "new",
			Description: "return a new channel",
			HelpName:    "duct new",
			Action: func(c *cli.Context) error {
				u1 := uuid.NewV4()
				fmt.Printf("https://duct.schollz.com/%s", u1)
				return nil
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug", Usage: "toggle debug mode"},
	}
	app.HideHelp = false
	app.HideVersion = false

	return app.Run(os.Args)
}

func setDebug(c *cli.Context) {
	if c.GlobalBool("debug") {
		log.SetLevel("debug")
	} else {
		log.SetLevel("info")
	}
}
