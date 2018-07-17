package libbuildpack_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stager", func() {
	var (
		manifest    *libbuildpack.Manifest
		buildDir    string
		cacheDir    string
		depsDir     string
		depsIdx     string
		profileDir  string
		logger      *libbuildpack.Logger
		s           *libbuildpack.Stager
		err         error
		oldCfStack  string
		buffer      *bytes.Buffer
		manifestDir string
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "build")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "cache")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "deps")
		Expect(err).To(BeNil())

		depsIdx = "0"
		err = os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)
		Expect(err).To(BeNil())

		profileDir, err = ioutil.TempDir("", "profiled")
		Expect(err).To(BeNil())

		manifestDir = filepath.Join("fixtures", "manifest", "standard")

		manifest, err = libbuildpack.NewManifest(manifestDir, logger, time.Now())
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(buffer)

		s = libbuildpack.NewStager([]string{buildDir, cacheDir, depsDir, depsIdx, profileDir}, logger, manifest)
	})

	AfterEach(func() {
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(profileDir)
		Expect(err).To(BeNil())
	})

	Describe("NewStager", func() {
		var args []string

		Context("A deps dir is provided", func() {
			It("sets it in the Stager struct", func() {
				args = []string{"buildDir", "cacheDir", "depsDir", "idx"}
				s = libbuildpack.NewStager(args, logger, manifest)
				Expect(err).To(BeNil())
				Expect(s.BuildDir()).To(Equal("buildDir"))
				Expect(s.CacheDir()).To(Equal("cacheDir"))
				Expect(s.DepsIdx()).To(Equal("idx"))
				Expect(s.DepDir()).To(Equal("depsDir/idx"))
				Expect(s.ProfileDir()).To(Equal("buildDir/.profile.d"))
			})
		})

		Context("A deps dir is not provided", func() {
			It("sets DepsDir to the empty string", func() {
				args = []string{"buildDir", "cacheDir"}
				s = libbuildpack.NewStager(args, logger, manifest)
				Expect(err).To(BeNil())
				Expect(s.BuildDir()).To(Equal("buildDir"))
				Expect(s.CacheDir()).To(Equal("cacheDir"))
				Expect(s.DepsIdx()).To(Equal(""))
				Expect(s.DepDir()).To(Equal(""))
				Expect(s.ProfileDir()).To(Equal("buildDir/.profile.d"))
			})
		})

		Context("A profile.d dir is provided", func() {
			It("sets ProfileDir", func() {
				args = []string{"buildDir", "cacheDir", "depsDir", "idx", "rootProfileD"}
				s = libbuildpack.NewStager(args, logger, manifest)
				Expect(err).To(BeNil())
				Expect(s.ProfileDir()).To(Equal("rootProfileD"))
			})
		})
	})

	Describe("WriteConfigYml", func() {
		It("creates a file in the <depDir>/idx directory", func() {
			Expect(s.WriteConfigYml(nil)).To(Succeed())

			config := struct {
				Name string `yaml:"name"`
			}{}
			Expect(libbuildpack.NewYAML().Load(filepath.Join(s.DepDir(), "config.yml"), &config)).To(Succeed())

			Expect(config.Name).To(Equal("dotnet-core"))
		})

		It("sets buildpack version in file", func() {
			Expect(s.WriteConfigYml(nil)).To(Succeed())

			config := struct {
				Version string `yaml:"version"`
			}{}
			libbuildpack.NewYAML().Load(filepath.Join(s.DepDir(), "config.yml"), &config)

			Expect(config.Version).To(Equal("99.99"))
		})

		It("writes passed config struct to file", func() {
			Expect(s.WriteConfigYml(map[string]string{"key": "value", "a": "b"})).To(Succeed())

			config := struct {
				Config map[string]string `yaml:"config"`
			}{}
			Expect(libbuildpack.NewYAML().Load(filepath.Join(s.DepDir(), "config.yml"), &config)).To(Succeed())

			Expect(config.Config).To(Equal(map[string]string{
				"key": "value",
				"a":   "b",
			}))
		})
	})

	Describe("CheckBuildpackValid", func() {
		BeforeEach(func() {
			oldCfStack = os.Getenv("CF_STACK")
			err = os.Setenv("CF_STACK", "cflinuxfs2")
			Expect(err).To(BeNil())
		})

		Context("buildpack is valid", func() {
			It("it logs the buildpack name and version", func() {
				err := s.CheckBuildpackValid()
				Expect(err).To(BeNil())
				Expect(buffer.String()).To(ContainSubstring("-----> Dotnet-Core Buildpack version 99.99"))
			})
		})
	})

	Describe("ClearCache", func() {
		Context("already empty", func() {
			It("returns successfully", func() {
				err = s.ClearCache()
				Expect(err).To(BeNil())
				Expect(cacheDir).To(BeADirectory())
			})
		})

		Context("cache dir does not exist", func() {
			BeforeEach(func() {
				cacheDir = filepath.Join("not", "real")
			})

			It("returns successfully", func() {
				err = s.ClearCache()
				Expect(err).To(BeNil())
				Expect(cacheDir).ToNot(BeADirectory())
			})
		})

		Context("not empty", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(cacheDir, "fred", "jane"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "fred", "jane", "jack.txt"), []byte("content"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "jill.txt"), []byte("content"), 0644)).To(Succeed())

				fi, err := ioutil.ReadDir(cacheDir)
				Expect(err).To(BeNil())
				Expect(len(fi)).To(Equal(2))
			})

			It("it clears the cache", func() {
				err = s.ClearCache()
				Expect(err).To(BeNil())
				Expect(cacheDir).To(BeADirectory())

				fi, err := ioutil.ReadDir(cacheDir)
				Expect(err).To(BeNil())
				Expect(len(fi)).To(Equal(0))
			})
		})
	})

	Describe("ClearDepDir", func() {
		Context("already empty", func() {
			It("returns successfully", func() {
				err = s.ClearDepDir()
				Expect(err).To(BeNil())
				Expect(s.DepDir()).To(BeADirectory())
			})
		})

		Context("not empty", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(depsDir, depsIdx, "fred", "jane"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(depsDir, depsIdx, "fred", "jane", "jack.txt"), []byte("content"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(depsDir, depsIdx, "jill.txt"), []byte("content"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(depsDir, depsIdx, "config.yml"), []byte("yaml"), 0644)).To(Succeed())

				fi, err := ioutil.ReadDir(filepath.Join(depsDir, depsIdx))
				Expect(err).To(BeNil())
				Expect(len(fi)).To(Equal(3))
			})

			It("it clears the depDir, leaving config.yml", func() {
				err = s.ClearDepDir()
				Expect(err).To(BeNil())
				Expect(s.DepDir()).To(BeADirectory())

				fi, err := ioutil.ReadDir(s.DepDir())
				Expect(err).To(BeNil())
				Expect(len(fi)).To(Equal(1))

				content, err := ioutil.ReadFile(filepath.Join(s.DepDir(), "config.yml"))
				Expect(err).To(BeNil())
				Expect(string(content)).To(Equal("yaml"))
			})
		})
	})

	Describe("WriteEnvFile", func() {
		It("creates a file in the <depDir>/env directory", func() {
			err := s.WriteEnvFile("ENVVAR", "value")
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(s.DepDir(), "env", "ENVVAR"))
			Expect(err).To(BeNil())

			Expect(string(contents)).To(Equal("value"))
		})
	})

	Describe("AddBinDependencyLink", func() {
		It("creates a symlink <depDir>/bin/<name> with the relative path to dest", func() {
			err := s.AddBinDependencyLink(filepath.Join(depsDir, depsIdx, "some", "long", "path"), "dep")
			Expect(err).To(BeNil())

			link, err := os.Readlink(filepath.Join(s.DepDir(), "bin", "dep"))
			Expect(err).To(BeNil())

			Expect(link).To(Equal("../some/long/path"))
		})
	})

	Describe("LinkDirectoryInDepDir", func() {
		var destDir string

		BeforeEach(func() {
			destDir, err = ioutil.TempDir("", "untarred-dependencies")
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(destDir, "thing1"), []byte("xxx"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(destDir, "thing2"), []byte("yyy"), 0644)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.RemoveAll(destDir)
			Expect(err).To(BeNil())
		})

		It("it creates a symlink <depDir>/<depSubDir>/<name> pointing to each file in dest dir", func() {
			err := s.LinkDirectoryInDepDir(destDir, "include")
			Expect(err).To(BeNil())

			link, err := os.Readlink(filepath.Join(s.DepDir(), "include", "thing1"))
			Expect(err).To(BeNil())
			Expect(link).To(Equal("../../../" + path.Base(destDir) + "/thing1"))

			data, err := ioutil.ReadFile(filepath.Join(s.DepDir(), "include", "thing1"))
			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal("xxx"))

			link, err = os.Readlink(filepath.Join(s.DepDir(), "include", "thing2"))
			Expect(err).To(BeNil())
			Expect(link).To(Equal("../../../" + path.Base(destDir) + "/thing2"))

			data, err = ioutil.ReadFile(filepath.Join(s.DepDir(), "include", "thing2"))
			Expect(err).To(BeNil())
			Expect(string(data)).To(Equal("yyy"))
		})

		It("overwrites existing links", func() {
			Expect(os.MkdirAll(filepath.Join(s.DepDir(), "include"), 0755)).To(Succeed())
			Expect(os.Symlink(filepath.Join(destDir, "thing2"), filepath.Join(s.DepDir(), "include", "thing1"))).To(Succeed())

			err := s.LinkDirectoryInDepDir(destDir, "include")
			Expect(err).To(BeNil())

			link, err := os.Readlink(filepath.Join(s.DepDir(), "include", "thing1"))
			Expect(err).To(BeNil())
			Expect(link).To(Equal("../../../" + path.Base(destDir) + "/thing1"))
		})
	})

	Describe("WriteProfileD", func() {
		var (
			info           os.FileInfo
			profileDScript string
			name           string
			contents       string
		)

		JustBeforeEach(func() {
			profileDScript = filepath.Join(s.DepDir(), "profile.d", name)

			err = s.WriteProfileD(name, contents)
			Expect(err).To(BeNil())
		})

		Context("profile.d directory exists", func() {
			BeforeEach(func() {
				name = "dir-exists.sh"
				contents = "used the dir"

				err = os.MkdirAll(filepath.Join(depsDir, depsIdx, "profile.d"), 0755)
				Expect(err).To(BeNil())
			})

			It("creates the file as an executable", func() {
				Expect(profileDScript).To(BeAnExistingFile())

				info, err = os.Stat(profileDScript)
				Expect(err).To(BeNil())

				// make sure at least 1 executable bit is set
				Expect(info.Mode().Perm() & 0111).NotTo(Equal(os.FileMode(0000)))
			})

			It("the script has the correct contents", func() {
				data, err := ioutil.ReadFile(profileDScript)
				Expect(err).To(BeNil())

				Expect(data).To(Equal([]byte("used the dir")))
			})
		})

		Context("profile.d directory does not exist", func() {
			BeforeEach(func() {
				name = "no-dir.sh"
				contents = "made the dir"
			})

			It("creates the file as an executable", func() {
				Expect(profileDScript).To(BeAnExistingFile())

				info, err = os.Stat(profileDScript)
				Expect(err).To(BeNil())

				// make sure at least 1 executable bit is set
				Expect(info.Mode().Perm() & 0111).NotTo(Equal(0000))
			})
		})

		It("the script has the correct contents", func() {
			data, err := ioutil.ReadFile(profileDScript)
			Expect(err).To(BeNil())

			Expect(data).To(Equal([]byte("made the dir")))
		})
	})

	Describe("Supply Environment", func() {
		BeforeEach(func() {
			err = os.MkdirAll(filepath.Join(depsDir, "00", "bin"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "01", "bin"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "01", "lib"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "02", "lib"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "03", "include"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "04", "pkgconfig"), 0755)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "05", "env"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "05", "env", "ENV_VAR"), []byte("value"), 0644)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "00", "profile.d"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "00", "profile.d", "supplied-script.sh"), []byte("first"), 0644)
			Expect(err).To(BeNil())

			err = os.MkdirAll(filepath.Join(depsDir, "01", "profile.d"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "01", "profile.d", "supplied-script.sh"), []byte("second"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(depsDir, "some-file.yml"), []byte("things"), 0644)
			Expect(err).To(BeNil())
		})

		Describe("SetStagingEnvironment", func() {
			var envVars = map[string]string{}

			BeforeEach(func() {
				vars := []string{"PATH", "LD_LIBRARY_PATH", "LIBRARY_PATH", "CPATH", "PKG_CONFIG_PATH", "ENV_VAR"}

				for _, envVar := range vars {
					envVars[envVar] = os.Getenv(envVar)
					os.Setenv(envVar, "existing_"+envVar)
				}
			})

			AfterEach(func() {
				for key, val := range envVars {
					err = os.Setenv(key, val)
					Expect(err).To(BeNil())
				}
			})

			It("sets PATH based on the supplied deps", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("PATH")
				Expect(newPath).To(Equal(fmt.Sprintf("%s/01/bin:%s/00/bin:existing_PATH", depsDir, depsDir)))
			})

			It("sets LD_LIBRARY_PATH based on the supplied deps", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("LD_LIBRARY_PATH")
				Expect(newPath).To(Equal(fmt.Sprintf("%s/02/lib:%s/01/lib:existing_LD_LIBRARY_PATH", depsDir, depsDir)))
			})

			It("sets LIBRARY_PATH based on the supplied deps", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("LIBRARY_PATH")
				Expect(newPath).To(Equal(fmt.Sprintf("%s/02/lib:%s/01/lib:existing_LIBRARY_PATH", depsDir, depsDir)))
			})

			It("sets CPATH based on the supplied deps", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("CPATH")
				Expect(newPath).To(Equal(fmt.Sprintf("%s/03/include:existing_CPATH", depsDir)))
			})

			It("sets PKG_CONFIG_PATH based on the supplied deps", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("PKG_CONFIG_PATH")
				Expect(newPath).To(Equal(fmt.Sprintf("%s/04/pkgconfig:existing_PKG_CONFIG_PATH", depsDir)))
			})

			It("sets environment variables from the env/ dir", func() {
				err = s.SetStagingEnvironment()
				Expect(err).To(BeNil())

				newPath := os.Getenv("ENV_VAR")
				Expect(newPath).To(Equal("value"))
			})

			Context("relevant env variable is empty", func() {
				BeforeEach(func() {
					for key, _ := range envVars {
						os.Setenv(key, "")
					}
				})

				It("sets PATH based on the supplied deps", func() {
					err = s.SetStagingEnvironment()
					Expect(err).To(BeNil())

					newPath := os.Getenv("PATH")
					Expect(newPath).To(Equal(fmt.Sprintf("%s/01/bin:%s/00/bin", depsDir, depsDir)))
				})

				It("sets LD_LIBRARY_PATH based on the supplied deps", func() {
					err = s.SetStagingEnvironment()
					Expect(err).To(BeNil())

					newPath := os.Getenv("LD_LIBRARY_PATH")
					Expect(newPath).To(Equal(fmt.Sprintf("%s/02/lib:%s/01/lib", depsDir, depsDir)))
				})

				It("sets LIBRARY_PATH based on the supplied deps", func() {
					err = s.SetStagingEnvironment()
					Expect(err).To(BeNil())

					newPath := os.Getenv("LIBRARY_PATH")
					Expect(newPath).To(Equal(fmt.Sprintf("%s/02/lib:%s/01/lib", depsDir, depsDir)))
				})

				It("sets CPATH based on the supplied deps", func() {
					err = s.SetStagingEnvironment()
					Expect(err).To(BeNil())

					newPath := os.Getenv("CPATH")
					Expect(newPath).To(Equal(fmt.Sprintf("%s/03/include", depsDir)))
				})

				It("sets PKG_CONFIG_PATH based on the supplied deps", func() {
					err = s.SetStagingEnvironment()
					Expect(err).To(BeNil())

					newPath := os.Getenv("PKG_CONFIG_PATH")
					Expect(newPath).To(Equal(fmt.Sprintf("%s/04/pkgconfig", depsDir)))
				})
			})
		})

		Describe("SetLaunchEnvironment", func() {
			It("writes a .profile.d script allowing the runtime container to use the supplied deps", func() {
				err = s.SetLaunchEnvironment()
				Expect(err).To(BeNil())

				contents, err := ioutil.ReadFile(filepath.Join(profileDir, "000_multi-supply.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(ContainSubstring(`export PATH=$DEPS_DIR/01/bin:$DEPS_DIR/00/bin$([[ ! -z "${PATH:-}" ]] && echo ":$PATH")`))
				Expect(string(contents)).To(ContainSubstring(`export LD_LIBRARY_PATH=$DEPS_DIR/02/lib:$DEPS_DIR/01/lib$([[ ! -z "${LD_LIBRARY_PATH:-}" ]] && echo ":$LD_LIBRARY_PATH")`))
				Expect(string(contents)).To(ContainSubstring(`export LIBRARY_PATH=$DEPS_DIR/02/lib:$DEPS_DIR/01/lib$([[ ! -z "${LIBRARY_PATH:-}" ]] && echo ":$LIBRARY_PATH")`))
			})

			It("copies scripts from <deps-dir>/<idx>/profile.d to the .profile.d directory, prepending <idx>", func() {
				err = s.SetLaunchEnvironment()
				Expect(err).To(BeNil())

				contents, err := ioutil.ReadFile(filepath.Join(profileDir, "00_supplied-script.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("first"))

				contents, err = ioutil.ReadFile(filepath.Join(profileDir, "01_supplied-script.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("second"))
			})
		})
	})

})
