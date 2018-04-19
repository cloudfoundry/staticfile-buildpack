package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"fmt"
)

var _ = Describe("deploy a staticfile app", func() {
	var app *cutlass.App
	var app_name string
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
		app_name = ""
	})
	JustBeforeEach(func() {
		Expect(app_name).ToNot(BeEmpty())
		app = cutlass.New(filepath.Join(bpDir, "fixtures", app_name))
		PushAppAndConfirm(app)
	})

	Context("Using ENV Variable", func() {
		BeforeEach(func() { app_name = "with_https" })

		It("receives a 301 redirect to https", func() {
			_, headers, err := app.Get("/", map[string]string{"NoFollow": "true"})
			Expect(err).To(BeNil())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"301"}))
			Expect(headers).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://"))))
		})

		It("injects x-forwarded-host into Location on redirect", func() {
			var upstreamHostName = "upstreamHostName.com"
			_, headers, err := app.Get("/", map[string]string{"NoFollow": "true", "X-Forwarded-Host": upstreamHostName})
			Expect(err).To(BeNil())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"301"}))
			Expect(headers).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix(fmt.Sprintf("https://%s", upstreamHostName)))))
		})
	})

	Context("Using Staticfile", func() {
		BeforeEach(func() { app_name = "with_https_in_staticfile" })

		It("receives a 301 redirect to https", func() {
			_, headers, err := app.Get("/", map[string]string{"NoFollow": "true"})
			Expect(err).To(BeNil())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"301"}))
			Expect(headers).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix("https://"))))
		})

		It("injects x-forwarded-host into Location on redirect", func() {
			var upstreamHostName = "upstreamHostName.com"
			_, headers, err := app.Get("/", map[string]string{"NoFollow": "true", "X-Forwarded-Host": upstreamHostName})
			Expect(err).To(BeNil())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"301"}))
			Expect(headers).To(HaveKeyWithValue("Location", ConsistOf(HavePrefix(fmt.Sprintf("https://%s", upstreamHostName)))))
		})

	})
})
