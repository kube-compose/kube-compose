package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/up"
)

func NewUpCommand() cli.Command {
	return cli.Command{
		Name:  "up",
		Usage: "Create and start containers",
		Action: func(c *cli.Context) error {
			cfg, err := NewConfig()
			if err != nil {
				return err
			}
			return up.Run(cfg)
		},
	}
}
