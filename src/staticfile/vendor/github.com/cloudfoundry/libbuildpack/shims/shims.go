package shims

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const setupPathContent = "export PATH={{ range $_, $path := . }}{{ $path }}:{{ end }}$PATH"

type Shimmer interface {
	Detect(binDir, buildpacksDir, groupMetadata, launchDir, orderMetadata, planMetadata string) error
	Supply(binDir, buildpacksDir, cacheDir, groupMetadata, launchDir, planMetadata, platformDir string) error
}

type Shim struct {}

func (s *Shim) Detect(binDir, buildpacksDir, groupMetadata, launchDir, orderMetadata, planMetadata string) error {
	cmd := exec.Command(
		filepath.Join(binDir, "v3-detector"),
		"-buildpacks", buildpacksDir,
		"-group", groupMetadata,
		"-launch", launchDir,
		"-order", orderMetadata,
		"-plan", planMetadata,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))
	return cmd.Run()
}

func (s *Shim) Supply(binDir, buildpacksDir, cacheDir, groupMetadata, launchDir, planMetadata, platformDir string) error {
	cmd := exec.Command(
		filepath.Join(binDir, "v3-builder"),
		"-buildpacks", buildpacksDir,
		"-cache", cacheDir,
		"-group", groupMetadata,
		"-launch", launchDir,
		"-plan", planMetadata,
		"-platform", platformDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PACK_STACK_ID=org.cloudfoundry.stacks."+os.Getenv("CF_STACK"))

	return cmd.Run()
}


func Detect(s Shimmer, buildpackDir, workspaceDir string) error {
	return s.Detect(
		filepath.Join(buildpackDir, "bin"),
		filepath.Join(buildpackDir, "cnbs"),
		filepath.Join(workspaceDir, "group.toml"),
		workspaceDir,
		filepath.Join(buildpackDir, "order.toml"),
		filepath.Join(workspaceDir, "plan.toml"),
	)
}

func Supply(s Shimmer, buildpackDir, buildDir, cacheDir, depsDir, depsIndex, workspaceDir, launchDir string) error {
	if err := os.Symlink(buildDir, filepath.Join(launchDir, "app")); err != nil {
		return err
	}

	_, groupErr := os.Stat(filepath.Join(workspaceDir, "group.toml"))
	_, planErr := os.Stat(filepath.Join(workspaceDir, "plan.toml"))

	if os.IsNotExist(groupErr) || os.IsNotExist(planErr) {
		Detect(s, buildpackDir, workspaceDir)
	}

	err := s.Supply(
		filepath.Join(buildpackDir, "bin"),
		filepath.Join(buildpackDir, "cnbs"),
		cacheDir,
		filepath.Join(workspaceDir, "group.toml"),
		launchDir,
		filepath.Join(workspaceDir, "plan.toml"),
		workspaceDir,
	)
	if err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(launchDir, "app")); err != nil {
		return err
	}

	layers, err := filepath.Glob(filepath.Join(launchDir, "*"))
	if err != nil {
		return err
	}

	for _, layer := range layers {
		if filepath.Base(layer) == "config" {
			err = os.Rename(filepath.Join(launchDir, "config", "metadata.toml"), filepath.Join(buildDir, "metadata.toml"))
			if err != nil {
				return err
			}
		} else {
			err := os.Rename(layer, filepath.Join(depsDir, depsIndex, filepath.Base(layer)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Finalize(depsDir, depsIndex, profileDir string) error {
	files, err := filepath.Glob(filepath.Join(depsDir, depsIndex, "*", "*", "profile.d", "*"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err := os.Rename(file, filepath.Join(profileDir, filepath.Base(file)))
		if err != nil {
			return err
		}
	}

	binDirs, err := filepath.Glob(filepath.Join(depsDir, depsIndex, "*", "*", "bin"))
	if err != nil {
		return err
	}

	for i, dir := range binDirs {
		binDirs[i] = strings.Replace(dir, filepath.Clean(depsDir), `$DEPS_DIR`, 1)
	}

	script, err := os.OpenFile(filepath.Join(profileDir, depsIndex+".sh"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer script.Close()

	setupPathTemplate, err := template.New("setupPathTemplate").Parse(setupPathContent)
	if err != nil {
		return err
	}

	return setupPathTemplate.Execute(script, binDirs)
}

type inputMetadata struct {
	Processes []struct {
		Type    string
		Command string
	}
}

func (i *inputMetadata) findCommand(processType string) (string, error) {
	for _, p := range i.Processes {
		if p.Type == processType {
			return p.Command, nil
		}
	}
	return "", fmt.Errorf("unable to find process with type %s in launch metadata", processType)
}

type outputMetadata struct {
	DefaultProcessTypes struct {
		Web string
	} `yaml:"default_process_types"`
}

func Release(buildDir string, writer io.Writer) error {
	metadataFile, input := filepath.Join(buildDir, "metadata.toml"), inputMetadata{}
	_, err := toml.DecodeFile(metadataFile, &input)

	defer os.Remove(metadataFile)

	webCommand, err := input.findCommand("web")
	if err != nil {
		return err
	}

	output := outputMetadata{DefaultProcessTypes: struct{ Web string }{Web: webCommand}}
	return yaml.NewEncoder(writer).Encode(output)
}