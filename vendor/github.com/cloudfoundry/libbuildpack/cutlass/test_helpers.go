package cutlass

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass/glow"
	"github.com/cloudfoundry/libbuildpack/packager"
	"github.com/cloudfoundry/packit"
	"gopkg.in/yaml.v2"
)

type VersionedBuildpackPackage struct {
	Version string
	File    string
}

var manifest struct {
	Language string `yaml:"language"`
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

func PackageUniquelyVersionedBuildpackExtra(name, version, stack string, cached, stackAssociationSupported bool) (VersionedBuildpackPackage, error) {
	bpDir, err := FindRoot()
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to find root: %v", err)
	}

	var file string
	if os.Getenv("BUILDPACK_FILE") == "" {
		if compileExtension, err := isCompileExtensionBuildpack(bpDir); err != nil {
			return VersionedBuildpackPackage{}, fmt.Errorf("Failed to decide if this is a compile extension buildpack: %v", err)
		} else if compileExtension {
			file, err = packager.CompileExtensionPackage(bpDir, version, cached, stack)
			if err != nil {
				return VersionedBuildpackPackage{}, fmt.Errorf("Failed to package as a compile extension buildpack: %v", err)
			}
		} else {
			file, err = packager.Package(bpDir, packager.CacheDir, version, stack, cached)
			if err != nil {
				return VersionedBuildpackPackage{}, fmt.Errorf("Failed to package buildpack: %v", err)
			}
		}
	} else {
		file = os.Getenv("BUILDPACK_FILE")
		version, err = readVersionFromZip(file)
		if err != nil {
			return VersionedBuildpackPackage{}, err
		}
	}

	if !stackAssociationSupported {
		stack = ""
	}

	err = CreateOrUpdateBuildpack(name, file, stack)
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to create or update buildpack: %v", err)
	}

	return VersionedBuildpackPackage{
		Version: version,
		File:    file,
	}, nil
}

func isCompileExtensionBuildpack(bpDir string) (bool, error) {
	var manifest struct {
		IncludeFiles []string `yaml:"include_files"`
	}
	if err := libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest); err != nil {
		return false, err
	}

	return len(manifest.IncludeFiles) == 0, nil
}

func PackageUniquelyVersionedBuildpack(stack string, stackAssociationSupported bool) (VersionedBuildpackPackage, error) {
	bpDir, err := FindRoot()
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to find root: %v", err)
	}

	data, err := ioutil.ReadFile(filepath.Join(bpDir, "VERSION"))
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to read VERSION file: %v", err)
	}
	buildpackVersion := strings.TrimSpace(string(data))
	buildpackVersion = fmt.Sprintf("%s.%s", buildpackVersion, time.Now().Format("20060102150405"))

	err = libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest)
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to load manifest.yml file: %v", err)
	}

	return PackageUniquelyVersionedBuildpackExtra(strings.Replace(manifest.Language, "-", "_", -1), buildpackVersion, stack, Cached, stackAssociationSupported)
}

func PackageShimmedBuildpack(bpDir, stack string) (VersionedBuildpackPackage, error) {
	var (
		name    string
		version string
		err     error
	)

	DefaultLogger.Debug("package-shimmed-buildpack")

	bpFilePath := os.Getenv("BUILDPACK_FILE")
	if bpFilePath == "" {
		session := DefaultLogger.Session("package-shim")

		cnb2cf := packit.NewExecutable(glow.ExecutableName, session)
		cli := glow.NewCLI(cnb2cf)
		archiver := glow.NewArchiver(cli)

		timestamp := time.Now().Format("20060102150405")
		bpFilePath, err = archiver.Archive(bpDir, stack, timestamp, Cached)
		if err != nil {
			return VersionedBuildpackPackage{}, fmt.Errorf("failed to package shimmed buildpack: %s", err)
		}
	}

	if !filepath.IsAbs(bpFilePath) {
		bpFilePath = filepath.Join(bpDir, bpFilePath)
	}

	version, name, err = extractVersionAndNameFromPackagedBuildpack(bpFilePath)
	if err != nil {
		return VersionedBuildpackPackage{}, err
	}

	name = strings.Replace(name, "-", "_", -1)
	err = CreateOrUpdateBuildpack(name, bpFilePath, stack)
	if err != nil {
		return VersionedBuildpackPackage{}, fmt.Errorf("Failed to create or update buildpack: %v", err)
	}

	return VersionedBuildpackPackage{
		Version: version,
		File:    bpFilePath,
	}, nil
}

func extractVersionAndNameFromPackagedBuildpack(bpFilePath string) (version string, name string, err error) {
	r, err := zip.OpenReader(bpFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "VERSION" {
			versionContents, err := readZipFile(f)
			if err != nil {
				return "", "", nil
			}

			version = versionContents.String()
		} else if f.Name == "manifest.yml" {
			manifestContents, err := readZipFile(f)
			if err != nil {
				return "", "", nil
			}

			if err := yaml.Unmarshal(manifestContents.Bytes(), &manifest); err != nil {
				return "", "", nil
			}

			name = manifest.Language
		}
	}
	return version, name, nil
}

func readZipFile(f *zip.File) (*bytes.Buffer, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, rc); err != nil {
		return nil, err
	}

	return buf, nil
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
	if os.Getenv("BUILDPACK_FILE") == "" {
		return os.Remove(buildpack.File)
	}
	return nil
}

func readVersionFromZip(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}

	defer r.Close()

	for _, f := range r.File {
		if f.Name != "VERSION" {
			continue
		}

		rc, err := f.Open()
		defer rc.Close()
		if err != nil {
			return "", err
		}

		out, err := ioutil.ReadAll(rc)
		if err != nil {
			return "", err

		}

		return strings.TrimSpace(fmt.Sprintf("%s", out)), nil
	}
	return "", fmt.Errorf("Could not find VERSION file from buildpack artifact: %s", filePath)
}
