package docker

import (
	"context"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/kube-compose/kube-compose/internal/pkg/container/service"
	dockerInternal "github.com/kube-compose/kube-compose/internal/pkg/docker"
)

type dockerContainerService struct {
	dc *dockerClient.Client
}

// New creates a new service.ContainerService backed by a docker daemon.
func New(dc *dockerClient.Client) service.ContainerService {
	bla := &dockerContainerService{
		dc: dc,
	}
	return bla
}

func (d *dockerContainerService) ImageList(ctx context.Context, listOptions dockerTypes.ImageListOptions) ([]dockerTypes.ImageSummary, error) {
	return d.ImageList(ctx, listOptions)
}

func (d *dockerContainerService) PullImage(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (digest string, err error) {
	return dockerInternal.PullImage(ctx, d.dc, image, registryAuth, func(pull *dockerInternal.PullOrPush) {
		onUpdate(pull)
	})
}

func (d *dockerContainerService) PushImage(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (digest string, err error) {
	return dockerInternal.PushImage(ctx, d.dc, image, registryAuth, func(push *dockerInternal.PullOrPush) {
		onUpdate(push)
	})
}

func (d *dockerContainerService) ResolveLocalImageAfterPull(
	ctx context.Context,
	named dockerRef.Named,
	digest string,
) (imageID, repoDigest string, err error) {
	return dockerInternal.ResolveLocalImageAfterPull(ctx, d.dc, named, digest)
}

func (d *dockerContainerService) ResolveLocalImageID(
	ref dockerRef.Reference,
	localImageIDSet *digestset.Set,
	localImagesCache []dockerTypes.ImageSummary,
) string {
	return dockerInternal.ResolveLocalImageID(ref, localImageIDSet, localImagesCache)
}
