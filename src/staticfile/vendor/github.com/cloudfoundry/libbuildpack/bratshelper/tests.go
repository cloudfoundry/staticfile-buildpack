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
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver"
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
		var bpName string
		var app *cutlass.App
		BeforeEach(func() {
			bpName = GenBpName("unbuilt")
			cmd := exec.Command("git", "archive", "-o", filepath.Join("/tmp", bpName+".zip"), "HEAD")
			cmd.Dir = Data.BpDir
			Expect(cmd.Run()).To(Succeed())
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, filepath.Join("/tmp", bpName+".zip"))).To(Succeed())
			Expect(os.Remove(filepath.Join("/tmp", bpName+".zip"))).To(Succeed())
			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
		})

		It("runs", func() {
			PushApp(app)
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Download go "))

			Expect(app.Stdout.String()).To(ContainSubstring("Installing " + depName))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
	})
}

func DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(copyBrats func(string) *cutlass.App) {
	Describe("deploying an app with an updated version of the same buildpack", func() {
		var bpName string
		var app *cutlass.App
		BeforeEach(func() {
			bpName = GenBpName("changing")
			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
		})
		AfterEach(func() {
			defaultCleanup(app)
			Expect(cutlass.DeleteBuildpack(bpName)).To(Succeed())
		})

		It("prints useful warning message to stdout", func() {
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, Data.UncachedFile)).To(Succeed())
			PushApp(app)
			Expect(app.Stdout.String()).ToNot(ContainSubstring("buildpack version changed from"))

			newFile, err := ModifyBuildpack(Data.UncachedFile, func(path string, r io.Reader) (io.Reader, error) {
				if path == "VERSION" {
					return strings.NewReader("NewVersion"), nil
				}
				return r, nil
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(cutlass.CreateOrUpdateBuildpack(bpName, newFile)).To(Succeed())
			PushApp(app)
			Expect(app.Stdout.String()).To(MatchRegexp(`buildpack version changed from (\S+) to NewVersion`))
		})
	})
}

func StagingWithBuildpackThatSetsEOL(depName string, copyBrats func(string) *cutlass.App) {
	Describe("staging with "+depName+" buildpack that sets EOL on dependency", func() {
		var (
			eolDate       string
			buildpackFile string
			bpName        string
			app           *cutlass.App
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
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, file)).To(Succeed())
			os.Remove(file)

			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
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

func StagingWithADepThatIsNotTheLatest(depName string, copyBrats func(string) *cutlass.App) {
	Describe("staging with a version of "+depName+" that is not the latest patch release in the manifest", func() {
		var app *cutlass.App
		BeforeEach(func() {
			manifest, err := libbuildpack.NewManifest(Data.BpDir, nil, time.Now())
			Expect(err).ToNot(HaveOccurred())
			raw := manifest.AllDependencyVersions(depName)
			vs := make([]*semver.Version, len(raw))
			for i, r := range raw {
				vs[i], err = semver.NewVersion(r)
				Expect(err).ToNot(HaveOccurred())
			}
			sort.Sort(semver.Collection(vs))
			version := vs[0].Original()

			app = copyBrats(version)
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

func StagingWithCustomBuildpackWithCredentialsInDependencies(depRegexp string, copyBrats func(string) *cutlass.App) {
	Describe("staging with custom buildpack that uses credentials in manifest dependency uris", func() {
		var (
			buildpackFile string
			bpName        string
			app           *cutlass.App
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
			Expect(cutlass.CreateOrUpdateBuildpack(bpName, file)).To(Succeed())
			os.Remove(file)

			app = copyBrats("")
			app.Buildpacks = []string{bpName + "_buildpack"}
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
				buildpackFile = Data.UncachedFile
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
			manifest, err := libbuildpack.NewManifest(Data.BpDir, nil, time.Now())
			dep, err := manifest.DefaultVersion(depName)
			Expect(err).ToNot(HaveOccurred())

			app = copyBrats(dep.Version)
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
