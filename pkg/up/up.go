package up

import (
	"github.com/jabrekelmans/k8s-docker-compose/config"
)

func Run (cfg *config.Config) error {
	fmt.Println("Hello World from package \"up\"!")
	return nil
}
