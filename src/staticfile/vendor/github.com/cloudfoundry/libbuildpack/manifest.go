package libbuildpack

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

const dateFormat = "2006-01-02"
const thirtyDays = time.Hour * 24 * 30

type Dependency struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type DeprecationDate struct {
	Name        string `yaml:"name"`
	VersionLine string `yaml:"version_line"`
	Date        string `yaml:"date"`
	Link        string `yaml:"link"`
}

type ManifestEntry struct {
	Dependency Dependency `yaml:",inline"`
	URI        string     `yaml:"uri"`
	MD5        string     `yaml:"md5"`
	CFStacks   []string   `yaml:"cf_stacks"`
}

type Manifest struct {
	LanguageString  string            `yaml:"language"`
	DefaultVersions []Dependency      `yaml:"default_versions"`
	ManifestEntries []ManifestEntry   `yaml:"dependencies"`
	Deprecations    []DeprecationDate `yaml:"dependency_deprecation_dates"`
	manifestRootDir string
	currentTime     time.Time
	log             *Logger
}

type BuildpackMetadata struct {
	Language string `yaml:"language"`
	Version  string `yaml:"version"`
}

func NewManifest(bpDir string, logger *Logger, currentTime time.Time) (*Manifest, error) {
	var m Manifest
	y := &YAML{}

	err := y.Load(filepath.Join(bpDir, "manifest.yml"), &m)
	if err != nil {
		return nil, err
	}

	m.manifestRootDir, err = filepath.Abs(bpDir)
	if err != nil {
		return nil, err
	}

	m.currentTime = currentTime
	m.log = logger

	return &m, nil
}

func (m *Manifest) RootDir() string {
	return m.manifestRootDir
}

func (m *Manifest) CheckBuildpackVersion(cacheDir string) {
	var md BuildpackMetadata
	y := &YAML{}

	err := y.Load(filepath.Join(cacheDir, "BUILDPACK_METADATA"), &md)
	if err != nil {
		return
	}

	if md.Language != m.Language() {
		return
	}

	version, err := m.Version()
	if err != nil {
		return
	}

	if md.Version != version {
		m.log.Warning("buildpack version changed from %s to %s", md.Version, version)
	}

	return
}

func (m *Manifest) StoreBuildpackMetadata(cacheDir string) {
	version, err := m.Version()
	if err != nil {
		return
	}

	md := BuildpackMetadata{Language: m.Language(), Version: version}

	if exists, _ := FileExists(cacheDir); exists {
		y := &YAML{}
		_ = y.Write(filepath.Join(cacheDir, "BUILDPACK_METADATA"), &md)
	}
}

func (m *Manifest) Language() string {
	return m.LanguageString
}

