package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() error {
	viper.SetEnvPrefix("kubecompose")
	envID := new(string)
	namespace := new(string)

	rootCmd := &cobra.Command{
		Use:     "kube-compose",
		Short:   "k8s",
		Long:    "Environments on k8s made easy",
		Version: "0.3.0",
	}

	rootCmd.AddCommand(newDownCli(), newUpCli(), newGetCli())
	rootCmd.PersistentFlags().StringP("file", "f", "", "Specify an alternate compose file")
	rootCmd.PersistentFlags().StringVarP(envID, "env-id", "e", "", "used to isolate environments deployed to a shared namespace, "+
		"by (1) using this value as a suffix of pod and service names and (2) using this value to isolate selectors. Either this flag or "+
		"the environment variable KUBECOMPOSE_ENVID must be set")
	viper.AutomaticEnv()

	if *namespace == "" && viper.GetString("namespace") != "" {
		// check if environment variable is set
		*namespace = viper.GetString("namespace")
	}
	// TODO https://github.com/jbrekelmans/kube-compose/issues/80 this does not have a hyphen whereas the flag does. What does AutomaticEnv
	// do?
	if *envID == "" && viper.GetString("envid") != "" {
		*envID = viper.GetString("envid")
	} else {
		_ = rootCmd.MarkPersistentFlagRequired("env-id")
	}
	return rootCmd.Execute()
}
