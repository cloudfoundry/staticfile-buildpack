package integration_test

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a basic auth app", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	It("the app uses Staticfile.auth", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "basic_auth"))
		PushAppAndConfirm(app)

		By("uses the provided credentials for authorization", func() {
			body, _, err := app.Get("/", map[string]string{"user": "bob", "password": "bob"})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>."))

			body, _, err = app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("401 Authorization Required"))

			body, _, err = app.Get("/", map[string]string{"user": "bob", "password": "bob1"})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("401 Authorization Required"))
		})

		By("does not write the contents of .htpasswd to the logs", func() {
			Expect(app.Stdout.String()).ToNot(ContainSubstring("bob:$"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("dave:$"))
		})

		By("logs the source of authentication credentials", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Enabling basic authentication using Staticfile.auth"))
		})
	})

	Context("and is missing Staticfile", func() {
		var appDir string
		BeforeEach(func() {
			var err error
			appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "basic_auth"))
			Expect(err).To(BeNil())

			Expect(os.Remove(filepath.Join(appDir, "Staticfile"))).To(Succeed())

			app = cutlass.New(appDir)
			app.Buildpack = "staticfile_buildpack"
			PushAppAndConfirm(app)
		})
		AfterEach(func() {
			if appDir != "" {
				os.RemoveAll(appDir)
			}
			appDir = ""
		})

		It("uses the provided credentials for authorization", func() {
			body, _, err := app.Get("/", map[string]string{"user": "bob", "password": "bob"})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>."))

			body, _, err = app.Get("/", map[string]string{})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("401 Authorization Required"))

			body, _, err = app.Get("/", map[string]string{"user": "bob", "password": "bob1"})
			Expect(err).To(BeNil())
			Expect(body).To(ContainSubstring("401 Authorization Required"))
		})
	})
})
