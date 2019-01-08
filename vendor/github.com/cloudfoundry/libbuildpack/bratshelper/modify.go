package bratshelper

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

func ModifyBuildpack(path string, cb func(path string, r io.Reader) (io.Reader, error)) (string, error) {
	if file, err := modifyZipfile(path, cb); err != nil {
		return file, err
	} else {
		if err := os.Rename(file, file+".zip"); err != nil {
			return "", err
		}
		return file + ".zip", nil
	}
}

func ModifyBuildpackManifest(path string, cb func(*Manifest)) (string, error) {
	return ModifyBuildpack(path, func(path string, r io.Reader) (io.Reader, error) {
		if path == "manifest.yml" {
			if r, err := changeManifest(r, cb); err != nil {
				return nil, err
			} else {
				return r, nil
			}
		}
		return r, nil
	})
}

type Manifest struct {
	Stack           string `yaml:"stack"`
	Language        string `yaml:"language"`
	DefaultVersions []*struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"default_versions"`
	PrePackage                 string `yaml:"pre_package"`
	DependencyDeprecationDates []*struct {
		VersionLine string `yaml:"version_line"`
		Name        string `yaml:"name"`
		Date        string `yaml:"date"`
		Link        string `yaml:"link"`
		Match       string `yaml:"match,omitempty"` // CompileExtensions
	} `yaml:"dependency_deprecation_dates"`
	Dependencies []*struct {
		Name     string   `yaml:"name"`
		Version  string   `yaml:"version"`
		URI      string   `yaml:"uri"`
		Md5      string   `yaml:"md5,omitempty"`
		Sha256   string   `yaml:"sha256,omitempty"`
		CfStacks []string `yaml:"cf_stacks"`
		Modules  []string `yaml:"modules,omitempty"` // CompileExtensions PHP
	} `yaml:"dependencies"`
	IncludeFiles       []string      `yaml:"include_files,omitempty"`
	ExcludeFiles       []string      `yaml:"exclude_files,omitempty"`         // CompileExtensions
	UrlToDependencyMap []interface{} `yaml:"url_to_dependency_map,omitempty"` // CompileExtensions
}

func modifyZipfile(path string, cb func(path string, r io.Reader) (io.Reader, error)) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	newfile, err := ioutil.TempFile("", "buildpack.zip")
	if err != nil {
		return "", err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		if f.FileInfo().IsDir() {
			// Nothing
		} else {
			header, err := zip.FileInfoHeader(f.FileInfo())
			if err != nil {
				return "", err
			}
			header.Method = zip.Deflate
			header.Name = f.Name

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return "", err
			}

			rc, err := cb(f.Name, rc)
			if err != nil {
				return "", err
			}

			_, err = io.Copy(writer, rc)
			if err != nil {
				return "", err
			}
		}
		rc.Close()
	}

	return newfile.Name(), nil
}

func changeManifest(r io.Reader, cb func(*Manifest)) (io.Reader, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	obj := Manifest{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	cb(&obj)

	if data, err := yaml.Marshal(&obj); err != nil {
		return nil, err
	} else {
		return bytes.NewReader(data), nil
	}
}
