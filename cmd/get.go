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
	getCmd.PersistentFlags().StringP("output", "o", "", "Go template string")
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
	var tmpl *template.Template
	if cmd.Flags().Changed("output") {
		var output string
		output, _ = cmd.Flags().GetString("output")
		tmpl, err = template.New("test").Parse(output)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	service := cfg.Services[args[0]]
	d, err := details.GetServiceDetails(cfg, service)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if tmpl != nil {
		err = tmpl.Execute(os.Stdout, d)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		output := util.FormatTable([][]string{
			{"NAME", "HOSTNAME", "CLUSTER-IP"},
			{d.Name, d.Hostname, d.ClusterIP},
		})
		fmt.Print(output)
	}
	return nil
}
