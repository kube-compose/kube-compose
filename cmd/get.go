package cmd

import (
	"log"
	"os"

	"text/template"

	"github.com/jbrekelmans/kube-compose/internal/pkg/get"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	"github.com/spf13/cobra"
)

func SetupGetCli() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Show details of a specific resource",
		Long:  "Print a detailed description of the selected resources, including related resources such as hostname or host IP.",
		Run:   getCommand,
	}
	getCmd.PersistentFlags().StringP("format", "", "", "Return specified format")
	return getCmd
}

func getCommand(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatal("No Args Provided")
	}
	file, err := getFileFlag(cmd)
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := getCommandConfig(cmd, args, file)
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
	result, err := get.Service(cfg, args[0])
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
		util.OutputTable([][]string{
			[]string{"NAME", "HOSTNAME", "CLUSTER-IP"},
			[]string{result.Service, result.Hostname, result.ClusterIP},
		})
	}
}
