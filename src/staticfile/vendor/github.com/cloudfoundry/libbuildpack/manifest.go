package libbuildpack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
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
	File       string     `yaml:"file"`
	SHA256     string     `yaml:"sha256"`
	CFStacks   []string   `yaml:"cf_stacks"`
}

type Manifest struct {
	LanguageString  string            `yaml:"language"`
	DefaultVersions []Dependency      `yaml:"default_versions"`
	ManifestEntries []ManifestEntry   `yaml:"dependencies"`
	Deprecations    []DeprecationDate `yaml:"dependency_deprecation_dates"`
	manifestRootDir string
	appCacheDir     string
	filesInAppCache map[string]interface{}
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
	m.filesInAppCache = make(map[string]interface{})

	return &m, nil
}

func (m *Manifest) SetAppCacheDir(appCacheDir string) (err error) {
	m.appCacheDir, err = filepath.Abs(filepath.Join(appCacheDir, "dependencies"))
	return
}
func (m *Manifest) replaceDefaultVersion(oDep Dependency) {
	replaced := false
	for idx, mDep := range m.DefaultVersions {
		if mDep.Name == oDep.Name {
			replaced = true
			m.DefaultVersions[idx] = oDep
		}
	}
	if !replaced {
		m.DefaultVersions = append(m.DefaultVersions, oDep)
	}
}
func (m *Manifest) replaceManifestEntry(oEntry ManifestEntry) {
	oDep := oEntry.Dependency
	replaced := false
	for idx, mEntry := range m.ManifestEntries {
		mDep := mEntry.Dependency
		if mDep.Name == oDep.Name && mDep.Version == oDep.Version {
			replaced = true
			m.ManifestEntries[idx] = mEntry
		}
	}
	if !replaced {
		m.ManifestEntries = append(m.ManifestEntries, oEntry)
	}
}

