package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Simple Integration Test", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	// TODO explain when to make these not pending
	PContext("app contains the example file", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple_test"))
			app.Buildpacks = []string{"{{LANGUAGE}}_buildpack"}
			PushAppAndConfirm(app)
		})
		It("has some_file.txt in the pushed app", func() {
			paths, err := app.Files("some_file.txt")
			Expect(err).To(BeNil())
			Expect(paths).ToNot(BeEmpty())
		})
	})
})
