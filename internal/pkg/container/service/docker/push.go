//nolint
// ignoring dupl of image_pull.go warning
package docker

import (
	"context"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

type ImagePusher interface {
	ImagePush(ctx context.Context, image string, pushOptions dockerTypes.ImagePushOptions) (io.ReadCloser, error)
}

func imagePush(ctx context.Context, pusher ImagePusher, image, registryAuth string, onUpdate func(*pullOrPush)) (string, error) {
	pushOptions := dockerTypes.ImagePushOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := pusher.ImagePush(ctx, image, pushOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	push := newPush(readCloser)
	return push.Wait(onUpdate)
}
