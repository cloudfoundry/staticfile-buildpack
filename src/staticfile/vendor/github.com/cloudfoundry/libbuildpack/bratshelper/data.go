package bratshelper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
)

type BpData struct {
	BpVersion    string
	BpLanguage   string
	BpDir        string
	Cached       string
	CachedFile   string
	Uncached     string
	UncachedFile string
}

var Data BpData

func InitBpData(stack string, stackAssociationSupported bool) *BpData {
	cutlass.SeedRandom()

	Data.BpVersion = cutlass.RandStringRunes(6)

	var err error
	Data.BpDir, err = cutlass.FindRoot()
	Expect(err).NotTo(HaveOccurred())

	file, err := ioutil.ReadFile(filepath.Join(Data.BpDir, "manifest.yml"))
	Expect(err).ToNot(HaveOccurred())
	obj := make(map[string]interface{})
	Expect(yaml.Unmarshal(file, &obj)).To(Succeed())

	var ok bool
	Data.BpLanguage, ok = obj["language"].(string)
	Expect(ok).To(BeTrue())

	Data.Cached = "brats_" + Data.BpLanguage + "_cached_" + Data.BpVersion
	Data.Uncached = "brats_" + Data.BpLanguage + "_uncached_" + Data.BpVersion

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer GinkgoRecover()
		fmt.Fprintln(os.Stderr, "Start build cached buildpack")
		cachedBuildpack, err := cutlass.PackageUniquelyVersionedBuildpackExtra(Data.Cached, Data.BpVersion, stack, true, stackAssociationSupported)
		Expect(err).NotTo(HaveOccurred())
		Data.CachedFile = cachedBuildpack.File
		fmt.Fprintln(os.Stderr, "Finish cached buildpack")
	}()
	go func() {
		defer wg.Done()
		defer GinkgoRecover()
		fmt.Fprintln(os.Stderr, "Start build uncached buildpack")
		uncachedBuildpack, err := cutlass.PackageUniquelyVersionedBuildpackExtra(Data.Uncached, Data.BpVersion, stack, false, stackAssociationSupported)
		Expect(err).NotTo(HaveOccurred())
		Data.UncachedFile = uncachedBuildpack.File
		fmt.Fprintln(os.Stderr, "Finish uncached buildpack")
	}()
	wg.Wait()

	Data.Cached = Data.Cached + "_buildpack"
	Data.Uncached = Data.Uncached + "_buildpack"

	return &Data
}

func (d *BpData) Marshal() []byte {
	data, err := json.Marshal(Data)
	Expect(err).NotTo(HaveOccurred())
	return data
}

func (d *BpData) Unmarshal(data []byte) {
	err := json.Unmarshal(data, d)
	Expect(err).NotTo(HaveOccurred())
}
