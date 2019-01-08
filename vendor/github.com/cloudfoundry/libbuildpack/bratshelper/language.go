package bratshelper

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
)

var cachedLanguage string

func bpLanguage() string {
	if cachedLanguage == "" {
		file, err := ioutil.ReadFile(filepath.Join(bpDir(), "manifest.yml"))
		Expect(err).ToNot(HaveOccurred())
		obj := make(map[string]interface{})
		Expect(yaml.Unmarshal(file, &obj)).To(Succeed())
		var ok bool
		cachedLanguage, ok = obj["language"].(string)
		Expect(ok).To(BeTrue())
	}
	return cachedLanguage
}

func GenBpName(name string) string {
	return "brats_" + bpLanguage() + "_" + name + "_" + cutlass.RandStringRunes(6)
}

var cachedBpDir string

func bpDir() string {
	if cachedBpDir == "" {
		var err error
		cachedBpDir, err = cutlass.FindRoot()
		Expect(err).ToNot(HaveOccurred())
	}
	return cachedBpDir
}
