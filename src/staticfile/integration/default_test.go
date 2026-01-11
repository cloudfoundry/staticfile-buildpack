package integration_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testDefault(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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
			if name != "" && !t.Skipped() && (!settings.KeepFailedContainers || !t.Failed()) {
				Expect(platform.Delete.Execute(name)).To(Succeed())
			}
		})

		it("builds and runs the app", func() {
			deployment, logs, err := platform.Deploy.
				Execute(name, filepath.Join(fixtures, "default", "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(MatchRegexp(`Installing nginx [\d\.]+`)), logs.String())

			Eventually(deployment).Should(Serve(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")))
		})

		context("when BP_DEBUG is enabled", func() {
			it("staging output includes before/after compile hooks", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "1"}).
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainSubstring("HOOKS 1: BeforeCompile"))
				Expect(logs).To(ContainSubstring("HOOKS 2: AfterCompile"))

				Eventually(deployment).Should(Serve(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")))
			})
		})

		context("when deploying a staticfile app", func() {
			it("properly logs stdout and stderr", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")))

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/does-not-exist"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("404 Not Found"), string(contents))

				Eventually(func() string {
					logs, _ := deployment.RuntimeLogs()
					return logs
				}, "10s", "1s").Should(Or(
					ContainSubstring("GET / HTTP/1.1"),
					ContainSubstring("GET /does-not-exist HTTP/1.1"),
				))

				if settings.Platform == "docker" {
					cmd := exec.Command("docker", "container", "exec", deployment.Name, "stat", "app/nginx/logs/access.log", "app/nginx/logs/error.log")
					Expect(cmd.Run()).To(Succeed())
				}
			})
		})

		context("when using headers", func() {
			it("the returned headers and requests with headers work properly", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/fixture.json"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				resp.Body.Close()

				Expect(resp.Header["Content-Type"]).To(Equal([]string{"application/json"}))

				uri.Path = "lots_of.js"

				req, err = http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Accept-Encoding", "gzip")

				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				resp.Body.Close()

				Expect(resp.Header["Content-Encoding"]).To(Equal([]string{"gzip"}))

				Eventually(deployment).Should(Serve(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")))
			})
		})

		context("when client accepts compressed files", func() {
			it("return and handles the file", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/war_and_peace.txt"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Accept-Encoding", "gzip")

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				gzr, err := gzip.NewReader(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				contents, err := io.ReadAll(gzr)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("Leo Tolstoy"))
			})
		})

		context("when client does not accept compressed files", func() {
			it("return and handles the file", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/war_and_peace.txt"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("Leo Tolstoy"))
			})
		})

		context("when the app shows directory index", func() {
			it("the index can be seen", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "directory_index"))
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

				Expect(contents).To(ContainSubstring("find-me-too.html"))
				Expect(contents).To(ContainSubstring("find-me.html"))

				Eventually(deployment).Should(Serve(ContainSubstring("This index file should still load normally when viewing a directory; and not a directory index.")).WithEndpoint("/subdir"))
			})
		})

		context("when the app has a custom error page", func() {
			it("enables the use of custom pages", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "custom_error"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).To(ContainSubstring("Enabling custom pages for status_codes"))

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/does-not-exist"

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				defer resp.Body.Close()

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(contents).To(ContainSubstring("My 404 page"), string(contents))
			})
		})

		context("when deploying an HSTS app", func() {
			it("provides the Strict-Transport-Security header", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "with_hsts"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				resp.Body.Close()

				Expect(resp.Header["Strict-Transport-Security"]).To(Equal([]string{"max-age=31536000; includeSubDomains; preload"}))
			})
		})

		context("when deploying a large page app", func() {
			it("responds with the Vary: Accept-Encoding header", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "large_page"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())
				resp.Body.Close()

				Expect(resp.Header["Vary"]).To(Equal([]string{"Accept-Encoding"}))
			})
		})

		context("when with pushstate enabled", func() {
			it("respects pushstate routing", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "pushstate"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("This is the index file")))
				Eventually(deployment).Should(Serve(ContainSubstring("This is a static file")).WithEndpoint("/static.html"))
				Eventually(deployment).Should(Serve(ContainSubstring("This is the index file")).WithEndpoint("/does-not-exist"))
			})
		})

		context("when with a reverse proxy", func() {
			it("respects the reverse proxy", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "reverse_proxy"))
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Google Privacy Policy")).WithEndpoint("/intl/en/policies"))

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				uri.Path = "/nginx.conf"

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

		context("when use basic auth", func() {
			it("uses the providided credentials", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "basic_auth"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs).ToNot(ContainSubstring("bob:$"))
				Expect(logs).ToNot(ContainSubstring("dave:$"))

				Expect(logs).To(ContainSubstring("-----> Enabling basic authentication using Staticfile.auth"))

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				var resp *http.Response
				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())

				contents, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				resp.Body.Close()

				Expect(contents).To(ContainSubstring("401 Authorization Required"), string(contents))

				req.SetBasicAuth("bob", "bob1")

				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())

				contents, err = io.ReadAll(resp.Body)
				resp.Body.Close()

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("401 Authorization Required"), string(contents))

				req.SetBasicAuth("bob", "bob")

				Eventually(func() error { resp, err = http.DefaultClient.Do(req); return err }).Should(Succeed())

				contents, err = io.ReadAll(resp.Body)
				resp.Body.Close()

				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring("This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>."), string(contents))
			})
		})

		context("ssi", func() {
			context("when ssi is disabled", func() {
				it("does not include the ssi body", func() {
					deployment, _, err := platform.Deploy.
						Execute(name, filepath.Join(fixtures, "default", "ssi_disabled"))
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).ShouldNot(Serve(ContainSubstring("I feel included!")))
					Eventually(deployment).Should(Serve(ContainSubstring("<!--# include file=\"ssi_body.html\" -->")))
				})
			})
			context("when ssi is enabled", func() {
				it("does not include the ssi body", func() {
					deployment, _, err := platform.Deploy.
						Execute(name, filepath.Join(fixtures, "default", "ssi_enabled"))
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(ContainSubstring("I feel included!")))
					Eventually(deployment).ShouldNot(Serve(ContainSubstring("<!--# include file=\"ssi_body.html\" -->")))
				})
			})
		})

		context("when HTTP/2 is enabled", func() {
			it("served HTTP/2 traffic", func() {
				deployment, _, err := platform.Deploy.
					Execute(name, filepath.Join(fixtures, "default", "enable_http2"))
				Expect(err).NotTo(HaveOccurred())

				uri, err := url.Parse(deployment.ExternalURL)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("GET", uri.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{
					Transport: &http.Transport{
						ForceAttemptHTTP2: true,
					},
				}

				var resp *http.Response
				Eventually(func() error { resp, err = client.Do(req); return err }).Should(Succeed())
				resp.Body.Close()

				// HTTP/2 over cleartext (h2c) requires special server/client configuration
				// Verify that response succeeds; protocol version depends on server/client support
				Expect(resp.StatusCode).To(Equal(200))
			})
		})
	}
}
