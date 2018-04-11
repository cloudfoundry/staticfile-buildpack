package main

import (
	_ "LANGUAGE/hooks"
	"LANGUAGE/supply"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	os.MkdirAll(filepath.Join(stager.DepDir(), "bin"), 0755)
	os.MkdirAll(filepath.Join(stager.DepDir(), "lib"), 0755)

	if err = manifest.SetAppCacheDir(stager.CacheDir()); err != nil {
		logger.Error("Unable to setup app cache dir: %s", err)
		os.Exit(18)
	}
	if err = manifest.ApplyOverride(stager.DepsDir()); err != nil {
		logger.Error("Unable to apply override.yml files: %s", err)
		os.Exit(17)
	}

	err = libbuildpack.RunBeforeCompile(stager)
	if err != nil {
		logger.Error("Before Compile: %s", err.Error())
		os.Exit(12)
	}

	if err := os.MkdirAll(filepath.Join(stager.DepDir(), "bin"), 0755); err != nil {
		logger.Error("Unable to create bin directory: %s", err.Error())
		os.Exit(13)
	}

	err = stager.SetStagingEnvironment()
	if err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(14)
	}

	s := supply.Supplier{
		Manifest: manifest,
		Stager:   stager,
		Command:  &libbuildpack.Command{},
		Log:      logger,
	}

	err = s.Run()
	if err != nil {
		logger.Error("Error: %s", err)
		os.Exit(15)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(16)
	}
	if err = manifest.CleanupAppCache(); err != nil {
		logger.Error("Unable to apply override.yml files: %s", err)
		os.Exit(19)
	}
}
