package main

import (
	"os"

	"github.com/kube-compose/kube-compose/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
