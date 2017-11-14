package packager

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

var CacheDir = filepath.Join(os.Getenv("HOME"), ".buildpack-packager", "cache")

func Package(bpDir, cacheDir, version string, cached bool) (string, error) {
	bpDir, err := filepath.Abs(bpDir)
	if err != nil {
		return "", err
	}
	dir, err := copyDirectory(bpDir)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(filepath.Join(dir, "VERSION"), []byte(version), 0644)
	if err != nil {
		return "", err
	}

	manifest := Manifest{}
	data, err := ioutil.ReadFile(filepath.Join(dir, "manifest.yml"))
	if err != nil {
		return "", err
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return "", err
	}

	if manifest.PrePackage != "" {
		cmd := exec.Command(manifest.PrePackage)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return "", err
		}
	}

	files := []File{}
	for _, name := range manifest.IncludeFiles {
		files = append(files, File{name, filepath.Join(dir, name)})
	}

	if cached {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			log.Fatalf("error: %v", err)
		}
		for _, d := range manifest.Dependencies {
			dest := filepath.Join("dependencies", fmt.Sprintf("%x", md5.Sum([]byte(d.URI))), filepath.Base(d.URI))

			if _, err := os.Stat(filepath.Join(cacheDir, dest)); err != nil {
				if err := downloadFromURI(d.URI, filepath.Join(cacheDir, dest)); err != nil {
					return "", err
				}
			}

			if err := checkSha256(filepath.Join(cacheDir, dest), d.SHA256); err != nil {
				return "", err
			}

			files = append(files, File{dest, filepath.Join(cacheDir, dest)})
		}
	}

	zipFile := fmt.Sprintf("%s_buildpack-v%s.zip", manifest.Language, version)
	if cached {
		zipFile = fmt.Sprintf("%s_buildpack-cached-v%s.zip", manifest.Language, version)
	}
	zipFile = filepath.Join(bpDir, zipFile)

	ZipFiles(zipFile, files)

	return zipFile, err
}

func downloadFromURI(uri, fileName string) error {
	err := os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	output, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer output.Close()

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	var source io.ReadCloser

	if u.Scheme == "file" {
		source, err = os.Open(u.Path)
		if err != nil {
			return err
		}
		defer source.Close()
	} else {
		response, err := http.Get(uri)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		source = response.Body

		if response.StatusCode < 200 || response.StatusCode > 299 {
			return fmt.Errorf("could not download: %d", response.StatusCode)
		}
	}

	_, err = io.Copy(output, source)

	return err
}

func checkSha256(filePath, expectedSha256 string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(content)

	actualSha256 := hex.EncodeToString(sum[:])

	if actualSha256 != expectedSha256 {
		return fmt.Errorf("dependency sha256 mismatch: expected sha256 %s, actual sha256 %s", expectedSha256, actualSha256)
	}
	return nil
}

func ZipFiles(filename string, files []File) error {
	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {

		zipfile, err := os.Open(file.Path)
		if err != nil {
			return err
		}
		defer zipfile.Close()

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate
		header.Name = file.Name

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, zipfile)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyDirectory(srcDir string) (string, error) {
	destDir, err := ioutil.TempDir("", "buildpack-packager")
	if err != nil {
		return "", err
	}
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.Rel(srcDir, path)

		if filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}

		dest := filepath.Join(destDir, path)
		if info.IsDir() {
			err = os.MkdirAll(dest, info.Mode())
			if err != nil {
				return err
			}
		} else {
			src, err := os.Open(filepath.Join(srcDir, path))
			if err != nil {
				return err
			}
			defer src.Close()

			err = os.MkdirAll(filepath.Dir(dest), 0755)
			if err != nil {
				return err
			}

			fh, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
			if err != nil {
				return err
			}

			_, err = io.Copy(fh, src)
			fh.Close()
			return err
		}
		return nil
	})
	return destDir, err
}
