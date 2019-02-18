package up

import (
	"fmt"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
)

func Run (cfg *config.Config) error {
	fmt.Println("Hello World from package \"up\"!")
	return nil
}
