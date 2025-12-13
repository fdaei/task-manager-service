package main

import (
	"log"

	"taskservice/cmd/taskservice/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("taskservice: %v", err)
	}
}
