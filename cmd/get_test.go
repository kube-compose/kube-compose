package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetCommand_NoArgsError(t *testing.T) {
	cmd := &cobra.Command{}
	args := []string{}
	err := getCommand(cmd, args)
	if err == nil {
		t.Fail()
	}
}
