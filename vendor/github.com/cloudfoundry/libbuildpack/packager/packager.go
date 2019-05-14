package packager

//go:generate go-bindata -pkg $GOPACKAGE -prefix scaffold scaffold/...

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
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

var CacheDir = filepath.Join(os.Getenv("HOME"), ".buildpack-packager", "cache")
var Stdout, Stderr io.Writer = os.Stdout, os.Stderr

func CompileExtensionPackage(bpDir, version string, cached bool, stack string) (string, error) {
	bpDir, err := filepath.Abs(bpDir)
	if err != nil {
		return "", fmt.Errorf("Failed to get the absolute path of %s: %v", bpDir, err)
	}
	dir, err := CopyDirectory(bpDir)
	if err != nil {
		return "", fmt.Errorf("Failed to copy %s: %v", bpDir, err)
	}

	err = ioutil.WriteFile(filepath.Join(dir, "VERSION"), []byte(version), 0644)
	if err != nil {
		return "", fmt.Errorf("Failed to write VERSION file: %v", err)
	}

	isCached := "--uncached"
	if cached {
		isCached = "--cached"
	}
	stackArg := "--stack=" + stack
	if stack == "any" {
		stackArg = "--any-stack"
	}
	cmd := exec.Command("bundle", "exec", "buildpack-packager", isCached, stackArg)
	cmd.Stdout = Stdout
	cmd.Stderr = Stderr
	cmd.Env = append(os.Environ(), "BUNDLE_GEMFILE=cf.Gemfile")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to run %s %s: %v", cmd.Path, strings.Join(cmd.Args, " "), err)
	}

	var manifest struct {
		Language string `yaml:"language"`
	}

	if err := libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest); err != nil {
		return "", fmt.Errorf("Failed to load manifest.yml: %v", err)
	}

	stackName := fmt.Sprintf("-%s", stack)
	if stackName == "any" {
		stackName = ""
	}
	zipFile := fmt.Sprintf("%s_buildpack%s-v%s.zip", manifest.Language, stackName, version)
	if cached {
		zipFile = fmt.Sprintf("%s_buildpack-cached%s-v%s.zip", manifest.Language, stackName, version)
	}
	if err := libbuildpack.CopyFile(filepath.Join(dir, zipFile), filepath.Join(bpDir, zipFile)); err != nil {
		return "", fmt.Errorf("Failed to copy %s from %s to %s: %v", zipFile, dir, bpDir, err)
	}

	return filepath.Join(dir, zipFile), nil
}

func validateStack(stack, bpDir string) error {
	manifest, err := readManifest(bpDir)
	if err != nil {
		return err
	}

	if manifest.Stack != "" {
		return fmt.Errorf("Cannot package from already packaged buildpack manifest")
	}

	if stack == "" {
		return nil
	}

	if len(manifest.Dependencies) > 0 && !manifest.hasStack(stack) {
		return fmt.Errorf("Stack `%s` not found in manifest", stack)
	}

	for _, d := range manifest.Defaults {
		if _, err := libbuildpack.FindMatchingVersion(d.Version, manifest.versionsOfDependencyWithStack(d.Name, stack)); err != nil {
			return fmt.Errorf("No matching default dependency `%s` for stack `%s`", d.Name, stack)
		}
	}

	return nil
}

func updateDependencyMap(dependencyMap interface{}, file File) error {
	dep, ok := dependencyMap.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("Could not cast deps[idx] to map[interface{}]interface{}")
	}
	dep["file"] = file.Name
	return nil
}

func downloadDependency(dependency Dependency, cacheDir string) (File, error) {
	file := filepath.Join("dependencies", fmt.Sprintf("%x", md5.Sum([]byte(dependency.URI))), filepath.Base(dependency.URI))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(cacheDir, file)); err != nil {
		if err := DownloadFromURI(dependency.URI, filepath.Join(cacheDir, file)); err != nil {
			return File{}, err
		}
	}

	if err := checkSha256(filepath.Join(cacheDir, file), dependency.SHA256); err != nil {
		return File{}, err
	}

	return File{file, filepath.Join(cacheDir, file)}, nil
}

