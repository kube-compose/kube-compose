package cmd

import (
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Test_GetLogLevelFlag_SuccessFlag(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + logLevelFlagName, "error"})
		logLevel, err := getLogLevelFlag(cmd.Flags())
		if err != nil {
			t.Error(err)
		} else if logLevel != log.ErrorLevel {
			t.Fail()
		}
	})
}

func Test_GetLogLevelFlag_SuccessEnv(t *testing.T) {
	withMockedEnv(map[string]string{
		logLevelEnvVarName: "error",
	}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		logLevel, err := getLogLevelFlag(cmd.Flags())
		if err != nil {
			t.Error(err)
		} else if logLevel != log.ErrorLevel {
			t.Fail()
		}
	})
}
func Test_GetLogLevelFlag_ErrorEnv(t *testing.T) {
	withMockedEnv(map[string]string{
		logLevelEnvVarName: "invalid",
	}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_, err := getLogLevelFlag(cmd.Flags())
		if err == nil {
			t.Fail()
		}
	})
}

func Test_SetupLogging_ErrorFlag(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + logLevelFlagName, "invalid"})
		err := setupLogging(cmd, nil)
		if err == nil {
			t.Fail()
		}
	})
}

func Test_SetupLogging_SuccessDefault(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		err := setupLogging(cmd, nil)
		if err != nil {
			t.Error(err)
		}
	})
}
