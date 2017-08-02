package cutlass

import (
	"github.com/cloudfoundry/libbuildpack"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func CopyFixture(srcDir string) (string, error) {
	destDir, err := ioutil.TempDir("", "cutlass-fixture-copy")
	if err != nil {
		return "", err
	}
	if err := libbuildpack.CopyDirectory(srcDir, destDir); err != nil {
		return "", err
	}
	return destDir, nil
}

func fileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}
