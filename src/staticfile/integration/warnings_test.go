package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy has nginx/conf directory", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("app has nginx include conf file", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("nginx_conf"))
			app.Buildpacks = []string{"staticfile_buildpack"}
			PushAppAndConfirm(app)
		})
		It("warns user to set root", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("You have an nginx/conf directory, but have not set *root*"))
			Expect(app.GetBody("/")).To(ContainSubstring("Test warnings"))
		})
	})

	Context("app as nginx.conf file", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("deprecated_nginx_conf"))
			PushAppAndConfirm(app)
		})
		It("warns user not to override nginx.conf", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("overriding nginx.conf is deprecated and highly discouraged, as it breaks the functionality of the Staticfile and Staticfile.auth configuration directives. Please use the NGINX buildpack available at: https://github.com/cloudfoundry/nginx-buildpack"))

			_, headers, err := app.Get("/", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(headers).To(HaveKeyWithValue("Custom-Nginx-Conf", []string{"true"}))
		})
	})
})
