package libbuildpack

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Stager struct {
	BuildDir string
	CacheDir string
	DepsDir  string
	DepsIdx  string
	Manifest Manifest
	Log      Logger
	Command  CommandRunner
}

func NewStager(args []string, logger Logger) (*Stager, error) {
	bpDir, err := GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		return nil, err
	}

	manifest, err := NewManifest(bpDir, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		return nil, err
	}

	buildDir := args[0]
	cacheDir := args[1]
	depsDir := ""
	depsIdx := ""

	if len(args) >= 4 {
		depsDir = args[2]
		depsIdx = args[3]
	}

	s := &Stager{BuildDir: buildDir,
		CacheDir: cacheDir,
		DepsDir:  depsDir,
		DepsIdx:  depsIdx,
		Manifest: manifest,
		Log:      logger,
		Command:  NewCommandRunner()}

	return s, nil
}

func GetBuildpackDir() (string, error) {
	var err error

	bpDir := os.Getenv("BUILDPACK_DIR")

	if bpDir == "" {
		bpDir, err = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), ".."))

		if err != nil {
			return "", err
		}
	}

	return bpDir, nil
}

func (s *Stager) DepDir() string {
	return filepath.Join(s.DepsDir, s.DepsIdx)
}

func (s *Stager) WriteConfigYml(config interface{}) error {
	if config == nil {
		config = map[interface{}]interface{}{}
	}
	data := map[string]interface{}{"name": s.Manifest.Language(), "config": config}
	return NewYAML().Write(filepath.Join(s.DepDir(), "config.yml"), data)
}

func (s *Stager) WriteEnvFile(envVar, envVal string) error {
	envDir := filepath.Join(s.DepDir(), "env")

	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err

	}

	return ioutil.WriteFile(filepath.Join(envDir, envVar), []byte(envVal), 0644)
}

func (s *Stager) AddBinDependencyLink(destPath, sourceName string) error {
	binDir := filepath.Join(s.DepDir(), "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	relPath, err := filepath.Rel(binDir, destPath)
	if err != nil {
		return err
	}

	return os.Symlink(relPath, filepath.Join(binDir, sourceName))
}

func (s *Stager) LinkDirectoryInDepDir(destDir, depSubDir string) error {
	srcDir := filepath.Join(s.DepDir(), depSubDir)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(destDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		relPath, err := filepath.Rel(srcDir, filepath.Join(destDir, file.Name()))
		if err != nil {
			return err
		}

		if err := os.Symlink(relPath, filepath.Join(srcDir, file.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (s *Stager) CheckBuildpackValid() error {
	version, err := s.Manifest.Version()
	if err != nil {
		s.Log.Error("Could not determine buildpack version: %s", err.Error())
		return err
	}

	s.Log.BeginStep("%s Buildpack version %s", strings.Title(s.Manifest.Language()), version)

	err = s.Manifest.CheckStackSupport()
	if err != nil {
		s.Log.Error("Stack not supported by buildpack: %s", err.Error())
		return err
	}

	s.Manifest.CheckBuildpackVersion(s.CacheDir)

	return nil
}

func (s *Stager) StagingComplete() {
	s.Manifest.StoreBuildpackMetadata(s.CacheDir)
}

func (s *Stager) ClearCache() error {
	files, err := ioutil.ReadDir(s.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(filepath.Join(s.CacheDir, file.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Stager) ClearDepDir() error {
	files, err := ioutil.ReadDir(s.DepDir())
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Name() != "config.yml" {
			if err := os.RemoveAll(filepath.Join(s.DepDir(), file.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Stager) WriteProfileD(scriptName, scriptContents string) error {
	profileDir := filepath.Join(s.DepDir(), "profile.d")

	err := os.MkdirAll(profileDir, 0755)
	if err != nil {
		return err
	}

	return writeToFile(strings.NewReader(scriptContents), filepath.Join(profileDir, scriptName), 0755)
}
