package cmd

import (
	"log"

	"github.com/jbrekelmans/kube-compose/pkg/down"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "A brief description of your command",
	Long:  "destroy all pods and services",
	Run:   downCommand,
}

func downCommand(cmd *cobra.Command, args []string) {
	composeFile, _ := cmd.Flags().GetString("file")
	cfg, err := newConfigFromEnv(composeFile)
	if err != nil {
		log.Fatal(err)
	}
	cfg.EnvironmentID, _ = cmd.Flags().GetString("env-id")
	if x, _ := cmd.Flags().GetString("namespace"); x != "" {
		cfg.Namespace, _ = cmd.Flags().GetString("namespace")
	}
	err = down.Run(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(downCmd)
	upCmd.PersistentFlags().StringP("file", "f", "", "Specify an alternate compose file")
}
