package integration_test

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/cloudfoundry/libbuildpack/packager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a staticfile app", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "staticfile_app"))
		app.Buildpacks = []string{"staticfile_buildpack"}
		app.SetEnv("BP_DEBUG", "1")
	})

	It("succeeds", func() {
		PushAppAndConfirm(app)

		Expect(app.Stdout.String()).To(ContainSubstring("HOOKS 1: BeforeCompile"))
		Expect(app.Stdout.String()).To(ContainSubstring("HOOKS 2: AfterCompile"))
		Expect(app.Stdout.String()).To(MatchRegexp("nginx -p .*/nginx -c .*/nginx/conf/nginx.conf"))

		Expect(app.GetBody("/")).To(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets."))

		_, headers, err := app.Get("/fixture.json", map[string]string{})
		Expect(err).To(BeNil())
		Expect(headers["Content-Type"]).To(Equal([]string{"application/json"}))

		_, headers, err = app.Get("/lots_of.js", map[string]string{"Accept-Encoding": "gzip"})
		Expect(err).To(BeNil())
		Expect(headers).To(HaveKeyWithValue("Content-Encoding", []string{"gzip"}))

		By("requesting a non-compressed version of a compressed file", func() {
			By("with a client that can handle receiving compressed content", func() {
				By("returns and handles the file", func() {
					url, err := app.GetUrl("/war_and_peace.txt")
					Expect(err).To(BeNil())
					command := exec.Command("curl", "-s", "--compressed", url)
					Expect(command.Output()).To(ContainSubstring("Leo Tolstoy"))
				})
			})

			By("with a client that cannot handle receiving compressed content", func() {
				By("returns and handles the file", func() {
					url, err := app.GetUrl("/war_and_peace.txt")
					Expect(err).To(BeNil())
					command := exec.Command("curl", "-s", url)
					Expect(command.Output()).To(ContainSubstring("Leo Tolstoy"))
				})
			})
		})

		apiVersionString, err := cutlass.ApiVersion()
		Expect(err).To(BeNil())
		apiVersion, err := semver.Make(apiVersionString)
		Expect(err).To(BeNil())
		apiHasTask, err := semver.ParseRange("> 2.75.0")
		Expect(err).To(BeNil())
		if apiHasTask(apiVersion) {
			By("running a task", func() {
				By("exits", func() {
					command := exec.Command("cf", "run-task", app.Name, "wc -l public/index.html")
					_, err := command.Output()
					Expect(err).To(BeNil())

					Eventually(func() string {
						output, err := exec.Command("cf", "tasks", app.Name).Output()
						Expect(err).To(BeNil())
						return string(output)
					}, "30s").Should(MatchRegexp("SUCCEEDED.*wc.*index.html"))
				})
			})
		}

		if cutlass.Cached {
			By("with a cached buildpack", func() {
				By("logs the files it downloads", func() {
					Expect(app.Stdout.String()).To(ContainSubstring("Copy [/"))
				})
			})
		} else {
			By("with a uncached buildpack", func() {
				By("logs the files it downloads", func() {
					Expect(app.Stdout.String()).To(ContainSubstring("Download [https://"))
				})
			})
		}
	})

	Describe("internet", func() {
		var bpFile string
		buildBpFile := func() {
			var err error
			localVersion := fmt.Sprintf("%s.%s", buildpackVersion, time.Now().Format("20060102150405"))
			bpFile, err = packager.Package(bpDir, packager.CacheDir, localVersion, os.Getenv("CF_STACK"), cutlass.Cached)
			Expect(err).To(BeNil())
		}
		AfterEach(func() { os.Remove(bpFile) })

		Context("with a cached buildpack", func() {
			BeforeEach(func() {
				if !cutlass.Cached {
					Skip("Running uncached tests")
				}
				buildBpFile()
			})

			It("does not call out over the internet", func() {
				traffic, _, _, err := cutlass.InternetTraffic(
					bpDir,
					"fixtures/staticfile_app",
					bpFile,
					[]string{},
				)
				Expect(err).To(BeNil())
				Expect(traffic).To(HaveLen(0))
			})
		})

		Context("with a uncached buildpack", func() {
			var proxy *httptest.Server
			BeforeEach(func() {
				var err error
				if cutlass.Cached {
					Skip("Running cached tests")
				}

				buildBpFile()

				proxy, err = cutlass.NewProxy()
				Expect(err).To(BeNil())
			})
			AfterEach(func() {
				os.Remove(bpFile)
				proxy.Close()
			})

			It("uses a proxy during staging if present", func() {
				traffic, _, _, err := cutlass.InternetTraffic(
					bpDir,
					"fixtures/staticfile_app",
					bpFile,
					[]string{"HTTP_PROXY=" + proxy.URL, "HTTPS_PROXY=" + proxy.URL},
				)
				Expect(err).To(BeNil())

				destUrl, err := url.Parse(proxy.URL)
				Expect(err).To(BeNil())

				Expect(cutlass.UniqueDestination(
					traffic, fmt.Sprintf("%s.%s", destUrl.Hostname(), destUrl.Port()),
				)).To(BeNil())
			})
		})
	})
})
