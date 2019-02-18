package up

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	
	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	v1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type outputHelper struct {
	outputDir string
}

func (o *outputHelper) init (cfg *config.Config) error {
	outputDir, err := filepath.Abs("output")
	if err != nil {
		return err
	}
	err = os.RemoveAll(outputDir)
	if err != nil {
		return err
	}
	o.outputDir = outputDir
	return nil
}

func (o *outputHelper) addResource (kind, name string, r interface{}) error {
	json, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(o.outputDir, name, kind + ".json")
	os.MkdirAll(filepath.Dir(file), 0700)
	err = ioutil.WriteFile(file, json, 0600)
	if err != nil {
		return err
	}
	return nil
}

func initObjectMeta (objectMeta *metav1.ObjectMeta, name string) {
	objectMeta.Name = name
	if objectMeta.Labels == nil {
		objectMeta.Labels = map[string]string{}
	}
	objectMeta.Labels["app"] = name
}

func Run (cfg *config.Config) error {
	o := outputHelper{}
	err := o.init(cfg)
	if err != nil {
		return err
	}
	for name, service := range cfg.ComposeYaml.Services {


		ports, err := config.ParsePorts(service.Ports)
		if err != nil {
			return err
		}

		containerPorts := []v1.ContainerPort{}
		servicePorts := []v1.ServicePort{}
		for _, port := range ports {
			containerPorts = append(containerPorts, v1.ContainerPort{
				ContainerPort: port.ContainerPort,
				Protocol: v1.Protocol(port.Protocol),
			})
			servicePorts = append(servicePorts, v1.ServicePort{
				Port: port.ExternalPort,
				Protocol: v1.Protocol(port.Protocol),
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
			})
		}

		pod := v1.Pod{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name: name,
						Image: service.Image,
						Ports: containerPorts,
						WorkingDir: service.WorkingDir,
					},
				},
			},
		}
		initObjectMeta(&pod.ObjectMeta, name)
		err = o.addResource("Pod", name, &pod)
		if err != nil {
			return err
		}

		service := v1.Service{
			Spec: v1.ServiceSpec{
				Ports: servicePorts,
				Selector: map[string]string {
					"app": name,
				},
				// This is the default value.
				// Type: v1.ServiceType("ClusterIP"),
			},
		}
		initObjectMeta(&service.ObjectMeta, name)
		err = o.addResource("Service", name, &service)
		if err != nil {
			return err
		}
	}
	return nil
}