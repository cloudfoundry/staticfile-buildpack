package integration_test

import (
	"path/filepath"

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
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "pushstate"))
		PushAppAndConfirm(app)
	})

	It("with pushstate", func() {
		By("requesting the index file returns the index file", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("This is the index file"))
		})
		By("requesting a static file returns the static file", func() {
			Expect(app.GetBody("/static.html")).To(ContainSubstring("This is a static file"))
		})
		By("requesting a inexistent file returns the index file", func() {
			Expect(app.GetBody("/inexistent")).To(ContainSubstring("This is the index file"))
		})
	})
})
