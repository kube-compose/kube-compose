package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kube-compose/kube-compose/internal/app/up"
	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
	"github.com/spf13/cobra"
)

func newUpCli() *cobra.Command {
	var upCmd = &cobra.Command{
		Use:   "up",
		Short: "Create and start containers running on K8s",
		Long:  "creates pods and services in an order that respects depends_on in the docker compose file",
		RunE:  upCommand,
	}
	upCmd.PersistentFlags().BoolP("detach", "d", false, "Detached mode: Run containers in the background")
	upCmd.PersistentFlags().BoolP("run-as-user", "", false, "When set, the runAsUser/runAsGroup will be set for each pod based on the "+
		"user of the pod's image and the \"user\" key of the pod's docker-compose service")
	return upCmd
}

func upCommand(cmd *cobra.Command, args []string) error {
	cfg, err := getCommandConfig(cmd, args)
	if err != nil {
		return err
	}
	opts := &up.Options{}
	opts.Context = context.Background()
	opts.Detach, _ = cmd.Flags().GetBool("detach")
	opts.Reporter = reporter.New(os.Stdout)
	opts.RunAsUser, _ = cmd.Flags().GetBool("run-as-user")
	go func() {
		for {
			opts.Reporter.Refresh()
			time.Sleep(reporter.RefreshInterval)
		}
	}()
	err = up.Run(cfg, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}
