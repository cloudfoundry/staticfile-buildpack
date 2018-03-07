package libbuildpack_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	httpmock "gopkg.in/jarcoal/httpmock.v1"

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
		})

		Context("Stack is not supported", func() {
			BeforeEach(func() {
				err = os.Setenv("CF_STACK", "notastack")
				Expect(err).To(BeNil())
			})

			It("returns nil", func() {
				Expect(manifest.CheckStackSupport()).To(MatchError(errors.New("required stack notastack was not found")))
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

	Describe("FetchDependency", func() {
		type ExpectedEntry struct {
			entry          libbuildpack.ManifestEntry
			content        []byte
			appCachePath   string
			buildpackCache string
		}

		var (
			tmpdir, outputFile, appCacheDir string
			entryToFetch                    ExpectedEntry
			allEntries                      []libbuildpack.ManifestEntry
		)

		BeforeEach(func() {
			tmpdir, err = ioutil.TempDir("", "downloads")
			Expect(err).To(BeNil())
			outputFile = filepath.Join(tmpdir, "out.tgz")

			manifestDir, err = ioutil.TempDir("", "buildpack")
			Expect(err).To(BeNil())
			appCacheDir, err = ioutil.TempDir("", "appCache")
			Expect(err).To(BeNil())

			entryToFetch = ExpectedEntry{
				entry: libbuildpack.ManifestEntry{
					Dependency: libbuildpack.Dependency{
						Name:    "thing",
						Version: "1"},
					URI:      "https://example.com/dependencies/thing-1-linux-x64.tgz",
					File:     "",
					SHA256:   "fdf72806b9bc1a1bc78be1bfc21978d03591dea5042304211b81235dbf87bd77",
					CFStacks: []string{"cflinuxfs2"},
				},
				content: []byte("exciting binary data"),
			}
			shaURI := sha256.Sum256([]byte(entryToFetch.entry.URI))
			entryToFetch.appCachePath = filepath.Join(appCacheDir, "dependencies", hex.EncodeToString(shaURI[:]), "thing-1-linux-x64.tgz")

			allEntries = []libbuildpack.ManifestEntry{entryToFetch.entry}
			for _, name := range []string{"thing", "some-dependency-name", "mysql"} {
				for _, version := range []string{"5", "6", "7.5.0", "albumin"} {
					entry := libbuildpack.ManifestEntry{
						Dependency: libbuildpack.Dependency{
							Name:    name,
							Version: version,
						},
					}
					allEntries = append(allEntries, entry)
				}
			}
		})
		AfterEach(func() {
			Expect(os.RemoveAll(tmpdir)).To(Succeed())
			Expect(os.RemoveAll(appCacheDir)).To(Succeed())
		})

		type CachedTestInputs struct {
			pathToCachedFile string
			dependency       libbuildpack.Dependency
			expectedContents []byte
		}

		usingCachedFileTests := func(inputs *CachedTestInputs) {
			Context("dependency exists cached on disk and matches checksum", func() {
				BeforeEach(func() {
					os.MkdirAll(filepath.Dir(inputs.pathToCachedFile), 0755)
					Expect(ioutil.WriteFile(inputs.pathToCachedFile, inputs.expectedContents, 0644)).To(Succeed())
				})
				It("copies the cached file to outputFile", func() {
					err = manifest.FetchDependency(inputs.dependency, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal(inputs.expectedContents))
				})
				It("makes intermediate directories", func() {
					outputFile = filepath.Join(tmpdir, "notexist", "out.tgz")
					err = manifest.FetchDependency(inputs.dependency, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal(inputs.expectedContents))
				})
			})

			Context("dependency exists cached on disk and does not match checksum", func() {
				BeforeEach(func() {
					os.MkdirAll(filepath.Dir(inputs.pathToCachedFile), 0755)
					Expect(ioutil.WriteFile(inputs.pathToCachedFile, append(inputs.expectedContents, []byte(" except not")...), 0644)).To(Succeed())
				})
				It("raises error", func() {
					err = manifest.FetchDependency(inputs.dependency, outputFile)

					Expect(err).ToNot(BeNil())
				})
				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(inputs.dependency, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
			})

			Context("dependency is not cached on disk", func() {
				It("raises error", func() {
					err = manifest.FetchDependency(inputs.dependency, outputFile)

					Expect(err).ToNot(BeNil())
				})
			})

		}

		type DownloadingTestInputs struct {
			Dependency       libbuildpack.Dependency
			DependencyURI    string
			ExpectedContent  []byte
			PathToCachedFile string
			CheckOnSuccess   func()
			CheckOnError     func()
		}

		BehaviorWhenDownloading := func(inputs *DownloadingTestInputs) {
			Context("url exists and matches checksum", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", inputs.DependencyURI,
						httpmock.NewStringResponder(200, string(inputs.ExpectedContent)))
				})

				It("downloads the file to the requested location", func() {
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal(inputs.ExpectedContent))
				})
				inputs.CheckOnSuccess()

				It("makes intermediate directories", func() {
					outputFile = filepath.Join(tmpdir, "notexist", "out.tgz")
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal(inputs.ExpectedContent))
				})
			})
			Context("url returns 404", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", inputs.DependencyURI,
						httpmock.NewStringResponder(404, string(inputs.ExpectedContent)))
				})
				It("raises error", func() {
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(err).ToNot(BeNil())
				})

				It("alerts the user that the url could not be downloaded", func() {
					Expect(inputs.Dependency.Name).To(Equal("thing"))
					err = manifest.FetchDependency(inputs.Dependency, outputFile)
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("could not download: 404"))
					Expect(buffer.String()).ToNot(ContainSubstring("to ["))
				})

				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
				inputs.CheckOnError()
			})

			Context("url exists but does not match checksum", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", inputs.DependencyURI,
						httpmock.NewStringResponder(200, string(append(inputs.ExpectedContent, []byte("other data")...))))
				})
				It("raises error", func() {
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(err).ToNot(BeNil())
				})
				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(inputs.Dependency, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
				inputs.CheckOnError()
			})
		}

		Context("uncached", func() {
			inputs := DownloadingTestInputs{}
			inputs.CheckOnSuccess = func() {}
			inputs.CheckOnError = func() {}
			BeforeEach(func() {
				entryToFetch.entry.File = "" // not cached in buildpack
				manifestForTest := libbuildpack.Manifest{
					LanguageString:  "sample",
					ManifestEntries: allEntries,
				}
				y := libbuildpack.NewYAML()
				Expect(y.Write(filepath.Join(manifestDir, "manifest.yml"), manifestForTest)).To(Succeed())

				inputs.Dependency = entryToFetch.entry.Dependency
				inputs.DependencyURI = entryToFetch.entry.URI
				inputs.ExpectedContent = entryToFetch.content
			})

			BehaviorWhenDownloading(&inputs)

		})

		Context("app cached", func() {
			var (
				err             error
				manifestForTest libbuildpack.Manifest
			)
			inputs := DownloadingTestInputs{}

			BeforeEach(func() {
				entryToFetch.entry.File = "" // not cached in buildpack
				manifestForTest = libbuildpack.Manifest{
					LanguageString:  "sample",
					ManifestEntries: []libbuildpack.ManifestEntry{entryToFetch.entry},
				}
				y := libbuildpack.NewYAML()
				Expect(y.Write(filepath.Join(manifestDir, "manifest.yml"), manifestForTest)).To(Succeed())

				inputs.Dependency = entryToFetch.entry.Dependency
				inputs.DependencyURI = entryToFetch.entry.URI
				inputs.ExpectedContent = entryToFetch.content
			})
			JustBeforeEach(func() {
				Expect(manifest.SetAppCacheDir(appCacheDir)).To(Succeed())
			})

			Context("when there is no cached file", func() {
				checkOnSuccess := func() {
					It("downloads the file to the cache location", func() {
						Expect(manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)).To(Succeed())

						Expect(ioutil.ReadFile(entryToFetch.appCachePath)).To(Equal(entryToFetch.content))
					})
				}
				checkOnError := func() {
					It("cached file does not exist", func() {
						err = manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)

						Expect(entryToFetch.appCachePath).ToNot(BeAnExistingFile())
					})
				}

				inputs.CheckOnSuccess = checkOnSuccess
				inputs.CheckOnError = checkOnError

				BehaviorWhenDownloading(&inputs)
			})

			Context("when there are other files in the app cache", func() {
				var extraFilePaths []string

				BeforeEach(func() {
					extraFilePaths = []string{}

					// create file in app cache dir
					extraFile := filepath.Join(appCacheDir, "dependencies", "abcdef0123456789", "decoyFile")
					Expect(os.MkdirAll(filepath.Dir(extraFile), 0755)).To(Succeed())
					Expect(ioutil.WriteFile(extraFile, []byte("decoy content"), 0644)).To(Succeed())
					extraFilePaths = append(extraFilePaths, extraFile)

					// create file for real dependency in manifest
					extraOtherDepFile := filepath.Join(appCacheDir, "dependencies", "662eacac1df6ae7eee9ccd1ac1eb1d0d8777c403e5375fd64d14907f875f50c0", "some-dependency-name-5.tgz")
					os.MkdirAll(filepath.Dir(extraOtherDepFile), 0755)
					Expect(ioutil.WriteFile(extraOtherDepFile, []byte("some super legit dependency content"), 0644)).To(Succeed())
					extraFilePaths = append(extraFilePaths, extraOtherDepFile)

					// create extra file for the fetched dependency
					extraDepFile := filepath.Join(filepath.Dir(entryToFetch.appCachePath), "decoyDep.zip")
					os.MkdirAll(filepath.Dir(extraDepFile), 0755)
					Expect(ioutil.WriteFile(extraDepFile, []byte("some more decoy content"), 0644)).To(Succeed())
					extraFilePaths = append(extraFilePaths, extraDepFile)

					// Add extra dependency to manifest & rewrite that file
					manifestForTest.ManifestEntries = append(manifestForTest.ManifestEntries,
						libbuildpack.ManifestEntry{
							Dependency: libbuildpack.Dependency{
								Name:    "some-dependency-name",
								Version: "5"},
							URI:      "http://www.example.com/some/dependency/uri/some-dependency-name-5.tgz",
							SHA256:   "shaofcontent",
							CFStacks: []string{"cflinuxfs2"},
						})
					y := libbuildpack.NewYAML()
					Expect(y.Write(filepath.Join(manifestDir, "manifest.yml"), manifestForTest)).To(Succeed())

					inputs.Dependency = entryToFetch.entry.Dependency
					inputs.DependencyURI = entryToFetch.entry.URI
					inputs.ExpectedContent = entryToFetch.content
				})

				checkOnSuccess := func() {
					It("downloads the file to the cache location", func() {
						Expect(manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)).To(Succeed())

						Expect(ioutil.ReadFile(entryToFetch.appCachePath)).To(Equal(entryToFetch.content))
					})
					It("everything else is deleted", func() {
						Expect(manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)).To(Succeed())
						Expect(manifest.CleanupAppCache()).To(Succeed())

						for _, extraFilePath := range extraFilePaths {
							Expect(extraFilePath).ToNot(BeAnExistingFile())
						}
					})
				}
				checkOnError := func() {
					It("cached file does not exist", func() {
						Expect(manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)).ToNot(Succeed())

						Expect(entryToFetch.appCachePath).ToNot(BeAnExistingFile())
					})
					It("other files remain", func() {
						Expect(manifest.FetchDependency(entryToFetch.entry.Dependency, outputFile)).ToNot(Succeed())

						for _, extraFilePath := range extraFilePaths {
							Expect(extraFilePath).To(BeAnExistingFile())
						}
					})
				}
				inputs.CheckOnError = checkOnError
				inputs.CheckOnSuccess = checkOnSuccess

				BehaviorWhenDownloading(&inputs)
			})
			Context("when file is in the app cache", func() {
				cachedInputs := CachedTestInputs{}
				BeforeEach(func() {
					cachedInputs.pathToCachedFile = entryToFetch.appCachePath
					cachedInputs.dependency = entryToFetch.entry.Dependency
					cachedInputs.expectedContents = entryToFetch.content
				})

				usingCachedFileTests(&cachedInputs)
			})

		})
		Context("buildpack cached", func() {
			cachedInputs := CachedTestInputs{}

			BeforeEach(func() {
				dependenciesDir := filepath.Join(manifestDir, "dependencies")
				os.MkdirAll(dependenciesDir, 0755)

				entryToFetch.entry.File = "dependencies/c4fef5682adf1c19c7f9b76fde9d0ecb/thing-1-linux-x64.tgz"
				manifestForTest := libbuildpack.Manifest{
					LanguageString:  "sample",
					ManifestEntries: []libbuildpack.ManifestEntry{entryToFetch.entry},
				}
				y := libbuildpack.NewYAML()
				Expect(y.Write(filepath.Join(manifestDir, "manifest.yml"), manifestForTest)).To(Succeed())

				outputFile = filepath.Join(tmpdir, "out.tgz")
				cachedInputs.pathToCachedFile = filepath.Join(manifestDir, entryToFetch.entry.File)
				cachedInputs.dependency = entryToFetch.entry.Dependency
				cachedInputs.expectedContents = entryToFetch.content
			})

			usingCachedFileTests(&cachedInputs)
		})
	})

	Describe("CleanupAppCache", func() {
		var (
			appCacheDir string
		)

		BeforeEach(func() {
			appCacheDir, err = ioutil.TempDir("", "appCache")
			Expect(err).To(BeNil())
		})
		JustBeforeEach(func() {
			Expect(manifest.SetAppCacheDir(appCacheDir)).To(Succeed())
		})

		Context("no dependencies were cached", func() {
			BeforeEach(func() {
				Expect(filepath.Join(appCacheDir, "dependencies")).ToNot(BeADirectory())
			})
			It("does nothing and succeeds", func() {
				Expect(manifest.CleanupAppCache()).To(Succeed())
			})
		})

		Context("dependencies were cached", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(filepath.Join(appCacheDir, "dependencies"), 0755)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(appCacheDir, "dependencies", "abcd"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(appCacheDir, "dependencies", "abcd", "file.tgz"), []byte("contents"), 0644)).To(Succeed())
			})
			It("deletes old files", func() {
				Expect(filepath.Join(appCacheDir, "dependencies", "abcd", "file.tgz")).To(BeARegularFile())

				Expect(manifest.CleanupAppCache()).To(Succeed())

				Expect(filepath.Join(appCacheDir, "dependencies", "abcd", "file.tgz")).ToNot(BeARegularFile())
			})
		})
	})

	Describe("InstallDependency", func() {
		var outputDir string

		BeforeEach(func() {
			outputDir, err = ioutil.TempDir("", "downloads")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.RemoveAll(outputDir)
			Expect(err).To(BeNil())
		})

		Context("uncached", func() {
			BeforeEach(func() {
				manifestDir = "fixtures/manifest/fetch"
			})
			Context("url exists and matches sha256", func() {
				BeforeEach(func() {
					tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
					Expect(err).To(BeNil())
					httpmock.RegisterResponder("GET", "https://example.com/dependencies/real_tar_file-3-linux-x64.tgz",
						httpmock.NewStringResponder(200, string(tgzContents)))
				})

				It("logs the name and version of the dependency", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_tar_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("-----> Installing real_tar_file 3"))
				})

				It("extracts a file at the root", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_tar_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "root.txt")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "root.txt"))).To(Equal([]byte("root\n")))
				})

				It("extracts a nested file", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_tar_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
				})

				It("makes intermediate directories", func() {
					outputDir = filepath.Join(outputDir, "notexist")
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_tar_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
				})

				Context("version is NOT latest in version line", func() {
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-6.2.2-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					It("warns the user", func() {
						patchWarning := "**WARNING** A newer version of thing is available in this buildpack. " +
							"Please adjust your app to use version 6.2.3 instead of version 6.2.2 as soon as possible. " +
							"Old versions of thing are only provided to assist in migrating to newer versions.\n"

						err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "6.2.2"}, outputDir)
						Expect(err).To(BeNil())
						Expect(buffer.String()).To(ContainSubstring(patchWarning))
					})
				})

				Context("version is latest in version line", func() {
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-6.2.3-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					It("does not warn the user", func() {
						err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "6.2.3"}, outputDir)
						Expect(err).To(BeNil())
						Expect(buffer.String()).NotTo(ContainSubstring("newer version"))
					})
				})

				Context("version is not semver", func() {
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://buildpacks.cloudfoundry.org/dependencies/godep/godep-v79-linux-x64-9e37ce0f.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					It("does not warn the user", func() {
						err = manifest.InstallDependency(libbuildpack.Dependency{Name: "godep", Version: "v79"}, outputDir)
						Expect(err).To(BeNil())
						Expect(buffer.String()).NotTo(ContainSubstring("newer version"))
					})
				})

				Context("version has an EOL, version line is major", func() {
					const warning = "**WARNING** thing 4.x will no longer be available in new buildpacks released after 2017-03-01."
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-4.6.1-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					Context("less than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2017-02-15")
							Expect(err).To(BeNil())
						})

						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "4.6.1"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})

						Context("dependency EOL has a link associated with it", func() {
							It("includes the link in the warning", func() {
								err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "4.6.1"}, outputDir)
								Expect(err).To(BeNil())
								Expect(buffer.String()).To(ContainSubstring("See: http://example.com/eol-policy"))
							})
						})

						Context("dependency EOL does not have a link associated with it", func() {
							BeforeEach(func() {
								tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
								Expect(err).To(BeNil())
								httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-5.2.3-linux-x64.tgz",
									httpmock.NewStringResponder(200, string(tgzContents)))
							})

							It("does not include the word 'See:' in the warning", func() {
								err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "5.2.3"}, outputDir)
								Expect(err).To(BeNil())
								Expect(buffer.String()).ToNot(ContainSubstring("See:"))
							})
						})
					})
					Context("in the past", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2017-12-15")
							Expect(err).To(BeNil())
						})
						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "4.6.1"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})
					})
					Context("more than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2016-10-15")
							Expect(err).To(BeNil())
						})
						It("does not warn the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "4.6.1"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).ToNot(ContainSubstring(warning))
						})
					})
				})

				Context("version has an EOL, version line is major + minor", func() {
					const warning = "**WARNING** thing 6.2.x will no longer be available in new buildpacks released after 2018-04-01"
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-6.2.3-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					Context("less than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2018-03-29")
							Expect(err).To(BeNil())
						})
						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "6.2.3"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})
					})
					Context("in the past", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2019-12-30")
							Expect(err).To(BeNil())
						})
						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "6.2.3"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})
					})
					Context("more than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2018-01-15")
							Expect(err).To(BeNil())
						})
						It("does not warn the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "6.2.3"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).ToNot(ContainSubstring(warning))
						})
					})
				})

				Context("version has an EOL, version line non semver", func() {
					const warning = "**WARNING** nonsemver abc-1.2.3-def-4.5.6 will no longer be available in new buildpacks released after 2018-04-01"
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/nonsemver-abc-1.2.3-def-4.5.6-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					Context("less than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2018-03-29")
							Expect(err).To(BeNil())
						})
						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "nonsemver", Version: "abc-1.2.3-def-4.5.6"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})
					})
					Context("in the past", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2019-12-30")
							Expect(err).To(BeNil())
						})
						It("warns the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "nonsemver", Version: "abc-1.2.3-def-4.5.6"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).To(ContainSubstring(warning))
						})
					})
					Context("more than 30 days in the future", func() {
						BeforeEach(func() {
							currentTime, err = time.Parse("2006-01-02", "2018-01-15")
							Expect(err).To(BeNil())
						})
						It("does not warn the user", func() {
							err = manifest.InstallDependency(libbuildpack.Dependency{Name: "nonsemver", Version: "abc-1.2.3-def-4.5.6"}, outputDir)
							Expect(err).To(BeNil())
							Expect(buffer.String()).ToNot(ContainSubstring(warning))
						})
					})
				})

				Context("version does not have an EOL", func() {
					const warning = "**WARNING** real_tar_file 3 will no longer be available in new buildpacks released after"
					BeforeEach(func() {
						tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
						Expect(err).To(BeNil())
						httpmock.RegisterResponder("GET", "https://example.com/dependencies/real_tar_file-3-linux-x64.tgz",
							httpmock.NewStringResponder(200, string(tgzContents)))
					})

					It("does not warn the user", func() {
						err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_tar_file", Version: "3"}, outputDir)
						Expect(err).To(BeNil())
						Expect(buffer.String()).ToNot(ContainSubstring(warning))
					})
				})
			})

			Context("url exists but does not match sha256", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-1-linux-x64.tgz",
						httpmock.NewStringResponder(200, "other data"))
				})

				It("logs the name and version of the dependency", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputDir)
					Expect(err).ToNot(BeNil())

					Expect(buffer.String()).To(ContainSubstring("-----> Installing thing 1"))
				})

				It("outputfile does not exist", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputDir)
					Expect(err).ToNot(BeNil())

					Expect(filepath.Join(outputDir, "root.txt")).ToNot(BeAnExistingFile())
				})
			})
		})

		Context("cached", func() {
			var (
				dependenciesDir string
				outputDir       string
			)

			BeforeEach(func() {
				manifestDir, err = ioutil.TempDir("", "cached")
				Expect(err).To(BeNil())

				dependenciesDir = filepath.Join(manifestDir, "dependencies")
				os.MkdirAll(dependenciesDir, 0755)

				data, err := ioutil.ReadFile("fixtures/manifest/fetch_cached/manifest.yml")
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(manifestDir, "manifest.yml"), data, 0644)
				Expect(err).To(BeNil())

				outputDir, err = ioutil.TempDir("", "downloads")
				Expect(err).To(BeNil())
			})

			Context("url exists cached on disk and matches sha256", func() {
				BeforeEach(func() {
					libbuildpack.CopyFile("fixtures/thing.zip", filepath.Join(dependenciesDir, "f666296d630cce4c94c62afcc6680b44", "real_zip_file-3-linux-x64.zip"))
				})

				It("logs the name and version of the dependency", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_zip_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(buffer.String()).To(ContainSubstring("-----> Installing real_zip_file 3"))
				})

				It("extracts a file at the root", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_zip_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "root.txt")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "root.txt"))).To(Equal([]byte("root\n")))
				})

				It("extracts a nested file", func() {
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_zip_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
				})

				It("makes intermediate directories", func() {
					outputDir = filepath.Join(outputDir, "notexist")
					err = manifest.InstallDependency(libbuildpack.Dependency{Name: "real_zip_file", Version: "3"}, outputDir)
					Expect(err).To(BeNil())

					Expect(filepath.Join(outputDir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
					Expect(ioutil.ReadFile(filepath.Join(outputDir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
				})
			})
		})
	})

	Describe("InstallOnlyVersion", func() {
		var outputDir string

		BeforeEach(func() {
			manifestDir = "fixtures/manifest/fetch"
			outputDir, err = ioutil.TempDir("", "downloads")
			Expect(err).To(BeNil())
		})
		AfterEach(func() { err = os.RemoveAll(outputDir); Expect(err).To(BeNil()) })

		Context("there is only one version of the dependency", func() {
			BeforeEach(func() {
				tgzContents, err := ioutil.ReadFile("fixtures/thing.tgz")
				Expect(err).To(BeNil())
				httpmock.RegisterResponder("GET", "https://example.com/dependencies/real_tar_file-3-linux-x64.tgz",
					httpmock.NewStringResponder(200, string(tgzContents)))
			})

			It("installs", func() {
				outputDir = filepath.Join(outputDir, "notexist")
				err = manifest.InstallOnlyVersion("real_tar_file", outputDir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(outputDir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(outputDir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
			})
		})

		Context("there is more than one version of the dependency", func() {
			It("fails", func() {
				outputDir = filepath.Join(outputDir, "notexist")
				err = manifest.InstallOnlyVersion("thing", outputDir)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("more than one version of thing found"))
			})
		})

		Context("there are no versions of the dependency", func() {
			It("fails", func() {
				outputDir = filepath.Join(outputDir, "notexist")
				err = manifest.InstallOnlyVersion("not_a_dependency", outputDir)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("no versions of not_a_dependency found"))
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
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("found 2 default versions for bower"))
			})
		})

		Context("requested name does not exist", func() {
			It("returns an error", func() {
				_, err := manifest.DefaultVersion("notexist")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("no default version for notexist"))
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
})
