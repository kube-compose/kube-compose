package details

import (
	"github.com/kube-compose/kube-compose/internal/app/config"
	"github.com/kube-compose/kube-compose/internal/app/k8smeta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type getRunner struct {
	cfg              *config.Config
	k8sClientset     *kubernetes.Clientset
	k8sServiceClient clientV1.ServiceInterface
	service          *config.Service
}

type ServiceDetails struct {
	Name      string
	ClusterIP string
	Hostname  string
}

func GetServiceDetails(cfg *config.Config, service *config.Service) (*ServiceDetails, error) {
	getRunner := &getRunner{
		cfg:     cfg,
		service: service,
	}
	return getRunner.run()
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

func (g *getRunner) run() (*ServiceDetails, error) {
	err := g.initKubernetesClientset()
	if err != nil {
		return nil, err
	}
	k8sName := k8smeta.GetK8sName(g.service, g.cfg)
	result, err := g.k8sServiceClient.Get(k8sName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	details := &ServiceDetails{
		Name:      g.service.Name,
		Hostname:  result.Name + "." + result.Namespace + ".svc.cluster.local",
		ClusterIP: result.Spec.ClusterIP,
	}
	return details, nil
}
