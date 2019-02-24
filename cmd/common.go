package cmd

import (
	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"k8s.io/client-go/tools/clientcmd"
)

func NewConfig() (*config.Config, error) {
	cfg, err := config.New()
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
	cfg.EnvironmentID = "test123"
	cfg.EnvironmentLabel = "environment-instance"
	cfg.Namespace = namespace
	return cfg, nil
}
