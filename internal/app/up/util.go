package up

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"time"

	log "github.com/Sirupsen/logrus"
	containerService "github.com/kube-compose/kube-compose/internal/pkg/container/service"
	"github.com/kube-compose/kube-compose/internal/pkg/docker"
	"github.com/kube-compose/kube-compose/internal/pkg/unix"
	dockerComposeConfig "github.com/kube-compose/kube-compose/pkg/docker/compose/config"
	v1 "k8s.io/api/core/v1"
)

// https://docs.docker.com/engine/reference/builder/#healthcheck
// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#configure-probes
func createReadinessProbeFromDockerHealthcheck(healthcheck *dockerComposeConfig.Healthcheck) *v1.Probe {
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
		// Add 2 to accommodate for /bin/sh -c
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

func inspectImageRawParseHealthcheck(inspectRaw []byte) (*dockerComposeConfig.Healthcheck, error) {
	// inspectInfo's type is similar to dockerClient.ImageInspect, but it allows us to detect absent fields so we can apply default values.
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
	if len(inspectInfo.Config.Healthcheck.Test) == 0 || inspectInfo.Config.Healthcheck.Test[0] == dockerComposeConfig.HealthcheckCommandNone {
		return nil, nil
	}
	healthcheck := &dockerComposeConfig.Healthcheck{
		Interval: dockerComposeConfig.HealthcheckDefaultInterval,
		Timeout:  dockerComposeConfig.HealthcheckDefaultTimeout,
		Retries:  dockerComposeConfig.HealthcheckDefaultRetries,
	}
	if inspectInfo.Config.Healthcheck.Test[0] == dockerComposeConfig.HealthcheckCommandShell {
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

func getUserinfoFromImage(
	ctx context.Context,
	cs containerService.ContainerService,
	image string,
	user *docker.Userinfo,
) error {
	containerID, err := cs.ContainerCreateForCopyFromContainer(ctx, image)
	if err != nil {
		return err
	}
	defer func() {
		err = cs.ContainerRemove(ctx, containerID)
		if err != nil {
			log.Error(err)
		}
	}()
	tmpDir, err := ioutil.TempDir("", "kube-compose-")
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.Error(err)
		}
	}()
	err = getUserinfoFromImageUID(ctx, cs, containerID, tmpDir, user)
	if err != nil {
		return err
	}
	return getUserinfoFromImageGID(ctx, cs, containerID, tmpDir, user)
}

func getUserinfoFromImageUID(
	ctx context.Context,
	cs containerService.ContainerService,
	containerID, tmpDir string,
	user *docker.Userinfo,
) error {
	// TODO https://github.com/kube-compose/kube-compose/issues/70 this is not correct for non-Linux containers
	if user.UID == nil {
		err := cs.CopyFromContainerToFile(ctx, containerID, unix.EtcPasswd, tmpDir)
		if err != nil {
			return err
		}
		var uid *int64
		uid, err = unix.FindUIDByNameInPasswd(path.Join(tmpDir, "passwd"), user.User)
		if err != nil {
			return err
		}
		if uid == nil {
			return fmt.Errorf("linux spec user: unable to find user %s no matching entries in passwd file", user.User)
		}
		user.UID = uid
	}
	return nil
}

func getUserinfoFromImageGID(
	ctx context.Context,
	cs containerService.ContainerService,
	containerID, tmpDir string,
	user *docker.Userinfo,
) error {
	// TODO https://github.com/kube-compose/kube-compose/issues/70 this is not correct for non-Linux containers
	if user.GID == nil && user.Group != "" {
		err := cs.CopyFromContainerToFile(ctx, containerID, unix.EtcGroup, tmpDir)
		if err != nil {
			return err
		}
		var gid *int64
		gid, err = unix.FindUIDByNameInPasswd(path.Join(tmpDir, "group"), user.Group)
		if err != nil {
			return err
		}
		if gid == nil {
			return fmt.Errorf("linux spec user: unable to find group %s no matching entries in group file", user.Group)
		}
		user.GID = gid
	}
	return nil
}
