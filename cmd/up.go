package cmd

import (
	"context"
	"log"

	"github.com/jbrekelmans/kube-compose/pkg/up"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create and start containers running on K8s",
	Long:  "creates pods and services in an order that respects depends_on in the docker compose file",
	Run:   upCommand,
}

func upCommand(cmd *cobra.Command, args []string) {
	cfg, err := upOrDownCommandCommon(cmd, args)
	if err != nil {
		log.Fatal(err)
	}
	opts := &up.Options{}
	opts.Detach, _ = cmd.Flags().GetBool("detach")
	opts.RunAsUser, _ = cmd.Flags().GetBool("run-as-user")
	err = up.Run(context.Background(), cfg, opts)
	if err != nil {
		log.Fatal(err)
	}
}

// This method is generated when cobra is initialized.
// Flags and configuration settings are meant to be
// configured here.
// nolint
func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.PersistentFlags().BoolP("detach", "d", false, "Detached mode: Run containers in the background")
	upCmd.PersistentFlags().BoolP("run-as-user", "", false, "When set, the runAsUser/runAsGroup will be set for each pod based on the "+
		"user of the pod's image and the \"user\" key of the pod's docker-compose service")
}
