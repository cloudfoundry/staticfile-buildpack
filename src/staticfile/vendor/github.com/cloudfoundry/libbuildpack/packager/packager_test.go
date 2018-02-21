package packager_test

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Packager", func() {
	var (
		buildpackDir string
		version      string
		cacheDir     string
	)

	BeforeEach(func() {
		var err error
		buildpackDir = "./fixtures/good"
		cacheDir, err = ioutil.TempDir("", "packager-cachedir")
		Expect(err).To(BeNil())
		version = fmt.Sprintf("1.23.45.%s", time.Now().Format("20060102150405"))

		httpmock.Reset()
	})

	Describe("Scaffold", func() {
		BeforeEach(func() {
			exists, err := libbuildpack.FileExists("bpdir")
			Expect(err).To(BeNil())
			Expect(exists).To(Equal(false))

			// run the code under test
			err = packager.Scaffold("bpdir", "mylanguage")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			os.RemoveAll("bpdir")
		})

		checkfileexists := func(path string) func() {
			return func() {
				exists, err := libbuildpack.FileExists(path)
				Expect(err).To(BeNil())
				Expect(exists).To(Equal(true))
			}
		}

		// top-level directories
		It("creates a named directory", checkfileexists("bpdir"))
		It("creates a bin directory", checkfileexists("bpdir/bin"))
		It("creates a scripts directory", checkfileexists("bpdir/scripts"))
		It("creates a src directory", checkfileexists("bpdir/src"))
		It("creates a fixtures directory", checkfileexists("bpdir/fixtures"))

		// top-level files
		It("creates a .envrc file", checkfileexists("bpdir/.envrc"))
		It("creates a .envrc file", checkfileexists("bpdir/.gitignore"))
		It("creates a manifest.yml file", checkfileexists("bpdir/manifest.yml"))
		It("creates a VERSION file", checkfileexists("bpdir/VERSION"))
		It("creates a README file", checkfileexists("bpdir/README.md"))

		// bin directory files
		It("creates a detect script", checkfileexists("bpdir/bin/detect"))
		It("creates a compile script", checkfileexists("bpdir/bin/compile"))
		It("creates a supply script", checkfileexists("bpdir/bin/supply"))
		It("creates a finalize script", checkfileexists("bpdir/bin/finalize"))
		It("creates a release script", checkfileexists("bpdir/bin/release"))

		// scripts directory files
		It("creates a brats test script", checkfileexists("bpdir/scripts/brats.sh"))
		It("creates a build script", checkfileexists("bpdir/scripts/build.sh"))
		It("creates a install_go script", checkfileexists("bpdir/scripts/install_go.sh"))
		It("creates a install_tools script", checkfileexists("bpdir/scripts/install_tools.sh"))
		It("creates a integration test script", checkfileexists("bpdir/scripts/integration.sh"))
		It("creates a unit test script", checkfileexists("bpdir/scripts/unit.sh"))

		It("creates a Gopkg.toml", checkfileexists("bpdir/src/mylanguage/Gopkg.toml"))

		// src/supply files
		It("creates a supply src directory", checkfileexists("bpdir/src/mylanguage/supply"))
		It("creates a supply src file", checkfileexists("bpdir/src/mylanguage/supply/supply.go"))
		It("creates a supply test file", checkfileexists("bpdir/src/mylanguage/supply/supply_test.go"))
		It("creates a supply cli src file", checkfileexists("bpdir/src/mylanguage/supply/cli/main.go"))

		// src/finalize files
		It("creates a finalize src directory", checkfileexists("bpdir/src/mylanguage/finalize"))
		It("creates a finalize src file", checkfileexists("bpdir/src/mylanguage/finalize/finalize.go"))
		It("creates a finalize test file", checkfileexists("bpdir/src/mylanguage/finalize/finalize.go"))
		It("creates a finalize cli src file", checkfileexists("bpdir/src/mylanguage/finalize/cli/main.go"))
	})

	Describe("Package", func() {
		var zipFile string
		var cached bool
		AfterEach(func() { os.Remove(zipFile) })

		Context("uncached", func() {
			BeforeEach(func() { cached = false })
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).To(BeNil())
			})

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-v%s.zip", version))))
			})

			It("includes files listed in manifest.yml", func() {
				Expect(ZipContents(zipFile, "bin/filename")).To(Equal("awesome content"))
			})

			It("overrides VERSION", func() {
				Expect(ZipContents(zipFile, "VERSION")).To(Equal(version))
			})

			It("runs pre-package script", func() {
				Expect(ZipContents(zipFile, "hi.txt")).To(Equal("hi mom\n"))
			})

			It("does not include files not in list", func() {
				_, err := ZipContents(zipFile, "ignoredfile")
				Expect(err.Error()).To(HavePrefix("ignoredfile not found in"))
			})

			It("does not include dependencies", func() {
				_, err := ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")
				Expect(err.Error()).To(HavePrefix("dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt not found in"))
			})

			It("does not set file on entries", func() {
				manifestYml, err := ZipContents(zipFile, "manifest.yml")
				Expect(err).To(BeNil())
				var m packager.Manifest
				Expect(yaml.Unmarshal([]byte(manifestYml), &m)).To(Succeed())
				Expect(m.Dependencies).ToNot(BeEmpty())
				Expect(m.Dependencies[0].File).To(Equal(""))
			})
		})

		Context("cached", func() {
			BeforeEach(func() { cached = true })
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).To(BeNil())
			})

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-cached-v%s.zip", version))))
			})

			It("includes files listed in manifest.yml", func() {
				Expect(ZipContents(zipFile, "bin/filename")).To(Equal("awesome content"))
			})

			It("overrides VERSION", func() {
				Expect(ZipContents(zipFile, "VERSION")).To(Equal(version))
			})

			It("runs pre-package script", func() {
				Expect(ZipContents(zipFile, "hi.txt")).To(Equal("hi mom\n"))
			})

			It("does not include files not in list", func() {
				_, err := ZipContents(zipFile, "ignoredfile")
				Expect(err.Error()).To(HavePrefix("ignoredfile not found in"))
			})

			It("includes dependencies", func() {
				Expect(ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")).To(ContainSubstring("Hyper Text Coffee Pot Control Protocol"))
			})

			It("sets file on entries", func() {
				manifestYml, err := ZipContents(zipFile, "manifest.yml")
				Expect(err).To(BeNil())
				var m packager.Manifest
				Expect(yaml.Unmarshal([]byte(manifestYml), &m)).To(Succeed())
				Expect(m.Dependencies).ToNot(BeEmpty())
				Expect(m.Dependencies[0].File).To(Equal("dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt"))
			})

			Context("dependency uses file://", func() {
				var tempfile string
				BeforeEach(func() {
					var err error
					tempdir, err := ioutil.TempDir("", "bp_fixture")
					Expect(err).ToNot(HaveOccurred())
					Expect(libbuildpack.CopyDirectory(buildpackDir, tempdir)).To(Succeed())

					fh, err := ioutil.TempFile("", "bp_dependency")
					Expect(err).ToNot(HaveOccurred())
					fh.WriteString("keaty")
					fh.Close()
					tempfile = fh.Name()

					manifestyml, err := ioutil.ReadFile(filepath.Join(tempdir, "manifest.yml"))
					Expect(err).ToNot(HaveOccurred())
					manifestyml2 := string(manifestyml)
					manifestyml2 = strings.Replace(manifestyml2, "https://www.ietf.org/rfc/rfc2324.txt", "file://"+tempfile, -1)
					manifestyml2 = strings.Replace(manifestyml2, "b11329c3fd6dbe9dddcb8dd90f18a4bf441858a6b5bfaccae5f91e5c7d2b3596", "f909ee4c4bec3280bbbff6b41529479366ab10c602d8aed33e3a86f0a9c5db4e", -1)
					Expect(ioutil.WriteFile(filepath.Join(tempdir, "manifest.yml"), []byte(manifestyml2), 0644)).To(Succeed())

					buildpackDir = tempdir
				})
				AfterEach(func() { os.RemoveAll(buildpackDir) })

				It("includes dependencies", func() {
					dest := filepath.Join("dependencies", fmt.Sprintf("%x", md5.Sum([]byte("file://"+tempfile))), filepath.Base(tempfile))
					Expect(ZipContents(zipFile, dest)).To(ContainSubstring("keaty"))
				})
			})
		})

		Context("cached dependency has wrong md5", func() {
			BeforeEach(func() {
				cached = true
				buildpackDir = "./fixtures/bad"
			})
			It("includes dependencies", func() {
				_, err := packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("dependency sha256 mismatch: expected sha256 fffffff, actual sha256 b11329c3fd6dbe9dddcb8dd90f18a4bf441858a6b5bfaccae5f91e5c7d2b3596"))
			})
		})
	})
})