func (m *Manifest) Version() (string, error) {
	version, err := ioutil.ReadFile(filepath.Join(m.manifestRootDir, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("unable to read VERSION file %s", err)
	}

	return strings.TrimSpace(string(version)), nil
}

func (m *Manifest) CheckStackSupport() error {
	requiredStack := os.Getenv("CF_STACK")

	if len(m.ManifestEntries) == 0 {
		return nil
	}
	for _, entry := range m.ManifestEntries {
		for _, stack := range entry.CFStacks {
			if stack == requiredStack {
				return nil
			}
		}
	}
	return fmt.Errorf("required stack %s was not found", requiredStack)
}

func (m *Manifest) DefaultVersion(depName string) (Dependency, error) {
	var defaultVersion string
	var err error
	numDefaults := 0

	for _, defaultDep := range m.DefaultVersions {
		if depName == defaultDep.Name {
			defaultVersion = defaultDep.Version
			numDefaults++
		}
	}

	if numDefaults == 0 {
		err = fmt.Errorf("no default version for %s", depName)
	} else if numDefaults > 1 {
		err = fmt.Errorf("found %d default versions for %s", numDefaults, depName)
	}

	if err != nil {
		m.log.Error(defaultVersionsError)
		return Dependency{}, err
	}

	depVersions := m.AllDependencyVersions(depName)
	highestVersion, err := FindMatchingVersion(defaultVersion, depVersions)

	if err != nil {
		m.log.Error(defaultVersionsError)
		return Dependency{}, err
	}

	return Dependency{Name: depName, Version: highestVersion}, nil
}

func (m *Manifest) InstallDependency(dep Dependency, outputDir string) error {
	m.log.BeginStep("Installing %s %s", dep.Name, dep.Version)

	tmpDir, err := ioutil.TempDir("", "downloads")
	if err != nil {
		return err
	}
	tmpFile := filepath.Join(tmpDir, "archive")

	entry, err := m.getEntry(dep)
	if err != nil {
		return err
	}

	err = m.FetchDependency(dep, tmpFile)
	if err != nil {
		return err
	}

	err = m.warnNewerPatch(dep)
	if err != nil {
		return err
	}

	err = m.warnEndOfLife(dep)
	if err != nil {
		return err
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	if strings.HasSuffix(entry.URI, ".zip") {
		return ExtractZip(tmpFile, outputDir)
	}

	return ExtractTarGz(tmpFile, outputDir)
}

func (m *Manifest) warnNewerPatch(dep Dependency) error {
	versions := m.AllDependencyVersions(dep.Name)

	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return err
	}

	constraint := fmt.Sprintf("%d.%d.x", v.Major(), v.Minor())
	latest, err := FindMatchingVersion(constraint, versions)
	if err != nil {
		return err
	}

	if latest != dep.Version {
		m.log.Warning(outdatedDependencyWarning(dep, latest))
	}

	return nil
}

func (m *Manifest) warnEndOfLife(dep Dependency) error {
	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return err
	}

	for _, deprecation := range m.Deprecations {
		if deprecation.Name != dep.Name {
			continue
		}

		versionLine, err := semver.NewConstraint(deprecation.VersionLine)
		if err != nil {
			return err
		}

		eolTime, err := time.Parse(dateFormat, deprecation.Date)
		if err != nil {
			return err
		}
		if versionLine.Check(v) && eolTime.Sub(m.currentTime) < thirtyDays {
			m.log.Warning(endOfLifeWarning(dep.Name, deprecation.VersionLine, deprecation.Date, deprecation.Link))
		}
	}
	return nil
}

func (m *Manifest) FetchDependency(dep Dependency, outputFile string) error {
	entry, err := m.getEntry(dep)
	if err != nil {
		return err
	}

	filteredURI, err := filterURI(entry.URI)
	if err != nil {
		return err
	}

	if m.isCached() {
		source := filepath.Join(m.manifestRootDir, "dependencies", fmt.Sprintf("%x", md5.Sum([]byte(entry.URI))), path.Base(entry.URI))
		exists, err := FileExists(source)
		if err != nil {
			m.log.Warning("Error determining if cached file exists: %s", err.Error())
		}
		if !exists {
			r := strings.NewReplacer("/", "_", ":", "_", "?", "_", "&", "_")
			source = filepath.Join(m.manifestRootDir, "dependencies", r.Replace(filteredURI))
		}
		m.log.Info("Copy [%s]", source)
		err = CopyFile(source, outputFile)
	} else {
		m.log.Info("Download [%s]", filteredURI)
		err = downloadFile(entry.URI, outputFile)
	}
	if err != nil {
		return err
	}

	err = checkMD5(outputFile, entry.MD5)
	if err != nil {
		os.Remove(outputFile)
		return err
	}

	return nil
}

func (m *Manifest) AllDependencyVersions(depName string) []string {
	var depVersions []string

	for _, e := range m.ManifestEntries {
		if e.Dependency.Name == depName {
			depVersions = append(depVersions, e.Dependency.Version)
		}
	}

	return depVersions
}

func (m *Manifest) InstallOnlyVersion(depName string, installDir string) error {
	depVersions := m.AllDependencyVersions(depName)

	if len(depVersions) > 1 {
		return fmt.Errorf("more than one version of %s found", depName)
	} else if len(depVersions) == 0 {
		return fmt.Errorf("no versions of %s found", depName)
	}

	dep := Dependency{Name: depName, Version: depVersions[0]}
	return m.InstallDependency(dep, installDir)
}

func (m *Manifest) getEntry(dep Dependency) (*ManifestEntry, error) {
	for _, e := range m.ManifestEntries {
		if e.Dependency == dep {
			return &e, nil
		}
	}

	m.log.Error(dependencyMissingError(m, dep))
	return nil, fmt.Errorf("dependency %s %s not found", dep.Name, dep.Version)
}

func (m *Manifest) isCached() bool {
	dependenciesDir := filepath.Join(m.manifestRootDir, "dependencies")

	isCached, err := FileExists(dependenciesDir)
	if err != nil {
		m.log.Warning("Error determining if buildpack is cached: %s", err.Error())
	}

	return isCached
}
