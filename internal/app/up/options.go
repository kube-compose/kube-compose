package up

import (
	"context"

	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
)

type Options struct {
	Context  context.Context
	Detach   bool
	Reporter *reporter.Reporter
	// True to set runAsUser/runAsGroup for each pod based on the user of the pod's image and the "user" key of the pod's docker-compose
	// service.
	RunAsUser bool

	RegistryUser string
	RegistryPass string
}
