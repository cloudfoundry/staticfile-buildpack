package packager

//go:generate go-bindata -pkg $GOPACKAGE -prefix scaffold scaffold/...

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// cfbindata.go is a collection of a few functions from bindata.go that we copied so that we could make our own changes to them.
// We don't want to make these changes within bindata.go because bindata.go is an auto-generated file
// These changes allow us to use go templating to setup the scaffold directory

// RestoreAsset restores an asset under the given directory
func OurRestoreAsset(dir, name string, funcMap template.FuncMap, shas map[string]string, force bool) error {
	data, err := Asset(name)

	if err != nil {
		return err
	}

	info, err := AssetInfo(name)
	if err != nil {
		return err
	}

	t, err := template.New("").Funcs(funcMap).Parse(string(data))
	if err != nil {
		return err
	}
	var b bytes.Buffer
	f := bufio.NewWriter(&b)
	if err := t.Execute(f, nil); err != nil {
		return err
	}
	if err := f.Flush(); err != nil {
		return err
	}

	if strings.HasPrefix(name, "src/LANGUAGE/") {
		langName, ok := funcMap["LANGUAGE"].(func() string)
		if !ok {
			return fmt.Errorf("Could not find language from funcmap")
		}
		name = strings.Replace(name, "src/LANGUAGE/", fmt.Sprintf("src/%s/", langName()), 1)
	}
	if basename := filepath.Base(name); strings.HasPrefix(basename, "_") {
		name = filepath.Join(filepath.Dir(name), string([]rune(basename)[1:]))
	}

	oldSha256 := ""
	if oldContents, err := ioutil.ReadFile(_filePath(dir, name)); err == nil {
		oldSha256 = checksumHex(oldContents)
	}

	if name == "bin/supply" {
		fmt.Println(name, oldSha256, shas[name])
	}
	if !force && shas[name] != "" && shas[name] != oldSha256 {
		fmt.Fprintf(Stdout, "***Ignoring %s because it has been modified***\n", name)
		return nil
	}
	shas[name] = checksumHex(b.Bytes())

	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(_filePath(dir, name), b.Bytes(), info.Mode()); err != nil {
		return err
	}

	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}

	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func OurRestoreAssets(dir, name string, funcMap template.FuncMap, shas map[string]string, force bool) error {
	//TODO: is passing in the shas map the best way to go about this?
	children, err := AssetDir(name)
	// File
	if err != nil {
		err = OurRestoreAsset(dir, name, funcMap, shas, force)
		if err != nil {
			return err
		}
	}

	// Dir
	for _, child := range children {
		err = OurRestoreAssets(dir, filepath.Join(name, child), funcMap, shas, force)
		if err != nil {
			return err
		}
	}
	return nil
}

func checksumHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
