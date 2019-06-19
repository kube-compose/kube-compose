package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	fileFlagName      = "file"
	namespaceFlagName = "namespace"
	envIDFlagName     = "env-id"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:     "kube-compose",
		Short:   "k8s",
		Long:    "Environments on k8s made easy",
		Version: "0.5.2",
	}
	rootCmd.AddCommand(newDownCli(), newUpCli(), newGetCli())
	setRootCommandFlags(rootCmd)
	return rootCmd.Execute()
}

func setRootCommandFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().StringP(fileFlagName, "f", "", "Specify an alternate compose file")
	rootCmd.PersistentFlags().StringP(namespaceFlagName, "n", "", fmt.Sprintf("namespace for environment. Can also be set via "+
		"environment variable %sNAMESPACE", envVarPrefix))
	rootCmd.PersistentFlags().StringP(envIDFlagName, "e", "", "used to isolate environments deployed to a shared namespace, "+
		"by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors. Either this flag or "+
		fmt.Sprintf("the environment variable %sENVID must be set", envVarPrefix))
}
