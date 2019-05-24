package up

import (
	"context"
)

type Options struct {
	Context context.Context
	Detach  bool
	// True to set runAsUser/runAsGroup for each pod based on the user of the pod's image and the "user" key of the pod's docker-compose
	// service.
	RunAsUser bool
}
