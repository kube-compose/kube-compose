package main

import (
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/cmd"
)

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		cmd.NewDownCommand(),
		cmd.NewUpCommand(),
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
