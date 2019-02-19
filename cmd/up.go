package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/up"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name:  "up",
		Usage: "Create and start containers",
		Action: func(c *cli.Context) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}
			cfg.EnvironmentID = "test123"
			cfg.EnvironmentLabel = "environment-instance"
			cfg.Namespace = "default"
			return up.Run(cfg)
		},
	}
}
