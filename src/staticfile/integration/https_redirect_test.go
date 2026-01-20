package integration_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testHttpsRedirect(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name string

			client *http.Client
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())

			client = &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
		})

		it.After(func() {
			if t.Failed() && name != "" {
				t.Logf("❌ FAILED TEST - App/Container: %s", name)
				t.Logf("   Platform: %s", settings.Platform)
			}
			if name != "" && (!settings.KeepFailedContainers || !t.Failed()) {
				Expect(platform.Delete.Execute(name)).To(Succeed())
			}
		})

		context("with HTTPS redirect set with an environment variable", func() {
			it("receives a 301 to HTTPS", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{
						"FORCE_HTTPS": "true",
					}).
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_env"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://"))))
			})

			it("injects X-Forwarded-Host into Location redirect", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{
						"FORCE_HTTPS": "true",
					}).
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_env"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("X-Forwarded-Host", "host.com")

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://host.com"))))
			})

			it("comma separated values in X-Forwarded headers", func() {
				deployment, _, err := platform.Deploy.
					WithEnv(map[string]string{
						"FORCE_HTTPS": "true",
					}).
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_env"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "path1/path2"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("X-Forwarded-Host", "host.com, something.else")
				req.Header.Add("X-Forwarded-Prefix", "/pre/fix1, /pre/fix2")

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://host.com/pre/fix1/path1/path2"))))
			})
		})

		context("with HTTPS redirect set with Staticfile setting", func() {
			it("receives a 301 to HTTPS", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_staticfile"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://"))))
			})

			it("injects X-Forwarded-Host into Location redirect", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_staticfile"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("X-Forwarded-Host", "host.com")

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://host.com"))))
			})

			it("comma separated values in X-Forwarded headers", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "https_redirect", "with_staticfile"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "path1/path2"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("X-Forwarded-Host", "host.com, something.else")
				req.Header.Add("X-Forwarded-Prefix", "/pre/fix1, /pre/fix2")

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(301))
				Expect(resp.Header).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://host.com/pre/fix1/path1/path2"))))
			})
		})
	}
}
