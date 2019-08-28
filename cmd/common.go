package cmd

import (
	"fmt"
	"os"

	log "github.com/github.com/sirupsen/logrus/logrus"
	"github.com/kube-compose/kube-compose/internal/app/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		return errors.Wrap(err, "could not load kube config")
	}
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	cfg.KubeConfig = kubeConfig
	cfg.Namespace = namespace
	return nil
}

func getFileFlags(flags *pflag.FlagSet) ([]string, error) {
	var files []string
	if flags.Changed(fileFlagName) {
		var err error
		files, err = flags.GetStringSlice(fileFlagName)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func getEnvIDFlag(flags *pflag.FlagSet) (string, error) {
	var envID string
	var exists bool
	if !flags.Changed(envIDFlagName) {
		envID, exists = envGetter(envIDEnvVarName)
		if !exists {
			return "", fmt.Errorf("either the flag --%s or the environment variable %s must be set", envIDFlagName, envIDEnvVarName)
		}
		if e := validation.IsValidLabelValue(envID); len(e) > 0 {
			return "", fmt.Errorf("the environment variable %s must be a valid label value: %s", envIDEnvVarName, e[0])
		}
	} else {
		envID, _ = flags.GetString(envIDFlagName)
		if e := validation.IsValidLabelValue(envID); len(e) > 0 {
			return "", fmt.Errorf("the --%s flag must be a valid label value: %s", envIDFlagName, e[0])
		}
	}
	return envID, nil
}

func getNamespaceFlag(flags *pflag.FlagSet) (string, bool) {
	var namespace string
	var exists bool
	if !flags.Changed(namespaceFlagName) {
		namespace, exists = envGetter(namespaceEnvVarName)
		if !exists {
			return "", false
		}
		return namespace, true
	}
	namespace, _ = flags.GetString(namespaceFlagName)
	return namespace, true
}

func getCommandConfig(cmd *cobra.Command, args []string) (*config.Config, error) {
	envID, err := getEnvIDFlag(cmd.Flags())
	if err != nil {
		return nil, err
	}
	files, err := getFileFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}
	cfg, err := config.New(files)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	if err := setFromKubeConfig(cfg); err != nil {
		log.Error(err)
		os.Exit(1)
	}
	cfg.EnvironmentID = envID
	if namespace, exists := getNamespaceFlag(cmd.Flags()); exists {
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
				log.Errorf("no service named %#v exists\n", arg)
				os.Exit(1)
			}
			cfg.AddToFilter(service)
		}
	}
	return cfg, nil
}
