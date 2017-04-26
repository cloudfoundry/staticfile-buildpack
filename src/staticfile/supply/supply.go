package supply

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Supplier struct {
	Stager *libbuildpack.Stager
}

func Run(ss *Supplier) error {
	if err := ss.InstallNginx(); err != nil {
		ss.Stager.Log.Error("Unable to install nginx: %s", err.Error())
		return err
	}

	return nil
}

func (ss *Supplier) InstallNginx() error {
	ss.Stager.Log.BeginStep("Installing nginx")

	nginx, err := ss.Stager.Manifest.DefaultVersion("nginx")
	if err != nil {
		return err
	}
	ss.Stager.Log.Info("Using nginx version %s", nginx.Version)

	if err := ss.Stager.Manifest.InstallDependency(nginx, ss.Stager.DepDir()); err != nil {
		return err
	}

	return ss.Stager.AddBinDependencyLink(filepath.Join(ss.Stager.DepDir(), "nginx", "sbin", "nginx"), "nginx")
}
