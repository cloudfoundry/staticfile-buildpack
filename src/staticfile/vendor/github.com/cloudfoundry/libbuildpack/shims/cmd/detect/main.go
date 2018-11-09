package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
	"path/filepath"
)

func main() {
	workspaceDir := filepath.Join(os.Args[1], "..")

	detector, err := shims.NewShim()
	if err != nil {
		log.Fatal(err)
	}

	err = shims.Detect(detector, workspaceDir)
	if err != nil {
		log.Fatal(err)
	}
}
