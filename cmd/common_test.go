package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetEnvIDFlag_EnvLookUpExists(t *testing.T) {
	cmd := &cobra.Command{}
	os.Setenv("KUBECOMPOSE_ENVID", "123")
	key, err := getEnvIDFlag(cmd)
	if key != "123" {
		t.Fail()
	}
	if err != nil {
		t.Fail()
	}
}

func TestGetEnvIDFlag_EnvLookUpNotExists(t *testing.T) {
	cmd := &cobra.Command{}
	key, err := getEnvIDFlag(cmd)
	if key == "123" {
		t.Fail()
	}
	if err == nil {
		t.Fail()
	}
}

func TestGetNamespaceFlag_EnvLookUpExists(t *testing.T) {
	cmd := &cobra.Command{}
	os.Setenv("KUBECOMPOSE_NAMESPACE", "123")
	key, exists := getNamespaceFlag(cmd)
	if key != "123" {
		t.Fail()
	}
	if exists == false {
		t.Fail()
	}
}

func TestGetNamespaceFlag_EnvLookUpNotExists(t *testing.T) {
	cmd := &cobra.Command{}
	key, exists := getNamespaceFlag(cmd)
	if key == "123" {
		t.Fail()
	}
	if exists == true {
		t.Fail()
	}
}
