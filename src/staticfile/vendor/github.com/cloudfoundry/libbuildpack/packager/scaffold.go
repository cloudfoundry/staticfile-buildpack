package packager

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/cloudfoundry/libbuildpack"
	"gopkg.in/yaml.v2"
)

type sha struct {
	Sha map[string]string `yaml:"sha"`
}

func generateAssets(bpDir, languageName string, force bool) error {

	language := func() string {
		return languageName
	}

	funcMap := template.FuncMap{
		"LANGUAGE": language,
	}

	shas, err := readShaYML(bpDir)
	if err != nil {
		return err
	}

	fmt.Fprintln(Stdout, "Creating directory and files")
	if err := OurRestoreAssets(bpDir, "", funcMap, shas, force); err != nil {
		return err
	}

	libbuildpack.NewYAML().Write(filepath.Join(bpDir, "sha.yml"), sha{
		Sha: shas,
	})

	if err := setupDep(bpDir, languageName); err != nil {
		return err
	}

	return nil

}

func setupDep(bpDir, languageName string) error {
	fmt.Fprintln(Stdout, "Installing dep")
	tmpDir, err := ioutil.TempDir("", "gopath")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "get", "-u", "github.com/golang/dep/cmd/dep")
	cmd.Stdout = Stdout
	cmd.Stderr = Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOBIN=%s/.bin", bpDir), fmt.Sprintf("GOPATH=%s", tmpDir))
	cmd.Dir = filepath.Join(bpDir, "src", languageName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go get -u github.com/golang/dep/cmd/dep: %s", err)
	}
	if err := os.RemoveAll(tmpDir); err != nil {
		return err
	}

	fmt.Fprintln(Stdout, "Running dep ensure")

	canonicalBpDir, err := filepath.EvalSymlinks(bpDir)
	if err != nil {
		return err
	}

	cmd = exec.Command(filepath.Join(bpDir, ".bin", "dep"), "ensure")
	cmd.Stdout = Stdout
	cmd.Stderr = Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOBIN=%s/.bin", bpDir), fmt.Sprintf("GOPATH=%s", canonicalBpDir))
	cmd.Dir = filepath.Join(bpDir, "src", languageName)

	if err := cmd.Run(); err != nil {
		fmt.Printf("GOPATH=%s\n", canonicalBpDir)
		return fmt.Errorf("dep ensure: %s", err)
	}
	return nil
}

// TODO: maybe make scaffold into a struct and split up packager and scaffold if they don't share anything
func Scaffold(bpDir string, languageName string) error {
	return generateAssets(bpDir, languageName, false)
}

func Upgrade(bpDir string, force bool) error {
	manifest, err := readManifest(bpDir)
	if err != nil {
		return fmt.Errorf("error opening manifest: %s", err)
	}

	return generateAssets(bpDir, manifest.Language, force)
}

func readShaYML(bpDir string) (map[string]string, error) {
	if found, err := libbuildpack.FileExists(filepath.Join(bpDir, "sha.yml")); err != nil {
		return map[string]string{}, err
	} else if found {
		shas := &sha{}
		data, err := ioutil.ReadFile(filepath.Join(bpDir, "sha.yml"))
		if err != nil {
			return map[string]string{}, err
		}
		if err := yaml.Unmarshal(data, shas); err != nil {
			return map[string]string{}, err
		}
		return shas.Sha, nil
	}
	return map[string]string{}, nil
}

func readManifest(bpDir string) (*Manifest, error) {
	manifest := &Manifest{}
	data, err := ioutil.ReadFile(filepath.Join(bpDir, "manifest.yml"))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}
