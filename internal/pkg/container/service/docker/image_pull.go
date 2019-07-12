//nolint
// ignoring dupl of image_push.go warning
package docker

import (
	"context"
	"io"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/kube-compose/kube-compose/internal/pkg/util"
)

type ImagePuller interface {
	ImagePull(ctx context.Context, image string, pushOptions dockerTypes.ImagePullOptions) (io.ReadCloser, error)
}

func imagePull(ctx context.Context, puller ImagePuller, image, registryAuth string, onUpdate func(*pullOrPush)) (string, error) {
	pullOptions := dockerTypes.ImagePullOptions{
		RegistryAuth: registryAuth,
	}
	readCloser, err := puller.ImagePull(ctx, image, pullOptions)
	if err != nil {
		return "", err
	}
	defer util.CloseAndLogError(readCloser)
	pull := newPull(readCloser)
	return pull.Wait(onUpdate)
}
