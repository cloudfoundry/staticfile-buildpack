package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("a staticfile app with no staticfile", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "without_staticfile"))
		app.Buildpack = "staticfile_buildpack"
	})

	It("runs", func() {
		PushAppAndConfirm(app)

		Expect(app.Stdout.String()).ToNot(ContainSubstring("grep: Staticfile: No such file or directory"))
	})
})
