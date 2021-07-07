package cmd

import (
	"fmt"
	"os"

	log "github.com/github.com/sirupsen/logrus/logrus"
	"github.com/spf13/cobra"
)

const (
	envVarPrefix        = "KUBECOMPOSE_"
	fileFlagName        = "file"
	namespaceEnvVarName = envVarPrefix + "NAMESPACE"
	namespaceFlagName   = "namespace"
	envIDEnvVarName     = envVarPrefix + "ENVID"
	envIDFlagName       = "env-id"
)

func Execute() error {
	log.SetOutput(os.Stdout)
	rootCmd := &cobra.Command{
		Use:               "kube-compose",
		Short:             "k8s",
		Long:              "Environments on k8s made easy",
		Version:           "0.6.1",
		PersistentPreRunE: setupLogging,
	}
	rootCmd.AddCommand(newDownCli(), newUpCli(), newGetCli())
	setRootCommandFlags(rootCmd)
	return rootCmd.Execute()
}

func setRootCommandFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().StringSliceP(fileFlagName, "f", []string{}, "Specify an alternate compose file")
	rootCmd.PersistentFlags().StringP(namespaceFlagName, "n", "", fmt.Sprintf("namespace for environment. Can also be set via "+
		"environment variable %s. Default to the namespace of the current kube config context", namespaceEnvVarName))
	rootCmd.PersistentFlags().StringP(envIDFlagName, "e", "", "used to isolate environments deployed to a shared namespace, "+
		"by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors. Either this flag or "+
		fmt.Sprintf("the environment variable %s must be set", envIDEnvVarName))
	rootCmd.PersistentFlags().StringP(logLevelFlagName, "l", "", fmt.Sprintf("Set to one of %s. Can also be set via environment variable "+
		"%s. Defaults to %s", formattedLogLevelList, logLevelEnvVarName, logLevelDefault.String()))
}
