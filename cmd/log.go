package cmd

import (
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	logLevelDefault    = log.WarnLevel
	logLevelEnvVarName = envVarPrefix + "LOGLEVEL"
	logLevelFlagName   = "log-level"
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
			return logLevelDefault, nil
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

func setupLogging(cmd *cobra.Command, _ []string) error {
	logLevel, err := getLogLevelFlag(cmd.Flags())
	if err != nil {
		return err
	}
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
	if reporter.IsTerminal(os.Stdout) {
		log.SetFormatter(createTerminalLogFormatter())
	} else {
		log.SetFormatter(&log.TextFormatter{
			DisableTimestamp: true,
		})
	}
	return nil
}

func createTerminalLogFormatter() log.Formatter {
	return &log.TextFormatter{
		ForceColors:               true,
		DisableTimestamp:          true,
		EnvironmentOverrideColors: true,
	}
}
