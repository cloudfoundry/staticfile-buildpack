package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a pushstate and reverse proxy app", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(Fixtures("pushstate_and_proxy_pass"))
		PushAppAndConfirm(app)
	})

	It("", func() {
		By("enables pushstate", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("This is the index file"))
			Expect(app.GetBody("/static.html")).To(ContainSubstring("This is a static file"))
			Expect(app.GetBody("/unknown")).To(ContainSubstring("This is the index file"))
		})

		By("proxies", func() {
			Expect(app.GetBody("/api")).To(ContainSubstring("This domain is established to be used for illustrative examples in documents"))
		})
	})
})
