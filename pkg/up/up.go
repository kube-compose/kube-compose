package up

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	"github.com/jbrekelmans/kube-compose/pkg/config"
	k8sUtil "github.com/jbrekelmans/kube-compose/pkg/k8s"
	goDigest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8swatch "k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/kubernetes"
	clientV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	k8sError "k8s.io/apimachinery/pkg/api/errors"
)

func errorResourcesModifiedExternally() error {
	return fmt.Errorf("one or more resources appear to have been modified by an external process, aborting")
}

type podStatus int

const (
	annotationName               = "kube-compose/service"
	podStatusReady     podStatus = 2
	podStatusStarted   podStatus = 1
	podStatusOther     podStatus = 0
	podStatusCompleted podStatus = 3
)

func (podStatus *podStatus) String() string {
	switch *podStatus {
	case podStatusReady:
		return "ready"
	case podStatusStarted:
		return "started"
	case podStatusCompleted:
		return "completed"
	}
	return "other"
}

type appImageInfo struct {
	err              error
	imageHealthcheck *config.Healthcheck
	once             *sync.Once
	podImage         string
	user             *userinfo
}

type app struct {
	serviceClusterIP                     string
	imageInfo                            appImageInfo
	hasService                           bool
	maxObservedPodStatus                 podStatus
	name                                 string
	nameEncoded                          string
	containersForWhichWeAreStreamingLogs map[string]bool
}

type hostAliasesOrError struct {
	v   []v1.HostAlias
	err error
}

type localImagesCache struct {
	imageIDSet *digestset.Set
	images     []dockerTypes.ImageSummary
	once       *sync.Once
	err        error
}

