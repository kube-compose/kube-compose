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
	if len(args) == 0 {
		return fmt.Errorf("no args provided")
	}
	cfg, err := getCommandConfig(cmd, args)
	if err != nil {
		return err
	}
	var format string
	if cmd.Flags().Changed("format") {
		format, err = cmd.Flags().GetString("format")
		if err != nil {
			return err
		}
	}
	service := cfg.FindServiceByName(args[0])
	if service == nil {
		return fmt.Errorf("no service named %#v exists", args[0])
	}
	result, err := details.GetServiceDetails(cfg, service)
	if err != nil {
		return err
	}
	if format != "" {
		tmpl, err := template.New(args[0]).Parse(format)
		if err != nil {
			return err
		}
		err = tmpl.Execute(os.Stdout, result)
		if err != nil {
			return err
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
