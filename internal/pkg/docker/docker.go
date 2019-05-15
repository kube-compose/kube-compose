package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
)

// EncodeRegistryAuth encodes a username and password into a base64 encoded value of a registry authentication header.
func EncodeRegistryAuth(username, password string) string {
	authConfig := dockerTypes.AuthConfig{
		Username: username,
		Password: password,
	}
	authConfigBytes, _ := json.Marshal(&authConfig)
	return base64.StdEncoding.EncodeToString(authConfigBytes)
}

type ImagePuller interface {
	ImagePull(ctx context.Context, image string, pullOptions dockerTypes.ImagePullOptions) (io.ReadCloser, error)
}

// PullImage pulls an image using a docker daemon, waits for the image to be pulled and returns its digest.
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
	ImagePush(ctx context.Context, image string, pullOptions dockerTypes.ImagePushOptions) (io.ReadCloser, error)
}

// PushImage pulls an image using a docker daemon, waits for the image to be pushed and returns its digest.
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
