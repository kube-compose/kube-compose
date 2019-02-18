package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/up"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name: "up",
		Usage: "Create and start containers",
		Action: func (c *cli.Context) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}
			return up.Run(cfg)
		},
	}
}
