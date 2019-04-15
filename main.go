package main

import (
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/jbrekelmans/kube-compose/cmd"
)

func main() {
	var err error

	app := cli.NewApp()
	app.Flags = cmd.GlobalFlags()
	app.Version = "3.0.2"
	app.Commands = []cli.Command{
		cmd.NewDownCommand(),
		cmd.NewUpCommand(),
	}
	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
