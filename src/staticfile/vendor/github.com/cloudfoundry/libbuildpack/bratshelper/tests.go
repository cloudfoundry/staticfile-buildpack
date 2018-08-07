package bratshelper

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(copyBrats func(string) *cutlass.App) {
	Describe("deploying an app with an updated version of the same buildpack", func() {
		var (
			bpName, stack string
			app           *cutlass.App
		)
		BeforeEach(func() {
			bpName = GenBpName("changing")
			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
			stackAssociationSupported, err := cutlass.ApiGreaterThan("2.113.0")
			Expect(err).ToNot(HaveOccurred())
			if stackAssociationSupported {
				stack = app.Stack
			} else {
				stack = ""
			}
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
			// With stacks, creating the buildpack twice will result in a second record, one with `nil` stack.
			// We need to clean up both.
			if count, err := cutlass.CountBuildpack(bpName); err == nil && count > 0 {
				Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed(), "Attempted to delete buildpack %s", bpName)
			}
			// LTS errors when running `cf buildpacks`. Ignore that output.
			count, err := cutlass.CountBuildpack(bpName)
			if err == nil {
				Expect(count).To(BeZero(), "There are %d %s buildpacks", count, bpName)
			}
		})

		It("prints useful warning message to stdout", func() {
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, Data.UncachedFile, stack)).To(Succeed())
			PushApp(app)
			Expect(app.Stdout.String()).ToNot(ContainSubstring("buildpack version changed from"))

			newFile, err := ModifyBuildpack(Data.UncachedFile, func(path string, r io.Reader) (io.Reader, error) {
				if path == "VERSION" {
					return strings.NewReader("NewVersion"), nil
				}
				return r, nil
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cutlass.CreateOrUpdateBuildpack(bpName, newFile, stack)).To(Succeed(), "Could not create or update %s on %s", bpName, stack)
			PushApp(app)
			Expect(app.Stdout.String()).To(MatchRegexp(`buildpack version changed from (\S+) to NewVersion`))
		})
	})
}

func StagingWithBuildpackThatSetsEOL(depName string, copyBrats func(string) *cutlass.App) {
	Describe("staging with "+depName+" buildpack that sets EOL on dependency", func() {
		var (
			eolDate, buildpackFile, bpName, stack string
			app                                   *cutlass.App
		)
		JustBeforeEach(func() {
			eolDate = time.Now().AddDate(0, 0, 10).Format("2006-01-02")
			file, err := ModifyBuildpackManifest(buildpackFile, func(m *Manifest) {
				for _, eol := range m.DependencyDeprecationDates {
					if eol.Name == depName {
						eol.Date = eolDate
					}
				}
			})
			Expect(err).ToNot(HaveOccurred())
			bpName = GenBpName("eol")
			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
			stackAssociationSupported, err := cutlass.ApiGreaterThan("2.113.0")
			Expect(err).ToNot(HaveOccurred())
			if stackAssociationSupported {
				stack = app.Stack
			} else {
				stack = ""
			}

			Expect(cutlass.CreateOrUpdateBuildpack(bpName, file, stack)).To(Succeed())
			os.Remove(file)

			PushApp(app)
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
		})

		Context("using an uncached buildpack", func() {
			BeforeEach(func() {
				buildpackFile = Data.UncachedFile
			})
			It("warns about end of life", func() {
				Expect(app.Stdout.String()).To(MatchRegexp(`WARNING.*` + depName + ` \S+ will no longer be available in new buildpacks released after`))
			})
		})

		Context("using a cached buildpack", func() {
			BeforeEach(func() {
				buildpackFile = Data.CachedFile
			})
			It("warns about end of life", func() {
				Expect(app.Stdout.String()).To(MatchRegexp(`WARNING.*` + depName + ` \S+ will no longer be available in new buildpacks released after`))
			})
		})
	})
}

