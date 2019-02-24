package up

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/docker"
	k8sUtil "github.com/jbrekelmans/k8s-docker-compose/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const annotationName = "k8s-docker-compose/service"

func errorResourcesModifiedExternally() error {
	return fmt.Errorf("one or more resources appear to have been modified by an external process, aborting")
}

type podStatus int

const (
	podStatusReady   podStatus = 2
	podStatusStarted podStatus = 1
	podStatusOther   podStatus = 0
)

func (podStatus *podStatus) String() string {
	switch *podStatus {
	case podStatusReady:
		return "ready"
	case podStatusStarted:
		return "started"
	}
	return "other"
}

type app struct {
	clusterIP            string
	desiredPod           *v1.Pod
	desiredPodImage      string
	desiredService       *v1.Service
	imageHealthcheck     *config.Healthcheck
	maxObservedPodStatus podStatus
	name                 string
	nameEncoded          string
}

type upRunner struct {
	apps             map[string]*app
	appsWithoutPods  map[*app]bool
	cfg              *config.Config
	dockerClient     *dockerClient.Client
	k8sClientset     *kubernetes.Clientset
	k8sServiceClient clientV1.ServiceInterface
	k8sPodClient     clientV1.PodInterface
	outputDir        string
}

func (u *upRunner) initKubernetesClientset() error {
	k8sClientset, err := kubernetes.NewForConfig(u.cfg.KubeConfig)
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
	u.apps = make(map[string]*app, len(u.cfg.CanonicalComposeFile.Services))
	u.appsWithoutPods = make(map[*app]bool, len(u.cfg.CanonicalComposeFile.Services))
	for name, service := range u.cfg.CanonicalComposeFile.Services {
		app := &app{
			name:        name,
			nameEncoded: k8sUtil.EncodeName(name),
		}
		u.appsWithoutPods[app] = true
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

		entrypoint := service.Entrypoint

		// TODO move this to a utility
		falseConst := false

		app.desiredPod = &v1.Pod{
			Spec: v1.PodSpec{
				AutomountServiceAccountToken: &falseConst,
				Containers: []v1.Container{
					v1.Container{
						Command:    entrypoint,
						Env:        envVars,
						Image:      service.Image,
						Name:       app.nameEncoded,
						Ports:      containerPorts,
						WorkingDir: service.WorkingDir,
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
			},
		}
		u.initResourceObjectMeta(&app.desiredPod.ObjectMeta, app.nameEncoded, name)
		u.writeResourceDebugFile("Pod", app.nameEncoded, app.desiredPod)
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
			err := u.writeResourceDebugFile("Service", app.nameEncoded, app.desiredService)
			if err != nil {
				return err
			}
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

func (u *upRunner) pushImage(ctx context.Context, app *app) error {
	sourceImage := u.cfg.CanonicalComposeFile.Services[app.name].Image
	if len(sourceImage) == 0 {
		return fmt.Errorf("service %s has no image or image is the empty string", app.name)
	}
	sourceImageCanonical, err := dockerRef.ParseNormalizedNamed(sourceImage)
	if err != nil {
		return err
	}
	destinationImage := fmt.Sprintf("%s/%s/%s", u.cfg.PushImages.DockerRegistry, u.cfg.Namespace, app.nameEncoded)
	destinationImagePush := destinationImage + ":" + u.cfg.EnvironmentID
	err = u.dockerClient.ImageTag(ctx, sourceImageCanonical.String(), destinationImagePush)
	if err != nil {
		message := fmt.Sprintf("%v", err)
		if !strings.HasPrefix(message, "Error response from daemon: No such image: ") {
			return err
		}
		lastLogTime := time.Now().Add(-2 * time.Second)
		digest, err := docker.PullImage(ctx, u.dockerClient, sourceImageCanonical.String(), func(pull *docker.PullOrPush) {
			t := time.Now()
			elapsed := t.Sub(lastLogTime)
			if elapsed >= 2*time.Second {
				lastLogTime = t
				progress := pull.Progress()
				fmt.Printf("app %s: pulling image %s (%.1f%%)\n", app.name, sourceImageCanonical, progress*100.0)
			}
		})
		if err != nil {
			return err
		}
		fmt.Printf("app %s: pulling image %s (%.1f%%) @%s\n", app.name, sourceImageCanonical, 100.0, digest)
		err = u.dockerClient.ImageTag(ctx, sourceImageCanonical.String(), destinationImagePush)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("app %s: already have image %s\n", app.name, sourceImage)
	}
	if u.cfg.PushImages != nil {
		lastLogTime := time.Now().Add(-2 * time.Second)
		digest, err := docker.PushImage(ctx, u.dockerClient, destinationImagePush, func(push *docker.PullOrPush) {
			t := time.Now()
			elapsed := t.Sub(lastLogTime)
			if elapsed >= 2*time.Second {
				lastLogTime = t
				progress := push.Progress()
				fmt.Printf("app %s: pushing image %s (%.1f%%)\n", app.name, destinationImagePush, progress*100.0)
			}
		})
		if err != nil {
			return err
		}
		fmt.Printf("app %s: pushing image %s (%.1f%%) @%s\n", app.name, destinationImagePush, 100.0, digest)
		app.desiredPod.Spec.Containers[0].Image = destinationImage + "@" + digest
		app.desiredPod.Spec.Containers[0].ImagePullPolicy = v1.PullAlways
	}
	filters := dockerFilters.NewArgs()
	filters.Add("reference", destinationImage)
	imageList, err := u.dockerClient.ImageList(ctx, dockerTypes.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return err
	}
	imageID := ""
	for _, image := range imageList {
		for _, repoTag := range image.RepoTags {
			if repoTag == destinationImagePush {
				imageID = image.ID
				break
			}
		}
		if len(imageID) > 0 {
			break
		}
	}
	if len(imageID) == 0 {
		return fmt.Errorf("error while trying to inspect image %s", sourceImageCanonical)
	}
	_, inspectRaw, err := u.dockerClient.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return err
	}
	var inspectInfo struct {
		Config struct {
			Healthcheck struct {
				Test     []string `json:"Test"`
				Timeout  *int64   `json:"Timeout"`
				Interval *int64   `json:"Interval"`
				Retries  *uint    `json:"Retries"`
			} `json:"Healthcheck"`
		} `json:"Config"`
	}
	err = json.Unmarshal(inspectRaw, &inspectInfo)
	if err != nil {
		return err
	}
	if inspectInfo.Config.Healthcheck.Test[0] != "NONE" {
		healthcheck := &config.Healthcheck{
			Interval: config.HealthcheckDefaultInterval,
			Timeout:  config.HealthcheckDefaultTimeout,
			Retries:  config.HealthcheckDefaultRetries,
		}
		if inspectInfo.Config.Healthcheck.Test[0] == config.HealthcheckCommandShell {
			healthcheck.IsShell = true
		}
		healthcheck.Test = inspectInfo.Config.Healthcheck.Test[1:]
		if inspectInfo.Config.Healthcheck.Timeout != nil {
			healthcheck.Timeout = time.Duration(*inspectInfo.Config.Healthcheck.Timeout)
		}
		if inspectInfo.Config.Healthcheck.Interval != nil {
			healthcheck.Interval = time.Duration(*inspectInfo.Config.Healthcheck.Interval)
		}
		if inspectInfo.Config.Healthcheck.Retries != nil {
			healthcheck.Retries = *inspectInfo.Config.Healthcheck.Retries
		}
		app.imageHealthcheck = healthcheck
	}
	var readinessProbe *v1.Probe
	service := u.cfg.CanonicalComposeFile.Services[app.name]
	if !service.HealthcheckDisabled {
		if service.Healthcheck != nil {
			readinessProbe = healthcheckToProbe(service.Healthcheck)
		} else if app.imageHealthcheck != nil {
			readinessProbe = healthcheckToProbe(app.imageHealthcheck)
		}
	}
	// TODO fix POD
	app.desiredPod.Spec.Containers[0].ReadinessProbe = readinessProbe
	return nil
}

func (u *upRunner) pushImageAsync(ctx context.Context, app *app) <-chan error {
	chn := make(chan error, 1)
	go func() {
		defer close(chn)
		err := u.pushImage(ctx, app)
		if err != nil {
			chn <- err
		}
	}()
	return chn
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

func (u *upRunner) findAppFromResourceObjectMeta(objectMeta *metav1.ObjectMeta) (*app, error) {
	if objectMeta.Annotations != nil {
		if name, ok := objectMeta.Annotations[annotationName]; ok {
			if app, ok := u.apps[name]; ok {
				return app, nil
			}
		}
	}
	nameEncoded := objectMeta.Name
	for _, app := range u.apps {
		if app.nameEncoded == nameEncoded {
			return nil, errorResourcesModifiedExternally()
		}
	}
	return nil, nil
}

func (u *upRunner) waitForServiceClusterIPUpdate(service *v1.Service) (*app, error) {
	app, err := u.findAppFromResourceObjectMeta(&service.ObjectMeta)
	if err != nil || app == nil {
		return app, err
	}
	if service.Spec.Type != "ClusterIP" {
		return app, errorResourcesModifiedExternally()
	}
	app.clusterIP = service.Spec.ClusterIP
	return app, nil
}

func (u *upRunner) waitForServiceClusterIPCountRemaining() int {
	remaining := 0
	for _, app := range u.apps {
		if app.desiredService != nil && len(app.clusterIP) == 0 {
			remaining++
		}
	}
	return remaining
}

func (u *upRunner) waitForServiceClusterIP(expected int) error {
	listOptions := metav1.ListOptions{
		LabelSelector: u.cfg.EnvironmentLabel + "=" + u.cfg.EnvironmentID,
	}
	serviceList, err := u.k8sServiceClient.List(listOptions)
	if err != nil {
		return err
	}
	if len(serviceList.Items) < expected {
		return errorResourcesModifiedExternally()
	}
	for _, service := range serviceList.Items {
		_, err = u.waitForServiceClusterIPUpdate(&service)
		if err != nil {
			return err
		}
	}
	remaining := u.waitForServiceClusterIPCountRemaining()
	if remaining == 0 {
		fmt.Printf("waiting for cluster IP assignment (%d/%d)\n", expected, expected)
		return nil
	}
	fmt.Printf("waiting for cluster IP assignment (%d/%d)\n", expected-remaining, expected)
	listOptions.ResourceVersion = serviceList.ResourceVersion
	listOptions.Watch = true
	watch, err := u.k8sServiceClient.Watch(listOptions)
	if err != nil {
		return err
	}
	defer watch.Stop()
	eventChannel := watch.ResultChan()
	for {
		event, ok := <-eventChannel
		if !ok {
			return fmt.Errorf("channel unexpectedly closed")
		}
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			service := event.Object.(*v1.Service)
			_, err := u.waitForServiceClusterIPUpdate(service)
			if err != nil {
				return err
			}
		} else if event.Type == "DELETED" {
			service := event.Object.(*v1.Service)
			app, err := u.findAppFromResourceObjectMeta(&service.ObjectMeta)
			if err != nil {
				return err
			}
			if app != nil {
				return errorResourcesModifiedExternally()
			}
		} else {
			fmt.Printf("got unexpected error event from channel: ")
			fmt.Println(event.Object)
			return fmt.Errorf("got unexpected error event from channel")
		}
		remainingNew := u.waitForServiceClusterIPCountRemaining()
		if remainingNew != remaining {
			remaining = remainingNew
			fmt.Printf("waiting for cluster IP assignment (%d/%d)\n", expected-remaining, expected)
			if remaining == 0 {
				break
			}
		}
	}
	return nil
}

func (u *upRunner) createServicesAndSetPodHostAliases() error {
	desiredServiceCount := 0
	for _, app := range u.apps {
		if app.desiredService != nil {
			desiredServiceCount++
		}
	}
	if desiredServiceCount == 0 {
		return nil
	}
	for _, app := range u.apps {
		if app.desiredService != nil {
			_, err := u.k8sServiceClient.Create(app.desiredService)
			if err != nil {
				return err
			}
			fmt.Printf("app %s: created service %s\n", app.name, app.desiredService.ObjectMeta.Name)
		}
	}
	err := u.waitForServiceClusterIP(desiredServiceCount)
	if err != nil {
		return err
	}
	hostAliases := make([]v1.HostAlias, desiredServiceCount)
	i := 0
	for _, app := range u.apps {
		if app.desiredService != nil {
			hostAliases[i] = v1.HostAlias{
				IP: app.clusterIP,
				Hostnames: []string{
					app.name,
				},
			}
			i++
		}
	}
	for _, app := range u.apps {
		app.desiredPod.Spec.HostAliases = hostAliases
	}
	return nil
}

func (u *upRunner) createAppPod(app *app, reason string) error {
	_, err := u.k8sPodClient.Create(app.desiredPod)
	if err != nil {
		return err
	}
	fmt.Printf("app %s: created pod %s because %s\n", app.name, app.desiredPod.ObjectMeta.Name, reason)
	return nil
}

func parsePodStatus(pod *v1.Pod) (podStatus, error) {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return podStatusReady, nil
		}
	}
	runningCount := 0
	for _, containerStatus := range pod.Status.ContainerStatuses {
		t := containerStatus.State.Terminated
		if t != nil {
			return podStatusOther, fmt.Errorf("aborting because container %s of pod %s terminated (code=%d,signal=%d,reason=%s): %s",
				containerStatus.Name,
				pod.ObjectMeta.Name,
				t.ExitCode,
				t.Signal,
				t.Reason,
				t.Message,
			)
		}

		if w := containerStatus.State.Waiting; w != nil && w.Reason == "ErrImagePull" {
			return podStatusOther, fmt.Errorf("aborting because container %s of pod %s could not pull image: %s",
				containerStatus.Name,
				pod.ObjectMeta.Name,
				w.Message,
			)
		}
		if containerStatus.State.Running != nil {
			runningCount++
		}
	}
	if runningCount == len(pod.Status.ContainerStatuses) {
		return podStatusStarted, nil
	}
	return podStatusOther, nil
}

func (u *upRunner) updateAppMaxObservedPodStatus(pod *v1.Pod) error {
	app, err := u.findAppFromResourceObjectMeta(&pod.ObjectMeta)
	if err != nil {
		return err
	}
	if app == nil {
		return nil
	}
	podStatus, err := parsePodStatus(pod)
	if err != nil {
		return err
	}
	if podStatus > app.maxObservedPodStatus {
		app.maxObservedPodStatus = podStatus
		fmt.Printf("app %s: pod status %s\n", app.name, &app.maxObservedPodStatus)
	}
	return nil
}

func (u *upRunner) createAppPodsIfNeeded() error {
	for app1 := range u.appsWithoutPods {
		dependsOn := u.cfg.CanonicalComposeFile.Services[app1.name].DependsOn
		createAppPod := true
		for service, healthiness := range dependsOn {
			app2 := u.apps[service.ServiceName]
			if healthiness == config.ServiceHealthy {
				if app2.maxObservedPodStatus != podStatusReady {
					createAppPod = false
				}
			} else {
				if app2.maxObservedPodStatus != podStatusStarted {
					createAppPod = false
				}
			}
		}
		if createAppPod {
			reason := strings.Builder{}
			reason.WriteString("its dependency conditions are met (")
			comma := false
			for service, healthiness := range dependsOn {
				if comma {
					reason.WriteString(", ")
				}
				reason.WriteString(service.ServiceName)
				if healthiness == config.ServiceHealthy {
					reason.WriteString(": healthy/ready")
				} else {
					reason.WriteString(": started")
				}
				comma = true
			}
			reason.WriteString(")")
			err := u.createAppPod(app1, reason.String())
			if err != nil {
				return err
			}
			delete(u.appsWithoutPods, app1)
		}
	}
	return nil
}

func (u *upRunner) doImageLogic() error {
	ctx := context.Background()
	n := len(u.apps)
	chnSlice := make([]<-chan error, n)
	i := 0
	for _, app := range u.apps {
		chnSlice[i] = u.pushImageAsync(ctx, app)
		i++
	}
	i = 0
	var lastError error
	for i < n {
		err := <-chnSlice[i]
		if err != nil {
			if lastError == nil {
				lastError = err
			} else {
				fmt.Println(err)
			}
		}
		i++
	}
	return lastError
}

func (u *upRunner) run() error {
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
	// Initialize docker client
	dockerClient, err := dockerClient.NewEnvClient()
	if err != nil {
		return err
	}
	u.dockerClient = dockerClient

	err = u.doImageLogic()
	if err != nil {
		return err
	}

	// Start actual deployment...
	err = u.createServicesAndSetPodHostAliases()
	if err != nil {
		return err
	}

	for _, app := range u.apps {
		if len(u.cfg.CanonicalComposeFile.Services[app.name].DependsOn) == 0 {
			u.createAppPod(app, "no dependencies")
			delete(u.appsWithoutPods, app)
		}
	}

	listOptions := metav1.ListOptions{
		LabelSelector: u.cfg.EnvironmentLabel + "=" + u.cfg.EnvironmentID,
	}
	podList, err := u.k8sPodClient.List(listOptions)
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		err = u.updateAppMaxObservedPodStatus(&pod)
		if err != nil {
			return err
		}
	}
	err = u.createAppPodsIfNeeded()
	if err != nil {
		return err
	}
	listOptions.ResourceVersion = podList.ResourceVersion
	listOptions.Watch = true
	watch, err := u.k8sPodClient.Watch(listOptions)
	if err != nil {
		return err
	}
	defer watch.Stop()
	eventChannel := watch.ResultChan()
	for {
		event, ok := <-eventChannel
		if !ok {
			return fmt.Errorf("channel unexpectedly closed")
		}
		if event.Type == "ADDED" || event.Type == "MODIFIED" {
			pod := event.Object.(*v1.Pod)
			err = u.updateAppMaxObservedPodStatus(pod)
			if err != nil {
				return err
			}
		} else if event.Type == "DELETED" {
			pod := event.Object.(*v1.Pod)
			app, err := u.findAppFromResourceObjectMeta(&pod.ObjectMeta)
			if err != nil {
				return err
			}
			if app != nil {
				return errorResourcesModifiedExternally()
			}
		} else {
			fmt.Printf("got unexpected error event from channel: ")
			fmt.Println(event.Object)
			return fmt.Errorf("got unexpected error event from channel")
		}
		err = u.createAppPodsIfNeeded()
		if err != nil {
			return err
		}
		allPodsReady := true
		for _, app := range u.apps {
			if app.maxObservedPodStatus != podStatusReady {
				allPodsReady = false
			}
		}
		if allPodsReady {
			break
		}
	}
	fmt.Printf("pods ready (%d/%d)\n", len(u.apps), len(u.apps))
	return nil
}

// Run runs an operation similar docker-compose up against a Kubernetes cluster.
func Run(cfg *config.Config) error {
	u := &upRunner{
		cfg: cfg,
	}
	return u.run()
}
