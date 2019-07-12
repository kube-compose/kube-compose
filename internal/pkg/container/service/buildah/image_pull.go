package buildah

import (
	"context"
	"fmt"

	"github.com/kube-compose/kube-compose/internal/pkg/container/service"
)

func (b *buildahContainerService) ImagePull(
	ctx context.Context,
	image, registryAuth string,
	onUpdate func(service.Progress),
) (digest string, err error) {

	// SignaturePolicyPath: iopts.signaturePolicy,
	// Store:               store,
	// SystemContext:       systemContext,
	// BlobDirectory:       iopts.blobCache,
	// AllTags:             iopts.allTags,
	// ReportWriter:        os.Stderr,

	// buildah.Pull(context.Background(), args[0], options)

	return "", fmt.Errorf("pulling images via Buildah and local storage is not supported")
}
