package main_test

import (
	c "compile"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"

	bp "github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=vendor/github.com/cloudfoundry/libbuildpack/yaml.go --destination=mocks_yaml_test.go --package=main_test
//go:generate mockgen -source=vendor/github.com/cloudfoundry/libbuildpack/manifest.go --destination=mocks_manifest_test.go --package=main_test --imports=.=github.com/cloudfoundry/libbuildpack

var _ = Describe("Compile", func() {
	var (
		sf           c.Staticfile
		err          error
		buildDir     string
		cacheDir     string
		compiler     *c.StaticfileCompiler
		logger       bp.Logger
		mockCtrl     *gomock.Controller
		mockYaml     *MockYAML
		mockManifest *MockManifest
		buffer       *bytes.Buffer
		data         []byte
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "staticfile-buildpack.build.")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "staticfile-buildpack.cache.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = bp.NewLogger()
		logger.SetOutput(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockYaml = NewMockYAML(mockCtrl)
		mockManifest = NewMockManifest(mockCtrl)
	})

	JustBeforeEach(func() {
		bpc := &bp.Compiler{BuildDir: buildDir,
			CacheDir: cacheDir,
			Manifest: mockManifest,
			Log:      logger}

		compiler = &c.StaticfileCompiler{Compiler: bpc,
			Config: sf,
			YAML:   mockYaml}
	})

	AfterEach(func() {
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())
	})

	Describe("LoadStaticfile", func() {
		Context("the staticfile does not exist", func() {
			BeforeEach(func() {
				mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Return(os.ErrNotExist)
			})
			It("does not return an error", func() {
				err = compiler.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			It("has default values", func() {
				err = compiler.LoadStaticfile()
				Expect(err).To(BeNil())
				Expect(compiler.Config.RootDir).To(Equal(""))
				Expect(compiler.Config.HostDotFiles).To(Equal(false))
				Expect(compiler.Config.LocationInclude).To(Equal(""))
				Expect(compiler.Config.DirectoryIndex).To(Equal(false))
				Expect(compiler.Config.SSI).To(Equal(false))
				Expect(compiler.Config.PushState).To(Equal(false))
				Expect(compiler.Config.HSTS).To(Equal(false))
				Expect(compiler.Config.ForceHTTPS).To(Equal(false))
				Expect(compiler.Config.BasicAuth).To(Equal(false))
			})

			It("does not log enabling statements", func() {
				Expect(buffer.String()).To(Equal(""))
			})
		})
		Context("the staticfile exists", func() {
			JustBeforeEach(func() {
				err = compiler.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			Context("and sets root", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["root"] = "root_test"
					})
				})
				It("sets RootDir", func() {
					Expect(compiler.Config.RootDir).To(Equal("root_test"))
				})
			})

			Context("and sets host_dot_files", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["host_dot_files"] = "true"
					})
				})
				It("sets HostDotFiles", func() {
					Expect(compiler.Config.HostDotFiles).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling hosting of dotfiles\n"))
				})
			})

			Context("and sets location_include", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["location_include"] = "a/b/c"
					})
				})
				It("sets location_include", func() {
					Expect(compiler.Config.LocationInclude).To(Equal("a/b/c"))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling location include file a/b/c\n"))
				})
			})

			Context("and sets directory", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["directory"] = "any_string"
					})
				})
				It("sets location_include", func() {
					Expect(compiler.Config.DirectoryIndex).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling directory index for folders without index.html files\n"))
				})
			})

			Context("and sets ssi", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["ssi"] = "enabled"
					})
				})
				It("sets ssi", func() {
					Expect(compiler.Config.SSI).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling SSI\n"))
				})
			})

			Context("and sets pushstate", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["pushstate"] = "enabled"
					})
				})
				It("sets pushstate", func() {
					Expect(compiler.Config.PushState).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling pushstate\n"))
				})
			})

			Context("and sets http_strict_transport_security", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["http_strict_transport_security"] = "true"
					})
				})
				It("sets pushstate", func() {
					Expect(compiler.Config.HSTS).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling HSTS\n"))
				})
			})

			Context("and sets force_https", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["force_https"] = "true"
					})
				})
				It("sets force_https", func() {
					Expect(compiler.Config.ForceHTTPS).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling HTTPS redirect\n"))
				})
			})
		})

		Context("Staticfile.auth is present", func() {
			BeforeEach(func() {
				mockYaml.EXPECT().Load(gomock.Any(), gomock.Any())

				err = ioutil.WriteFile(filepath.Join(buildDir, "Staticfile.auth"), []byte("some credentials"), 0644)
				Expect(err).To(BeNil())
			})

			JustBeforeEach(func() {
				err = compiler.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			It("sets BasicAuth", func() {
				Expect(compiler.Config.BasicAuth).To(Equal(true))
			})
			It("Logs", func() {
				Expect(buffer.String()).To(ContainSubstring("-----> Enabling basic authentication using Staticfile.auth\n"))
			})
		})

		Context("the staticfile exists and is not valid", func() {
			BeforeEach(func() {
				mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Return(errors.New("a yaml parsing error"))
			})

			It("returns an error", func() {
				err = compiler.LoadStaticfile()
				Expect(err).NotTo(BeNil())
			})
		})
	})

	Describe("GetAppRootDir", func() {
		var (
			returnDir string
		)

		JustBeforeEach(func() {
			returnDir, err = compiler.GetAppRootDir()
		})

		Context("the staticfile has a root directory specified", func() {
			Context("the directory does not exist", func() {
				BeforeEach(func() {
					sf.RootDir = "not_exist"
				})

				It("logs the staticfile's root directory", func() {
					Expect(buffer.String()).To(ContainSubstring("-----> Root folder"))
					Expect(buffer.String()).To(ContainSubstring("not_exist"))

				})

				It("returns an error", func() {
					Expect(returnDir).To(Equal(""))
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("the application Staticfile specifies a root directory"))
					Expect(err.Error()).To(ContainSubstring("that does not exist"))
				})
			})

			Context("the directory exists but is actually a file", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(buildDir, "actually_a_file"), []byte("xxx"), 0644)
					sf.RootDir = "actually_a_file"
				})

				It("logs the staticfile's root directory", func() {
					Expect(buffer.String()).To(ContainSubstring("-----> Root folder"))
					Expect(buffer.String()).To(ContainSubstring("actually_a_file"))
				})

				It("returns an error", func() {
					Expect(returnDir).To(Equal(""))
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("the application Staticfile specifies a root directory"))
					Expect(err.Error()).To(ContainSubstring("that is a plain file"))
				})
			})

			Context("the directory exists", func() {
				BeforeEach(func() {
					os.Mkdir(filepath.Join(buildDir, "a_directory"), 0755)
					sf.RootDir = "a_directory"
				})

				It("logs the staticfile's root directory", func() {
					Expect(buffer.String()).To(ContainSubstring("-----> Root folder"))
					Expect(buffer.String()).To(ContainSubstring("a_directory"))
				})

				It("returns the full directory path", func() {
					Expect(err).To(BeNil())
					Expect(returnDir).To(Equal(filepath.Join(buildDir, "a_directory")))
				})
			})
		})

		Context("the staticfile does not have an root directory", func() {
			BeforeEach(func() {
				sf.RootDir = ""
			})

			It("logs the build directory as the root directory", func() {
				Expect(buffer.String()).To(ContainSubstring("-----> Root folder"))
				Expect(buffer.String()).To(ContainSubstring(buildDir))
			})
			It("returns the build directory", func() {
				Expect(err).To(BeNil())
				Expect(returnDir).To(Equal(buildDir))
			})
		})
	})

	Describe("ConfigureNginx", func() {
		BeforeEach(func() {
			err = os.MkdirAll(filepath.Join(buildDir, "nginx", "conf"), 0755)
			Expect(err).To(BeNil())
		})

		JustBeforeEach(func() {
			err = compiler.ConfigureNginx()
			Expect(err).To(BeNil())
		})

		Context("custom nginx.conf exists", func() {
			BeforeEach(func() {
				err = os.MkdirAll(filepath.Join(buildDir, "public"), 0755)
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(buildDir, "public", "nginx.conf"), []byte("nginx configuration"), 0644)
				Expect(err).To(BeNil())
			})

			It("uses the custom configuration", func() {
				data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
				Expect(err).To(BeNil())
				Expect(data).To(Equal([]byte("nginx configuration")))
			})
		})

		Context("custom nginx.conf does NOT exist", func() {
			hostDotConf := `
    location ~ /\. {
      deny all;
      return 404;
    }
`
			pushStateConf := `
        if (!-e $request_filename) {
          rewrite ^(.*)$ / break;
        }
`

			forceHTTPSConf := `
        if ($http_x_forwarded_proto != "https") {
          return 301 https://$host$request_uri;
        }
`
			forceHTTPSErb := `
      <% if ENV["FORCE_HTTPS"] %>
        if ($http_x_forwarded_proto != "https") {
          return 301 https://$host$request_uri;
        }
      <% end %>
`

			basicAuthConf := `
        auth_basic "Restricted";  #For Basic Auth
        auth_basic_user_file <%= ENV["APP_ROOT"] %>/nginx/conf/.htpasswd;
`
			Context("host_dot_files is set in staticfile", func() {
				BeforeEach(func() {
					sf.HostDotFiles = true
				})
				It("allows dotfiles to be hosted", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(hostDotConf))
				})
			})

			Context("host_dot_files is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.HostDotFiles = false
				})
				It("allows dotfiles to be hosted", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(hostDotConf))
				})
			})

			Context("location_include is set in staticfile", func() {
				BeforeEach(func() {
					sf.LocationInclude = "a/b/c"
				})
				It("includes the file", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("include a/b/c;"))
				})
			})

			Context("location_include is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.LocationInclude = ""
				})
				It("does not include the file", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("include ;"))
				})
			})

			Context("directory is set in staticfile", func() {
				BeforeEach(func() {
					sf.DirectoryIndex = true
				})
				It("sets autoindex on", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("autoindex on;"))
				})
			})

			Context("directory is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.DirectoryIndex = false
				})
				It("does not set autoindex on", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("autoindex on;"))
				})
			})

			Context("ssi is set in staticfile", func() {
				BeforeEach(func() {
					sf.SSI = true
				})
				It("enables SSI", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("ssi on;"))
				})
			})

			Context("ssi is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.SSI = false
				})
				It("does not enable SSI", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("ssi on;"))
				})
			})

			Context("pushstate is set in staticfile", func() {
				BeforeEach(func() {
					sf.PushState = true
				})
				It("it adds the configuration", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(pushStateConf))
				})
			})

			Context("pushstate is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.PushState = false
				})
				It("it does not add the configuration", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(pushStateConf))
				})
			})

			Context("http_strict_transport_security is set in staticfile", func() {
				BeforeEach(func() {
					sf.HSTS = true
				})
				It("it adds the HSTS header", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(`add_header Strict-Transport-Security "max-age=31536000";`))
				})
			})

			Context("http_strict_transport_security is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.HSTS = false
				})
				It("it does not add the HSTS header", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(`add_header Strict-Transport-Security "max-age=31536000";`))
				})
			})

			Context("force_https is set in staticfile", func() {
				BeforeEach(func() {
					sf.ForceHTTPS = true
				})
				It("the 301 redirect does not depend on ENV['FORCE_HTTPS']", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(forceHTTPSConf))
					Expect(string(data)).NotTo(ContainSubstring(`<% if ENV["FORCE_HTTPS"] %>`))
					Expect(string(data)).NotTo(ContainSubstring(`<% end %>`))
				})
			})

			Context("force_https is NOT set in staticfile", func() {
				BeforeEach(func() {
					sf.ForceHTTPS = false
				})
				It("the 301 redirect does depend on ENV['FORCE_HTTPS']", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(forceHTTPSErb))
				})
			})

			Context("there is a Staticfile.auth", func() {
				BeforeEach(func() {
					sf.BasicAuth = true
					err = ioutil.WriteFile(filepath.Join(buildDir, "Staticfile.auth"), []byte("authentication info"), 0644)
					Expect(err).To(BeNil())
				})

				It("it enables basic authentication", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(basicAuthConf))
				})

				It("copies the Staticfile.auth to .htpasswd", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", ".htpasswd"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(Equal("authentication info"))
				})
			})

			Context("there is not a Staticfile.auth", func() {
				BeforeEach(func() {
					sf.BasicAuth = false
				})
				It("it does not enable basic authenticaiont", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(basicAuthConf))
				})

				It("does not create an .htpasswd", func() {
					Expect(filepath.Join(buildDir, "nginx", "conf", ".htpasswd")).NotTo(BeAnExistingFile())
				})
			})
		})

		Context("custom mime.types exists", func() {
			BeforeEach(func() {
				err = os.MkdirAll(filepath.Join(buildDir, "public"), 0755)
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(buildDir, "public", "mime.types"), []byte("mime types info"), 0644)
				Expect(err).To(BeNil())
			})

			It("uses the custom configuration", func() {
				data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "mime.types"))
				Expect(err).To(BeNil())
				Expect(data).To(Equal([]byte("mime types info")))
			})
		})

		Context("custom mime.types does NOT exist", func() {
			It("uses the provided mime.types", func() {
				data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "mime.types"))
				Expect(err).To(BeNil())
				Expect(string(data)).To(Equal(c.MimeTypes))
			})
		})
	})

	Describe("WriteProfileD", func() {
		var (
			info           os.FileInfo
			profileDScript string
		)
		BeforeEach(func() {
			profileDScript = filepath.Join(buildDir, ".profile.d", "staticfile.sh")
		})

		JustBeforeEach(func() {
			err = compiler.WriteProfileD()
			Expect(err).To(BeNil())
		})

		Context(".profile.d directory exists", func() {
			BeforeEach(func() {
				err = os.Mkdir(filepath.Join(buildDir, ".profile.d"), 0755)
				Expect(err).To(BeNil())
			})

			It("creates the file as an executable", func() {
				Expect(profileDScript).To(BeAnExistingFile())

				info, err = os.Stat(profileDScript)
				Expect(err).To(BeNil())

				// make sure at least 1 executable bit is set
				Expect(info.Mode().Perm() & 0111).NotTo(Equal(os.FileMode(0000)))
			})

		})
		Context(".profile.d directory does not exist", func() {
			It("creates the file as an executable", func() {
				Expect(profileDScript).To(BeAnExistingFile())

				info, err = os.Stat(profileDScript)
				Expect(err).To(BeNil())

				// make sure at least 1 executable bit is set
				Expect(info.Mode().Perm() & 0111).NotTo(Equal(0000))
			})
		})
	})

	Describe("InstallNginx", func() {
		It("Installs nginx to builddir", func() {
			dep := bp.Dependency{Name: "nginx", Version: "99.99"}

			mockManifest.EXPECT().DefaultVersion("nginx").Return(dep, nil)
			mockManifest.EXPECT().InstallDependency(dep, buildDir)

			compiler.InstallNginx()
			Expect(buffer.String()).To(ContainSubstring("-----> Installing nginx"))
			Expect(buffer.String()).To(ContainSubstring("       Using nginx version 99.99"))
		})
	})

	Describe("CopyFilesToPublic", func() {
		var (
			appRootDir    string
			buildDirFiles []string
		)

		JustBeforeEach(func() {
			buildDirFiles = []string{"Staticfile", "Staticfile.auth", "manifest.yml", ".profile", "stackato.yml", ".hidden.html", "index.html"}

			for _, file := range buildDirFiles {
				err = ioutil.WriteFile(filepath.Join(appRootDir, file), []byte(file+"contents"), 0644)
				Expect(err).To(BeNil())
			}

			err = compiler.CopyFilesToPublic(appRootDir)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.RemoveAll(appRootDir)
			Expect(err).To(BeNil())
		})

		Context("The appRootDir is <buildDir>/public", func() {
			BeforeEach(func() {
				appRootDir = filepath.Join(buildDir, "public")
				err = os.MkdirAll(appRootDir, 0755)
				Expect(err).To(BeNil())

				err = ioutil.WriteFile(filepath.Join(appRootDir, "index2.html"), []byte("html contents"), 0644)
			})

			It("doesn't copy any files", func() {
				for _, file := range buildDirFiles {
					_, err = os.Stat(filepath.Join(buildDir, file))
					Expect(os.IsNotExist(err)).To(BeTrue())
				}

				Expect(filepath.Join(appRootDir, "index2.html")).To(BeAnExistingFile())
			})
		})

		Context("The appRootDir is NOT <buildDir>/public", func() {
			Context("host dotfiles is set", func() {
				BeforeEach(func() {
					sf.HostDotFiles = true
					appRootDir, err = ioutil.TempDir("", "staticfile-buildpack.app_root.")
					Expect(err).To(BeNil())
				})

				It("Moves the dot files to public/", func() {
					Expect(filepath.Join(buildDir, "public", ".hidden.html")).To(BeAnExistingFile())
				})

				It("Moves the regular files to public/", func() {
					Expect(filepath.Join(buildDir, "public", "index.html")).To(BeAnExistingFile())
				})

				It("Does not move the blacklisted files to public/", func() {
					Expect(filepath.Join(buildDir, "public", "Staticfile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "Staticfile.auth")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "manifest.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "stackato.yml")).ToNot(BeAnExistingFile())
				})
			})
			Context("host dotfiles is NOT set", func() {
				BeforeEach(func() {
					sf.HostDotFiles = false
					appRootDir = buildDir
				})

				It("does NOT move the dot files to public/", func() {
					Expect(filepath.Join(buildDir, "public", ".hidden.html")).NotTo(BeAnExistingFile())
				})

				It("Moves the regular files to public/", func() {
					Expect(filepath.Join(buildDir, "public", "index.html")).To(BeAnExistingFile())
				})

				It("Does not move the blacklisted files to public/", func() {
					Expect(filepath.Join(buildDir, "public", "Staticfile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "Staticfile.auth")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "manifest.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "stackato.yml")).ToNot(BeAnExistingFile())
				})
			})
		})
	})
})
