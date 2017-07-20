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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("Manifest", func() {
	var (
		manifest    *libbuildpack.Manifest
		manifestDir string
		err         error
		version     string
		currentTime time.Time
		buffer      *bytes.Buffer
		logger      *libbuildpack.Logger
	)

	BeforeEach(func() {
		manifestDir = "fixtures/manifest/standard"
		currentTime = time.Now()
		httpmock.Reset()

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))
	})

	JustBeforeEach(func() {
		manifest, err = libbuildpack.NewManifest(manifestDir, logger, currentTime)
		Expect(err).To(BeNil())
	})

	Describe("NewManifest", func() {
		It("has a language", func() {
			Expect(manifest.Language()).To(Equal("dotnet-core"))
		})
	})

	Describe("CheckStackSupport", func() {
		var (
			oldCfStack string
		)

		BeforeEach(func() { oldCfStack = os.Getenv("CF_STACK") })
		AfterEach(func() { err = os.Setenv("CF_STACK", oldCfStack); Expect(err).To(BeNil()) })

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
			Expect(err).To(BeNil())

			Expect(versions).To(Equal([]string{"1.0.0", "1.0.1", "1.0.3", "1.1.0"}))
		})
	})

	Describe("FetchDependency", func() {
		var tmpdir, outputFile string

		BeforeEach(func() {
			manifestDir = "fixtures/manifest/fetch"
			tmpdir, err = ioutil.TempDir("", "downloads")
			Expect(err).To(BeNil())
			outputFile = filepath.Join(tmpdir, "out.tgz")
		})
		AfterEach(func() { err = os.RemoveAll(tmpdir); Expect(err).To(BeNil()) })

		Context("uncached", func() {
			Context("url exists and matches md5", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-1-linux-x64.tgz",
						httpmock.NewStringResponder(200, "exciting binary data"))
				})

				It("downloads the file to the requested location", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("exciting binary data")))
				})

				It("makes intermediate directories", func() {
					outputFile = filepath.Join(tmpdir, "notexist", "out.tgz")
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("exciting binary data")))
				})
			})

			Context("url returns 404", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-1-linux-x64.tgz",
						httpmock.NewStringResponder(404, "exciting binary data"))
				})
				It("raises error", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(err).ToNot(BeNil())
				})

				It("alerts the user that the url could not be downloaded", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("could not download: 404"))
					Expect(buffer.String()).ToNot(ContainSubstring("to ["))
				})

				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
			})

			Context("url exists but does not match md5", func() {
				BeforeEach(func() {
					httpmock.RegisterResponder("GET", "https://example.com/dependencies/thing-1-linux-x64.tgz",
						httpmock.NewStringResponder(200, "other data"))
				})
				It("raises error", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(err).ToNot(BeNil())
				})
				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "1"}, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
			})

		})

		Context("cached", func() {
			var dependenciesDir string

			BeforeEach(func() {
				var err error
				manifestDir, err = ioutil.TempDir("", "cached")
				Expect(err).To(BeNil())

				dependenciesDir = filepath.Join(manifestDir, "dependencies")
				os.MkdirAll(dependenciesDir, 0755)

				data, err := ioutil.ReadFile("fixtures/manifest/fetch/manifest.yml")
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(manifestDir, "manifest.yml"), data, 0644)
				Expect(err).To(BeNil())

				outputFile = filepath.Join(tmpdir, "out.tgz")
			})

			Context("url exists cached on disk under old format and matches md5", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(dependenciesDir, "https___example.com_dependencies_thing-2-linux-x64.tgz"), []byte("awesome binary data"), 0644)
				})
				It("copies the cached file to outputFile", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("awesome binary data")))
				})
				It("makes intermediate directories", func() {
					outputFile = filepath.Join(tmpdir, "notexist", "out.tgz")
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("awesome binary data")))
				})
			})

			Context("url exists cached on disk under new format and matches md5", func() {
				BeforeEach(func() {
					os.MkdirAll(filepath.Join(dependenciesDir, "c4fef5682adf1c19c7f9b76fde9d0ecb"), 0755)
					Expect(ioutil.WriteFile(filepath.Join(dependenciesDir, "c4fef5682adf1c19c7f9b76fde9d0ecb", "thing-2-linux-x64.tgz"), []byte("awesome binary data"), 0644)).To(Succeed())
				})
				It("copies the cached file to outputFile", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("awesome binary data")))
				})
				It("makes intermediate directories", func() {
					outputFile = filepath.Join(tmpdir, "notexist", "out.tgz")
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).To(BeNil())
					Expect(ioutil.ReadFile(outputFile)).To(Equal([]byte("awesome binary data")))
				})
			})

			Context("url exists cached on disk under old format and does not match md5", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(dependenciesDir, "https___example.com_dependencies_thing-2-linux-x64.tgz"), []byte("different binary data"), 0644)
				})
				It("raises error", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).ToNot(BeNil())
				})
				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
			})

			Context("url exists cached on disk under new format and does not match md5", func() {
				BeforeEach(func() {
					os.MkdirAll(filepath.Join(dependenciesDir, "c4fef5682adf1c19c7f9b76fde9d0ecb"), 0755)
					Expect(ioutil.WriteFile(filepath.Join(dependenciesDir, "c4fef5682adf1c19c7f9b76fde9d0ecb", "thing-2-linux-x64.tgz"), []byte("different binary data"), 0644)).To(Succeed())
				})
				It("raises error", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).ToNot(BeNil())
				})
				It("outputfile does not exist", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(outputFile).ToNot(BeAnExistingFile())
				})
			})

			Context("url is not cached on disk", func() {
				It("raises error", func() {
					err = manifest.FetchDependency(libbuildpack.Dependency{Name: "thing", Version: "2"}, outputFile)

					Expect(err).ToNot(BeNil())
				})
			})
		})
	})

	Describe("InstallDependency", func() {
		var outputDir string

		BeforeEach(func() {
			manifestDir = "fixtures/manifest/fetch"
			outputDir, err = ioutil.TempDir("", "downloads")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.RemoveAll(outputDir)
			Expect(err).To(BeNil())
		})

		Context("uncached", func() {
			Context("url exists and matches md5", func() {
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

			Context("url exists but does not match md5", func() {
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

				data, err := ioutil.ReadFile("fixtures/manifest/fetch/manifest.yml")
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(manifestDir, "manifest.yml"), data, 0644)
				Expect(err).To(BeNil())

				outputDir, err = ioutil.TempDir("", "downloads")
				Expect(err).To(BeNil())
			})

			Context("url exists cached on disk and matches md5", func() {
				BeforeEach(func() {
					libbuildpack.CopyFile("fixtures/thing.zip", filepath.Join(dependenciesDir, "https___example.com_dependencies_real_zip_file-3-linux-x64.zip"))
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
