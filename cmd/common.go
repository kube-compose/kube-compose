package cmd

import (
	"fmt"
	"os"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	"github.com/pkg/errors"
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
		return errors.Wrap(err, "error loading kubernetes config file")
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	cfg.KubeConfig = kubeConfig
	cfg.Namespace = namespace
	return nil
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

func getEnvIDFlag(cmd *cobra.Command) (string, error) {
	var envID string
	var exists bool
	if !cmd.Flags().Changed("env-id") {
		envID, exists = os.LookupEnv("KUBECOMPOSE_ENVID")
		if !exists {
			return "", fmt.Errorf("either the flag --env-id or the environment variable KUBECOMPOSE_ENVID must be set")
		}
		return envID, nil
	}
	envID, _ = cmd.Flags().GetString("env-id")
	return envID, nil
}

func getNamespaceFlag(cmd *cobra.Command) (string, bool) {
	var namespace string
	var exists bool
	if !cmd.Flags().Changed("namespace") {
		namespace, exists = os.LookupEnv("KUBECOMPOSE_NAMESPACE")
		if !exists {
			return "", false
		}
		return namespace, true
	}
	namespace, _ = cmd.Flags().GetString("namespace")
	return namespace, false
}

func getCommandConfig(cmd *cobra.Command, args []string) (*config.Config, error) {
	envID, err := getEnvIDFlag(cmd)
	if err != nil {
		return nil, err
	}
	file, err := getFileFlag(cmd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.New(file)
	if err != nil {
		return nil, err
	}
	if err := setFromKubeConfig(cfg); err != nil {
		return nil, err
	}
	cfg.EnvironmentID = envID
	if namespace, exists := getNamespaceFlag(cmd); exists {
		cfg.Namespace = namespace
	}
	if len(args) == 0 {
		for _, service := range cfg.Services {
			cfg.AddToFilter(service)
		}
	} else {
		for _, arg := range args {
			service := cfg.FindServiceByName(arg)
			if service == nil {
				return nil, fmt.Errorf("no service named %#v does exists", arg)
			}
			cfg.AddToFilter(service)
		}
	}
	return cfg, nil
}
