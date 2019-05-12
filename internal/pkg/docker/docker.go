package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/jbrekelmans/kube-compose/internal/pkg/util"
)

func EncodeRegistryAuth(username, password string) (string, error) {
	authConfig := dockerTypes.AuthConfig{
		Username: username,
		Password: password,
	}
	authConfigBytes, err := json.Marshal(&authConfig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(authConfigBytes), nil
}

func PullImage(ctx context.Context, dc *dockerClient.Client, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pullOptions := dockerTypes.ImagePullOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := dc.ImagePull(ctx, image, pullOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	pull := NewPull(readCloser)
	return pull.Wait(onUpdate)
}

func PushImage(ctx context.Context, dc *dockerClient.Client, image, registryAuth string, onUpdate func(*PullOrPush)) (string, error) {
	pushOptions := dockerTypes.ImagePushOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := dc.ImagePush(ctx, image, pushOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	push := NewPush(readCloser)
	return push.Wait(onUpdate)
}