package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "kube-compose",
	Short: "k8s",
	Long:  "Environments on k8s made easy",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	env := &struct {
		namespace string
		envID     string
	}{}
	viper.SetEnvPrefix("kubecompose")
	rootCmd.PersistentFlags().StringVarP(&env.namespace, "namespace", "n", "", "namespace for environment")
	rootCmd.PersistentFlags().StringVarP(&env.envID, "env-id", "e", "", "used to isolate environments deployed to a shared namespace, by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors(required)")
	viper.AutomaticEnv()
	if env.namespace == "" && viper.GetString("namespace") != "" {
		// check if environment variable is set
		env.namespace = viper.GetString("namespace")
	}
	if env.envID == "" && viper.GetString("envid") != "" {
		env.envID = viper.GetString("envid")

	} else {
		_ = rootCmd.MarkPersistentFlagRequired("env-id")
	}
}
