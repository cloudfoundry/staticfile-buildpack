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

type Installer struct {
	manifest        *Manifest
	appCacheDir     string
	filesInAppCache map[string]interface{}
	versionLine     *map[string]string
}

func NewInstaller(manifest *Manifest) *Installer {
	return &Installer{manifest, "", make(map[string]interface{}), &map[string]string{}}
}

func (i *Installer) SetAppCacheDir(appCacheDir string) (err error) {
	i.appCacheDir, err = filepath.Abs(filepath.Join(appCacheDir, "dependencies"))
	return
}

func (i *Installer) InstallDependency(dep Dependency, outputDir string) error {
	i.manifest.log.BeginStep("Installing %s %s", dep.Name, dep.Version)

	tmpDir, err := ioutil.TempDir("", "downloads")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "archive")

	entry, err := i.manifest.GetEntry(dep)
	if err != nil {
		return err
	}

	err = i.FetchDependency(dep, tmpFile)
	if err != nil {
		return err
	}

	err = i.warnNewerPatch(dep)
	if err != nil {
		return err
	}

	err = i.warnEndOfLife(dep)
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

func (i *Installer) warnNewerPatch(dep Dependency) error {
	versions := i.manifest.AllDependencyVersions(dep.Name)

	v, err := semver.NewVersion(dep.Version)
	if err != nil {
		return nil
	}

	minor := fmt.Sprintf("%v", v.Minor())
	versionLine := *i.GetVersionLine()
	if versionLine[dep.Name] == "minor" {
		minor = "x"
	}
	constraint := fmt.Sprintf("%d.%s.x", v.Major(), minor)

	latest, err := FindMatchingVersion(constraint, versions)
	if err != nil {
		return err
	}

	if latest != dep.Version {
		i.manifest.log.Warning(outdatedDependencyWarning(dep, latest))
	}

	return nil
}

func (i *Installer) warnEndOfLife(dep Dependency) error {
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

	for _, deprecation := range i.manifest.Deprecations {
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

		if eolTime.Sub(i.manifest.currentTime) < thirtyDays {
			i.manifest.log.Warning(endOfLifeWarning(dep.Name, deprecation.VersionLine, deprecation.Date, deprecation.Link))
		}
	}
	return nil
}

func (i *Installer) FetchDependency(dep Dependency, outputFile string) error {
	entry, err := i.manifest.GetEntry(dep)
	if err != nil {
		return err
	}

	if entry.File != "" { // this file is cached by the buildpack
		return fetchCachedBuildpackDependency(entry, outputFile, i.manifest.manifestRootDir, i.manifest.log)
	}

	if i.appCacheDir != "" { // this buildpack caches dependencies in the app cache
		return i.fetchAppCachedBuildpackDependency(entry, outputFile)
	}

	return downloadDependency(entry, outputFile, i.manifest.log)
}

func (i *Installer) CleanupAppCache() error {
	pathsToDelete := []string{}

	if err := filepath.Walk(i.appCacheDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Failed while cleaning up app cache; couldn't look at %s because: %v", path, err)
		}
		if path == i.appCacheDir {
			return nil
		}
		if _, ok := i.filesInAppCache[path]; !ok {
			pathsToDelete = append(pathsToDelete, path)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, path := range pathsToDelete {
		i.manifest.log.Debug("Deleting cached file: %s", path)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("Failed while cleaning up app cache; couldn't delete %s because: %v", path, err)
		}
	}

	return nil
}

func (i *Installer) InstallOnlyVersion(depName string, installDir string) error {
	depVersions := i.manifest.AllDependencyVersions(depName)

	if len(depVersions) > 1 {
		return fmt.Errorf("more than one version of %s found", depName)
	} else if len(depVersions) == 0 {
		return fmt.Errorf("no versions of %s found", depName)
	}

	dep := Dependency{Name: depName, Version: depVersions[0]}
	return i.InstallDependency(dep, installDir)
}

func (i *Installer) fetchAppCachedBuildpackDependency(entry *ManifestEntry, outputFile string) error {
	shaURI := sha256.Sum256([]byte(entry.URI))
	cacheFile := filepath.Join(i.appCacheDir, hex.EncodeToString(shaURI[:]), filepath.Base(entry.URI))

	i.filesInAppCache[cacheFile] = true
	i.filesInAppCache[filepath.Dir(cacheFile)] = true

	foundCacheFile, err := FileExists(cacheFile)
	if err != nil {
		return err
	}

	if foundCacheFile {
		i.manifest.log.Info("Copy [%s]", cacheFile)
		if err := CopyFile(cacheFile, outputFile); err != nil {
			return err
		}
		return deleteBadFile(entry, outputFile)
	}

	if err := downloadDependency(entry, outputFile, i.manifest.log); err != nil {
		return err
	}
	if err := CopyFile(outputFile, cacheFile); err != nil {
		return err
	}

	return nil
}

func (i *Installer) SetVersionLine(depName string, line string) {
	(*i.versionLine)[depName] = line
}

func (i *Installer) GetVersionLine() *map[string]string {
	return i.versionLine
}
