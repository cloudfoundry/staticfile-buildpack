package shims

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Detector interface {
	Detect() error
}

type Supplier struct {
	BinDir string

	V2AppDir    string
	V2CacheDir  string
	V2DepsDir   string
	V2DepsIndex string

	V3AppDir        string
	V3BuildpacksDir string
	V3LaunchDir     string
	V3WorkspaceDir  string

	OrderMetadata string
	GroupMetadata string
	PlanMetadata  string

	Detector  Detector
	Installer Installer
}

func (s *Supplier) Supply() error {
	if err := s.Installer.InstallOnlyVersion("v3-builder", s.BinDir); err != nil {
		return err
	}

	if err := s.Installer.InstallCNBS(s.OrderMetadata, s.V3BuildpacksDir); err != nil {
		return err
	}

	if err := os.RemoveAll(s.V3AppDir); err != nil {
		return err
	}

	if err := os.Rename(s.V2AppDir, s.V3AppDir); err != nil {
		return err
	}

	if err := s.GetBuildPlan(); err != nil {
		return err
	}

	if err := s.RunLifeycleBuild(); err != nil {
		return err
	}

	if err := os.Rename(s.V3AppDir, s.V2AppDir); err != nil {
		return err
	}

	if err := s.Installer.InstallOnlyVersion("v3-launcher", s.V2DepsDir); err != nil {
		return err
	}

	return s.MoveLayers()
}

func (s *Supplier) MoveLayers() error {
	layers, err := filepath.Glob(filepath.Join(s.V3LaunchDir, "*"))
	if err != nil {
		return err
	}

	for _, layer := range layers {
		if filepath.Base(layer) == "config" {
			if err := os.Mkdir(filepath.Join(s.V2DepsDir, s.V2DepsIndex, "config"), 0777); err != nil {
				return err
			}

			err = libbuildpack.CopyFile(filepath.Join(s.V3LaunchDir, "config", "metadata.toml"), filepath.Join(s.V2DepsDir, s.V2DepsIndex, "config", "metadata.toml"))
			if err != nil {
				return err
			}

			if err := os.Mkdir(filepath.Join(s.V2AppDir, ".cloudfoundry"), 0777); err != nil {
				return err
			}

			err = os.Rename(filepath.Join(s.V3LaunchDir, "config", "metadata.toml"), filepath.Join(s.V2AppDir, ".cloudfoundry", "metadata.toml"))
			if err != nil {
				return err
			}
		} else {
			err := os.Rename(layer, filepath.Join(s.V2DepsDir, s.V2DepsIndex, filepath.Base(layer)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Supplier) GetBuildPlan() error {
	_, groupErr := os.Stat(s.GroupMetadata)
	_, planErr := os.Stat(s.PlanMetadata)

	if os.IsNotExist(groupErr) || os.IsNotExist(planErr) {
		return s.Detector.Detect()
	}

	return nil
}

func (s *Supplier) RunLifeycleBuild() error {
	cmd := exec.Command(
		filepath.Join(s.BinDir, "v3-builder"),
		"-app", s.V3AppDir,
		"-buildpacks", s.V3BuildpacksDir,
		"-cache", s.V2CacheDir,
		"-group", s.GroupMetadata,
		"-launch", s.V3LaunchDir,
		"-plan", s.PlanMetadata,
		"-platform", s.V3WorkspaceDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))

	return cmd.Run()
}
