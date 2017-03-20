package libbuildpack

import (
	"os"
	"path/filepath"
	"strings"
)

func WriteProfileD(buildDir, scriptName, scriptContents string) error {
	err := os.MkdirAll(filepath.Join(buildDir, ".profile.d"), 0755)
	if err != nil {
		return err
	}

	return writeToFile(strings.NewReader(scriptContents), filepath.Join(buildDir, ".profile.d", scriptName), 0755)
}
