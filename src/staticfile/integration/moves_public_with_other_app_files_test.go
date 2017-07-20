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
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "recursive_public"))
		PushAppAndConfirm(app)
	})

	It("should have a copy of the original public dir in the new public dir", func() {
		Expect(app.Files("app")).To(ContainElement("app/public/public/file_in_public"))
	})
})
