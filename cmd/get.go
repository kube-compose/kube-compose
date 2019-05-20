package cmd

import (
	"log"
	"os"

	"text/template"

	"github.com/jbrekelmans/kube-compose/internal/get"
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
	Service        string
	Hostname       string
	Namespace      string
	ClusterIP      string
	ExternalIP     []string
	LoadBalancerIP string
}

func getCommand(cmd *cobra.Command, args []string) {
	filter := "Service\tHostname\tClusterIP\n{{.Service}}\t{{.Hostname}}\t{{.ClusterIP}}"
	var err error
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
	if cmd.Flags().Changed("format") {
		filter, err = cmd.Flags().GetString("format")
		if err != nil {
			log.Fatal(err)
		}
	}
	result, err := get.Service(cfg, args[0])
	if err != nil {
		log.Fatal(err)
	}
	service := KubeComposeService{
		Service:        result.Name,
		Hostname:       result.Name + "." + result.Namespace + ".svc.cluster.local",
		Namespace:      result.Namespace,
		ClusterIP:      result.Spec.ClusterIP,
		ExternalIP:     result.Spec.ExternalIPs,
		LoadBalancerIP: result.Spec.LoadBalancerIP,
	}
	tmpl, err := template.New(args[0]).Parse(filter)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, service)
	if err != nil {
		panic(err)
	}
}
