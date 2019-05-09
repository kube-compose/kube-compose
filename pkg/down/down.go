package down

import (
	"fmt"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type deleter func(name string, options *metav1.DeleteOptions) error

type lister func(listOptions metav1.ListOptions) ([]*v1.ObjectMeta, error)

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

func (d *downRunner) deleteCommon(errorChannel chan<- error, kind string, lister lister, deleter deleter) {
	defer close(errorChannel)
	listOptions := metav1.ListOptions{
		LabelSelector: d.cfg.EnvironmentLabel + "=" + d.cfg.EnvironmentID,
	}
	list, err := lister(listOptions)
	if err != nil {
		errorChannel <- err
		return
	}
	deleteOptions := &metav1.DeleteOptions{}
	for _, item := range list {
		err := deleter(item.Name, deleteOptions)
		if err != nil {
			errorChannel <- err
			return
		}
		fmt.Printf("deleted %s %s\n", kind, item.Name)
	}
}

func (d *downRunner) deleteK8sResource(resource string, errorChannel chan<- error) {
	lister := func(listOptions metav1.ListOptions) ([]*v1.ObjectMeta, error) {
		serviceList, err := d.k8sServiceClient.List(listOptions)
		if err != nil {
			return nil, err
		}
		list := make([]*v1.ObjectMeta, len(serviceList.Items))
		for i := 0; i < len(serviceList.Items); i++ {
			list[i] = &serviceList.Items[i].ObjectMeta
		}
		return list, nil
	}
	d.deleteCommon(errorChannel, resource, lister, d.k8sServiceClient.Delete)
}

func (d *downRunner) run() error {
	err := d.initKubernetesClientset()
	if err != nil {
		return err
	}
	errorChannels := make([]chan error, 2)
	for i := 0; i < len(errorChannels); i++ {
		errorChannels[i] = make(chan error, 1)
	}
	go d.deleteK8sResource("Service", errorChannels[0])
	go d.deleteK8sResource("Pod", errorChannels[1])
	var firstError error
	for i := 0; i < len(errorChannels); i++ {
		err, more := <-errorChannels[i]
		if !more {
			continue
		}
		if firstError == nil {
			firstError = err
		} else {
			fmt.Println(err)
		}
	}
	return firstError
}

// Run runs a docker-compose down command...
func Run(cfg *config.Config) error {
	d := &downRunner{
		cfg: cfg,
	}
	return d.run()
}
