package main

import (
	"log"

	"github.com/kube-compose/kube-compose/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
