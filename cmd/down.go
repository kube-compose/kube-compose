package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/jompose/pkg/down"
)

func NewDownCommand() cli.Command {
	return cli.Command{
		Name:  "down",
		Usage: "deletes pods and services",
		Action: func(c *cli.Context) error {
			cfg, err := newConfigFromEnv()
			if err != nil {
				return err
			}
			err = updateConfigFromCli(cfg, c)
			if err != nil {
				return err
			}
			return down.Run(cfg)
		},
	}
}
