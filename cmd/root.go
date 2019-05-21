package cmd

import (
	"github.com/spf13/cobra"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:     "kube-compose",
		Short:   "k8s",
		Long:    "Environments on k8s made easy",
		Version: "0.3.0",
	}
	rootCmd.AddCommand(newDownCli(), newUpCli(), newGetCli())
	rootCmd.PersistentFlags().StringP("file", "f", "", "Specify an alternate compose file")
	rootCmd.PersistentFlags().StringP("env-id", "e", "", "used to isolate environments deployed to a shared namespace, "+
		"by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors. Either this flag or "+
		"the environment variable KUBECOMPOSE_ENVID must be set")
	return rootCmd.Execute()
}
