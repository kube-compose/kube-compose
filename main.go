package main

<<<<<<< HEAD
import "github.com/jbrekelmans/kube-compose/cmd"
=======
import (
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/jbrekelmans/kube-compose/cmd"
)
>>>>>>> 670f0fc... issue #16: rename jompose to kube-compose

func main() {
	cmd.Execute()
}
