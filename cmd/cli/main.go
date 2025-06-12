package main

import (
	"os"

	"github.com/jaywantadh/DisktroByte/pkg/env"
	"github.com/jaywantadh/DisktroByte/pkg/logging"
	"github.com/urfave/cli/v2"
)

func main(){

	env.LoadEnv()
	logging.InitLogger(true)

		app := &cli.App{
		Name:  "DisktroByte",
		Usage: "A P2P distributed file system",
		Commands: []*cli.Command{
			{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start the DisktroByte node",
				Action: func(c *cli.Context) error {
					logging.Log.Info("🚀 DisktroByte node started")
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "Check node status",
				Action: func(c *cli.Context) error {
					logging.Log.Info("✅ Node is healthy")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil{
			logging.Log.Fatal(err)
	}

	port := env.GetEnv("PORT", "8000")
	logging.Log.Infof("Node will run on port: %s", port)
}