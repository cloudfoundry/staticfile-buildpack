package packager_test

import (
	"fmt"
	"io/ioutil"
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

	Describe("Summary", func() {
		It("Renders tables of dependencies", func() {
			Expect(packager.Summary(buildpackDir)).To(Equal(`Packaged binaries:

| name | version | cf_stacks |
|-|-|-|
| ruby | 1.2.3 | cflinuxfs2 |
| ruby | 1.2.3 | cflinuxfs3 |

Default binary versions:

| name | version |
|-|-|
| ruby | 1.2.3 |
`))
		})

		Context("modules exist", func() {
			BeforeEach(func() {
				buildpackDir = "./fixtures/modules"
			})
			It("Renders tables of dependencies (including modules)", func() {
				Expect(packager.Summary(buildpackDir)).To(Equal(`Packaged binaries:

| name | version | cf_stacks | modules |
|-|-|-|-|
| nginx | 1.7.3 | cflinuxfs2 |  |
| php | 1.6.1 | cflinuxfs2 | gearman, geoip, zlib |
`))
			})
		})

		Context("no dependencies", func() {
			BeforeEach(func() {
				buildpackDir = "./fixtures/no_dependencies"
			})
			It("Produces no output", func() {
				Expect(packager.Summary(buildpackDir)).To(Equal(""))
			})
		})
	})
})
