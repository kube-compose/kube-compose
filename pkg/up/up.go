package up

import (
	"encoding/json"
	"io/ioutil"
	"math"
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

func (o *outputHelper) init () error {
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

// https://docs.docker.com/engine/reference/builder/#healthcheck
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#configure-probes
func healthcheckToProbe (healthcheck *config.Healthcheck) *v1.Probe {
	if healthcheck == nil {
		return nil
	}
	
	var retriesInt32 int32
	if healthcheck.Retries > math.MaxInt32 {
		retriesInt32 = math.MaxInt32
	} else {
		retriesInt32 = int32(healthcheck.Retries)
	}

	offset := 0
	if healthcheck.IsShell {
		// Assume the Shell is /bin/sh
		// Add 2 to accomodate for /bin/sh -c
		offset = 2
	}
	n := len(healthcheck.Test) + offset
	execCommand := make([]string, n)
	if offset > 0 {
		execCommand[0] = "/bin/sh"
		execCommand[1] = "-c"
	}
	for i := offset; i < n; i++ {
		execCommand[i] = healthcheck.Test[i - offset]
	}
	probe := &v1.Probe{
		FailureThreshold: retriesInt32,
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: execCommand,
			},
		},
		// StartPeriod and InitialDelaySeconds are not exactly the same, but this is good enough.
		InitialDelaySeconds: int32(math.RoundToEven(healthcheck.StartPeriod.Seconds())),
		PeriodSeconds: int32(math.RoundToEven(healthcheck.Interval.Seconds())),
		TimeoutSeconds: int32(math.RoundToEven(healthcheck.Timeout.Seconds())),
		// This is the default value.
		// SuccessThreshold: 1,
	}
	return probe
}

func Run (cfg *config.Config) error {
	o := outputHelper{}
	err := o.init()
	if err != nil {
		return err
	}
	for name, service := range cfg.DockerComposeFile.Services {
		var containerPorts []v1.ContainerPort
		var servicePorts []v1.ServicePort
		ports := service.Ports
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
		envVarCount := len(service.Environment)
		if envVarCount > 0 {
			envVars = make([]v1.EnvVar, envVarCount)
			i := 0
			for key, value := range service.Environment {
				envVars[i] = v1.EnvVar{
					Name: key,
					Value: value,
				}
				i++
			}
		}

		// TODO use HEALTHCHECK from docker image, unless service.HealthcheckDisabled is true
		probe := healthcheckToProbe(service.Healthcheck)

		pod := v1.Pod{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Env: envVars,
						Image: service.Image,
						Name: name,
						Ports: containerPorts,
						ReadinessProbe: probe,
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