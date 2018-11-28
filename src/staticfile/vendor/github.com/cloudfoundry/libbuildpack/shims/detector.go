package shims

import (
	"os"
	"os/exec"
	"path/filepath"
)

type Installer interface {
	InstallOnlyVersion(depName string, installDir string) error
	InstallCNBS(orderFile string, installDir string) error
}

type DefaultDetector struct {
	BinDir string

	V2AppDir string

	V3BuildpacksDir string

	OrderMetadata string
	GroupMetadata string
	PlanMetadata  string

	Installer Installer
}

func (d DefaultDetector) Detect() error {
	err := d.Installer.InstallOnlyVersion("v3-detector", d.BinDir)
	if err != nil {
		return err
	}

	err = d.Installer.InstallCNBS(d.OrderMetadata, d.V3BuildpacksDir)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		filepath.Join(d.BinDir, "v3-detector"),
		"-app", d.V2AppDir,
		"-buildpacks", d.V3BuildpacksDir,
		"-order", d.OrderMetadata,
		"-group", d.GroupMetadata,
		"-plan", d.PlanMetadata,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))
	return cmd.Run()
}
