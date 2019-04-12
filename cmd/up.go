package cmd

import (
	"github.com/urfave/cli"

	"github.com/jbrekelmans/kube-compose/pkg/up"
)

func NewUpCommand() cli.Command {
	return cli.Command{
		Name:  "up",
		Usage: "creates pods and services in an order that respects depends_on in the docker compose file",
		Action: func(c *cli.Context) error {
			cfg, err := newConfigFromEnv()
			if err != nil {
				return err
			}
			err = updateConfigFromCli(cfg, c)
			if err != nil {
				return err
			}
			return up.Run(cfg)
		},
	}
}
