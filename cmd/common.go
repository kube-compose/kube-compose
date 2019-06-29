package cmd

import (
	"fmt"
	"os"

	"github.com/kube-compose/kube-compose/internal/app/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"

	// Plugin does not export any functions therefore it is ignored IE. "_"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var envGetter = os.LookupEnv

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

func getFileFlags(cmd *cobra.Command) ([]string, error) {
	var files []string
	if cmd.Flags().Changed(fileFlagName) {
		var err error
		files, err = cmd.Flags().GetStringSlice(fileFlagName)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func getEnvIDFlag(cmd *cobra.Command) (string, error) {
	var envID string
	var exists bool
	if !cmd.Flags().Changed(envIDFlagName) {
		envID, exists = envGetter(envIDEnvVarName)
		if !exists {
			return "", fmt.Errorf("either the flag --env-id or the environment variable %s must be set", envIDEnvVarName)
		}
		if e := validation.IsValidLabelValue(envID); len(e) > 0 {
			return "", fmt.Errorf("the environment variable %s must be a valid label value: %s", envIDEnvVarName, e[0])
		}
	} else {
		envID, _ = cmd.Flags().GetString(envIDFlagName)
		if e := validation.IsValidLabelValue(envID); len(e) > 0 {
			return "", fmt.Errorf("the --env-id flag must be a valid label value: %s", e[0])
		}
	}
	return envID, nil
}

func getNamespaceFlag(cmd *cobra.Command) (string, bool) {
	var namespace string
	var exists bool
	if !cmd.Flags().Changed(namespaceFlagName) {
		namespace, exists = envGetter(namespaceEnvVarName)
		if !exists {
			return "", false
		}
		return namespace, true
	}
	namespace, _ = cmd.Flags().GetString(namespaceFlagName)
	return namespace, true
}

func getCommandConfig(cmd *cobra.Command, args []string) (*config.Config, error) {
	envID, err := getEnvIDFlag(cmd)
	if err != nil {
		return nil, err
	}
	files, err := getFileFlags(cmd)
	if err != nil {
		return nil, err
	}
	cfg, err := config.New(files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := setFromKubeConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
			service := cfg.Services[arg]
			if service == nil {
				fmt.Fprintf(os.Stderr, "no service named %#v exists\n", arg)
				os.Exit(1)
			}
			cfg.AddToFilter(service)
		}
	}
	return cfg, nil
}
