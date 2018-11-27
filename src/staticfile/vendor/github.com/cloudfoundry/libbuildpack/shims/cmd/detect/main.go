package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal(errors.New("incorrect number of arguments"))
	}

	appDir := os.Args[1]

	buildpackDir, err := filepath.Abs(filepath.Join(os.Args[0], "..", ".."))
	if err != nil {
		log.Fatal(err)
	}

	workspaceDir, err := filepath.Abs(filepath.Join(appDir, ".."))
	if err != nil {
		log.Fatal(err)
	}

	detector := shims.DefaultDetector{
		BinDir:        filepath.Join(buildpackDir, "bin"),
		AppDir:        appDir,
		BuildpacksDir: filepath.Join(buildpackDir, "cnbs"),
		GroupMetadata: filepath.Join(workspaceDir, "group.toml"),
		LaunchDir:     workspaceDir,
		OrderMetadata: filepath.Join(buildpackDir, "order.toml"),
		PlanMetadata:  filepath.Join(workspaceDir, "plan.toml"),
	}

	err = detector.Detect()
	if err != nil {
		log.Fatal(err)
	}
}
