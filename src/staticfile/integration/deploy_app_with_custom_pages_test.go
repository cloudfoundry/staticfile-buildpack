package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("a staticfile app with custom error pages", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(Fixtures("custom_error"))
		app.Buildpacks = []string{"staticfile_buildpack"}
	})

	It("runs", func() {
		PushAppAndConfirm(app)

		Expect(app.Stdout.String()).To(ContainSubstring("Enabling custom pages for status_codes"))
	})
})
