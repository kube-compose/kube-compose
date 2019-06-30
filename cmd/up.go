package cmd

import (
	"context"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
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
	opts.RunAsUser, _ = cmd.Flags().GetBool("run-as-user")

	opts.Reporter = reporter.New(os.Stdout)
	if opts.Reporter.IsTerminal() {
		log.StandardLogger().SetFormatter(&log.TextFormatter{
			ForceColors:               true,
			EnvironmentOverrideColors: true,
		})
		log.StandardLogger().SetOutput(opts.Reporter.LogWriter())
		go func() {
			for {
				opts.Reporter.Refresh()
				time.Sleep(reporter.RefreshInterval)
			}
		}()
	}

	err = up.Run(cfg, opts)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	return nil
}
