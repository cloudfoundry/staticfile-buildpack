package packager_test

import (
	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("Packager", func() {
var (
	buildpackDir string
)
	BeforeEach(func() {
		buildpackDir = "./fixtures/good"

		httpmock.Reset()
	})

	Describe("Summary", func() {
		It("Renders tables of dependencies", func() {
			s, e := packager.Summary(buildpackDir)
			Expect(e).NotTo(HaveOccurred())
			Expect(s, e).To(Equal(`
Packaged binaries:

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
				Expect(packager.Summary(buildpackDir)).To(Equal(`
Packaged binaries:

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
