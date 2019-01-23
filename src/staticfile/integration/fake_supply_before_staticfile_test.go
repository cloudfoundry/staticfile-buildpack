package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply buildpacks before the staticfile buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("the app is pushed once", func() {
		BeforeEach(func() {
			if ok, err := cutlass.ApiGreaterThan("2.65.1"); err != nil || !ok {
				Skip("API version does not have multi-buildpack support")
			}

			app = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_supply_staticfile_app"))
			app.Buildpacks = []string{
				"https://github.com/cloudfoundry/dotnet-core-buildpack#master",
				"staticfile_buildpack",
			}
			app.Disk = "1G"
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("Supplying Dotnet Core"))
			Expect(app.GetBody("/")).To(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets."))
		})
	})
})
