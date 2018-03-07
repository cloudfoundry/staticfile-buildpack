package finalize_test

//go:generate mockgen -source=finalize.go --destination=mocks_test.go --package=finalize_test
import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Finalize", func() {
	It("succeeds", func() {
		Expect(false).To(Equal(false))
	})
	// TODO: Add tests here to check configure dependency functions work
})
