package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func withMockedEnv(mockEnv map[string]string, callback func()) {
	oldEnvGetter := envGetter
	defer func() {
		envGetter = oldEnvGetter
	}()
	envGetter = func(name string) (string, bool) {
		value, ok := mockEnv[name]
		return value, ok
	}
	callback()
}

func TestGetEnvIDFlag_EnvLookUpExists(t *testing.T) {
	withMockedEnv(map[string]string{
		"KUBECOMPOSE_ENVID": "12345",
	}, func() {
		cmd := &cobra.Command{}
		key, err := getEnvIDFlag(cmd)
		if key != "12345" || err != nil {
			t.Fail()
		}
	})
}

func TestGetEnvIDFlag_EnvLookUpNotExists(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		key, err := getEnvIDFlag(cmd)
		if key != "" || err == nil {
			t.Fail()
		}
	})
}

func TestGetEnvIDFlag_FlagIsSet(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + envIDFlagName, "123"})
		key, err := getEnvIDFlag(cmd)
		if key != "123" || err != nil {
			t.Fail()
		}
	})
}

func TestGetNamespaceFlag_EnvLookUpExists(t *testing.T) {
	withMockedEnv(map[string]string{
		"KUBECOMPOSE_NAMESPACE": "1234",
	}, func() {
		cmd := &cobra.Command{}
		key, exists := getNamespaceFlag(cmd)
		if key != "1234" || exists == false {
			t.Fail()
		}
	})
}

func TestGetNamespaceFlag_EnvLookUpNotExists(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		key, exists := getNamespaceFlag(cmd)
		if key != "" || exists == true {
			t.Fail()
		}
	})
}

func TestGetNamespaceFlag_FlagIsSet(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + namespaceFlagName, "test"})
		key, exists := getNamespaceFlag(cmd)
		if key != "test" || exists != true {
			t.Fail()
		}
	})
}
