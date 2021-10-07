package integration_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a HTTP/2 app", func() {
	var app *cutlass.App
	var app_name string
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
		app_name = ""
	})

	BeforeEach(func() {
		apiVersionString, err := cutlass.ApiVersion()
		Expect(err).To(BeNil())
		apiVersion, err := semver.Make(apiVersionString)
		Expect(err).To(BeNil())
		apiHasHttp2, err := semver.ParseRange(">= 2.172.0")
		Expect(err).To(BeNil())
		if !apiHasHttp2(apiVersion) {
			Skip("HTTP/2 not supported for this API version.")
		}
	})

	JustBeforeEach(func() {
		Expect(app_name).ToNot(BeEmpty())
		app = cutlass.New(Fixtures(app_name))
		PushAppAndConfirm(app)

		By("inserting the app name and default route into the manifest with HTTP/2 configuration")
		manifestPath := filepath.Join(app.Path, "v3-manifest.yml")
		manifestTemplate, err := ioutil.ReadFile(manifestPath)
		Expect(err).To(BeNil())

		appName := app.Name
		appRoute, err := app.GetUrl("")
		Expect(err).To(BeNil())

		manifestBody := strings.ReplaceAll(string(manifestTemplate), "SED_APP_NAME", appName)
		manifestBody = strings.ReplaceAll(manifestBody, "SED_ROUTE", appRoute)

		By("applying manifest to set http2 protocol on route")
		err = v3ApplyManifest(app, manifestBody)
		Expect(err).To(BeNil())
	})

	Context("Using ENV Variable", func() {
		BeforeEach(func() { app_name = "enable_http2" })

		It("serves HTTP/2 traffic", func() {
			By("ensuring that the manifest background job has set the env variable", func() {
				runEnvCommand := func() string {
					envCommand := exec.Command("cf", "env", app.Name)
					output, err := envCommand.Output()
					Expect(err).To(BeNil())

					return string(output)
				}
				Eventually(runEnvCommand).Should(ContainSubstring("ENABLE_HTTP2"))
			})

			By("restarting the app to apply the environment variable change", func() {
				Expect(app.Restart()).To(Succeed())
				Eventually(app.InstanceStates).Should(Equal([]string{"RUNNING"}))
			})

			_, headers, err := app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"200"}))
		})
	})

	Context("Using Staticfile", func() {
		BeforeEach(func() { app_name = "enable_http2_in_staticfile" })

		It("serves HTTP/2 traffic", func() {
			getAppRoot := func() map[string][]string {
				_, headers, err := app.Get("/", map[string]string{})
				Expect(err).To(BeNil())
				return headers
			}
			By("polling the app until the HTTP/2 route configuration has propogated", func() {
				Eventually(getAppRoot).Should(HaveKeyWithValue("StatusCode", []string{"200"}))
			})
		})
	})
})

func v3ApplyManifest(a *cutlass.App, manifestBody string) error {
	spaceGUID, err := a.SpaceGUID()
	if err != nil {
		return err
	}
	manifestURL := fmt.Sprintf("/v3/spaces/%s/actions/apply_manifest", spaceGUID)
	command := exec.Command(
		"cf", "curl",
		"-X", "POST",
		"-H", "Content-Type: application/x-yaml",
		manifestURL,
		"-d", manifestBody,
	)
	command.Stdout = cutlass.DefaultStdoutStderr
	command.Stderr = cutlass.DefaultStdoutStderr
	if err = command.Run(); err != nil {
		return err
	}

	return nil
}
