package packager_test

import (
	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
	"sort"
)

var _ = Describe("Packager", func() {
	BeforeEach(func() {
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
