package integration_test

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testIncludeHeaders(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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
			if t.Failed() && name != "" {
				t.Logf("❌ FAILED TEST - App/Container: %s", name)
				t.Logf("   Platform: %s", settings.Platform)
			}
			if name != "" && (!settings.KeepFailedContainers || !t.Failed()) {
				Expect(platform.Delete.Execute(name)).To(Succeed())
			}
		})

		context("with a public folder", func() {
			it("adds headers", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "include_headers", "public"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("Test add headers"), string(contents))
				Expect(resp.Header).To(HaveKey("X-Superspecialpublic"))
			})
		})

		context("with a dist folder", func() {
			it("adds headers", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "include_headers", "dist"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("Test add headers"), string(contents))
				Expect(resp.Header).To(HaveKey("X-Superspecialdist"))
			})
		})

		context("with a root folder", func() {
			it("adds headers", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "include_headers", "root_pwd"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("Test add headers"), string(contents))
				Expect(resp.Header).To(HaveKey("X-Superspecialroot"))
			})
		})
	}
}
