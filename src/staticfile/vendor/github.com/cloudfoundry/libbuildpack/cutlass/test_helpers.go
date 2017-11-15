package cutlass

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/packager"
)

type VersionedBuildpackPackage struct {
	Version string
	File    string
}

func FindRoot() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	for {
		if dir == "/" {
			return "", fmt.Errorf("Could not find VERSION in the directory hierarchy")
		}
		if exist, err := libbuildpack.FileExists(filepath.Join(dir, "VERSION")); err != nil {
			return "", err
		} else if exist {
			return dir, nil
		}
		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}
	}
}

func PackageUniquelyVersionedBuildpackExtra(name, version string, cached bool) (VersionedBuildpackPackage, error) {
	bpDir, err := FindRoot()
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	file, err := packager.Package(bpDir, packager.CacheDir, version, cached)
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	err = CreateOrUpdateBuildpack(name, file)
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	return VersionedBuildpackPackage{
		Version: version,
		File:    file,
	}, nil
}

func PackageUniquelyVersionedBuildpack() (VersionedBuildpackPackage, error) {
	bpDir, err := FindRoot()
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	data, err := ioutil.ReadFile(filepath.Join(bpDir, "VERSION"))
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}
	buildpackVersion := strings.TrimSpace(string(data))
	buildpackVersion = fmt.Sprintf("%s.%s", buildpackVersion, time.Now().Format("20060102150405"))

	var manifest struct {
		Language string `yaml:"language"`
	}
	err = libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest)
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	return PackageUniquelyVersionedBuildpackExtra(manifest.Language, buildpackVersion, Cached)
}

func CopyCfHome() error {
	cf_home := os.Getenv("CF_HOME")
	if cf_home == "" {
		cf_home = os.Getenv("HOME")
	}
	cf_home_new, err := ioutil.TempDir("", "cf-home-copy")
	if err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(cf_home_new, ".cf"), 0755); err != nil {
		return err
	}
	if err := libbuildpack.CopyDirectory(filepath.Join(cf_home, ".cf"), filepath.Join(cf_home_new, ".cf")); err != nil {
		return err
	}
	return os.Setenv("CF_HOME", cf_home_new)

}

func SeedRandom() {
	seed := int64(time.Now().Nanosecond() + os.Getpid())
	rand.Seed(seed)
}

func RemovePackagedBuildpack(buildpack VersionedBuildpackPackage) error {
	return os.Remove(buildpack.File)
}
