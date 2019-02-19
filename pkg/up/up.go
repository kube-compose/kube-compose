package up

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	k8sUtil "github.com/jbrekelmans/k8s-docker-compose/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const annotationName = "k8s-docker-compose/service"

type app struct {
	clusterIP      string
	nameEncoded    string
	desiredPod     *v1.Pod
	desiredService *v1.Service
}

type upRunner struct {
	cfg              *config.Config
	apps             map[string]*app
	k8sClientset     *kubernetes.Clientset
	k8sServiceClient clientV1.ServiceInterface
	k8sPodClient     clientV1.PodInterface
	outputDir        string
}

func (u *upRunner) initKubernetesClientset() error {
	k8sConfig := &rest.Config{
		Host: "https://192.168.64.2:8443",
		TLSClientConfig: rest.TLSClientConfig{
			ServerName: "localhost",
			CertData:   clientCertData,
			KeyData:    clientKeyData,
			CAData:     caData,
		},
	}
	k8sClientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}
	u.k8sClientset = k8sClientset
	u.k8sServiceClient = u.k8sClientset.CoreV1().Services(u.cfg.Namespace)
	u.k8sPodClient = u.k8sClientset.CoreV1().Pods(u.cfg.Namespace)
	return nil
}

func (u *upRunner) initOutputDir() error {
	outputDir, err := filepath.Abs("output")
	if err != nil {
		return err
	}
	err = os.RemoveAll(outputDir)
	if err != nil {
		return err
	}
	u.outputDir = outputDir
	return nil
}

func (u *upRunner) initApps() error {
	u.apps = make(map[string]*app, len(u.cfg.DockerComposeFile.Services))
	for name, service := range u.cfg.DockerComposeFile.Services {
		app := &app{
			nameEncoded: k8sUtil.EncodeName(name),
		}
		var containerPorts []v1.ContainerPort
		var servicePorts []v1.ServicePort
		ports := service.Ports
		if len(ports) > 0 {
			containerPorts = make([]v1.ContainerPort, len(ports))
			servicePorts = make([]v1.ServicePort, len(ports))
			for i, port := range ports {
				containerPorts[i] = v1.ContainerPort{
					ContainerPort: port.ContainerPort,
					Protocol:      v1.Protocol(port.Protocol),
				}
				servicePorts[i] = v1.ServicePort{
					Port:       port.ExternalPort,
					Protocol:   v1.Protocol(port.Protocol),
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
					Name:  key,
					Value: value,
				}
				i++
			}
		}

		// TODO use HEALTHCHECK from docker image, unless service.HealthcheckDisabled is true
		readinessProbe := healthcheckToProbe(service.Healthcheck)

		app.desiredPod = &v1.Pod{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Env:   envVars,
						Image: service.Image,
						// TODO set ImagePullPolicy based on the image we have here...

						Name:           app.nameEncoded,
						Ports:          containerPorts,
						ReadinessProbe: readinessProbe,
						WorkingDir:     service.WorkingDir,
					},
				},
			},
		}
		u.initResourceObjectMeta(&app.desiredPod.ObjectMeta, app.nameEncoded, name)
		if len(servicePorts) > 0 {
			app.desiredService = &v1.Service{
				Spec: v1.ServiceSpec{
					Ports: servicePorts,
					Selector: map[string]string{
						"app":                  app.nameEncoded,
						u.cfg.EnvironmentLabel: u.cfg.EnvironmentID,
					},
					// This is the default value.
					// Type: v1.ServiceType("ClusterIP"),
				},
			}
			u.initResourceObjectMeta(&app.desiredService.ObjectMeta, app.nameEncoded, name)
		}
		u.apps[name] = app
	}
	return nil
}

func (u *upRunner) initResourceObjectMeta(objectMeta *metav1.ObjectMeta, nameEncoded, name string) {
	objectMeta.Name = nameEncoded + "-" + u.cfg.EnvironmentID
	if objectMeta.Labels == nil {
		objectMeta.Labels = map[string]string{}
	}
	objectMeta.Labels["app"] = nameEncoded
	objectMeta.Labels[u.cfg.EnvironmentLabel] = u.cfg.EnvironmentID
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = map[string]string{}
	}
	objectMeta.Annotations[annotationName] = name
}

