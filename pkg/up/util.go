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

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerContainers "github.com/docker/docker/api/types/container"
	dockerFilters "github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	dockerArchive "github.com/docker/docker/pkg/archive"
	"github.com/jbrekelmans/kube-compose/internal/pkg/docker"
	"github.com/jbrekelmans/kube-compose/internal/pkg/linux"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
	"github.com/jbrekelmans/kube-compose/pkg/config"
	"github.com/pkg/errors"
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
<<<<<<< HEAD
		// Add 2 to accommodate for /bin/sh -c
=======
		// Add 2 to accomomdate for /bin/sh -c
>>>>>>> 63e2104... Fixing lint issues
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

type hasTag interface {
	Tag() string
}

func inspectImageRawParseHealthcheck(inspectRaw []byte) (*config.Healthcheck, error) {
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

func copyFileFromContainer(ctx context.Context, dc *dockerClient.Client, containerID, srcFile, dstFile string) error {
	readCloser, stat, err := dc.CopyFromContainer(ctx, containerID, srcFile)
	if err != nil {
		return err
	}
	defer util.CloseAndLogError(readCloser)
	if (stat.Mode & os.ModeType) != 0 {
		// TODO https://github.com/jbrekelmans/kube-compose/issues/70 we should follow symlinks
		return fmt.Errorf("could not copy %#v because it is not a regular file", srcFile)
	}
	srcInfo := dockerArchive.CopyInfo{
		Path:       srcFile,
		Exists:     true,
		IsDir:      false,
		RebaseName: "",
	}
	err = dockerArchive.CopyTo(readCloser, srcInfo, dstFile)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error while copying image file %#v to local file %#v", srcFile, dstFile))
	}
	return nil
}

func getUserinfoFromImage(ctx context.Context, dc *dockerClient.Client, image string, user *docker.Userinfo) error {
	containerConfig := &dockerContainers.Config{
		Entrypoint: []string{"sh"},
		Image:      image,
		WorkingDir: "/",
	}
	resp, err := dc.ContainerCreate(ctx, containerConfig, nil, nil, "")
	if err != nil {
		return err
	}
	defer func() {
		err = dc.ContainerRemove(ctx, resp.ID, dockerTypes.ContainerRemoveOptions{})
		if err != nil {
			fmt.Println(err)
		}
	}()
	tmpDir, err := ioutil.TempDir("", "kube-compose-")
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			fmt.Println(err)
		}
	}()
	// TODO https://github.com/jbrekelmans/kube-compose/issues/70 this is not correct for non-Linux containers
	if user.UID == nil {
		err = copyFileFromContainer(ctx, dc, resp.ID, "/etc/passwd", tmpDir)
		if err != nil {
			return err
		}
		var uid *int64
		uid, err = linux.FindUserInPasswd(path.Join(tmpDir, "passwd"), user.User)
		if err != nil {
			return err
		}
		if uid == nil {
			return fmt.Errorf("linux spec user: unable to find user %s no matching entries in passwd file", user.User)
		}
		user.UID = uid
	}
	if user.GID == nil && user.Group != "" {
		err = copyFileFromContainer(ctx, dc, resp.ID, "/etc/group", tmpDir)
		if err != nil {
			return err
		}
		var gid *int64
		gid, err = linux.FindUserInPasswd(path.Join(tmpDir, "group"), user.Group)
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

// resolveLocalImageID resolves an image ID against a cached list (like the one output by the command "docker images").
// ref is assumed not to be a partial image ID.
<<<<<<< HEAD
// TODO: https://github.com/jbrekelmans/kube-compose/issues/64
=======
>>>>>>> 63e2104... Fixing lint issues
// nolint
func resolveLocalImageID(ref dockerRef.Reference, localImageIDSet *digestset.Set, localImagesCache []dockerTypes.ImageSummary) string {
	named, isNamed := ref.(dockerRef.Named)
	digested, isDigested := ref.(dockerRef.Digested)
	// By definition of dockerRef.ParseAnyReferenceWithSet isNamed or isDigested is true
	if !isNamed {
		imageID := digested.String()
		if _, err := localImageIDSet.Lookup(imageID); err == digestset.ErrDigestNotFound {
			return ""
		}
		// The only other error returned by Lookup is a digestset.ErrDigestAmbiguous, which cannot
		// happen by our assumption that ref cannot be a partial image ID
		return imageID
	}
	familiarName := dockerRef.FamiliarName(named)
	// The source image must be named
	if isDigested {
		// docker images returns RepoDigests as a familiar name with a digest
		repoDigest := familiarName + "@" + string(digested.Digest())
		for i := 0; i < len(localImagesCache); i++ {
			for _, repoDigest2 := range localImagesCache[i].RepoDigests {
				if repoDigest == repoDigest2 {
					return localImagesCache[i].ID
				}
			}
		}
	}
	tag := getTag(ref)
	if len(tag) > 0 {
		// docker images returns RepoTags as a familiar name with a tag
		repoTag := familiarName + ":" + tag
		for i := 0; i < len(localImagesCache); i++ {
			for _, repoTag2 := range localImagesCache[i].RepoTags {
				if repoTag == repoTag2 {
					return localImagesCache[i].ID
				}
			}
		}
	}
	return ""
}

// resolveLocalImageAfterPull resolves an image based on a repository and digest by querying the docker daemon.
// This is exactly the information we have available after pulling an image.
// Returns the image ID, repo digest and optionally an error.
func resolveLocalImageAfterPull(ctx context.Context, dc *dockerClient.Client, named dockerRef.Named, digest string) (
	imageID, repoDigest string, err error) {
	filters := dockerFilters.NewArgs()
	familiarName := dockerRef.FamiliarName(named)
	filters.Add("reference", familiarName)
	imageSummaries, err := dc.ImageList(ctx, dockerTypes.ImageListOptions{
		All:     false,
		Filters: filters,
	})
	if err != nil {
		return "", "", err
	}
	repoDigest = familiarName + "@" + digest
	for i := 0; i < len(imageSummaries); i++ {
		for _, repoDigest2 := range imageSummaries[i].RepoDigests {
			if repoDigest == repoDigest2 {
				return imageSummaries[i].ID, repoDigest, nil
			}
		}
	}
	return "", "", nil
}

func getTag(ref dockerRef.Reference) string {
	refWithTag, ok := ref.(hasTag)
	if !ok {
		return ""
	}
	return refWithTag.Tag()
}

func pullImageWithLogging(ctx context.Context, dc *dockerClient.Client, appName, image string) (string, error) {
	lastLogTime := time.Now().Add(-2 * time.Second)
	digest, err := docker.PullImage(ctx, dc, image, "123", func(pull *docker.PullOrPush) {
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

func pushImageWithLogging(ctx context.Context, dc *dockerClient.Client, appName, image, bearerToken string) (string, error) {
	lastLogTime := time.Now().Add(-2 * time.Second)
	registryAuth, err := docker.EncodeRegistryAuth("unused", bearerToken)
	if err != nil {
		return "", err
	}
	digest, err := docker.PushImage(ctx, dc, image, registryAuth, func(push *docker.PullOrPush) {
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
