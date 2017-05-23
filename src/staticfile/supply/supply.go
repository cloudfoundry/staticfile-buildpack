package supply

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
}

type Stager interface {
	AddBinDependencyLink(string, string) error
	DepDir() string
}

type Supplier struct {
	Stager   Stager
	Manifest Manifest
	Log      *libbuildpack.Logger
}

func Run(ss *Supplier) error {
	if err := ss.InstallNginx(); err != nil {
		ss.Log.Error("Unable to install nginx: %s", err.Error())
		return err
	}

	return nil
}

func (ss *Supplier) InstallNginx() error {
	ss.Log.BeginStep("Installing nginx")

	nginx, err := ss.Manifest.DefaultVersion("nginx")
	if err != nil {
		return err
	}
	ss.Log.Info("Using nginx version %s", nginx.Version)

	if err := ss.Manifest.InstallDependency(nginx, ss.Stager.DepDir()); err != nil {
		return err
	}

	return ss.Stager.AddBinDependencyLink(filepath.Join(ss.Stager.DepDir(), "nginx", "sbin", "nginx"), "nginx")
}
