package glow

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

var PackagerBuildpackRegexp = regexp.MustCompile(`Packaged Shimmed Buildpack at: ([\w-\.^]*.zip)`)

//go:generate faux --interface Packager --output fakes/packager.go
type Packager interface {
	Package(dir string, stack string, options PackageOptions) (stdout string, stderr string, err error)
}

type Archiver struct {
	packager Packager
}

func NewArchiver(packager Packager) Archiver {
	return Archiver{
		packager: packager,
	}
}

func (a Archiver) Archive(dir, stack, tag string, cached bool) (string, error) {
	version, err := ioutil.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return "", err
	}

	_, stderr, err := a.packager.Package(dir, stack, PackageOptions{
		Cached:  cached,
		Version: fmt.Sprintf("%s-%s", string(version), tag),
	})
	if err != nil {
		return "", fmt.Errorf("running package command failed: %s", err)
	}

	matches := PackagerBuildpackRegexp.FindStringSubmatch(stderr)
	if len(matches) != 2 {
		return "", fmt.Errorf("failed to find archive file path in output:\n%s", stderr)
	}

	return filepath.Join(dir, matches[1]), nil
}
