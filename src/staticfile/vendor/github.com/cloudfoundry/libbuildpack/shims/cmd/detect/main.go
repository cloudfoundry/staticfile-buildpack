package main

import (
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/shims"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stderr)

	if len(os.Args) != 2 {
		logger.Error("Incorrect number of arguments")
		os.Exit(1)
	}

	appDir := os.Args[1]

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to find buildpack directory: %s", err.Error())
		os.Exit(1)
	}

	workspaceDir, err := filepath.Abs(filepath.Join(appDir, ".."))
	if err != nil {
		logger.Error("Unable to find workspace directory: %s", err.Error())
		os.Exit(1)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(1)
	}

	detector := shims.DefaultDetector{
		BinDir: filepath.Join(workspaceDir, "bin"),

		V2AppDir: appDir,

		V3BuildpacksDir: filepath.Join(workspaceDir, "cnbs"),

		OrderMetadata: filepath.Join(buildpackDir, "order.toml"),
		GroupMetadata: filepath.Join(workspaceDir, "group.toml"),
		PlanMetadata:  filepath.Join(workspaceDir, "plan.toml"),

		Installer: shims.NewCNBInstaller(manifest),
	}

	err = detector.Detect()
	if err != nil {
		logger.Error("Failed detection step: %s", err.Error())
		os.Exit(1)
	}
}
