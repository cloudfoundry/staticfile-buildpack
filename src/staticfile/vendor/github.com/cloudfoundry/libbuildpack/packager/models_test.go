package packager_test

import (
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("Packager", func() {
	var (
		buildpackDir string
		version      string
		cacheDir     string
	)

	BeforeEach(func() {
		var err error
		buildpackDir = "./fixtures/good"
		cacheDir, err = ioutil.TempDir("", "packager-cachedir")
		Expect(err).To(BeNil())
		version = fmt.Sprintf("1.23.45.%s", time.Now().Format("20060102150405"))

		httpmock.Reset()
	})

	Describe("Sort Dependencies", func() {
		It("....", func() {
			deps := packager.Dependencies{
				{Name: "ruby", Version: "1.2.3"},
				{Name: "ruby", Version: "3.2.1"},
				{Name: "zesty", Version: "2.1.3"},
				{Name: "ruby", Version: "1.11.3"},
				{Name: "jruby", Version: "2.1.3"},
			}
			sort.Sort(deps)
			Expect(deps).To(Equal(packager.Dependencies{
				{Name: "jruby", Version: "2.1.3"},
				{Name: "ruby", Version: "1.2.3"},
				{Name: "ruby", Version: "1.11.3"},
				{Name: "ruby", Version: "3.2.1"},
				{Name: "zesty", Version: "2.1.3"},
			}))
		})
	})
})
