package integration_test

import (
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

	Context("with a public folder", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("include_headers_public"))
			PushAppAndConfirm(app)
		})

		It("adds headers", func() {
			body, headers, err := app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("Test add headers"))
			Expect(headers).To(HaveKey("X-Superspecialpublic"))
		})
	})

	Context("with a dist folder", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("include_headers_dist"))
			PushAppAndConfirm(app)
		})

		It("adds headers", func() {
			body, headers, err := app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("Test add headers"))
			Expect(headers).To(HaveKey("X-Superspecialdist"))
		})
	})

	Context("with a root folder", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("include_headers_root_pwd"))
			PushAppAndConfirm(app)
		})

		It("adds headers", func() {
			body, headers, err := app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("Test add headers"))
			Expect(headers).To(HaveKey("X-Superspecialroot"))
		})
	})
})
