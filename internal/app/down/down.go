package down

import (
	log "github.com/Sirupsen/logrus"
	"github.com/kube-compose/kube-compose/internal/app/config"
	"github.com/kube-compose/kube-compose/internal/app/k8smeta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type deleter func(name string, options *metav1.DeleteOptions) error

type lister func(listOptions metav1.ListOptions) ([]*metav1.ObjectMeta, error)

type downRunner struct {
	cfg              *config.Config
	k8sClientset     *kubernetes.Clientset
	k8sServiceClient clientV1.ServiceInterface
	k8sPodClient     clientV1.PodInterface
}

func (d *downRunner) initKubernetesClientset() error {
	k8sClientset, err := kubernetes.NewForConfig(d.cfg.KubeConfig)
	if err != nil {
		return err
	}
	d.k8sClientset = k8sClientset
	d.k8sServiceClient = d.k8sClientset.CoreV1().Services(d.cfg.Namespace)
	d.k8sPodClient = d.k8sClientset.CoreV1().Pods(d.cfg.Namespace)
	return nil
}

func (d *downRunner) deleteCommon(kind string, lister lister, deleter deleter) (bool, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: d.cfg.EnvironmentLabel + "=" + d.cfg.EnvironmentID,
	}
	list, err := lister(listOptions)
	if err != nil {
		return false, err
	}
	deleteOptions := &metav1.DeleteOptions{}
	deletedAll := true
	for _, item := range list {
		composeService := k8smeta.FindFromObjectMeta(d.cfg, item)
		if composeService == nil || d.cfg.MatchesFilter(composeService) {
			err = deleter(item.Name, deleteOptions)
			if err != nil {
				return false, err
			}
			log.Infof("deleted %s %s\n", kind, item.Name)
		} else {
			deletedAll = false
		}
	}
	return deletedAll, nil
}

// Linter reports code duplication amongst deleteServices and deletePods. Although this is true, deduplicating would require the use of
// generics, so we choose to nolint.
// nolint
func (d *downRunner) deleteServices() (bool, error) {
	lister := func(listOptions metav1.ListOptions) ([]*metav1.ObjectMeta, error) {
		serviceList, err := d.k8sServiceClient.List(listOptions)
		if err != nil {
			return nil, err
		}
		list := make([]*metav1.ObjectMeta, len(serviceList.Items))
		for i := 0; i < len(serviceList.Items); i++ {
			list[i] = &serviceList.Items[i].ObjectMeta
		}
		return list, nil
	}
	return d.deleteCommon("Service", lister, d.k8sServiceClient.Delete)
}

// Linter reports code duplication amongst deleteServices and deletePods. Although this is true, deduplicating would require the use of
// generics, so we choose to nolint.
// nolint
func (d *downRunner) deletePods() (bool, error) {
	lister := func(listOptions metav1.ListOptions) ([]*metav1.ObjectMeta, error) {
		podList, err := d.k8sPodClient.List(listOptions)
		if err != nil {
			return nil, err
		}
		list := make([]*metav1.ObjectMeta, len(podList.Items))
		for i := 0; i < len(podList.Items); i++ {
			list[i] = &podList.Items[i].ObjectMeta
		}
		return list, nil
	}
	return d.deleteCommon("Pod", lister, d.k8sPodClient.Delete)
}

func (d *downRunner) run() error {
	err := d.initKubernetesClientset()
	if err != nil {
		return err
	}

	deletedAllPods, err := d.deletePods()
	if err != nil {
		return err
	}

	// Only delete services if all pods are to be deleted. This is so that existing pods will not have
	// their host aliases invalidated.
	if deletedAllPods {
		_, err = d.deleteServices()
		if err != nil {
			return err
		}
	}
	return nil
}

// Run runs a docker-compose down command...
func Run(cfg *config.Config) error {
	d := &downRunner{
		cfg: cfg,
	}
	return d.run()
}
