package packager_test

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/jarcoal/httpmock.v1"
	"gopkg.in/yaml.v2"

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
		stack        string
		err          error
	)

	BeforeEach(func() {
		stack = "cflinuxfs2"
		buildpackDir = "./fixtures/good"
		cacheDir, err = ioutil.TempDir("", "packager-cachedir")
		Expect(err).To(BeNil())
		version = fmt.Sprintf("1.23.45.%s", time.Now().Format("20060102150405"))

		httpmock.Reset()
	})

	Describe("Package", func() {
		var zipFile string
		var cached bool
		AfterEach(func() { os.Remove(zipFile) })

		AssertStack := func() {
			var manifest *packager.Manifest
			Context("stack specified and matches any dependency in manifest.yml", func() {
				BeforeEach(func() { stack = "cflinuxfs2" })
				JustBeforeEach(func() {
					manifestYml, err := ZipContents(zipFile, "manifest.yml")
					Expect(err).To(BeNil())
					manifest = &packager.Manifest{}
					Expect(yaml.Unmarshal([]byte(manifestYml), manifest)).To(Succeed())
				})

				It("removes dependencies for other stacks from the manifest", func() {
					Expect(len(manifest.Dependencies)).To(Equal(1))
					Expect(manifest.Dependencies[0].SHA256).To(Equal("b11329c3fd6dbe9dddcb8dd90f18a4bf441858a6b5bfaccae5f91e5c7d2b3596"))
				})

				It("removes cfstacks from the remaining dependencies", func() {
					Expect(manifest.Dependencies[0].Stacks).To(BeNil())
				})

				It("adds a top-level stack: key to the manifest", func() {
					Expect(manifest.Stack).To(Equal(stack))
				})
			})

			Context("empty stack specified", func() {
				BeforeEach(func() { stack = "" })
				JustBeforeEach(func() {
					manifestYml, err := ZipContents(zipFile, "manifest.yml")
					Expect(err).To(BeNil())
					manifest = &packager.Manifest{}
					Expect(yaml.Unmarshal([]byte(manifestYml), manifest)).To(Succeed())
				})

				It("includes dependencies for all stacks in the manifest", func() {
					Expect(len(manifest.Dependencies)).To(Equal(2))
				})

				It("does not add a top-level stack: key to the manifest", func() {
					Expect(manifest.Stack).To(Equal(""))
				})

				It("does not remove cf_stacks from dependencies", func() {
					Expect(manifest.Dependencies[0].Stacks).To(Equal([]string{"cflinuxfs2"}))
					Expect(manifest.Dependencies[1].Stacks).To(Equal([]string{"cflinuxfs3"}))
				})
			})
		}

		Context("manifest.yml was already packaged", func() {
			Context("setting specific stack", func() {
				BeforeEach(func() { stack = "cflinuxfs2" })
				It("returns an error", func() {
					zipFile, err = packager.Package("./fixtures/prepackaged", cacheDir, version, stack, cached)
					Expect(err).To(MatchError("Cannot package from already packaged buildpack manifest"))
				})
			})

			Context("setting any stack", func() {

				BeforeEach(func() { stack = "" })

				It("returns an error", func() {
					zipFile, err = packager.Package("./fixtures/prepackaged", cacheDir, version, stack, cached)
					Expect(err).To(MatchError("Cannot package from already packaged buildpack manifest"))
				})
			})
		})

		Context("manifest.yml has no dependencies", func() {
			BeforeEach(func() { stack = "cflinuxfs2" })

			It("allows stack when packaging", func() {
				zipFile, err = packager.Package("./fixtures/no_dependencies", cacheDir, version, stack, cached)
				Expect(err).To(BeNil())
			})
		})

		Context("stack is invalid", func() {
			Context("stack not found in any dependencies", func() {
				BeforeEach(func() { stack = "nonexistent-stack" })

				It("returns an error", func() {
					zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
					Expect(err).To(MatchError("Stack `nonexistent-stack` not found in manifest"))
				})
			})
			Context("stack not found in any default dependencies", func() {
				BeforeEach(func() {
					stack = "cflinuxfs3"
					buildpackDir = "./fixtures/missing_default_fs3"
				})

				It("returns an error", func() {
					zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
					Expect(err).To(MatchError("No matching default dependency `ruby` for stack `cflinuxfs3`"))
				})
			})
		})

		Context("uncached", func() {
			BeforeEach(func() { cached = false })
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
				Expect(err).To(BeNil())
			})

			AssertStack()

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-cflinuxfs2-v%s.zip", version))))
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
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
				Expect(err).To(BeNil())
			})

			AssertStack()

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-cached-cflinuxfs2-v%s.zip", version))))
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

			Describe("including dependencies", func() {
				Context("when a stack is specified", func() {
					It("includes ONLY dependencies for the specified stack", func() {
						Expect(ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")).To(ContainSubstring("Hyper Text Coffee Pot Control Protocol"))
						_, err := ZipContents(zipFile, "dependencies/ff1eb131521acf5bc95db59b2a2c29c0/rfc2549.txt")
						Expect(err.Error()).To(HavePrefix("dependencies/ff1eb131521acf5bc95db59b2a2c29c0/rfc2549.txt not found in"))
					})
				})
				Context("when the empty stack is specified", func() {
					BeforeEach(func() { stack = "" })

					It("includes dependencies for ALL stacks if the empty stack is used", func() {
						Expect(ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")).To(ContainSubstring("Hyper Text Coffee Pot Control Protocol"))
						Expect(ZipContents(zipFile, "dependencies/ff1eb131521acf5bc95db59b2a2c29c0/rfc2549.txt")).To(ContainSubstring("IP over Avian Carriers with Quality of Service"))
					})
				})
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

		Context("when buildpack includes symlink to directory", func() {
			BeforeEach(func() {
				cached = true
				buildpackDir = "./fixtures/symlink_dir"
			})
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
				Expect(err).To(BeNil())
			})
			It("gets zipfile name", func() {
				Expect(zipFile).ToNot(BeEmpty())
			})
			It("generates a zipfile with name", func() {
				var cachedStr string
				if cached {
					cachedStr = "-cached"
				}
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack%s-cflinuxfs2-v%s.zip", cachedStr, version))))
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
		})

		Context("cached dependency has wrong md5", func() {
			BeforeEach(func() {
				cached = true
				buildpackDir = "./fixtures/bad"
			})
			It("includes dependencies", func() {
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("dependency sha256 mismatch: expected sha256 fffffff, actual sha256 b11329c3fd6dbe9dddcb8dd90f18a4bf441858a6b5bfaccae5f91e5c7d2b3596"))
			})
		})

		Context("packaging with no stack", func() {
			BeforeEach(func() {
				cached = false
				stack = ""
			})

			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, stack, cached)
				Expect(err).To(BeNil())
			})

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-v%s.zip", version))))
			})
		})
	})
})
