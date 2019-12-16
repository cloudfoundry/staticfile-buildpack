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

	hasSubDependencies := false
	for _, d := range manifest.Dependencies {
		if len(d.SubDependencies) > 0 {
			hasSubDependencies = true
			break
		}
	}

	var out string
	if len(manifest.Dependencies) > 0 {
		out = "\nPackaged binaries:\n\n"
		sort.Sort(manifest.Dependencies)
		if hasSubDependencies {
			out += "| name | version | cf_stacks | modules |\n|-|-|-|-|\n"
		} else {
			out += "| name | version | cf_stacks |\n|-|-|-|\n"
		}
	}

	for _, d := range manifest.Dependencies {
		sort.Strings(d.Stacks)
		if hasSubDependencies {
			moduleNames := []string{}
			for _, dep := range d.SubDependencies {
				moduleNames = append(moduleNames, dep.Name)
			}
			sort.Strings(moduleNames)

			out += fmt.Sprintf("| %s | %s | %s | %s |\n", d.Name, d.Version, strings.Join(d.Stacks, ", "), strings.Join(moduleNames, ", "))
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
