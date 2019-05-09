package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kube-compose",
	Short: "k8s",
	Long:  `Environments on k8s made easy`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("namespace", "n", "", "namespace for environment (required)")
	rootCmd.MarkPersistentFlagRequired("namespace")
	rootCmd.PersistentFlags().StringP("env-id", "e", "", "used to isolate environments deployed to a shared namespace, by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors(required)")
	rootCmd.MarkPersistentFlagRequired("env-id")
}
