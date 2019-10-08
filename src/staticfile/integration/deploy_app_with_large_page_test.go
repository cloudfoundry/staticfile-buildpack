package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a staticfile app", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(Fixtures("large_page"))
		PushAppAndConfirm(app)
	})

	It("responds with the Vary: Accept-Encoding header", func() {
		_, headers, err := app.Get("/", map[string]string{})
		Expect(err).To(BeNil())
		Expect(headers).To(HaveKeyWithValue("Vary", []string{"Accept-Encoding"}))
	})
})
