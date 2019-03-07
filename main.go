package main

import (
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/jbrekelmans/jompose/cmd"
)

func main() {
	app := cli.NewApp()
	app.Flags = cmd.GlobalFlags()
	app.Commands = []cli.Command{
		cmd.NewDownCommand(),
		cmd.NewUpCommand(),
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
