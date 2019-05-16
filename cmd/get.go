package cmd

import (
	"log"
	"os"

	"text/template"

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

type KubeComposeService struct {
	Service  string
	Hostname string
	IP       string
}

func getCommand(cmd *cobra.Command, args []string) {
	filter := "Service\tHostname\tIP\n{{.Service}}\t{{.Hostname}}\t{{.IP}}"
	var err error
	if len(args) == 0 {
		log.Fatal("No Args Provided")
	}
	if cmd.Flags().Changed("format") {
		filter, err = cmd.Flags().GetString("format")
		if err != nil {
			log.Fatal(err)
		}
	}
	service := KubeComposeService{args[0], "TestServiceName", "0.0.0.0"}
	tmpl, err := template.New(args[0]).Parse(filter)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, service)
	if err != nil {
		panic(err)
	}
}