type upRunner struct {
	apps                  map[string]*app
	appsThatNeedToBeReady map[*app]bool
	appsWithoutPods       map[*app]bool
	cfg                   *config.Config
	ctx                   context.Context
	dockerClient          *dockerClient.Client
	localImagesCache      localImagesCache
	k8sClientset          *kubernetes.Clientset
	k8sServiceClient      clientV1.ServiceInterface
	k8sPodClient          clientV1.PodInterface
	hostAliasesOnce       *sync.Once
	hostAliases           hostAliasesOrError
	completedChannels     []chan interface{}
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

func (u *upRunner) initAppsToBeStarted() error {
	appNames := make([]string, len(u.apps))
	podsRequired := []string{}
	n := 0
	for _, app := range u.apps {
		if contains(u.cfg.Services, app.name) {
			appNames[n] = app.name
			n++
			for n > 0 {
				appName := appNames[n-1]
				n--
				for dependencyApp := range u.cfg.CanonicalComposeFile.Services[appName].DependsOn {
					appNames[n] = u.apps[dependencyApp.ServiceName].name
					n++
				}
				podsRequired = append(podsRequired, appName)
			}
		} else if len(u.cfg.Services) == 0 {
			podsRequired = append(podsRequired, app.name)
		}
	}
	for app := range u.appsWithoutPods {
		if !contains(podsRequired, app.name) {
			delete(u.appsWithoutPods, app)
		}
	}
	return nil
}

func (u *upRunner) initApps() error {
	u.apps = make(map[string]*app, len(u.cfg.CanonicalComposeFile.Services))
	u.appsThatNeedToBeReady = map[*app]bool{}
	u.appsWithoutPods = make(map[*app]bool, len(u.cfg.CanonicalComposeFile.Services))
	for name, dcService := range u.cfg.CanonicalComposeFile.Services {
		app := &app{
			name:                                 name,
			nameEncoded:                          k8sUtil.EncodeName(name),
			containersForWhichWeAreStreamingLogs: make(map[string]bool),
		}
		app.imageInfo.once = &sync.Once{}
		u.appsWithoutPods[app] = true
		app.hasService = len(dcService.Ports) > 0
		u.apps[name] = app
	}
	if err := u.initAppsToBeStarted(); err != nil {
		return err
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

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) getAppImageInfo(app *app) error {
	sourceImage := u.cfg.CanonicalComposeFile.Services[app.name].Image
	if sourceImage == "" {
		return fmt.Errorf("docker compose service %s has no image or is a empty string, and building images is not supported",
			app.name)
	}
	localImageIDSet, err := u.getLocalImageIDSet()
	if err != nil {
		return err
	}
	// Use the same interpretation of images as docker-compose (use ParseAnyReferenceWithSet)
	sourceImageRef, err := dockerRef.ParseAnyReferenceWithSet(sourceImage, localImageIDSet)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error while parsing image %s", sourceImage))
	}

	// We need the image locally always, so we can parse its healthcheck
	sourceImageNamed, sourceImageIsNamed := sourceImageRef.(dockerRef.Named)
	sourceImageID := resolveLocalImageID(sourceImageRef, localImageIDSet, u.localImagesCache.images)

	var podImage string
	if sourceImageID == "" {
		if !sourceImageIsNamed {
			return fmt.Errorf("could not find image %s locally, and building images is not supported", sourceImage)
		}
		var digest string
		digest, err = pullImageWithLogging(u.ctx, u.dockerClient, app.name, sourceImageRef.String())
		if err != nil {
			return err
		}
		sourceImageID, podImage, err = resolveLocalImageAfterPull(u.ctx, u.dockerClient, sourceImageNamed, digest)
		if err != nil {
			return err
		}
		if sourceImageID == "" {
			return fmt.Errorf("could get ID of image %s, this is either a bug or images were removed by an external process (please try again)",
				sourceImage)
		}
		// len(podImage) > 0 by definition of resolveLocalImageAfterPull
	}
	inspect, inspectRaw, err := u.dockerClient.ImageInspectWithRaw(u.ctx, sourceImageID)
	if err != nil {
		return err
	}
	if u.cfg.PushImages != nil {
		destinationImage := fmt.Sprintf("%s/%s/%s", u.cfg.PushImages.DockerRegistry, u.cfg.Namespace, app.nameEncoded)
		destinationImagePush := destinationImage + ":latest"
		err = u.dockerClient.ImageTag(u.ctx, sourceImageID, destinationImagePush)
		if err != nil {
			return err
		}
		var digest string
		digest, err = pushImageWithLogging(u.ctx, u.dockerClient, app.name,
			destinationImagePush,
			u.cfg.KubeConfig.BearerToken)
		if err != nil {
			return err
		}
		podImage = destinationImage + "@" + digest
	} else if podImage == "" {
		if !sourceImageIsNamed {
			// TODO https://github.com/jbrekelmans/kube-compose/issues/6
			return fmt.Errorf("image reference %s is likely unstable, please enable pushing of images or use named image references to "+
				"improve reliability", sourceImage)
		}
		podImage = sourceImage
	}
	app.imageInfo.podImage = podImage
	imageHealthcheck, err := inspectImageRawParseHealthcheck(inspectRaw)
	if err != nil {
		return err
	}
	app.imageInfo.imageHealthcheck = imageHealthcheck

	if u.cfg.RunAsUser {
		var user *userinfo
		if u.cfg.CanonicalComposeFile.Services[app.name].User == nil {
			user, err = parseUserinfo(inspect.Config.User)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("image %s has an invalid user %#v", sourceImageRef.String(), inspect.Config.User))
			}
		} else {
			user, err = parseUserinfo(*u.cfg.CanonicalComposeFile.Services[app.name].User)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("service %s in the docker-compose file has an invalid user", app.name))
			}
		}
		if user.UID == nil || (user.Group != "" && user.GID == nil) {
			// TODO https://github.com/jbrekelmans/kube-compose/issues/70 confirm whether docker and our pod spec will produce the same default
			// group if a UID is set but no GID
			err = getUserinfoFromImage(u.ctx, u.dockerClient, sourceImageID, user)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error getting uid/gid from file system of image %s", sourceImageRef.String()))
			}
		}
		app.imageInfo.user = user
	}
	return nil
}

func (u *upRunner) getAppImageInfoOnce(app *app) error {
	app.imageInfo.once.Do(func() {
		app.imageInfo.err = u.getAppImageInfo(app)
	})
	return app.imageInfo.err
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
	app.serviceClusterIP = service.Spec.ClusterIP
	return app, nil
}

