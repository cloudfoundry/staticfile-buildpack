package libbuildpack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var envVarDirs = map[string]string{
	"PATH":            "bin",
	"LD_LIBRARY_PATH": "ld_library_path",
}

func SetEnvironmentFromSupply(depsDir string) error {
	for envVar, dir := range envVarDirs {
		oldVal := os.Getenv(envVar)

		depsPaths, err := joinExistingDepsDirs(depsDir, dir, depsDir)
		if err != nil {
			return err
		}

		if depsPaths != "" {
			os.Setenv(envVar, fmt.Sprintf("%s:%s", depsPaths, oldVal))
		}
	}

	return nil
}

func joinExistingDepsDirs(depsDir, subDir, prefix string) (string, error) {
	dirs, err := ioutil.ReadDir(depsDir)
	if err != nil {
		return "", err
	}

	existingDirs := ""

	for _, dir := range dirs {
		filesystemDir := filepath.Join(depsDir, dir.Name(), subDir)
		dirToJoin := filepath.Join(prefix, dir.Name(), subDir)

		addToDirs, err := FileExists(filesystemDir)
		if err != nil {
			return "", err
		}

		if addToDirs {
			existingDirs = fmt.Sprintf("%s:%s", dirToJoin, existingDirs)
		}
	}

	return strings.TrimRight(existingDirs, ":"), nil
}

func WriteProfileDFromSupply(depsDir, buildDir string) error {
	scriptContents := ""

	for envVar, dir := range envVarDirs {
		depsPaths, err := joinExistingDepsDirs(depsDir, dir, "$DEPS_DIR")
		if err != nil {
			return err
		}

		if depsPaths != "" {
			scriptContents += fmt.Sprintf("export %s=%s:$%s\n", envVar, depsPaths, envVar)
		}
	}

	return WriteProfileD(buildDir, "00-multi-supply.sh", scriptContents)
}
