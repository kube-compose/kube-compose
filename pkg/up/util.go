package up

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
	"github.com/jbrekelmans/k8s-docker-compose/pkg/docker"
	v1 "k8s.io/api/core/v1"
)

// https://docs.docker.com/engine/reference/builder/#healthcheck
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#configure-probes
func createReadinessProbeFromDockerHealthcheck(healthcheck *config.Healthcheck) *v1.Probe {
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
		// The Shell is hardcoded by docker to be /bin/sh
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
		// InitialDelaySeconds must always be zero so we start the healthcheck immediately.
		// Irrespective of Docker's StartPeriod we should set this to zero.
		// If this was a liveness probe we would have to set InitialDelaySeconds to StartPeriod.
		InitialDelaySeconds: 0,

		PeriodSeconds:  int32(math.RoundToEven(healthcheck.Interval.Seconds())),
		TimeoutSeconds: int32(math.RoundToEven(healthcheck.Timeout.Seconds())),
		// This is the default value.
		// SuccessThreshold: 1,
	}
	return probe
}

func newFalsePointer() *bool {
	f := false
	return &f
}

func inspectImageRawParseHealthcheck(inspectRaw []byte) (*config.Healthcheck, error) {
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
	err := json.Unmarshal(inspectRaw, &inspectInfo)
	if err != nil {
		return nil, err
	}
	if len(inspectInfo.Config.Healthcheck.Test) == 0 || inspectInfo.Config.Healthcheck.Test[0] == config.HealthcheckCommandNone {
		return nil, nil
	}
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
	return healthcheck, nil
}

func inspectImageRaw(ctx context.Context, dockerClient *dockerClient.Client, repoTag string) ([]byte, string, error) {
	i := strings.LastIndexByte(repoTag, ':')
	if i < 0 {
		panic(fmt.Errorf("repoTag must have a tag"))
	}
	repo := repoTag[:i]
	filters := dockerFilters.NewArgs()
	filters.Add("reference", repo)
	imageList, err := dockerClient.ImageList(ctx, dockerTypes.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, "", err
	}
	imageID := ""
	for _, image := range imageList {
		for _, repoTag := range image.RepoTags {
			if repoTag == repoTag {
				imageID = image.ID
				break
			}
		}
		if len(imageID) > 0 {
			break
		}
	}
	if len(imageID) == 0 {
		return nil, "", nil
	}
	_, inspectRaw, err := dockerClient.ImageInspectWithRaw(ctx, imageID)
	return inspectRaw, imageID, err
}

func pullImageWithLogging(ctx context.Context, dockerClient *dockerClient.Client, appName, image string) (string, error) {
	lastLogTime := time.Now().Add(-2 * time.Second)
	digest, err := docker.PullImage(ctx, dockerClient, image, func(pull *docker.PullOrPush) {
		t := time.Now()
		elapsed := t.Sub(lastLogTime)
		if elapsed >= 2*time.Second {
			lastLogTime = t
			progress := pull.Progress()
			fmt.Printf("app %s: pulling image %s (%.1f%%)\n", appName, image, progress*100.0)
		}
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("app %s: pulling image %s (%.1f%%)   @%s\n", appName, image, 100.0, digest)
	return digest, nil
}

func pushImageWithLogging(ctx context.Context, dockerClient *dockerClient.Client, appName, image string) (string, error) {
	lastLogTime := time.Now().Add(-2 * time.Second)
	digest, err := docker.PushImage(ctx, dockerClient, image, func(push *docker.PullOrPush) {
		t := time.Now()
		elapsed := t.Sub(lastLogTime)
		if elapsed >= 2*time.Second {
			lastLogTime = t
			progress := push.Progress()
			fmt.Printf("app %s: pushing image %s (%.1f%%)\n", appName, image, progress*100.0)
		}
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("app %s: pushing image %s (%.1f%%) @%s\n", appName, image, 100.0, digest)
	return digest, err
}
