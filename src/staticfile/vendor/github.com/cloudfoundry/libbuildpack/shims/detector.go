package shims

import (
	"os"
	"os/exec"
	"path/filepath"
)

type DefaultDetector struct {
	BinDir        string
	AppDir        string
	BuildpacksDir string
	GroupMetadata string
	LaunchDir     string
	OrderMetadata string
	PlanMetadata  string
}

func (d *DefaultDetector) Detect() error {
	cmd := exec.Command(
		filepath.Join(d.BinDir, "v3-detector"),
		"-app", d.AppDir,
		"-buildpacks", d.BuildpacksDir,
		"-group", d.GroupMetadata,
		"-order", d.OrderMetadata,
		"-plan", d.PlanMetadata,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))
	return cmd.Run()
}
