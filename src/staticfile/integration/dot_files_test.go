package integration_test

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testDotFiles(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when deploying an app it dotfile enabled", func() {
			it("builds and runs the app and the dotfiles are accessible", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "dotfiles", "with_dotfiles"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainSubstring("Enabling hosting of dotfiles"), logs.String())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello from a hidden file")).WithEndpoint(".hidden.html"))
			})

			context("when root is /public", func() {
				it("builds and runs the app and the dotfiles are accessible", func() {
					deployment, logs, err := platform.Deploy.
						Execute(name, filepath.Join(fixtures, "dotfiles", "with_dotfiles_public_root"))
					Expect(err).NotTo(HaveOccurred())

					Expect(logs).To(ContainSubstring("Enabling hosting of dotfiles"), logs.String())

					Eventually(deployment).Should(Serve(ContainSubstring("Hello from a hidden file")).WithEndpoint(".hidden.html"))
				})
			})
		})

		context("when deploying an app it dotfile disabled", func() {
			it("builds and runs the app and the dotfiles are not accessible", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "dotfiles", "without_dotfiles"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).ToNot(ContainSubstring("Enabling hosting of dotfiles"), logs.String())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = ".hidden.html"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("404 Not Found"), string(contents))
			})

			context("when root is /public", func() {
				it("builds and runs the app and the dotfiles are not accessible", func() {
					deployment, logs, err := platform.Deploy.
						Execute(name, filepath.Join(fixtures, "dotfiles", "without_dotfiles_public_root"))
					Expect(err).NotTo(HaveOccurred())

					Expect(logs).ToNot(ContainSubstring("Enabling hosting of dotfiles"), logs.String())

					uri, err := url.Parse(deployment.ExternalURL)
					Expect(err).NotTo(HaveOccurred())

					uri.Path = ".hidden.html"

					req, err := http.NewRequest("GET", uri.String(), nil)
					Expect(err).NotTo(HaveOccurred())

					var resp *http.Response
					Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
					defer resp.Body.Close()

					contents, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					Expect(contents).To(ContainSubstring("404 Not Found"), string(contents))
				})
			})
		})
	}
}
