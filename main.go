package main

import (
	"log"

	"github.com/jbrekelmans/kube-compose/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
