package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/down"
)

func NewDownCommand() cli.Command {
	return cli.Command{
		Name:  "down",
		Usage: "Stop and delete containers",
		Action: func(c *cli.Context) error {
			cfg, err := NewConfig()
			if err != nil {
				return err
			}
			return down.Run(cfg)
		},
	}
}
