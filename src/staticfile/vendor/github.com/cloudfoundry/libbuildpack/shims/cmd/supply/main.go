package main

import (
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	if len(os.Args) != 5 {
		logger.Error("Incorrect number of arguments")
		os.Exit(1)
	}

	if err := supply(logger); err != nil {
		logger.Error("Failed supply step: %s", err.Error())
		os.Exit(1)
	}
}

func supply(logger *libbuildpack.Logger) error {
	v2AppDir := os.Args[1]
	v2CacheDir := os.Args[2]
	v2DepsDir := os.Args[3]
	v2DepsIndex := os.Args[4]

	buildpackDir, err := filepath.Abs(filepath.Join(os.Args[0], "..", ".."))
	if err != nil {
		return err
	}

	v3WorkspaceDir, err := filepath.Abs(filepath.Join(v2AppDir, ".."))
	if err != nil {
		return err
	}

	binDir := filepath.Join(v3WorkspaceDir, "bin")

	v3LaunchDir := filepath.Join(string(filepath.Separator), "home", "vcap", "deps", v2DepsIndex)
	err = os.MkdirAll(v3LaunchDir, 0777)
	if err != nil {
		return err
	}
	defer os.RemoveAll(v3LaunchDir)

	v3AppDir := filepath.Join(string(filepath.Separator), "home", "vcap", "app")

	v3BuildpacksDir := filepath.Join(v3WorkspaceDir, "cnbs")
	err = os.MkdirAll(v3BuildpacksDir, 0777)
	if err != nil {
		return err
	}
	defer os.RemoveAll(v3BuildpacksDir)

	orderMetadata := filepath.Join(buildpackDir, "order.toml")
	groupMetadata := filepath.Join(v3WorkspaceDir, "group.toml")
	planMetadata := filepath.Join(v3WorkspaceDir, "plan.toml")

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		return err
	}

	installer := shims.NewCNBInstaller(manifest)

	detector := shims.DefaultDetector{
		BinDir: binDir,

		V2AppDir: v2AppDir,

		V3BuildpacksDir: v3BuildpacksDir,

		OrderMetadata: orderMetadata,
		GroupMetadata: groupMetadata,
		PlanMetadata:  planMetadata,

		Installer: installer,
	}

	supplier := shims.Supplier{
		BinDir: binDir,

		V2AppDir:    v2AppDir,
		V2CacheDir:  v2CacheDir,
		V2DepsDir:   v2DepsDir,
		V2DepsIndex: v2DepsIndex,

		V3AppDir:        v3AppDir,
		V3BuildpacksDir: v3BuildpacksDir,
		V3LaunchDir:     v3LaunchDir,
		V3WorkspaceDir:  v3WorkspaceDir,

		OrderMetadata: orderMetadata,
		GroupMetadata: groupMetadata,
		PlanMetadata:  planMetadata,

		Detector:  detector,
		Installer: installer,
	}

	return supplier.Supply()
}
