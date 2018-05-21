package libbuildpack_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	var (
		oldCfStack  string
		manifest    *libbuildpack.Manifest
		manifestDir string
		err         error
		version     string
		currentTime time.Time
		buffer      *bytes.Buffer
		logger      *libbuildpack.Logger
	)

	BeforeEach(func() {
		oldCfStack = os.Getenv("CF_STACK")
		os.Setenv("CF_STACK", "cflinuxfs2")

		manifestDir = "fixtures/manifest/standard"
		currentTime = time.Now()
		httpmock.Reset()

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))
	})
	AfterEach(func() { err = os.Setenv("CF_STACK", oldCfStack); Expect(err).To(BeNil()) })

	JustBeforeEach(func() {
		manifest, err = libbuildpack.NewManifest(manifestDir, logger, currentTime)
		Expect(err).To(BeNil())
	})

	Describe("NewManifest", func() {
		It("has a language", func() {
			Expect(manifest.Language()).To(Equal("dotnet-core"))
		})
	})

	Describe("ApplyOverride", func() {
		var depsDir string
		BeforeEach(func() {
			depsDir, err = ioutil.TempDir("", "libbuildpack_override")
			Expect(err).ToNot(HaveOccurred())
			Expect(os.Mkdir(filepath.Join(depsDir, "0"), 0755)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(depsDir, "1"), 0755)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(depsDir, "2"), 0755)).To(Succeed())

			data := `---
dotnet-core:
  default_versions:
  - name: node
    version: 1.7.x
  - name: thing
    version: 9.3.x
  dependencies:
  - name: node
    version: 1.7.6
    cf_stacks: ['cflinuxfs2']
  - name: thing
    version: 9.3.6
    cf_stacks: ['cflinuxfs2']
ruby:
  default_versions:
  - name: node
    version: 2.2.x
  dependencies:
  - name: node
    version: 2.2.2
    cf_stacks: ['cflinuxfs2']
`
			Expect(ioutil.WriteFile(filepath.Join(depsDir, "1", "override.yml"), []byte(data), 0644)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(depsDir)).To(Succeed())
		})

		It("updates default version", func() {
			Expect(manifest.DefaultVersion("node")).To(Equal(libbuildpack.Dependency{Name: "node", Version: "6.9.4"}))

			Expect(manifest.ApplyOverride(depsDir)).To(Succeed())

			Expect(manifest.DefaultVersion("node")).To(Equal(libbuildpack.Dependency{Name: "node", Version: "1.7.6"}))
		})

		It("doesn't remove data which is not overriden", func() {
			Expect(manifest.DefaultVersion("ruby")).To(Equal(libbuildpack.Dependency{Name: "ruby", Version: "2.3.3"}))

			Expect(manifest.ApplyOverride(depsDir)).To(Succeed())

			Expect(manifest.DefaultVersion("ruby")).To(Equal(libbuildpack.Dependency{Name: "ruby", Version: "2.3.3"}))
		})

		It("adds new default versions", func() {
			Expect(manifest.ApplyOverride(depsDir)).To(Succeed())

			Expect(manifest.DefaultVersion("thing")).To(Equal(libbuildpack.Dependency{Name: "thing", Version: "9.3.6"}))
		})
	})

	Describe("CheckStackSupport", func() {
		Context("Stack is supported", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/stacks"
				err = os.Setenv("CF_STACK", "cflinuxfs2")
				Expect(err).To(BeNil())
			})

			It("returns nil", func() {
				Expect(manifest.CheckStackSupport()).To(Succeed())
			})

			Context("with no dependencies listed", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/no-deps"
				})
				It("returns nil", func() {
					Expect(manifest.CheckStackSupport()).To(Succeed())
				})
			})

			Context("by a single dependency", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/stacks"
					err = os.Setenv("CF_STACK", "xenial")
					Expect(err).To(BeNil())
				})
				It("returns nil", func() {
					Expect(manifest.CheckStackSupport()).To(Succeed())
				})
			})

			Context("by the whole manifest", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/packaged-with-stack"
					err = os.Setenv("CF_STACK", "cflinuxfs2")
					Expect(err).To(BeNil())
				})
				It("returns nil", func() {
					Expect(manifest.CheckStackSupport()).To(Succeed())
				})
			})
		})

		Context("Stack is not supported", func() {
			Context("stacks specified in dependencies", func() {
				BeforeEach(func() {
					err = os.Setenv("CF_STACK", "notastack")
					Expect(err).To(BeNil())
				})

				It("returns an error", func() {
					Expect(manifest.CheckStackSupport()).To(MatchError(errors.New("required stack notastack was not found")))
				})
			})
			Context("stack specified in top-level of manifest", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/packaged-with-stack"
					err = os.Setenv("CF_STACK", "notastack")
					Expect(err).To(BeNil())
				})

				It("returns an error", func() {
					Expect(manifest.CheckStackSupport()).To(MatchError(errors.New("required stack notastack was not found")))
				})
			})
		})
	})

	Describe("Version", func() {
		Context("VERSION file exists", func() {
			It("returns the version", func() {
				version, err = manifest.Version()
				Expect(err).To(BeNil())

				Expect(version).To(Equal("99.99"))
			})
		})

		Context("VERSION file does not exist", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/duplicate"
			})

			It("returns an error", func() {
				version, err = manifest.Version()
				Expect(version).To(Equal(""))
				Expect(err).ToNot(BeNil())

				Expect(err.Error()).To(ContainSubstring("unable to read VERSION file"))
			})
		})
	})

	Describe("AllDependencyVersions", func() {
		It("returns all the versions of the dependency", func() {
			versions := manifest.AllDependencyVersions("dotnet-framework")
			Expect(versions).To(Equal([]string{"1.0.0", "1.0.1", "1.0.3", "1.1.0"}))
		})

		Context("CF_STACK = xenial", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/stacks"
				os.Setenv("CF_STACK", "xenial")
			})
			It("limits to dependencies matching CF_STACK", func() {
				versions := manifest.AllDependencyVersions("thing")
				Expect(versions).To(Equal([]string{"1"}))
			})
		})

		Context("CF_STACK = cflinuxfs2", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/stacks"
				os.Setenv("CF_STACK", "cflinuxfs2")
			})
			It("limits to dependencies matching CF_STACK", func() {
				versions := manifest.AllDependencyVersions("thing")
				Expect(versions).To(Equal([]string{"1", "2"}))
			})
		})

		Context("CF_STACK = empty string", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/stacks"
				os.Setenv("CF_STACK", "cflinuxfs2")
			})
			It("lists all dependencies matching name", func() {
				versions := manifest.AllDependencyVersions("thing")
				Expect(versions).To(Equal([]string{"1", "2"}))
			})
		})

		Context("stack specified in top-level of manifest", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/packaged-with-stack"
			})

			Context("stack matches", func() {
				BeforeEach(func() {
					os.Setenv("CF_STACK", "cflinuxfs2")
				})
			    It("returns all versions of the dependency", func() {
					versions := manifest.AllDependencyVersions("jruby")
					Expect(versions).To(Equal([]string{"9.3.4", "9.3.5", "9.4.4"}))
			    })
			})

			Context("stack does not match", func() {
				BeforeEach(func() {
					os.Setenv("CF_STACK", "inanestack")
				})

				It("returns an empty list", func() {
					versions := manifest.AllDependencyVersions("jruby")
					Expect(versions).To(BeEmpty())
				})
			})
		})
	})

	Describe("IsCached", func() {
		BeforeEach(func() {
			var err error
			manifestDir, err = ioutil.TempDir("", "cached")
			Expect(err).To(BeNil())

			data, err := ioutil.ReadFile("fixtures/manifest/fetch/manifest.yml")
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(manifestDir, "manifest.yml"), data, 0644)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			Expect(os.RemoveAll(manifestDir)).To(Succeed())
		})

		Context("uncached buildpack", func() {
			It("is false", func() {
				Expect(manifest.IsCached()).To(BeFalse())
			})
		})

		Context("cached buildpack", func() {
			BeforeEach(func() {
				dependenciesDir := filepath.Join(manifestDir, "dependencies")
				Expect(os.MkdirAll(dependenciesDir, 0755)).To(Succeed())
			})
			It("is true", func() {
				Expect(manifest.IsCached()).To(BeTrue())
			})
		})
	})

	Describe("DefaultVersion", func() {
		Context("requested name exists and default version is locked to the patch", func() {
			It("returns the default", func() {
				dep, err := manifest.DefaultVersion("node")
				Expect(err).To(BeNil())

				Expect(dep).To(Equal(libbuildpack.Dependency{Name: "node", Version: "6.9.4"}))
			})
		})

		Context("requested name exists multiple times in dependencies and default version is locked to minor line", func() {
			It("returns the default", func() {
				dep, err := manifest.DefaultVersion("jruby")
				Expect(err).To(BeNil())

				Expect(dep).To(Equal(libbuildpack.Dependency{Name: "jruby", Version: "9.3.5"}))
			})
		})

		Context("requested name exists multiple times in dependencies and default version is locked to major line", func() {
			It("returns the default", func() {
				dep, err := manifest.DefaultVersion("ruby")
				Expect(err).To(BeNil())

				Expect(dep).To(Equal(libbuildpack.Dependency{Name: "ruby", Version: "2.3.3"}))
			})
		})

		Context("requested name exists (twice) in default version section", func() {
			BeforeEach(func() { manifestDir = "fixtures/manifest/duplicate" })
			It("returns an error", func() {
				_, err := manifest.DefaultVersion("bower")
				Expect(err.Error()).To(Equal("found 2 default versions for bower"))
			})
		})

		Context("requested name does not exist", func() {
			It("returns an error", func() {
				_, err := manifest.DefaultVersion("notexist")
				Expect(err.Error()).To(Equal("no default version for notexist"))
			})
		})

		Context("stack specified in top-level of manifest", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/packaged-with-stack"
			})

			Context("stack matches", func() {
				BeforeEach(func() {
					os.Setenv("CF_STACK", "cflinuxfs2")
				})
				It("returns all versions of the dependency", func() {
					dep, err := manifest.DefaultVersion("jruby")
					Expect(err).To(BeNil())
					Expect(dep).To(Equal(libbuildpack.Dependency{Name: "jruby", Version: "9.3.5"}))
				})
			})

			Context("stack does not match", func() {
				BeforeEach(func() {
					os.Setenv("CF_STACK", "inanestack")
				})

				It("returns an error", func() {
					_, err := manifest.DefaultVersion("jruby")
					Expect(err.Error()).To(Equal("no match found for 9.3.x in []"))
				})
			})
		})
	})

	Describe("CheckBuildpackVersion", func() {
		var cacheDir string

		BeforeEach(func() {
			cacheDir, err = ioutil.TempDir("", "cache")
		})

		AfterEach(func() {
			err = os.RemoveAll(cacheDir)
			Expect(err).To(BeNil())
		})

		Context("BUILDPACK_METADATA exists", func() {
			Context("The language does not match", func() {
				BeforeEach(func() {
					metadata := "---\nlanguage: diffLang\nversion: 99.99"
					ioutil.WriteFile(filepath.Join(cacheDir, "BUILDPACK_METADATA"), []byte(metadata), 0666)
				})

				It("Does not log anything", func() {
					manifest.CheckBuildpackVersion(cacheDir)
					Expect(buffer.String()).To(Equal(""))
				})
			})
			Context("The language matches", func() {
				Context("The version matches", func() {
					BeforeEach(func() {
						metadata := "---\nlanguage: dotnet-core\nversion: 99.99"
						ioutil.WriteFile(filepath.Join(cacheDir, "BUILDPACK_METADATA"), []byte(metadata), 0666)
					})

					It("Does not log anything", func() {
						manifest.CheckBuildpackVersion(cacheDir)
						Expect(buffer.String()).To(Equal(""))

					})
				})

				Context("The version does not match", func() {
					BeforeEach(func() {
						metadata := "---\nlanguage: dotnet-core\nversion: 33.99"
						ioutil.WriteFile(filepath.Join(cacheDir, "BUILDPACK_METADATA"), []byte(metadata), 0666)
					})

					It("Logs a warning that the buildpack version has changed", func() {
						manifest.CheckBuildpackVersion(cacheDir)
						Expect(buffer.String()).To(ContainSubstring("buildpack version changed from 33.99 to 99.99"))

					})
				})
			})
		})

		Context("BUILDPACK_METADATA does not exist", func() {
			It("Does not log anything", func() {
				manifest.CheckBuildpackVersion(cacheDir)
				Expect(buffer.String()).To(Equal(""))

			})
		})
	})

	Describe("StoreBuildpackMetadata", func() {
		var cacheDir string

		BeforeEach(func() {
			cacheDir, err = ioutil.TempDir("", "cache")
		})

		AfterEach(func() {
			err = os.RemoveAll(cacheDir)
			Expect(err).To(BeNil())
		})

		Context("VERSION file exists", func() {
			Context("cache dir exists", func() {
				It("writes to the BUILDPACK_METADATA file", func() {
					manifest.StoreBuildpackMetadata(cacheDir)

					var md libbuildpack.BuildpackMetadata

					y := &libbuildpack.YAML{}
					err = y.Load(filepath.Join(cacheDir, "BUILDPACK_METADATA"), &md)
					Expect(err).To(BeNil())

					Expect(md.Language).To(Equal("dotnet-core"))
					Expect(md.Version).To(Equal("99.99"))
				})
			})

			Context("cache dir does not exist", func() {
				It("Does not log anything", func() {
					manifest.StoreBuildpackMetadata(filepath.Join(cacheDir, "not_exist"))
					Expect(buffer.String()).To(Equal(""))
					Expect(filepath.Join(cacheDir, "not_exist")).ToNot(BeADirectory())
				})
			})
		})

		Context("VERSION file does not exist", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/stacks"
			})

			It("Does not log anything", func() {
				manifest.StoreBuildpackMetadata(cacheDir)
				Expect(buffer.String()).To(Equal(""))
			})
		})
	})

	Describe("GetEntry", func() {
		var depToFind libbuildpack.Dependency

		Context("dependency matches", func() {
			BeforeEach(func() {
			    depToFind = libbuildpack.Dependency{"jruby", "9.3.5"}
			})

			Context("top-level manifest stack exists", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/packaged-with-stack"
				})

				Context("top-level manifest stack matches", func() {
					BeforeEach(func() {
						os.Setenv("CF_STACK", "cflinuxfs2")
					})

					It("returns matched dependency", func() {
						entry, err := manifest.GetEntry(depToFind)
						Expect(err).To(BeNil())
						Expect(entry.Dependency).To(Equal(depToFind))
					})
				})

				Context("top-level manifest stack does not match", func() {
					BeforeEach(func() {
						os.Setenv("CF_STACK", "inanestack")
					})

					It("returns an error", func() {
						_, err := manifest.GetEntry(depToFind)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("top-level manifest stack does not exist", func() {
				BeforeEach(func() {
					manifestDir = "fixtures/manifest/standard"
				})

				Context("dependency stack matches", func() {
					BeforeEach(func() {
						os.Setenv("CF_STACK", "cflinuxfs2")
					})

					It("returns matched dependency", func() {
						entry, err := manifest.GetEntry(depToFind)
						Expect(err).To(BeNil())
						Expect(entry.Dependency).To(Equal(depToFind))
					})
				})

				Context("dependency stack does not match", func() {
					BeforeEach(func() {
						os.Setenv("CF_STACK", "inanestack")
					})

					It("returns an error", func() {
						_, err := manifest.GetEntry(depToFind)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("dependency does not match ", func() {
				BeforeEach(func() {
					depToFind = libbuildpack.Dependency{"inanedep", "11"}
				})

				It("returns an error", func() {
					_, err := manifest.GetEntry(depToFind)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
