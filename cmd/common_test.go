package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func withMockedEnv(mockEnv map[string]string, callback func()) {
	orig := envGetter
	defer func() {
		envGetter = orig
	}()
	envGetter = func(name string) (string, bool) {
		value, ok := mockEnv[name]
		return value, ok
	}
	callback()
}

func Test_GetEnvIDFlag_EnvLookupSuccess(t *testing.T) {
	withMockedEnv(map[string]string{
		"KUBECOMPOSE_ENVID": "12345",
	}, func() {
		cmd := &cobra.Command{}
		key, err := getEnvIDFlag(cmd.Flags())
		if err != nil {
			t.Error(err)
		} else if key != "12345" {
			t.Fail()
		}
	})
}

func Test_GetEnvIDFlag_NotSetError(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		_, err := getEnvIDFlag(cmd.Flags())
		if err == nil {
			t.Fail()
		}
	})
}

func Test_GetEnvIDFlag_InvalidEnvError(t *testing.T) {
	withMockedEnv(map[string]string{
		"KUBECOMPOSE_ENVID": ".",
	}, func() {
		cmd := &cobra.Command{}
		_, err := getEnvIDFlag(cmd.Flags())
		if err == nil {
			t.Fail()
		}
	})
}

func Test_GetEnvIDFlag_FlagSuccess(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + envIDFlagName, "123"})
		key, err := getEnvIDFlag(cmd.Flags())
		if err != nil {
			t.Error(err)
		} else if key != "123" {
			t.Fail()
		}
	})
}

func Test_GetEnvIDFlag_InvalidFlagError(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + envIDFlagName, ","})
		_, err := getEnvIDFlag(cmd.Flags())
		if err == nil {
			t.Fail()
		}
	})
}

func Test_GetNamespaceFlag_EnvLookupSuccess(t *testing.T) {
	withMockedEnv(map[string]string{
		"KUBECOMPOSE_NAMESPACE": "1234",
	}, func() {
		cmd := &cobra.Command{}
		key, exists := getNamespaceFlag(cmd.Flags())
		if key != "1234" || !exists {
			t.Fail()
		}
	})
}

func Test_GetNamespaceFlag_NotSet(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		key, exists := getNamespaceFlag(cmd.Flags())
		if key != "" || exists {
			t.Fail()
		}
	})
}

func Test_GetNamespaceFlag_FlagSuccess(t *testing.T) {
	withMockedEnv(map[string]string{}, func() {
		cmd := &cobra.Command{}
		setRootCommandFlags(cmd)
		_ = cmd.ParseFlags([]string{"--" + namespaceFlagName, "test"})
		key, exists := getNamespaceFlag(cmd.Flags())
		if key != "test" || !exists {
			t.Fail()
		}
	})
}
