package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/kube-compose/kube-compose/internal/app/up"
	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/spf13/cobra"
)

const (
	registryUserEnvVarName = envVarPrefix + "REGISTRY_USER"
	registryUserFlagName   = "registry-user"
	registryPassEnvVarName = envVarPrefix + "REGISTRY_PASS"
	registryPassFlagName   = "registry-pass"
)

var registryUserFromEnv = util.Ternary(os.Getenv(registryUserEnvVarName), "unused")
var registryPassFromEnv = os.Getenv(registryPassEnvVarName)

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
	upCmd.PersistentFlags().StringP(registryUserFlagName, "", registryUserFromEnv,
		fmt.Sprintf("The docker registry user to authenticate as. Can also be set via environment variable %s. The default is common for Openshift clusters.", registryUserEnvVarName))
	upCmd.PersistentFlags().StringP(registryPassFlagName, "", registryPassFromEnv,
		fmt.Sprintf("The docker registry password to authenticate with. Can also be set via environment variable %s. When unset, will use the Bearer Token from Kube config as is common for Openshift clusters.", registryPassEnvVarName))
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
		log.StandardLogger().SetOutput(opts.Reporter.LogSink())
		go func() {
			for {
				opts.Reporter.Refresh()
				time.Sleep(reporter.RefreshInterval)
			}
		}()
	}

	opts.RegistryUser, _ = cmd.Flags().GetString(registryUserFlagName)
	opts.RegistryPass, _ = cmd.Flags().GetString(registryPassFlagName)

	err = up.Run(cfg, opts)
	if err != nil {
		log.Error(err)
		opts.Reporter.Refresh()
		os.Exit(1)
	}
	opts.Reporter.Refresh()
	return nil
}
