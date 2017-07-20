package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a staticfile app", func() {
	var app *cutlass.App
	var app_name string
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
		app_name = ""
	})
	JustBeforeEach(func() {
		Expect(app_name).ToNot(BeEmpty())
		app = cutlass.New(filepath.Join(bpDir, "fixtures", app_name))
		PushAppAndConfirm(app)
	})

	Context("ssi is toggled on", func() {
		BeforeEach(func() { app_name = "ssi_enabled" })

		It("", func() {
			body, err := app.GetBody("/")
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("I feel included!"))
			Expect(body).ToNot(ContainSubstring("<!--# include file=\"ssi_body.html\" -->"))
		})
	})

	Context("ssi is toggled off", func() {
		BeforeEach(func() { app_name = "ssi_disabled" })

		It("", func() {
			body, err := app.GetBody("/")
			Expect(err).To(BeNil())
			Expect(body).ToNot(ContainSubstring("I feel included!"))
			Expect(body).To(ContainSubstring("<!--# include file=\"ssi_body.html\" -->"))
		})
	})
})
