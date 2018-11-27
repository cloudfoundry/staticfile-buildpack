package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	if err := supply(); err != nil {
		log.Fatal(err)
	}
}

func supply() error {
	if len(os.Args) != 5 {
		return errors.New("incorrect number of arguments")
	}

	buildpackDir, err := filepath.Abs(filepath.Join(os.Args[0], "..", ".."))
	if err != nil {
		return err
	}

	buildDir := os.Args[1]
	cacheDir := os.Args[2]
	depsDir := os.Args[3]
	depsIndex := os.Args[4]

	workspaceDir, err := filepath.Abs(filepath.Join(buildDir, ".."))
	if err != nil {
		return err
	}

	launchDir := filepath.Join(string(filepath.Separator), "home", "vcap", "deps", depsIndex)

	err = os.MkdirAll(launchDir, 0777)
	if err != nil {
		return err
	}
	defer os.RemoveAll(launchDir)

	cnbAppDir := filepath.Join(string(filepath.Separator), "home", "vcap", "app")

	supplier := shims.Supplier{
		BinDir:        filepath.Join(buildpackDir, "bin"),
		V2BuildDir:    buildDir,
		CNBAppDir:     cnbAppDir,
		BuildpacksDir: filepath.Join(buildpackDir, "cnbs"),
		CacheDir:      cacheDir,
		DepsDir:       depsDir,
		DepsIndex:     depsIndex,
		GroupMetadata: filepath.Join(workspaceDir, "group.toml"),
		LaunchDir:     launchDir,
		OrderMetadata: filepath.Join(buildpackDir, "order.toml"),
		PlanMetadata:  filepath.Join(workspaceDir, "plan.toml"),
		WorkspaceDir:  workspaceDir,
	}

	return supplier.Supply()
}
