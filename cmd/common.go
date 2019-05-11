package cmd

import (
	"github.com/jbrekelmans/kube-compose/pkg/config"
	"github.com/spf13/cobra"

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
		return nil, err
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, err
	}
	cfg.KubeConfig = kubeConfig
	cfg.Namespace = namespace
	return cfg, nil
}

func getFileFlag(cmd *cobra.Command) (*string, error) {
	var file *string
	if cmd.Flags().Changed("file") {
		fileStr, err := cmd.Flags().GetString("file")
		if err != nil {
			return nil, err
		}
		file = new(string)
		*file = fileStr
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
	cfg.Services = args
	return cfg, nil
}
