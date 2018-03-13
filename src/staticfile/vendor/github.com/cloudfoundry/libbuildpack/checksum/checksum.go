package checksum

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Checksum struct {
	dir           string
	timestampFile string
}

func New(dir string) *Checksum {
	c := &Checksum{dir: dir}

	if f, err := ioutil.TempFile("", "checksum"); err == nil {
		f.Close()
		c.timestampFile = f.Name()
	}

	return c
}

func Do(dir string, debug func(format string, args ...interface{}), exec func() error) error {
	checksum := New(dir)
	if sum, err := checksum.calc(); err == nil {
		debug("Checksum Before (%s): %s", dir, sum)
	}
	err := exec()
	if sum, err := checksum.calc(); err == nil {
		debug("Checksum After (%s): %s", dir, sum)
	}

	if checksum.timestampFile != "" {
		if filesChanged, err := (&libbuildpack.Command{}).Output(checksum.dir, "find", ".", "-newer", checksum.timestampFile, "-not", "-path", "./.cloudfoundry/*", "-not", "-path", "./.cloudfoundry"); err == nil && filesChanged != "" {
			debug("Below files changed:")
			debug(filesChanged)
		}
	}
	return err
}

func (c *Checksum) calc() (string, error) {
	h := md5.New()
	err := filepath.Walk(c.dir, func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			relpath, err := filepath.Rel(c.dir, path)
			if strings.HasPrefix(relpath, ".cloudfoundry/") {
				return nil
			}
			if err != nil {
				return err
			}
			if _, err := io.WriteString(h, relpath); err != nil {
				return err
			}
			if f, err := os.Open(path); err != nil {
				return err
			} else {
				if _, err := io.Copy(h, f); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
