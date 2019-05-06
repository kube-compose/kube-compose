package cmd

import (
	"github.com/jbrekelmans/kube-compose/pkg/config"
<<<<<<< HEAD
	"github.com/spf13/cobra"
=======
	"github.com/urfave/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
>>>>>>> adf01d6... Start independent services in kube-compose defined in docker-compose.yml (#49)

	// Plugin does not export any functions therefore it is ignored IE. "_"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func setFromKubeConfig(cfg *config.Config) error {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, &overrides)
	kubeConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	cfg.KubeConfig = kubeConfig
	cfg.Namespace = namespace
	return nil
}

<<<<<<< HEAD
func getFileFlag(cmd *cobra.Command) (*string, error) {
	var file *string
	if cmd.Flags().Changed("file") {
		fileStr, err := cmd.Flags().GetString("file")
		if err != nil {
			return nil, err
		}
		file = new(string)
		*file = fileStr
=======
func updateConfigFromCli(cfg *config.Config, c *cli.Context) error {
	environmentID := c.GlobalString(environmentIDFlagName)
	cfg.Services = c.Args()
	if len(environmentID) == 0 && !c.GlobalIsSet(environmentIDFlagName) {
		return fmt.Errorf("the environment id is required")
	} else if len(environmentID) == 0 {
		return fmt.Errorf("environment id must not be empty")
>>>>>>> adf01d6... Start independent services in kube-compose defined in docker-compose.yml (#49)
	}
	return file, nil
}

func upOrDownCommandCommon(cmd *cobra.Command, args []string) (*config.Config, error) {
	file, err := getFileFlag(cmd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.New(file)
	if err != nil {
		return nil, err
	}
	err = setFromKubeConfig(cfg)
	if err != nil {
		return nil, err
	}
	cfg.EnvironmentID, _ = cmd.Flags().GetString("env-id")
	if namespace, _ := cmd.Flags().GetString("namespace"); namespace != "" {
		cfg.Namespace = namespace
	}
	if len(args) == 0 {
		cfg.SetFilterToMatchAll()
	} else {
		err = cfg.SetFilter(args)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}
