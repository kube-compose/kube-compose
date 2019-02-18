package up

import (
	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
)

func Run (cfg *config.Config) error {
	fmt.Println("Hello World from package \"up\"!")
	return nil
}
