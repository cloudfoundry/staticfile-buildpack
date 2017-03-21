package libbuildpack

import (
	"os"
	"path/filepath"
	"strings"
)

type Compiler struct {
	BuildDir string
	CacheDir string
	DepsDir  string
	Manifest Manifest
	Log      Logger
}

func NewCompiler(args []string, logger Logger) (*Compiler, error) {
	bpDir, err := GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		return nil, err
	}

	manifest, err := NewManifest(bpDir)
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		return nil, err
	}

	buildDir := args[0]
	cacheDir := args[1]
	depsDir := ""

	if len(args) >= 4 {
		depsDir = args[3]
	}

	c := &Compiler{BuildDir: buildDir,
		CacheDir: cacheDir,
		DepsDir:  depsDir,
		Manifest: manifest,
		Log:      logger}

	return c, nil
}

func GetBuildpackDir() (string, error) {
	var err error

	bpDir := os.Getenv("BUILDPACK_DIR")

	if bpDir == "" {
		bpDir, err = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), ".."))

		if err != nil {
			return "", err
		}
	}

	return bpDir, nil
}

func (c *Compiler) CheckBuildpackValid() error {
	version, err := c.Manifest.Version()
	if err != nil {
		c.Log.Error("Could not determine buildpack version: %s", err.Error())
		return err
	}

	c.Log.BeginStep("%s Buildpack version %s", strings.Title(c.Manifest.Language()), version)

	err = c.Manifest.CheckStackSupport()
	if err != nil {
		c.Log.Error("Stack not supported by buildpack: %s", err.Error())
		return err
	}

	c.Manifest.CheckBuildpackVersion(c.CacheDir)

	return nil
}

func (c *Compiler) LoadSuppliedDeps() error {
	if c.DepsDir == "" {
		return nil
	}

	err := SetEnvironmentFromSupply(c.DepsDir)
	if err != nil {
		c.Log.Error("Unable to setup environment variables: %s", err.Error())
		return err
	}

	err = WriteProfileDFromSupply(c.DepsDir, c.BuildDir)
	if err != nil {
		c.Log.Error("Unable to write .profile.d supply script: %s", err.Error())
		return err
	}

	return nil
}

func (c *Compiler) StagingComplete() {
	c.Manifest.StoreBuildpackMetadata(c.CacheDir)
}
