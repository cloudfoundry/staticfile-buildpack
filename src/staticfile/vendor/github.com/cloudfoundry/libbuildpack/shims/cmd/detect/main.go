package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
	"path/filepath"
)

func main() {
	buildpackDir := filepath.Join(os.Args[0], "..", "..")
	workspaceDir := filepath.Join(os.Args[1], "..")

	err := shims.Detect(&shims.Shim{}, buildpackDir, workspaceDir)
	if err != nil {
		log.Fatal(err)
	}
}
