// +build linux

package buildah

import (
	"context"
	"fmt"

	containerStorage "github.com/containers/storage"
	dockerRef "github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/kube-compose/kube-compose/internal/pkg/container/service"
)

type buildahContainerService struct {
	store containerStorage.Store
}

// New creates a new container service backed by Buildah and local storage.
func New() (service.ContainerService, error) {
	store, err := containerStorage.GetStore(containerStorage.StoreOptions{})
	if err != nil {
		return nil, err
	}
	s := &buildahContainerService{
		store: store,
	}
	return s, nil
}

func (b *buildahContainerService) Close() error {
	return nil
}

func (b *buildahContainerService) ContainerCreateForCopyFromContainer(ctx context.Context, image string) (string, error) {
	return "", fmt.Errorf("creating Buildah containers is not supported")
}

func (b *buildahContainerService) ContainerRemove(ctx context.Context, containerID string) error {
	return fmt.Errorf("removing Buildah containers is not supported")
}

func (b *buildahContainerService) CopyFromContainerToFile(ctx context.Context, containerID, srcFile, dstFile string) error {
	return fmt.Errorf("copying files from containers created via Buildah is not supported")
}

func (b *buildahContainerService) ImageBuild(opts *service.ImageBuildOptions) (string, error) {
	return "", fmt.Errorf("building images via Buildah is not supported")
}

func (b *buildahContainerService) ImageInspectWithRaw(ctx context.Context, imageID string) (dockerTypes.ImageInspect, []byte, error) {
	return dockerTypes.ImageInspect{}, nil, fmt.Errorf("inspecting images in local storage (pulled via Buildah) is not supported")
}

func (b *buildahContainerService) ImageList(ctx context.Context, listOptions dockerTypes.ImageListOptions) (
	[]dockerTypes.ImageSummary, error) {
	return nil, fmt.Errorf("listing images in local storage (pulled via Buildah) is not supported")
}

func (b *buildahContainerService) ImageTag(ctx context.Context, source, target string) error {
	return fmt.Errorf("tagging images in local storage (pulled via Buildah) is not supported")
}

func (b *buildahContainerService) ImagePull(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(Progress),
) (digest string, err error) {

	// SystemContext:       systemContext,
	// BlobDirectory:       iopts.blobCache,
	// ReportWriter:        os.Stderr,

	// should image be converted to a normalized name, so that the default docker domain and
	// library are resolved?
	// https://github.com/containers/buildah/blob/master/docs/buildah-pull.md
	imageID, err := buildah.Pull(ctx, image, buildah.PullOptions{
		AllTags:             false,
		BlobDirectory:       "",
		SignaturePolicyPath: "",
		Store:               b.store,
		SystemContext:       nil,
		ReportWriter:        os.Stdout,
	})
	if err != nil {
		return "", err
	}
	_ = imageID
	return "", fmt.Errorf("pulling images via Buildah and local storage is not supported")
}

func (b *buildahContainerService) ImagePush(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (digest string, err error) {
	return "", fmt.Errorf("pushing images via Buildah and local storage is not supported")
}

func (b *buildahContainerService) ImagePullResolve(
	ctx context.Context,
	named dockerRef.Named,
	digest string,
) (imageID, repoDigest string, err error) {
	return "", "", fmt.Errorf("resolving images after Buildah pull (based a local storage) is not supported")
}
