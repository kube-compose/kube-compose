package cmd

import (
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	envVarPrefix        = "KUBECOMPOSE_"
	fileFlagName        = "file"
	logLevelEnvVarName  = envVarPrefix + "LOGLEVEL"
	logLevelFlagName    = "log-level"
	namespaceEnvVarName = envVarPrefix + "NAMESPACE"
	namespaceFlagName   = "namespace"
	envIDEnvVarName     = envVarPrefix + "ENVID"
	envIDFlagName       = "env-id"
)

var formattedLogLevelList = formatLogLevelList()

func formatLogLevelList() string {
	var sb strings.Builder
	sb.WriteString(log.AllLevels[0].String())
	iLast := len(log.AllLevels) - 1
	for i := 1; i < iLast; i++ {
		sb.WriteString(", ")
		sb.WriteString(log.AllLevels[i].String())
	}
	sb.WriteString(" and ")
	sb.WriteString(log.AllLevels[iLast].String())
	return sb.String()
}

func getLogLevelFlag(flags *pflag.FlagSet) (log.Level, error) {
	if !flags.Changed(logLevelFlagName) {
		s, exists := envGetter(logLevelEnvVarName)
		if !exists {
			return log.WarnLevel, nil
		}
		logLevel, err := log.ParseLevel(s)
		if err != nil {
			return 0, fmt.Errorf("the environment variable %s can only be set to one of %s", logLevelEnvVarName, formattedLogLevelList)
		}
		return logLevel, nil
	}
	s, _ := flags.GetString(logLevelFlagName)
	logLevel, err := log.ParseLevel(s)
	if err != nil {
		return 0, fmt.Errorf("the flag --%s can only be set to one of %s", logLevelFlagName, formattedLogLevelList)
	}
	return logLevel, nil
}

func Execute() error {
	log.SetOutput(os.Stdout)
	rootCmd := &cobra.Command{
		Use:     "kube-compose",
		Short:   "k8s",
		Long:    "Environments on k8s made easy",
		Version: "0.6.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logLevel, err := getLogLevelFlag(cmd.Flags())
			if err != nil {
				return err
			}
			log.SetLevel(logLevel)
			return nil
		},
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
		"%s. Defaults to warn", formattedLogLevelList, logLevelEnvVarName))
}
