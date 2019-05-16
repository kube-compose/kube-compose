package cmd

import (
	"github.com/spf13/cobra"
)

func SetupDescribeCli() *cobra.Command {
	var describeCmd = &cobra.Command{
		Use:   "describe",
		Short: "Show details of a specific resource or group of resources",
		Long:  "Print a detailed description of the selected resources, including related resources such as hostname or host IP.",
	}
	describeCmd.AddCommand(hostnameCmd(), hostIPCmd())
	return describeCmd
}

func hostnameCmd() *cobra.Command {
	var hostnameCmd = &cobra.Command{
		Use:   "hostname",
		Short: "Return the hostname of the running service",
		Run:   hostnameCommand,
	}
	hostnameCmd.PersistentFlags().StringP("service", "s", "", "Specify running target service")
	return hostnameCmd
}

func hostIPCmd() *cobra.Command {
	var hostIPCmd = &cobra.Command{
		Use:   "ip",
		Short: "Return the clusterIP of the running service",
		Run:   hostIPCommand,
	}
	hostIPCmd.PersistentFlags().StringP("service", "s", "", "Specify running target service")
	return hostIPCmd
}

func hostnameCommand(cmd *cobra.Command, args []string) {

}

func hostIPCommand(cmd *cobra.Command, args []string) {

}
