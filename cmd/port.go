package cmd

import (
	"github.com/spf13/cobra"
)

func SetupPortCli() *cobra.Command {

	var portCmd = &cobra.Command{
		Use:   "port",
		Short: "Print the public port for a port binding",
		Long:  "creates pods and services in an order that respects depends_on in the docker compose file",
		Run:   portCommand,
	}
	return portCmd
}

func portCommand(cmd *cobra.Command, args []string) {

}
