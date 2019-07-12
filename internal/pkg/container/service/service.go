package service

import (
	"context"

	"github.com/docker/distribution/digestset"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
)

type Progress interface {
	Progress() float64
}

type ContainerService interface {
	PullImage(
		ctx context.Context,
		image, registryAuth string,
		onUpdate func(Progress),
	) (digest string, err error)
	PushImage(
		ctx context.Context,
		image, registryAuth string,
		onUpdate func(Progress),
	) (digest string, err error)
	ResolveLocalImageAfterPull(
		ctx context.Context,
		named dockerRef.Named,
		digest string,
	) (imageID, repoDigest string, err error)
	ResolveLocalImageID(
		ref dockerRef.Reference,
		localImageIDSet *digestset.Set,
		localImagesCache []dockerTypes.ImageSummary,
	) string
}
