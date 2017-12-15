package libbuildpack_test

import (
	"fmt"

	bp "github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("versions", func() {

	Describe("FindMatchingVersion", func() {
		var versions []string

		BeforeEach(func() {
			versions = []string{"1.2.3", "1.2.4", "1.2.2", "1.3.3", "1.3.4", "1.3.2", "2.0.0"}
		})

		It("returns the greatest version", func() {
			ver, err := bp.FindMatchingVersion("x", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("2.0.0"))
		})

		It("returns the greatest version in a minor line", func() {
			ver, err := bp.FindMatchingVersion("1.x", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("1.3.4"))
		})

		It("returns the greatest version in a patch line", func() {
			ver, err := bp.FindMatchingVersion("1.2.x", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("1.2.4"))
		})

		It("returns the greatest version less than the above", func() {
			ver, err := bp.FindMatchingVersion(">=1.2.0, <1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("1.2.3"))
		})

		It("returns the greatest version less than the above (without comma)", func() {
			ver, err := bp.FindMatchingVersion(">=1.2.0 <1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("1.2.3"))
		})

		It("returns the greatest version less or equal than the above (without comma)", func() {
			ver, err := bp.FindMatchingVersion(">1.2.0 <=1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(ver).To(Equal("1.2.4"))
		})

		It("returns an error if nothing matches", func() {
			_, err := bp.FindMatchingVersion("1.4.x", versions)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(fmt.Sprintf("no match found for 1.4.x in %v", versions)))
		})
	})
	Describe("FindMatchingVersions", func() {
		var versions []string

		BeforeEach(func() {
			versions = []string{"1.2.3", "1.2.4", "1.2.2", "1.3.3", "1.3.4", "1.3.2", "2.0.0"}
		})

		It("Returns every version for x, sorted", func() {
			vers, err := bp.FindMatchingVersions("x", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.2", "1.2.3", "1.2.4", "1.3.2", "1.3.3", "1.3.4", "2.0.0"}))
		})

		It("returns all versions in a minor line, sorted", func() {
			vers, err := bp.FindMatchingVersions("1.x", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.2", "1.2.3", "1.2.4", "1.3.2", "1.3.3", "1.3.4"}))
		})

		It("returns all versions in a patch line", func() {
			vers, err := bp.FindMatchingVersions("1.2.x", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.2", "1.2.3", "1.2.4"}))
		})

		It("returns all versions less than the above", func() {
			vers, err := bp.FindMatchingVersions(">=1.2.0, <1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.2", "1.2.3"}))
		})

		It("returns all versions less than the above (without comma)", func() {
			vers, err := bp.FindMatchingVersions(">=1.2.0 <1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.2", "1.2.3"}))
		})

		It("returns all versions less or equal than the above (without comma)", func() {
			vers, err := bp.FindMatchingVersions(">1.2.2 <=1.2.4", versions)
			Expect(err).To(BeNil())
			Expect(vers).To(Equal([]string{"1.2.3", "1.2.4"}))
		})

		It("returns an error if nothing matches", func() {
			_, err := bp.FindMatchingVersions("1.4.x", versions)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(fmt.Sprintf("no match found for 1.4.x in %v", versions)))
		})
	})
})
