package cmd

import (
	"log"

	"github.com/urfave/cli"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/up"
)

func NewCommand() cli.Command {
	return cli.Command{
		Name: "up",
		Usage: "Create and start containers",
		Action: func (c *cli.Context) error {
			log.Println("Loading config...")
			cfg, err := config.New()
			if err != nil {
				return err
			}
			log.Println("Done loading config...")
			return up.Run(cfg)
		},
	}
}
