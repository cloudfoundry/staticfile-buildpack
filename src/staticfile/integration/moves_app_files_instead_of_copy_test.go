package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pushing a static app with dummy file in root", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "public_unspecified"))
		PushAppAndConfirm(app)
	})

	It("should only have dummy file in public", func() {
		files, err := app.Files("app")
		Expect(err).To(BeNil())

		Expect(files).To(ContainElement("app/public/dummy_file"))
		Expect(files).ToNot(ContainElement("app/dummy_file"))
	})
})
