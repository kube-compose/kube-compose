package main

import (
	"fmt"
	"log"

	"github.com/urfave/cli"

	"github.com/jabrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jabrekelmans/k8s-docker-compose/pkg/up"
)

func main () {

	config := config.Config{
		A: "hello"
	}

	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name: "up",
			Usage: "Create and start containers",
			Action: func (c *cli.Context) error {
				cfg := config.New()
				return up.Run(cfg)
			},
		},
	}

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}