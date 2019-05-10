package cmd

import (
	"github.com/jbrekelmans/kube-compose/pkg/config"

	// Plugin does not export any functions therefore it is ignored IE. "_"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

func newConfigFromEnv(file string) (*config.Config, error) {
	cfg, err := config.New(file)
	if err != nil {
		return nil, err
	}
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
