package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
	"path/filepath"
)

func main() {
	buildDir := os.Args[1]
	cacheDir := os.Args[2]
	depsDir := os.Args[3]
	depsIndex := os.Args[4]
	workspaceDir := filepath.Join(buildDir, "..")
	launchDir, err := filepath.Abs(filepath.Join("home", "vcap", "deps", depsIndex))
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(launchDir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(launchDir)

	shim, err := shims.NewShim()
	if err != nil {
		log.Fatal(err)
	}

	err = shims.Supply(shim, buildDir, cacheDir, depsDir, depsIndex, workspaceDir, launchDir)
	if err != nil {
		log.Fatal(err)
	}
}
