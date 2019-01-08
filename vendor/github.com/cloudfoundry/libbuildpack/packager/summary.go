package packager

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func Summary(bpDir string) (string, error) {
	manifest := Manifest{}
	data, err := ioutil.ReadFile(filepath.Join(bpDir, "manifest.yml"))
	if err != nil {
		return "", err
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return "", err
	}

	hasModules := false
	for _, d := range manifest.Dependencies {
		if len(d.Modules) > 0 {
			hasModules = true
			break
		}
	}

	var out string
	if len(manifest.Dependencies) > 0 {
		out = "\nPackaged binaries:\n\n"
		sort.Sort(manifest.Dependencies)
		if hasModules {
			out += "| name | version | cf_stacks | modules |\n|-|-|-|-|\n"
		} else {
			out += "| name | version | cf_stacks |\n|-|-|-|\n"
		}
	}

	for _, d := range manifest.Dependencies {
		sort.Strings(d.Stacks)
		if hasModules {
			sort.Strings(d.Modules)
			out += fmt.Sprintf("| %s | %s | %s | %s |\n", d.Name, d.Version, strings.Join(d.Stacks, ", "), strings.Join(d.Modules, ", "))
		} else {
			out += fmt.Sprintf("| %s | %s | %s |\n", d.Name, d.Version, strings.Join(d.Stacks, ", "))
		}
	}

	if len(manifest.Defaults) > 0 {
		out += "\nDefault binary versions:\n\n"
		out += "| name | version |\n|-|-|\n"
		for _, d := range manifest.Defaults {
			out += fmt.Sprintf("| %s | %s |\n", d.Name, d.Version)
		}
	}

	return out, nil
}
