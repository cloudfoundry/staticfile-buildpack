package main

import (
	"github.com/cloudfoundry/libbuildpack/shims"
	"log"
	"os"
	"path/filepath"
)

func main() {
	buildpackDir := filepath.Join(os.Args[0], "..", "..")
	buildDir := os.Args[1]
	cacheDir := os.Args[2]
	depsDir := os.Args[3]
	depsIndex := os.Args[4]
	workspaceDir := filepath.Join(buildDir, "..")
	launchDir := filepath.Join(string(filepath.Separator), "home", "vcap", "deps", depsIndex)

	err := os.MkdirAll(launchDir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(launchDir)


	err = shims.Supply(&shims.Shim{}, buildpackDir, buildDir, cacheDir, depsDir, depsIndex, workspaceDir, launchDir)
	if err != nil {
		log.Fatal(err)
	}
}