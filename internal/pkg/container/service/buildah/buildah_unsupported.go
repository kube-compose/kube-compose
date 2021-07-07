// +build !linux

package buildah

import (
	"github.com/kube-compose/kube-compose/internal/pkg/container/service"
)

// New creates a new container service backed by Buildah and local storage.
func New() (service.ContainerService, error) {
	return nil, errNotSupported
}