func (u *upRunner) writeResourceDebugFile(kind, name string, r interface{}) error {
	json, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(u.outputDir, name, kind+".json")
	os.MkdirAll(filepath.Dir(file), 0700)
	err = ioutil.WriteFile(file, json, 0600)
	if err != nil {
		return err
	}
	return nil
}

// https://docs.docker.com/engine/reference/builder/#healthcheck
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#configure-probes
func healthcheckToProbe(healthcheck *config.Healthcheck) *v1.Probe {
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
		execCommand[i] = healthcheck.Test[i-offset]
	}
	probe := &v1.Probe{
		FailureThreshold: retriesInt32,
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: execCommand,
			},
		},
		// InitialDelaySeconds must always be zero so we start the healthcheck immediately. Irrespective of Docker's StartPeriod we should set this to zero.
		InitialDelaySeconds: 0,

		PeriodSeconds:  int32(math.RoundToEven(healthcheck.Interval.Seconds())),
		TimeoutSeconds: int32(math.RoundToEven(healthcheck.Timeout.Seconds())),
		// This is the default value.
		// SuccessThreshold: 1,
	}
	return probe
}

func (u *upRunner) createServicesAndSetPodHostAliases() error {
	desiredServiceCount := 0
	for _, app := range u.apps {
		if app.desiredService != nil {
			desiredServiceCount++
		}
	}
	if desiredServiceCount > 0 {
		waitForClusterIPChannel := make(chan error)
		go func() {
			// TOOD perform watch after create service so that ResourceVersion is used and better efficiency is achieved
			// see https://stackoverflow.com/questions/52717497/correct-way-to-use-kubernetes-watches
			defer close(waitForClusterIPChannel)
			watch, err := u.k8sServiceClient.Watch(metav1.ListOptions{
				LabelSelector: u.cfg.EnvironmentLabel + "=" + u.cfg.EnvironmentID,
				Watch:         true,
			})
			if err != nil {
				waitForClusterIPChannel <- err
				return
			}
			defer watch.Stop()
			eventChannel := watch.ResultChan()
			remaining := desiredServiceCount
			for {
				event, ok := <-eventChannel
				if !ok {
					waitForClusterIPChannel <- fmt.Errorf("Channel unexpectedly closed")
					break
				}
				if event.Type == "ADDED" || event.Type == "MODIFIED" {
					service := event.Object.(*v1.Service)
					if len(service.Spec.ClusterIP) > 0 && service.Spec.Type == "ClusterIP" && service.ObjectMeta.Annotations != nil {
						if name, ok := service.ObjectMeta.Annotations[annotationName]; ok {
							if app, ok := u.apps[name]; ok && len(app.clusterIP) == 0 {
								app.clusterIP = service.Spec.ClusterIP
								remaining--
								if remaining == 0 {
									break
								}
							}
						}
					}
				} else if event.Type != "DELETED" {
					fmt.Println(event.Object)
				}
			}
		}()
		for _, app := range u.apps {
			if app.desiredService != nil {
				_, err := u.k8sServiceClient.Create(app.desiredService)
				if err != nil {
					return err
				}
				err = u.writeResourceDebugFile("Service", app.nameEncoded, app.desiredService)
				if err != nil {
					return err
				}
			}
		}
		err, ok := <-waitForClusterIPChannel
		if ok {
			return err
		}
		hostAliases := make([]v1.HostAlias, desiredServiceCount)
		i := 0
		for name, app := range u.apps {
			if app.desiredService != nil {
				hostAliases[i] = v1.HostAlias{
					IP: app.clusterIP,
					Hostnames: []string{
						name,
					},
				}
				i++
			}
		}
		for _, app := range u.apps {
			app.desiredPod.Spec.HostAliases = hostAliases
		}
	}
	return nil
}

func (u *upRunner) run() error {
	err := u.createServicesAndSetPodHostAliases()
	if err != nil {
		return err
	}
	// TODO depends_on???
	for _, app := range u.apps {
		_, err := u.k8sPodClient.Create(app.desiredPod)
		if err != nil {
			return err
		}
		err = u.writeResourceDebugFile("Pod", app.nameEncoded, app.desiredPod)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run runs an operation similar docker-compose up against a Kubernetes cluster.
func Run(cfg *config.Config) error {
	u := &upRunner{
		cfg: cfg,
	}
	err := u.initOutputDir()
	if err != nil {
		return err
	}
	err = u.initApps()
	if err != nil {
		return err
	}
	err = u.initKubernetesClientset()
	if err != nil {
		return err
	}

	return u.run()
}
