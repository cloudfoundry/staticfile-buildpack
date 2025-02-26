package integration_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testWarnings(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		context("when app has an nginx include conf file", func() {
			it("warns user to set root", func() {
				deployment, logs, err := platform.Deploy.
					WithBuildpacks(
						"staticfile_buildpack",
					).
					Execute(name, filepath.Join(fixtures, "warnings", "nginx_conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainSubstring("You have an nginx/conf directory, but have not set *root*"), logs.String())

				Eventually(deployment).Should(Serve(ContainSubstring("Test warnings")))
			})
		})

		context("when app has an nginx conf file", func() {
			it("warns user to set root", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "warnings", "deprecated_nginx_conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainSubstring("overriding nginx.conf is deprecated and highly discouraged, as it breaks the functionality of the Staticfile and Staticfile.auth configuration directives. Please use the NGINX buildpack available at: https://github.com/cloudfoundry/nginx-buildpack"))

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.Header["Custom-Nginx-Conf"]).To(Equal([]string{"true"}))
			})
		})
	}
}
