package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy includes headers", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "include_headers"))
		PushAppAndConfirm(app)
	})

	It("adds headers", func() {
		body, headers, err := app.Get("/", map[string]string{})
		Expect(err).To(BeNil())
		Expect(body).To(ContainSubstring("Test add headers"))
		Expect(headers).To(HaveKey("X-Superspecial"))
	})
})
