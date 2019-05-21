package cmd

import (
	"github.com/jbrekelmans/kube-compose/pkg/down"
	"github.com/spf13/cobra"
)

func newDownCli() *cobra.Command {
	var downCmd = &cobra.Command{
		Use: "down",
		Short: "Deletes the pods of the specified docker compose services. " +
			"If all docker compose services would be deleted then the Kubernetes services are also deleted.",
		Long: "destroy all pods and services",
		RunE: downCommand,
	}
	return downCmd
}

func downCommand(cmd *cobra.Command, args []string) error {
	cfg, err := getCommandConfig(cmd, args)
	if err != nil {
		return err
	}
	if err := down.Run(cfg); err != nil {
		return err
	}
	return nil
}
