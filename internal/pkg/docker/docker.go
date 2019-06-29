package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

// Defaults useful when constructing fully qualified image refs.
// Sourced from https://github.com/moby/moby/blob/master/vendor/github.com/docker/distribution/reference/normalize.go#L14
const (
	DefaultDomain    = "docker.io"
	OfficialRepoName = "library"
)

func EncodeRegistryAuth(username, password string) string {
	authConfig := dockerTypes.AuthConfig{
		Username: username,
		Password: password,
	}
	authConfigBytes, _ := json.Marshal(&authConfig)
	return base64.StdEncoding.EncodeToString(authConfigBytes)
}

type ImagePuller interface {
	ImagePull(ctx context.Context, image string, pushOptions dockerTypes.ImagePullOptions) (io.ReadCloser, error)
}

func PullImage(ctx context.Context, puller ImagePuller, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pullOptions := dockerTypes.ImagePullOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := puller.ImagePull(ctx, image, pullOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	pull := NewPull(readCloser)
	return pull.Wait(onUpdate)
}

type ImagePusher interface {
	ImagePush(ctx context.Context, image string, pushOptions dockerTypes.ImagePushOptions) (io.ReadCloser, error)
}

func PushImage(ctx context.Context, pusher ImagePusher, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pushOptions := dockerTypes.ImagePushOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := pusher.ImagePush(ctx, image, pushOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	push := NewPush(readCloser)
	return push.Wait(onUpdate)
}
