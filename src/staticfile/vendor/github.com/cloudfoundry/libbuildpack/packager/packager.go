package packager

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

type Manifest struct {
	Language     string   `yaml:"language"`
	IncludeFiles []string `yaml:"include_files"`
	PrePackage   string   `yaml:"pre_package"`
	Dependencies []struct {
		URI string `yaml:"uri"`
		MD5 string `yaml:"md5"`
	} `yaml:"dependencies"`
}

type File struct {
	Name, Path string
}

var CacheDir string = filepath.Join(os.Getenv("HOME"), ".buildpack-packager", "cache")

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

			if _, err := os.Stat(dest); err != nil {
				if os.IsNotExist(err) {
					err = downloadFromUrl(d.URI, filepath.Join(cacheDir, dest))
				}
				if err != nil {
					return "", err
				}
			}

			if err := checkMD5(filepath.Join(cacheDir, dest), d.MD5); err != nil {
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

func downloadFromUrl(url, fileName string) error {
	err := os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	output, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("could not download: %d", response.StatusCode)
	}

	if _, err := io.Copy(output, response.Body); err != nil {
		return err
	}
	return nil
}

func checkMD5(filePath, expectedMD5 string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	hashInBytes := hash.Sum(nil)[:16]
	actualMD5 := hex.EncodeToString(hashInBytes)

	if actualMD5 != expectedMD5 {
		return fmt.Errorf("dependency md5 mismatch: expected md5 %s, actual md5 %s", expectedMD5, actualMD5)
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
