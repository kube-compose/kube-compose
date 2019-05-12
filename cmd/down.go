package cmd

import (
	"log"

	"github.com/jbrekelmans/kube-compose/pkg/down"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove containers, networks, images, and volumes running on K8s",
	Long:  "destroy all pods and services",
	Run:   downCommand,
}

func downCommand(cmd *cobra.Command, args []string) {
	cfg, err := upOrDownCommandCommon(cmd, args)
	if err != nil {
		log.Fatal(err)
	}
	err = down.Run(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

// This method is generated when cobra is initialized.
// Flags and configuration settings are meant to be
// configured here.
// nolint
func init() {
	rootCmd.AddCommand(downCmd)
}