func (u *upRunner) waitForServiceClusterIPCountRemaining() int {
	remaining := 0
	for _, app := range u.apps {
		if app.hasService && app.serviceClusterIP == "" {
			remaining++
		}
	}
	return remaining
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
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
	for i := 0; i < len(serviceList.Items); i++ {
		_, err = u.waitForServiceClusterIPUpdate(&serviceList.Items[i])
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
		switch event.Type {
		case k8swatch.Added, k8swatch.Modified:
			service := event.Object.(*v1.Service)
			_, err := u.waitForServiceClusterIPUpdate(service)
			if err != nil {
				return err
			}
		case k8swatch.Deleted:
			service := event.Object.(*v1.Service)
			app, err := u.findAppFromResourceObjectMeta(&service.ObjectMeta)
			if err != nil {
				return err
			}
			if app != nil {
				return errorResourcesModifiedExternally()
			}
		default:
			return fmt.Errorf("got unexpected error event from channel: %+v", event.Object)
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

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) createServicesAndGetPodHostAliases() ([]v1.HostAlias, error) {
	expectedServiceCount := 0
	for _, app := range u.apps {
		if !app.hasService {
			continue
		}
		expectedServiceCount++
		dcService := u.cfg.CanonicalComposeFile.Services[app.name]
		servicePorts := make([]v1.ServicePort, len(dcService.Ports))
		for i, port := range dcService.Ports {
			servicePorts[i] = v1.ServicePort{
				Name:       fmt.Sprintf("%s%d", port.Protocol, port.Internal),
				Port:       port.Internal,
				Protocol:   v1.Protocol(strings.ToUpper(port.Protocol)),
				TargetPort: intstr.FromInt(int(port.Internal)),
			}
		}
		service := &v1.Service{
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
		u.initResourceObjectMeta(&service.ObjectMeta, app.nameEncoded, app.name)
		_, err := u.k8sServiceClient.Create(service)
		switch {
		case k8sError.IsAlreadyExists(err):
			fmt.Printf("app %s: service %s already exists\n", app.name, service.ObjectMeta.Name)
		case err != nil:
			return nil, err
		default:
			fmt.Printf("app %s: created service %s\n", app.name, service.ObjectMeta.Name)
		}
	}
	if expectedServiceCount == 0 {
		return nil, nil
	}
	err := u.waitForServiceClusterIP(expectedServiceCount)
	if err != nil {
		return nil, err
	}
	hostAliases := make([]v1.HostAlias, expectedServiceCount)
	i := 0
	for _, app := range u.apps {
		if app.hasService {
			hostAliases[i] = v1.HostAlias{
				IP: app.serviceClusterIP,
				Hostnames: []string{
					app.name,
				},
			}
			i++
		}
	}
	return hostAliases, nil
}

func (u *upRunner) initLocalImages() error {
	u.localImagesCache.once.Do(func() {
		imageSummarySlice, err := u.dockerClient.ImageList(u.ctx, dockerTypes.ImageListOptions{
			All: true,
		})
		var imageIDSet *digestset.Set
		if err == nil {
			imageIDSet = digestset.NewSet()
			for i := 0; i < len(imageSummarySlice); i++ {
				_ = imageIDSet.Add(goDigest.Digest(imageSummarySlice[i].ID))
			}
		}
		u.localImagesCache = localImagesCache{
			imageIDSet: imageIDSet,
			images:     imageSummarySlice,
			err:        err,
		}
	})
	return u.localImagesCache.err
}

func (u *upRunner) getLocalImageIDSet() (*digestset.Set, error) {
	err := u.initLocalImages()
	if err != nil {
		return nil, err
	}
	return u.localImagesCache.imageIDSet, nil
}

func (u *upRunner) createServicesAndGetPodHostAliasesOnce() ([]v1.HostAlias, error) {
	u.hostAliasesOnce.Do(func() {
		hostAliases, err := u.createServicesAndGetPodHostAliases()
		u.hostAliases = hostAliasesOrError{
			v:   hostAliases,
			err: err,
		}
	})
	return u.hostAliases.v, u.hostAliases.err
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) createPod(app *app) (*v1.Pod, error) {
	err := u.getAppImageInfoOnce(app)
	if err != nil {
		return nil, err
	}
	dcService := u.cfg.CanonicalComposeFile.Services[app.name]

	// We convert the image/docker-compose healthcheck to a readiness probe to implement
	// depends_on condition: service_healthy in docker compose files.
	// Kubernetes does not appear to have disabled the healthcheck of docker images:
	// https://stackoverflow.com/questions/41475088/when-to-use-docker-healthcheck-vs-livenessprobe-readinessprobe
	// ... so we're not doubling up on healthchecks.
	// We accept that this may lead to calls failing due to removal backend pods from load balancers.
	var readinessProbe *v1.Probe
	if !dcService.HealthcheckDisabled {
		if dcService.Healthcheck != nil {
			readinessProbe = createReadinessProbeFromDockerHealthcheck(dcService.Healthcheck)
		} else if app.imageInfo.imageHealthcheck != nil {
			readinessProbe = createReadinessProbeFromDockerHealthcheck(app.imageInfo.imageHealthcheck)
		}
	}
	var containerPorts []v1.ContainerPort
	if len(dcService.Ports) > 0 {
		containerPorts = make([]v1.ContainerPort, len(dcService.Ports))
		for i, port := range dcService.Ports {
			containerPorts[i] = v1.ContainerPort{
				ContainerPort: port.Internal,
				Protocol:      v1.Protocol(strings.ToUpper(port.Protocol)),
			}
		}
	}
	var envVars []v1.EnvVar
	envVarCount := len(dcService.Environment)
	if envVarCount > 0 {
		envVars = make([]v1.EnvVar, envVarCount)
		i := 0
		for key, value := range dcService.Environment {
			envVars[i] = v1.EnvVar{
				Name:  key,
				Value: value,
			}
			i++
		}
	}
	hostAliases, err := u.createServicesAndGetPodHostAliasesOnce()
	if err != nil {
		return nil, err
	}

	var securityContext *v1.PodSecurityContext
	if u.cfg.RunAsUser {
		securityContext = &v1.PodSecurityContext{
			RunAsUser: app.imageInfo.user.UID,
		}
		if app.imageInfo.user.GID != nil {
			securityContext.RunAsGroup = app.imageInfo.user.GID
		}
	}
	pod := &v1.Pod{
		Spec: v1.PodSpec{
			// new(bool) allocates a bool, sets it to false, and returns a pointer to it.
			AutomountServiceAccountToken: new(bool),
			Containers: []v1.Container{
				{
					Command:         dcService.Entrypoint,
					Env:             envVars,
					Image:           app.imageInfo.podImage,
					ImagePullPolicy: v1.PullAlways,
					Name:            app.nameEncoded,
					Ports:           containerPorts,
					ReadinessProbe:  readinessProbe,
					WorkingDir:      dcService.WorkingDir,
				},
			},
			HostAliases:     hostAliases,
			RestartPolicy:   v1.RestartPolicyNever,
			SecurityContext: securityContext,
		},
	}
	u.initResourceObjectMeta(&pod.ObjectMeta, app.nameEncoded, app.name)
	podServer, err := u.k8sPodClient.Create(pod)
	if k8sError.IsAlreadyExists(err) {
		fmt.Printf("app %s: pod %s already exists\n", app.name, app.nameEncoded)
	} else if err != nil {
		return nil, err
	}
	u.appsThatNeedToBeReady[app] = true
	return podServer, nil
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
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
			if t.Reason != "Completed" {
				return podStatusOther, fmt.Errorf("aborting because container %s of pod %s terminated (code=%d,signal=%d,reason=%s): %s",
					containerStatus.Name,
					pod.ObjectMeta.Name,
					t.ExitCode,
					t.Signal,
					t.Reason,
					t.Message,
				)
			}
			return podStatusCompleted, nil
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

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) updateAppMaxObservedPodStatus(pod *v1.Pod) error {

	app, err := u.findAppFromResourceObjectMeta(&pod.ObjectMeta)
	if err != nil {
		return err
	}
	if app == nil {
		return nil
	}
	// For each container of the pod:
	// 		if the container is running
	//			// use app.containersForWhichWeAreStreamingLogs to determine the following condition
	// 			if we are not already streaming logs for the container
	//				start streaming logs for the container
	if !u.cfg.Detach {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			_, ok := app.containersForWhichWeAreStreamingLogs[containerStatus.Name]
			if !ok && containerStatus.State.Running != nil {
				app.containersForWhichWeAreStreamingLogs[containerStatus.Name] = true
				getPodLogOptions := &v1.PodLogOptions{
					Follow:    true,
					Container: containerStatus.Name,
				}
				completedChannel := make(chan interface{})
				u.completedChannels = append(u.completedChannels, completedChannel)
				go func() {
					getLogsRequest := u.k8sPodClient.GetLogs(pod.ObjectMeta.Name, getPodLogOptions)
					var bodyReader io.ReadCloser
					bodyReader, err = getLogsRequest.Stream()
					if err != nil {
						panic(err)
					}
					defer util.CloseAndLogError(bodyReader)
					scanner := bufio.NewScanner(bodyReader)
					for scanner.Scan() {
						logline := app.name + " | " + scanner.Text()
						fmt.Println(logline)
					}
					if err = scanner.Err(); err != nil {
						fmt.Println(err)
					}
					close(completedChannel)
				}()
			}
		}
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

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) createPodsIfNeeded() error {
	for app1 := range u.appsWithoutPods {
		dependsOn := u.cfg.CanonicalComposeFile.Services[app1.name].DependsOn
		createPod := true
		for dcService, healthiness := range dependsOn {
			app2 := u.apps[dcService.ServiceName]
			if healthiness == config.ServiceHealthy {
				if app2.maxObservedPodStatus != podStatusReady {
					createPod = false
				}
			} else {
				if app2.maxObservedPodStatus != podStatusStarted {
					createPod = false
				}
			}
		}
		if createPod {
			reason := strings.Builder{}
			reason.WriteString("its dependency conditions are met (")
			comma := false
			for dcService, healthiness := range dependsOn {
				if comma {
					reason.WriteString(", ")
				}
				reason.WriteString(dcService.ServiceName)
				if healthiness == config.ServiceHealthy {
					reason.WriteString(": ready")
				} else {
					reason.WriteString(": running")
				}
				comma = true
			}
			reason.WriteString(")")
			pod, err := u.createPod(app1)
			if err != nil {
				return err
			}
			fmt.Printf("app %s: created pod %s because %s\n", app1.name, pod.ObjectMeta.Name, reason.String())
			delete(u.appsWithoutPods, app1)
		}
	}
	return nil
}

// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
// nolint
func (u *upRunner) run() error {
	err := u.initApps()
	if err != nil {
		return err
	}
	err = u.initKubernetesClientset()
	if err != nil {
		return err
	}
	// Initialize docker client
	var dc *dockerClient.Client
	dc, err = dockerClient.NewEnvClient()
	if err != nil {
		return err
	}
	u.dockerClient = dc

	for app := range u.appsWithoutPods {
		// Begin pulling and pushing images immediately...
		//nolint
		go u.getAppImageInfoOnce(app)
	}
	// Begin creating services and collecting their cluster IPs (we'll need this to
	// set the hostAliases of each pod)
	// nolint
	go u.createServicesAndGetPodHostAliasesOnce()
	for app := range u.appsWithoutPods {
		if len(u.cfg.CanonicalComposeFile.Services[app.name].DependsOn) != 0 {
			continue
		}
		var pod *v1.Pod
		pod, err = u.createPod(app)
		if err != nil {
			return err
		}
		fmt.Printf("app %s: created pod %s because all its dependency conditions are met\n", app.name, pod.ObjectMeta.Name)
		delete(u.appsWithoutPods, app)

	}

	listOptions := metav1.ListOptions{
		LabelSelector: u.cfg.EnvironmentLabel + "=" + u.cfg.EnvironmentID,
	}
	podList, err := u.k8sPodClient.List(listOptions)
	if err != nil {
		return err
	}
	for i := 0; i < len(podList.Items); i++ {
		err = u.updateAppMaxObservedPodStatus(&podList.Items[i])
		if err != nil {
			return err
		}
	}
	err = u.createPodsIfNeeded()
	if err != nil {
		return err
	}

	if u.checkIfPodsReady() {
		fmt.Printf("pods ready (%d/%d)\n", len(u.appsThatNeedToBeReady), len(u.appsThatNeedToBeReady))
		return nil
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
		switch event.Type {
		case k8swatch.Added, k8swatch.Modified:
			pod := event.Object.(*v1.Pod)
			err = u.updateAppMaxObservedPodStatus(pod)
			if err != nil {
				return err
			}
		case k8swatch.Deleted:
			pod := event.Object.(*v1.Pod)
			var app *app
			app, err = u.findAppFromResourceObjectMeta(&pod.ObjectMeta)
			if err != nil {
				return err
			}
			if app != nil {
				return errorResourcesModifiedExternally()
			}
		default:
			return fmt.Errorf("got unexpected error event from channel: %+v", event.Object)
		}
		err = u.createPodsIfNeeded()
		if err != nil {
			return err
		}

		if u.checkIfPodsReady() {
			break
		}
	}
	fmt.Printf("pods ready (%d/%d)\n", len(u.appsThatNeedToBeReady), len(u.appsThatNeedToBeReady))
	// Wait for completed channels
	for _, completedChannel := range u.completedChannels {
		<-completedChannel
	}

	return nil
}

func (u *upRunner) checkIfPodsReady() bool {
	allPodsReady := true
	for app := range u.appsThatNeedToBeReady {
		if app.maxObservedPodStatus < podStatusReady {
			allPodsReady = false
		}
	}
	return allPodsReady
}

// Run runs an operation similar docker-compose up against a Kubernetes cluster.
func Run(cfg *config.Config) error {
	// TODO https://github.com/jbrekelmans/kube-compose/issues/2 accept context as a parameter
	u := &upRunner{
		cfg:             cfg,
		ctx:             context.Background(),
		hostAliasesOnce: &sync.Once{},
	}
	u.localImagesCache.once = &sync.Once{}
	return u.run()
}
