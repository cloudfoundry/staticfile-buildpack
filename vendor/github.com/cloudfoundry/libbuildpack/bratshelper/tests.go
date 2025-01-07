package bratshelper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func defaultCleanup(app *cutlass.App) {
	DestroyApp(app)
	os.RemoveAll(app.Path)
}

func UnbuiltBuildpack(depName string, copyBrats func(string) *cutlass.App) {
	Context("Unbuilt buildpack (eg github)", func() {
		var (
			bpName string
			app    *cutlass.App
		)
		BeforeEach(func() {
			bpName = GenBpName("unbuilt")
			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
			cmd := exec.Command("git", "archive", "-o", filepath.Join("/tmp", bpName+".zip"), "HEAD")
			cmd.Dir = Data.BpDir
			Expect(cmd.Run()).To(Succeed())
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, filepath.Join("/tmp", bpName+".zip"), "")).To(Succeed())
			Expect(os.Remove(filepath.Join("/tmp", bpName+".zip"))).To(Succeed())
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
		})

		It("runs", func() {
			PushApp(app)
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Download go "))

			if depName != "" {
				Expect(app.Stdout.String()).To(ContainSubstring("Installing " + depName))
			}
			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
	})
}

func DeployAppWithExecutableProfileScript(depName string, copyBrats func(string) *cutlass.App) {
	Describe("deploying an app that has an executable .profile script", func() {
		var app *cutlass.App
		BeforeEach(func() {
			if depName != "" {
				manifest, err := libbuildpack.NewManifest(Data.BpDir, nil, time.Now())
				Expect(err).ToNot(HaveOccurred())
				dep, err := manifest.DefaultVersion(depName)
				Expect(err).ToNot(HaveOccurred())

				app = copyBrats(dep.Version)
			} else {
				app = copyBrats("")
			}
			AddDotProfileScriptToApp(app.Path)
			app.Buildpacks = []string{Data.Cached}
			PushApp(app)
		})
		AfterEach(func() {
			defaultCleanup(app)
		})

		It("executes the .profile script", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("PROFILE_SCRIPT_IS_PRESENT_AND_RAN"))
		})
		It("does not let me view the .profile script", func() {
			_, headers, err := app.Get("/.profile", map[string]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(headers).To(HaveKeyWithValue("StatusCode", []string{"404"}))
		})
	})
}

func ForAllSupportedVersions(depName string, copyBrats func(string) *cutlass.App, runTests func(string, *cutlass.App)) {
	Describe("For all supported "+depName+" versions", func() {
		bpDir, err := cutlass.FindRoot()
		if err != nil {
			panic(err)
		}
		manifest, err := libbuildpack.NewManifest(bpDir, nil, time.Now())
		if err != nil {
			panic(err)
		}
		versions := manifest.AllDependencyVersions(depName)

		var app *cutlass.App
		AfterEach(func() {
			defaultCleanup(app)
		})

		for _, v := range versions {
			version := v
			It("with "+depName+" "+version, func() {
				app = copyBrats(version)
				app.Buildpacks = []string{Data.Cached}

				runTests(version, app)
			})
		}
	})
}

func ForAllSupportedVersions2(depName1, depName2 string, compatible func(string, string) bool, itString string, copyBrats func(string, string) *cutlass.App, runTests func(string, string, *cutlass.App)) {
	Describe("For all supported "+depName1+" and "+depName2+" versions", func() {
		bpDir, err := cutlass.FindRoot()
		if err != nil {
			panic(err)
		}
		manifest, err := libbuildpack.NewManifest(bpDir, nil, time.Now())
		if err != nil {
			panic(err)
		}
		versions1 := manifest.AllDependencyVersions(depName1)
		versions2 := manifest.AllDependencyVersions(depName2)

		var app *cutlass.App
		AfterEach(func() {
			defaultCleanup(app)
		})

		for _, v1 := range versions1 {
			version1 := v1
			for _, v2 := range versions2 {
				version2 := v2
				if compatible(v1, v2) {
					It(fmt.Sprintf(itString, version1, version2), func() {
						app = copyBrats(version1, version2)
						app.Buildpacks = []string{Data.Cached}

						runTests(version1, version2, app)
					})
				}
			}
		}
	})
}
