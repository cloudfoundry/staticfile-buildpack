package libbuildpack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// ExtractZip extracts zipfile to destDir
func ExtractZip(zipfile, destDir string) error {
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(destDir, f.Name)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(path, f.Mode())
		} else {
			err = writeToFile(rc, path, f.Mode())
		}

		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// ExtractTarGz extracts tar.gz to destDir
func ExtractTarGz(tarfile, destDir string) error {
	file, err := os.Open(tarfile)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(gz, destDir)
}

// CopyFile copies source file to destFile, creating all intermediate directories in destFile
func CopyFile(source, destFile string) error {
	fh, err := os.Open(source)
	if err != nil {
		Log.Error("Could not be found")
		return err
	}

	fileInfo, err := fh.Stat()
	if err != nil {
		Log.Error("Could not stat")
		return err
	}

	defer fh.Close()

	err = os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		Log.Error("Could not create %s", filepath.Dir(destFile))
		return err
	}
	return writeToFile(fh, destFile, fileInfo.Mode())
}

func FileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func extractTar(src io.Reader, destDir string) error {
	tr := tar.NewReader(src)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		path := filepath.Join(destDir, hdr.Name)

		if hdr.FileInfo().IsDir() {
			err = os.MkdirAll(path, hdr.FileInfo().Mode())
		} else {
			err = writeToFile(tr, path, hdr.FileInfo().Mode())
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func filterURI(rawURL string) (string, error) {
	unsafeURL, err := url.Parse(rawURL)

	if err != nil {
		return "", err
	}

	var safeURL string

	if unsafeURL.User == nil {
		safeURL = rawURL
		return safeURL, nil
	}

	redactedUserInfo := url.UserPassword("-redacted-", "-redacted-")

	unsafeURL.User = redactedUserInfo
	safeURL = unsafeURL.String()

	return safeURL, nil
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
		Log.Error("DEPENDENCY_MD5_MISMATCH: expected md5: %s, actual md5: %s", expectedMD5, actualMD5)
		return fmt.Errorf("expected md5: %s actual md5: %s", expectedMD5, actualMD5)
	}
	return nil
}

func downloadFile(url, destFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		Log.Error("Could not download: %d", resp.StatusCode)
		return errors.New("file download failed")
	}

	err = os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		Log.Error("Could not create %s", filepath.Dir(destFile))
		return err
	}
	return writeToFile(resp.Body, destFile, 0666)
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		Log.Error("Could not create %s", destFile)
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		Log.Error("Could not write to %s", destFile)
		return err
	}

	return nil
}
