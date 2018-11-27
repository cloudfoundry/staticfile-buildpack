package shims

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Detector interface {
	Detect() error
}

type Supplier struct {
	Detector      Detector
	BinDir        string
	V2BuildDir    string
	CNBAppDir     string
	BuildpacksDir string
	CacheDir      string
	DepsDir       string
	DepsIndex     string
	LaunchDir     string
	OrderMetadata string
	GroupMetadata string
	PlanMetadata  string
	WorkspaceDir  string
}

func (s *Supplier) Supply() error {
	if err := os.RemoveAll(s.CNBAppDir); err != nil {
		return err
	}

	if err := os.Rename(s.V2BuildDir, s.CNBAppDir); err != nil {
		return err
	}

	if err := s.GetBuildPlan(); err != nil {
		return err
	}

	if err := s.RunLifeycleBuild(); err != nil {
		return err
	}

	if err := os.Rename(s.CNBAppDir, s.V2BuildDir); err != nil {
		return err
	}

	if err := s.InstallV3Launcher(s.DepsDir); err != nil {
		return err
	}

	return s.MoveLayers()
}

func (s *Supplier) InstallV3Launcher(dstDir string) error {
	contents, err := ioutil.ReadFile(filepath.Join(s.BinDir, "v3-launcher")) // don't copy
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(dstDir, "v3-launcher"), contents, 0777)
}

func (s *Supplier) MoveLayers() error {
	layers, err := filepath.Glob(filepath.Join(s.LaunchDir, "*"))
	if err != nil {
		return err
	}

	for _, layer := range layers {
		if filepath.Base(layer) == "config" {
			if err := os.Mkdir(filepath.Join(s.DepsDir, s.DepsIndex, "config"), 0777); err != nil {
				return err
			}

			err = libbuildpack.CopyFile(filepath.Join(s.LaunchDir, "config", "metadata.toml"), filepath.Join(s.DepsDir, s.DepsIndex, "config", "metadata.toml"))
			if err != nil {
				return err
			}

			if err := os.Mkdir(filepath.Join(s.V2BuildDir, ".cloudfoundry"), 0777); err != nil {
				return err
			}

			err = os.Rename(filepath.Join(s.LaunchDir, "config", "metadata.toml"), filepath.Join(s.V2BuildDir, ".cloudfoundry", "metadata.toml"))
			if err != nil {
				return err
			}
		} else {
			err := os.Rename(layer, filepath.Join(s.DepsDir, s.DepsIndex, filepath.Base(layer)))
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
		if err := s.Detector.Detect(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supplier) RunLifeycleBuild() error {
	cmd := exec.Command(
		filepath.Join(s.BinDir, "v3-builder"),
		"-app", s.CNBAppDir,
		"-buildpacks", s.BuildpacksDir,
		"-cache", s.CacheDir,
		"-group", s.GroupMetadata,
		"-launch", s.LaunchDir,
		"-plan", s.PlanMetadata,
		"-platform", s.WorkspaceDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))

	return cmd.Run()
}
