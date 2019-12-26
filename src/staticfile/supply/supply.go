package supply

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest interface {
	DefaultVersion(string) (libbuildpack.Dependency, error)
}

type Installer interface {
	InstallDependency(libbuildpack.Dependency, string) error
}

type Stager interface {
	AddBinDependencyLink(string, string) error
	DepDir() string
}

type Supplier struct {
	Stager    Stager
	Manifest  Manifest
	Installer Installer
	Log       *libbuildpack.Logger
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

	nginxDir := filepath.Join(ss.Stager.DepDir(), "nginx")
	if err := ss.Installer.InstallDependency(nginx, nginxDir); err != nil {
		return err
	}

	return ss.Stager.AddBinDependencyLink(filepath.Join(nginxDir, "sbin", "nginx"), "nginx")
}