func Package(bpDir, cacheDir, version, stack string, cached bool) (string, error) {
	bpDir, err := filepath.Abs(bpDir)
	if err != nil {
		return "", err
	}
	err = validateStack(stack, bpDir)
	if err != nil {
		return "", err
	}
	dir, err := CopyDirectory(bpDir)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	err = ioutil.WriteFile(filepath.Join(dir, "VERSION"), []byte(version), 0644)
	if err != nil {
		return "", err
	}

	manifest, err := readManifest(dir)
	if err != nil {
		return "", err
	}

	if manifest.PrePackage != "" {
		cmd := exec.Command(manifest.PrePackage)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintln(Stdout, string(out))
			return "", err
		}
	}

	files := []File{}
	for _, name := range manifest.IncludeFiles {
		files = append(files, File{name, filepath.Join(dir, name)})
	}

	var m map[string]interface{}
	if err := libbuildpack.NewYAML().Load(filepath.Join(dir, "manifest.yml"), &m); err != nil {
		return "", err
	}

	if stack != "" {
		m["stack"] = stack
	}

	deps, ok := m["dependencies"].([]interface{})
	if !ok {
		return "", fmt.Errorf("Could not cast dependencies to []interface{}")
	}
	dependenciesForStack := []interface{}{}
	for idx, d := range manifest.Dependencies {
		for _, s := range d.Stacks {
			if stack == "" || s == stack {
				dependencyMap := deps[idx]
				if cached {
					if file, err := downloadDependency(d, cacheDir); err != nil {
						return "", err
					} else {
						updateDependencyMap(dependencyMap, file)
						files = append(files, file)
					}
				}
				if stack != "" {
					delete(dependencyMap.(map[interface{}]interface{}), "cf_stacks")
				}
				dependenciesForStack = append(dependenciesForStack, dependencyMap)
				break
			}
		}
	}
	m["dependencies"] = dependenciesForStack

	if err := libbuildpack.NewYAML().Write(filepath.Join(dir, "manifest.yml"), m); err != nil {
		return "", err
	}

	stackPart := ""
	if stack != "" {
		stackPart = "-" + stack
	}

	cachedPart := ""
	if cached {
		cachedPart = "-cached"
	}

	fileName := fmt.Sprintf("%s_buildpack%s%s-v%s.zip", manifest.Language, cachedPart, stackPart, version)
	zipFile := filepath.Join(bpDir, fileName)

	if err := ZipFiles(zipFile, files); err != nil {
		return "", err
	}

	return zipFile, err
}

func DownloadFromURI(uri, fileName string) error {
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
			returnErr := fmt.Errorf("failed to open included_file: %s, %v", file.Path, err)
			err = os.Remove(filename)
			if err != nil {
				returnErr = fmt.Errorf("%s. Failed to remove broken buildpack file: %s", returnErr.Error(), filename)
			}
			return returnErr
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
		if !info.IsDir() {
			if _, err = io.Copy(writer, zipfile); err != nil {
				return err
			}
		}
	}
	return nil
}

func CopyDirectory(srcDir string) (string, error) {
	destDir, err := ioutil.TempDir("", "buildpack-packager")
	if err != nil {
		return "", err
	}
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if path == ".git" || path == "tests" {
			return filepath.SkipDir
		}

		dest := filepath.Join(destDir, path)
		if m := info.Mode(); m&os.ModeSymlink != 0 {
			srcPath := filepath.Join(srcDir, path)
			target, err := os.Readlink(srcPath)
			if err != nil {
				return fmt.Errorf("Error while reading symlink '%s': %v", srcPath, err)
			}
			if err := os.Symlink(target, dest); err != nil {
				return fmt.Errorf("Error while creating '%s' as symlink to '%s': %v", dest, target, err)
			}
		} else if info.IsDir() {
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
