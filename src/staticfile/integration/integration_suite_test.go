package integration_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/cloudfoundry/libbuildpack/packager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var bpDir string
var buildpackVersion string

func init() {
	flag.StringVar(&buildpackVersion, "version", "", "version to use (builds if empty)")
	flag.BoolVar(&cutlass.Cached, "cached", true, "cached buildpack")
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "128M", "default disk for pushed apps")
	flag.Parse()
	fmt.Println("cutlass.Cached", cutlass.Cached)
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func PushAppAndConfirm(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 10*time.Second).Should(Equal([]string{"RUNNING"}))
	Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
}

func findRoot() string {
	file := "VERSION"
	for {
		files, err := filepath.Glob(file)
		Expect(err).To(BeNil())
		if len(files) == 1 {
			file, err = filepath.Abs(filepath.Dir(file))
			Expect(err).To(BeNil())
			return file
		}
		file = filepath.Join("..", file)
	}
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Run once
	bpDir = findRoot()

	if buildpackVersion == "" {
		data, err := ioutil.ReadFile(filepath.Join(bpDir, "VERSION"))
		Expect(err).NotTo(HaveOccurred())
		buildpackVersion = string(data)
		buildpackVersion = fmt.Sprintf("%s.%s", buildpackVersion, time.Now().Format("20060102150405"))

		file, err := packager.Package(bpDir, packager.CacheDir, buildpackVersion, cutlass.Cached)
		Expect(err).To(BeNil())

		var manifest struct {
			Language string `yaml:"language"`
		}
		Expect(libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &manifest)).To(Succeed())
		Expect(cutlass.UpdateBuildpack(manifest.Language, file)).To(Succeed())

		os.Remove(file)
	}

	return nil
}, func(_ []byte) {
	// Run on all nodes
	bpDir = findRoot()
	cutlass.DefaultStdoutStderr = GinkgoWriter
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all nodes
}, func() {
	// Run once
	cutlass.DeleteOrphanedRoutes()
})
