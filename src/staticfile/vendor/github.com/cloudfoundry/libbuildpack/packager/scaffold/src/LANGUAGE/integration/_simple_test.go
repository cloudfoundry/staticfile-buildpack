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

	// TODO This test is pending because it currently fails. It is just an example
	PIt("app deploys", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "simple_test"))
		app.Buildpacks = []string{"{{LANGUAGE}}_buildpack"}
		PushAppAndConfirm(app)
		Expect(app.GetBody("/")).To(ContainSubstring("Something on your website"))
	})
})
