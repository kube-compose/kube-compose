package get

import (
	"fmt"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type getRunner struct {
	cfg              *config.Config
	k8sClientset     *kubernetes.Clientset
	k8sServiceClient clientV1.ServiceInterface
}

func Service(cfg *config.Config, serviceName string) (*v1.Service, error) {
	g := &getRunner{
		cfg: cfg,
	}
	service := cfg.CanonicalComposeFile.Services[serviceName]
	if service == nil {
		return nil, fmt.Errorf("no service named %#v exists", serviceName)
	}
	return g.run(service)
}

func (g *getRunner) initKubernetesClientset() error {
	k8sClientset, err := kubernetes.NewForConfig(g.cfg.KubeConfig)
	if err != nil {
		return err
	}
	g.k8sClientset = k8sClientset
	g.k8sServiceClient = g.k8sClientset.CoreV1().Services(g.cfg.Namespace)
	return nil
}

func (g *getRunner) getK8sServiceResource(service *config.Service) (*v1.Service, error) {
	options := &metav1.GetOptions{}
	result, err := g.k8sServiceClient.Get(service.NameEscaped()+"-"+g.cfg.EnvironmentID, *options)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (g *getRunner) run(service *config.Service) (*v1.Service, error) {
	err := g.initKubernetesClientset()
	if err != nil {
		return nil, err
	}
	result, err := g.getK8sServiceResource(service)
	if err != nil {
		return result, err
	}
	return result, nil
}
