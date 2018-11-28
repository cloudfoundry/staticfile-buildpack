package shims

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
)

type buildpack struct {
	Id      string `toml:"id"`
	Version string `toml:"version"`
}

type group struct {
	Labels     []string    `toml:"labels"`
	Buildpacks []buildpack `toml:"buildpacks"`
}

type order struct {
	Groups []group `toml:"groups"`
}

type CNBInstaller struct {
	*libbuildpack.Installer
	manifest *libbuildpack.Manifest
}

func NewCNBInstaller(manifest *libbuildpack.Manifest) *CNBInstaller {
	return &CNBInstaller{libbuildpack.NewInstaller(manifest), manifest}
}

func (c *CNBInstaller) InstallCNBS(orderFile string, installDir string) error {
	o := order{}
	_, err := toml.DecodeFile(orderFile, &o)
	if err != nil {
		return err
	}

	bpSet := make(map[string]interface{})
	for _, group := range o.Groups {
		for _, bp := range group.Buildpacks {
			bpSet[bp.Id] = nil
		}
	}

	for buildpack := range bpSet {
		versions := c.manifest.AllDependencyVersions(buildpack)
		if len(versions) != 1 {
			return fmt.Errorf("unable to find a unique version of %s in the manifest", buildpack)
		}

		buildpackDest := filepath.Join(installDir, buildpack, versions[0])
		if exists, err := libbuildpack.FileExists(buildpackDest); err != nil {
			return err
		} else if exists {
			continue
		}

		err := c.InstallOnlyVersion(buildpack, buildpackDest)
		if err != nil {
			return err
		}

		err = os.Symlink(buildpackDest, filepath.Join(installDir, buildpack, "latest"))
		if err != nil {
			return err
		}
	}

	return nil
}
