package integration_test

import (
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("nginx logs go to stdout and stderr", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "staticfile_app"))
		PushAppAndConfirm(app)
	})

	It("", func() {
		By("writes regular logs to stdout and does not write to actual log files", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets."))
			Eventually(app.Stdout.String, 5*time.Second).Should(MatchRegexp("OUT.*GET / HTTP/1.1"))
		})

		By("writes error logs to stderr and does not write to actual log files", func() {
			Expect(app.GetBody("/idontexist")).To(ContainSubstring("404 Not Found"))
			Eventually(app.Stdout.String, 5*time.Second).Should(MatchRegexp("ERR.*GET /idontexist HTTP/1.1"))
		})
	})
})