func StagingWithADepThatIsNotTheLatestConstrained(depName string, versionConstraint string, copyBrats func(string) *cutlass.App) {
	Describe("staging with a version of "+depName+" that is not the latest patch release in the manifest", func() {
		var app *cutlass.App
		BeforeEach(func() {
			manifest, err := libbuildpack.NewManifest(Data.BpDir, nil, time.Now())
			Expect(err).ToNot(HaveOccurred(), "Making new manifest from %s: error is %v", Data.BpDir, err)
			versions, err := libbuildpack.FindMatchingVersions(versionConstraint, manifest.AllDependencyVersions(depName))
			Expect(err).ToNot(HaveOccurred(), "Finding matching version: error is %v", err)
			app = copyBrats(versions[0])
			app.Buildpacks = []string{Data.Cached}
			PushApp(app)
		})
		AfterEach(func() {
			defaultCleanup(app)
		})

		It("logs a warning that tells the user to upgrade the dependency", func() {
			Expect(app.Stdout.String()).To(MatchRegexp("WARNING.*A newer version of " + depName + " is available in this buildpack"))
		})
	})
}

func StagingWithADepThatIsNotTheLatest(depName string, copyBrats func(string) *cutlass.App) {
	StagingWithADepThatIsNotTheLatestConstrained(depName, "x", copyBrats)
}

func StagingWithCustomBuildpackWithCredentialsInDependencies(depRegexp string, copyBrats func(string) *cutlass.App) {
	Describe("staging with custom buildpack that uses credentials in manifest dependency uris", func() {
		var (
			buildpackFile, bpName, stack string
			app                          *cutlass.App
		)
		JustBeforeEach(func() {
			file, err := ModifyBuildpackManifest(buildpackFile, func(m *Manifest) {
				for _, d := range m.Dependencies {
					uri, err := url.Parse(d.URI)
					uri.User = url.UserPassword("login", "password")
					Expect(err).ToNot(HaveOccurred())
					d.URI = uri.String()
				}
			})
			Expect(err).ToNot(HaveOccurred())
			bpName = GenBpName("eol")

			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}

			stackAssociationSupported, err := cutlass.ApiGreaterThan("2.113.0")
			Expect(err).ToNot(HaveOccurred())
			if stackAssociationSupported {
				stack = app.Stack
			} else {
				stack = ""
			}

			Expect(cutlass.CreateOrUpdateBuildpack(bpName, file, stack)).To(Succeed())
			os.Remove(file)
			PushApp(app)
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
		})
		Context("using an uncached buildpack", func() {
			BeforeEach(func() {
				buildpackFile = Data.UncachedFile
			})
			It("does not include credentials in logged dependency uris", func() {
				Expect(app.Stdout.String()).To(MatchRegexp(depRegexp))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("login"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("password"))
			})
		})
		Context("using a cached buildpack", func() {
			BeforeEach(func() {
				buildpackFile = Data.CachedFile
			})
			It("does not include credentials in logged dependency file paths", func() {
				Expect(app.Stdout.String()).To(MatchRegexp(depRegexp))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("login"))
				Expect(app.Stdout.String()).ToNot(ContainSubstring("password"))
			})
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

func DeployAnAppWithSensitiveEnvironmentVariables(copyBrats func(string) *cutlass.App) {
	Describe("deploying an app that has sensitive environment variables", func() {
		var app *cutlass.App
		BeforeEach(func() {
			app = copyBrats("")
			app.Buildpacks = []string{Data.Cached}
			app.SetEnv("MY_SPECIAL_VAR", "SUPER SENSITIVE DATA")
			PushApp(app)
		})
		AfterEach(func() {
			defaultCleanup(app)
		})

		It("will not write credentials to the app droplet", func() {
			Expect(app.DownloadDroplet(filepath.Join(app.Path, "droplet.tgz"))).To(Succeed())
			file, err := os.Open(filepath.Join(app.Path, "droplet.tgz"))
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()
			gz, err := gzip.NewReader(file)
			Expect(err).ToNot(HaveOccurred())
			defer gz.Close()
			tr := tar.NewReader(gz)

			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				b, err := ioutil.ReadAll(tr)
				for _, content := range []string{"MY_SPECIAL_VAR", "SUPER SENSITIVE DATA"} {
					if strings.Contains(string(b), content) {
						Fail(fmt.Sprintf("Found sensitive string %s in %s", content, hdr.Name))
					}
				}
			}
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
