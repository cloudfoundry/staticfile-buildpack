package snapshot

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

type DirSnapshot struct {
	dir             string
	initialPaths    []string
	initialChecksum string
	logger          Logger
	command         libbuildpack.Command
}

type Logger interface {
	Debug(format string, args ...interface{})
}

func Dir(dir string, logger Logger) *DirSnapshot {
	dirSnapshot := &DirSnapshot{
		dir:          dir,
		initialPaths: []string{},
		logger:       logger,
		command:      libbuildpack.Command{},
	}

	if os.Getenv("BP_DEBUG") != "" {
		if paths, checksum, err := dirSnapshot.calcChecksum(); err == nil {
			logger.Debug("Initial dir checksum %s", checksum)
			dirSnapshot.initialPaths = paths
			dirSnapshot.initialChecksum = checksum
		}

		dirSnapshot.command.Execute(dir, ioutil.Discard, ioutil.Discard, "touch", "/tmp/checkpoint")
	}

	return dirSnapshot
}

func (d *DirSnapshot) Diff() {
	if d.initialChecksum == "" {
		return
	}
	if paths, checksum, err := d.calcChecksum(); err == nil {
		if checksum == d.initialChecksum {
			d.logger.Debug("Dir checksum unchanged")
		} else {
			d.logger.Debug("New dir checksum is %s", checksum)
			d.logger.Debug("paths changed:")
			d.logDiffs(paths)
		}
	}
}

func (d *DirSnapshot) logDiffs(newPaths []string) {
	if filesChanged, err := d.command.Output(d.dir, "find", ".", "-newer", "/tmp/checkpoint", "-not", "-path", "./.cloudfoundry/*", "-not", "-path", "./.cloudfoundry"); err == nil && filesChanged != "" {
		changedFiles := strings.Split(filesChanged, "\n")
		for _, file := range changedFiles {
			if file != "." && file != "" {
				d.logger.Debug(file)
			}
		}
	}
	for _, path := range d.initialPaths {
		found := false
		for _, newPath := range newPaths {
			if newPath == path {
				found = true
				break
			}
		}
		if !found {
			d.logger.Debug(fmt.Sprintf("./%s", path))
		}
	}
}

func (d *DirSnapshot) calcChecksum() ([]string, string, error) {
	paths := []string{}
	h := md5.New()
	err := filepath.Walk(d.dir, func(path string, info os.FileInfo, err error) error {
		relpath, err := filepath.Rel(d.dir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relpath, ".cloudfoundry/") || relpath == ".cloudfoundry" {
			return nil
		}
		paths = append(paths, relpath)
		if _, err := io.WriteString(h, relpath); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			if f, err := os.Open(path); err != nil {
				return err
			} else {
				if _, err := io.Copy(h, f); err != nil {
					return err
				}
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			if dest, err := os.Readlink(path); err != nil {
				return err
			} else {
				reldestpath, err := filepath.Rel(d.dir, dest)
				if err != nil {
					return err
				}
				if _, err := io.WriteString(h, reldestpath); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return paths, "", err
	}
	return paths, fmt.Sprintf("%x", h.Sum(nil)), nil
}
