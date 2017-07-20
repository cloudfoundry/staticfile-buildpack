package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy an app using hsts", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_hsts"))
	})

	It("provides the Strict-Transport-Security header", func() {
		PushAppAndConfirm(app)

		_, headers, err := app.Get("/", map[string]string{})
		Expect(err).To(BeNil())
		Expect(headers).To(HaveKey("Strict-Transport-Security"))
	})
})
