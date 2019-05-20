package cmd

import (
	"fmt"
	"log"
	"os"

	"text/template"

	details "github.com/jbrekelmans/kube-compose/internal/app/get"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	"github.com/spf13/cobra"
)

func newGetCli() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Show details of a specific resource",
		Long:  "Print a detailed description of the selected resources, including related resources such as hostname or host IP.",
		Run:   getCommand,
	}
	getCmd.PersistentFlags().StringP("format", "", "", "Return specified format")
	return getCmd
}

// TODO: If no service is specified then it should iterate through all services in the docker-compsoe
// https://github.com/jbrekelmans/kube-compose/issues/126
func getCommand(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatal("No Args Provided")
	}
	cfg, err := getCommandConfig(cmd, args)
	if err != nil {
		log.Fatal(err)
	}
	var format string
	if cmd.Flags().Changed("format") {
		format, err = cmd.Flags().GetString("format")
		if err != nil {
			log.Fatal(err)
		}
	}
	service := cfg.FindServiceByName(args[0])
	if service == nil {
		log.Fatalf("no service named %#v exists", args[0])
	}
	result, err := details.GetServiceDetails(cfg, service)
	if err != nil {
		log.Fatal(err)
	}
	if format != "" {
		tmpl, err := template.New(args[0]).Parse(format)
		if err != nil {
			panic(err)
		}
		err = tmpl.Execute(os.Stdout, result)
		if err != nil {
			panic(err)
		}
	} else {
		output := util.FormatTable([][]string{
			{"NAME", "NAMESPACE", "HOSTNAME", "CLUSTER-IP"},
			{result.Service, result.Namespace, result.Hostname, result.ClusterIP},
		})
		fmt.Print(output)
	}
}
