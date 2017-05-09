package libbuildpack

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var stagingEnvVarDirs = map[string]string{
	"PATH":            "bin",
	"LD_LIBRARY_PATH": "lib",
	"INCLUDE_PATH":    "include",
	"CPATH":           "include",
	"CPPPATH":         "include",
	"PKG_CONFIG_PATH": "pkgconfig",
}

var launchEnvVarDirs = map[string]string{
	"PATH":            "bin",
	"LD_LIBRARY_PATH": "lib",
}

func SetStagingEnvironment(depsDir string) error {
	for envVar, dir := range stagingEnvVarDirs {
		oldVal := os.Getenv(envVar)

		depsPaths, err := existingDepsDirs(depsDir, dir, depsDir)
		if err != nil {
			return err
		}

		if len(depsPaths) != 0 {
			if len(oldVal) > 0 {
				depsPaths = append(depsPaths, oldVal)
			}
			os.Setenv(envVar, strings.Join(depsPaths, ":"))
		}
	}

	depsPaths, err := existingDepsDirs(depsDir, "env", depsDir)
	if err != nil {
		return err
	}

	for _, dir := range depsPaths {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.Mode().IsRegular() {
				val, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
				if err != nil {
					return err
				}

				if err := os.Setenv(file.Name(), string(val)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SetLaunchEnvironment(depsDir, buildDir string) error {
	scriptContents := ""

	for envVar, dir := range launchEnvVarDirs {
		depsPaths, err := existingDepsDirs(depsDir, dir, "$DEPS_DIR")
		if err != nil {
			return err
		}

		if len(depsPaths) != 0 {
			scriptContents += fmt.Sprintf(`export %[1]s=%[2]s$([[ ! -z "${%[1]s:-}" ]] && echo ":$%[1]s")`, envVar, strings.Join(depsPaths, ":"))
		  scriptContents += "\n"
	  }
	}

	if err := os.MkdirAll(filepath.Join(buildDir, ".profile.d"), 0755); err != nil {
		return err
	}

	scriptLocation := filepath.Join(buildDir, ".profile.d", "000_multi-supply.sh")
	if err := writeToFile(strings.NewReader(scriptContents), scriptLocation, 0755); err != nil {
		return err
	}

	profileDirs, err := existingDepsDirs(depsDir, "profile.d", depsDir)
	if err != nil {
		return err
	}

	for _, dir := range profileDirs {
		sections := strings.Split(dir, string(filepath.Separator))
		if len(sections) < 2 {
			return errors.New("invalid dep dir")
		}

		depsIdx := sections[len(sections)-2]

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.Mode().IsRegular() {
				src := filepath.Join(dir, file.Name())
				dest := filepath.Join(buildDir, ".profile.d", depsIdx+"_"+file.Name())

				if err := CopyFile(src, dest); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func existingDepsDirs(depsDir, subDir, prefix string) ([]string, error) {
	files, err := ioutil.ReadDir(depsDir)
	if err != nil {
		return nil, err
	}

	var existingDirs []string

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		filesystemDir := filepath.Join(depsDir, file.Name(), subDir)
		dirToJoin := filepath.Join(prefix, file.Name(), subDir)

		addToDirs, err := FileExists(filesystemDir)
		if err != nil {
			return nil, err
		}

		if addToDirs {
			existingDirs = append([]string{dirToJoin}, existingDirs...)
		}
	}

	return existingDirs, nil
}
