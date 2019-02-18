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

		var containerPorts []v1.ContainerPort
		var servicePorts []v1.ServicePort
		if len(ports) > 0 {
			containerPorts = make([]v1.ContainerPort, len(ports))
			servicePorts = make([]v1.ServicePort, len(ports))
			for i, port := range ports {
				containerPorts[i] = v1.ContainerPort{
					ContainerPort: port.ContainerPort,
					Protocol: v1.Protocol(port.Protocol),
				}
				servicePorts[i] = v1.ServicePort{
					Port: port.ExternalPort,
					Protocol: v1.Protocol(port.Protocol),
					TargetPort: intstr.FromInt(int(port.ContainerPort)),
				}
			}
		}

		var envVars []v1.EnvVar
		if len(service.Environment) > 0 {
			envVars := make([]v1.EnvVar, len(service.Environment))[:0]
			for key, value := range service.Environment {
				envVars = append(envVars, v1.EnvVar{
					Name: key,
					Value: value,
				})
			}
		}

		pod := v1.Pod{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Env: envVars,
						Image: service.Image,
						Name: name,
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
		
		if len(servicePorts) > 0 {
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
	}
	return nil
}