package up

import (
	"fmt"

	"github.com/jbrekelmans/k8s-docker-compose/pkg/config"
)

func Run (cfg *config.Config) error {
	image := cfg.ComposeYaml.Services["k8s-docker-compose"].Image
	fmt.Println("Hello World from package \"up\"!")
	fmt.Println(image)
	return nil
}
