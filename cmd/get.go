package cmd

import (
	"fmt"
	"os"

	"text/template"

	details "github.com/kube-compose/kube-compose/internal/app/get"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
	"github.com/spf13/cobra"
)

func newGetCli() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Show details of a specific resource",
		Long:  "Print a detailed description of the selected resources, including related resources such as hostname or host IP.",
		RunE:  getCommand,
	}
	getCmd.PersistentFlags().StringP("format", "", "", "Return specified format")
	return getCmd
}

// TODO: If no service is specified then it should iterate through all services in the docker-compose
// https://github.com/kube-compose/kube-compose/issues/126
func getCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one positional argument is required")
	}
	cfg, err := getCommandConfig(cmd, args)
	if err != nil {
		return err
	}
	var template *template.Template
	if cmd.Flags().Changed("format") {
		var format string
		format, _ = cmd.Flags().GetString("format")
		var err error
		template, err = template.New("testasdf").Parse(format)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	service := cfg.FindServiceByName(args[0])
	result, err := details.GetServiceDetails(cfg, service)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if template != nil {
		err = template.Execute(os.Stdout, result)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		output := util.FormatTable([][]string{
			{"NAME", "NAMESPACE", "HOSTNAME", "CLUSTER-IP"},
			{result.Service, result.Namespace, result.Hostname, result.ClusterIP},
		})
		fmt.Print(output)
	}
	return nil
}
