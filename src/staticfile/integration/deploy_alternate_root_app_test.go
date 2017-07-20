package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy an app with contents in an alternate root", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	It("default path", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "alternate_root"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("This index file comes from an alternate root <code>dist/</code>."))
	})

	It("not default path", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "alternate_root_not_default"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("This index file comes from an alternate root dist/public/index.html"))
	})
})