func (m *Manifest) ApplyOverride(depsDir string) error {
	files, err := filepath.Glob(filepath.Join(depsDir, "*", "override.yml"))
	if err != nil {
		return err
	}

	for _, file := range files {
		var overrideYml map[string]Manifest
		y := &YAML{}
		if err := y.Load(file, &overrideYml); err != nil {
			return err
		}

		if o, found := overrideYml[m.Language()]; found {
			for _, oDep := range o.DefaultVersions {
				m.replaceDefaultVersion(oDep)
			}
			for _, oEntry := range o.ManifestEntries {
				m.replaceManifestEntry(oEntry)
			}
		}
	}

	return nil
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

	if strings.HasSuffix(entry.URI, ".sh") {
		return os.Rename(tmpFile, outputDir)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	if strings.HasSuffix(entry.URI, ".zip") {
		return ExtractZip(tmpFile, outputDir)
	}

	if strings.HasSuffix(entry.URI, ".tar.xz") {
		return ExtractTarXz(tmpFile, outputDir)
	}

	return ExtractTarGz(tmpFile, outputDir)
}

func (m *Manifest) warnNewerPatch(dep Dependency) error {
	versions := m.AllDependencyVersions(dep.Name)

	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return nil
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
	matchVersion := func(versionLine, depVersion string) bool {
		return versionLine == depVersion
	}

	v, err := semver.NewVersion(dep.Version)
	if err == nil {
		matchVersion = func(versionLine, depVersion string) bool {
			constraint, err := semver.NewConstraint(versionLine)
			if err != nil {
				return false
			}

			return constraint.Check(v)
		}
	}

	for _, deprecation := range m.Deprecations {
		if deprecation.Name != dep.Name {
			continue
		}
		if !matchVersion(deprecation.VersionLine, dep.Version) {
			continue
		}

		eolTime, err := time.Parse(dateFormat, deprecation.Date)
		if err != nil {
			return err
		}

		if eolTime.Sub(m.currentTime) < thirtyDays {
			m.log.Warning(endOfLifeWarning(dep.Name, deprecation.VersionLine, deprecation.Date, deprecation.Link))
		}
	}
	return nil
}

func fetchCachedBuildpackDependency(entry *ManifestEntry, outputFile, manifestRootDir string, manifestLog *Logger) error {
	source := entry.File
	if !filepath.IsAbs(source) {
		source = filepath.Join(manifestRootDir, source)
	}
	manifestLog.Info("Copy [%s]", source)
	if err := CopyFile(source, outputFile); err != nil {
		return err
	}
	return deleteBadFile(entry, outputFile)
}

func deleteBadFile(entry *ManifestEntry, outputFile string) error {
	if err := checkSha256(outputFile, entry.SHA256); err != nil {
		os.Remove(outputFile)
		return err
	}
	return nil
}

func downloadDependency(entry *ManifestEntry, outputFile string, logger *Logger) error {
	filteredURI, err := filterURI(entry.URI)
	if err != nil {
		return err
	}
	logger.Info("Download [%s]", filteredURI)
	err = downloadFile(entry.URI, outputFile)
	if err != nil {
		return err
	}

	return deleteBadFile(entry, outputFile)
}

func (m *Manifest) fetchAppCachedBuildpackDependency(entry *ManifestEntry, outputFile string) error {
	shaURI := sha256.Sum256([]byte(entry.URI))
	cacheFile := filepath.Join(m.appCacheDir, hex.EncodeToString(shaURI[:]), filepath.Base(entry.URI))

	m.filesInAppCache[cacheFile] = true
	m.filesInAppCache[filepath.Dir(cacheFile)] = true

	foundCacheFile, err := FileExists(cacheFile)
	if err != nil {
		return err
	}

	if foundCacheFile {
		m.log.Info("Copy [%s]", cacheFile)
		if err := CopyFile(cacheFile, outputFile); err != nil {
			return err
		}
		return deleteBadFile(entry, outputFile)
	}

	if err := downloadDependency(entry, outputFile, m.log); err != nil {
		return err
	}
	if err := CopyFile(outputFile, cacheFile); err != nil {
		return err
	}

	return nil
}

func (m *Manifest) FetchDependency(dep Dependency, outputFile string) error {
	entry, err := m.getEntry(dep)
	if err != nil {
		return err
	}

	if entry.File != "" { // this file is cached by the buildpack
		return fetchCachedBuildpackDependency(entry, outputFile, m.manifestRootDir, m.log)
	}

	if m.appCacheDir != "" { // this buildpack caches dependencies in the app cache
		return m.fetchAppCachedBuildpackDependency(entry, outputFile)
	}

	return downloadDependency(entry, outputFile, m.log)
}

func (m *Manifest) CleanupAppCache() error {
	pathsToDelete := []string{}

	if err := filepath.Walk(m.appCacheDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Failed while cleaning up app cache; couldn't look at %s because: %v", path, err)
		}
		if path == m.appCacheDir {
			return nil
		}
		if _, ok := m.filesInAppCache[path]; !ok {
			pathsToDelete = append(pathsToDelete, path)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, path := range pathsToDelete {
		m.log.Debug("Deleting cached file: %s", path)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("Failed while cleaning up app cache; couldn't delete %s because: %v", path, err)
		}
	}

	return nil
}

func (m *Manifest) entrySupportsCurrentStack(entry *ManifestEntry) bool {
	stack := os.Getenv("CF_STACK")
	if stack == "" {
		return true
	}

	for _, s := range entry.CFStacks {
		if s == stack {
			return true
		}
	}

	return false
}

func (m *Manifest) AllDependencyVersions(depName string) []string {
	var depVersions []string

	for _, e := range m.ManifestEntries {
		if e.Dependency.Name == depName && m.entrySupportsCurrentStack(&e) {
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
		if e.Dependency == dep && m.entrySupportsCurrentStack(&e) {
			return &e, nil
		}
	}

	m.log.Error(dependencyMissingError(m, dep))
	return nil, fmt.Errorf("dependency %s %s not found", dep.Name, dep.Version)
}

func (m *Manifest) IsCached() bool {
	dependenciesDir := filepath.Join(m.manifestRootDir, "dependencies")

	isCached, err := FileExists(dependenciesDir)
	if err != nil {
		m.log.Warning("Error determining if buildpack is cached: %s", err.Error())
	}

	return isCached
}
