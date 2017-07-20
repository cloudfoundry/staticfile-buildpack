package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploy a an app with dot files", func() {
	var app *cutlass.App
	var app_name string
	var appDir string
	var staticfile_contents string
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		if appDir != "" {
			os.RemoveAll(appDir)
		}
		app = nil
		app_name = ""
		appDir = ""
		staticfile_contents = ""
	})
	JustBeforeEach(func() {
		Expect(app_name).ToNot(BeEmpty())
		Expect(staticfile_contents).ToNot(BeEmpty())

		var err error
		appDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", app_name))
		Expect(err).To(BeNil())
		app = cutlass.New(appDir)
		ioutil.WriteFile(filepath.Join(appDir, "Staticfile"), []byte(staticfile_contents), 0644)

		PushAppAndConfirm(app)
	})

	Describe("host_dot_files: true is present in Staticfile", func() {
		Describe("the app uses the default root location", func() {
			BeforeEach(func() {
				app_name = "with_dotfile"
				staticfile_contents = "host_dot_files: true"
			})
			It("hosts the dotfiles", func() {
				Expect(app.Stdout.String()).To(ContainSubstring("Enabling hosting of dotfiles"))
				Expect(app.GetBody("/.hidden.html")).To(ContainSubstring("Hello from a hidden file"))
			})
		})
		Describe("the app specifies /public as the root location", func() {
			BeforeEach(func() {
				app_name = "dotfile_public"
				staticfile_contents = "host_dot_files: true\nroot: public"
			})
			It("hosts the dotfiles", func() {
				Expect(app.Stdout.String()).To(ContainSubstring("Enabling hosting of dotfiles"))
				Expect(app.GetBody("/.hidden.html")).To(ContainSubstring("Hello from a hidden file"))
			})
		})
	})

	Describe("host_dot_files: true not present in Staticfile", func() {
		Describe("the app uses the default root location", func() {
			BeforeEach(func() {
				app_name = "with_dotfile"
				staticfile_contents = "host_dot_files: false"
			})
			It("does not host the dotfiles", func() {
				Expect(app.Stdout.String()).ToNot(ContainSubstring("Enabling hosting of dotfiles"))
				Expect(app.GetBody("/.hidden.html")).To(ContainSubstring("404 Not Found"))
			})
		})
		Describe("the app specifies /public as the root location", func() {
			BeforeEach(func() {
				app_name = "dotfile_public"
				staticfile_contents = "host_dot_files: false\nroot: public"
			})
			It("does not host the dotfiles", func() {
				Expect(app.Stdout.String()).ToNot(ContainSubstring("Enabling hosting of dotfiles"))
				Expect(app.GetBody("/.hidden.html")).To(ContainSubstring("404 Not Found"))
			})
		})
	})
})
