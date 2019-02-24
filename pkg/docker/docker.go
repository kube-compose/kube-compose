package docker

import (
	"context"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
)

func PullImage(ctx context.Context, dockerClient *dockerClient.Client, image string, onUpdate func(*PullOrPush)) (string, error) {
	pullOptions := dockerTypes.ImagePullOptions{}
	readCloser, err := dockerClient.ImagePull(ctx, image, pullOptions)
	if err != nil {
		return "", err
	}
	defer readCloser.Close()
	pull := NewPullOrPush(readCloser)
	return pull.Wait(onUpdate)
}

func PushImage(ctx context.Context, dockerClient *dockerClient.Client, image string, onUpdate func(*PullOrPush)) (string, error) {
	pushOptions := dockerTypes.ImagePushOptions{
		RegistryAuth: "123",
	}
	readCloser, err := dockerClient.ImagePush(ctx, image, pushOptions)
	if err != nil {
		return "", err
	}
	defer readCloser.Close()
	push := NewPullOrPush(readCloser)
	return push.Wait(onUpdate)
}
