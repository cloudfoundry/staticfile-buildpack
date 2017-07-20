package finalize_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"staticfile/finalize"
	"syscall"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalize.go --destination=mocks_test.go --package=finalize_test

var _ = Describe("Compile", func() {
	var (
		staticfile finalize.Staticfile
		err        error
		buildDir   string
		depDir     string
		finalizer  *finalize.Finalizer
		logger     *libbuildpack.Logger
		mockCtrl   *gomock.Controller
		mockYaml   *MockYAML
		buffer     *bytes.Buffer
		data       []byte
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "staticfile-buildpack.build.")
		Expect(err).To(BeNil())

		depDir, err = ioutil.TempDir("", "staticfile-buildpack.depDir.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockYaml = NewMockYAML(mockCtrl)
	})

	JustBeforeEach(func() {
		finalizer = &finalize.Finalizer{
			BuildDir: buildDir,
			DepDir:   depDir,
			Config:   staticfile,
			YAML:     mockYaml,
			Log:      logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depDir)
		Expect(err).To(BeNil())
	})

	Describe("WriteStartupFiles", func() {
		It("writes staticfile.sh to the profile.d directory", func() {
			err = finalizer.WriteStartupFiles()
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(depDir, "profile.d", "staticfile.sh"))
			Expect(err).To(BeNil())
			Expect(string(contents)).To(ContainSubstring("export LD_LIBRARY_PATH=$APP_ROOT/nginx/lib:$LD_LIBRARY_PATH"))
		})

		It("writes start_logging.sh in appdir", func() {
			err = finalizer.WriteStartupFiles()
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(buildDir, "start_logging.sh"))
			Expect(err).To(BeNil())
			Expect(string(contents)).To(Equal("\ncat < $APP_ROOT/nginx/logs/access.log &\n(>&2 cat) < $APP_ROOT/nginx/logs/error.log &\n"))
		})

		It("start_logging.sh is an executable file", func() {
			err = finalizer.WriteStartupFiles()
			Expect(err).To(BeNil())

			fi, err := os.Stat(filepath.Join(buildDir, "start_logging.sh"))
			Expect(err).To(BeNil())
			Expect(fi.Mode().Perm() & 0111).NotTo(Equal(os.FileMode(0000)))
		})

		It("writes boot.sh in appdir", func() {
			err = finalizer.WriteStartupFiles()
			Expect(err).To(BeNil())

			contents, err := ioutil.ReadFile(filepath.Join(buildDir, "boot.sh"))
			Expect(err).To(BeNil())
			Expect(string(contents)).To(Equal("#!/bin/sh\nset -ex\n$APP_ROOT/start_logging.sh\nnginx -p $APP_ROOT/nginx -c $APP_ROOT/nginx/conf/nginx.conf\n"))
		})

		It("boot.sh is an executable file", func() {
			err = finalizer.WriteStartupFiles()
			Expect(err).To(BeNil())

			fi, err := os.Stat(filepath.Join(buildDir, "boot.sh"))
			Expect(err).To(BeNil())
			Expect(fi.Mode().Perm() & 0111).NotTo(Equal(os.FileMode(0000)))
		})
	})

	Describe("LoadStaticfile", func() {
		Context("the staticfile does not exist", func() {
			BeforeEach(func() {
				mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Return(os.ErrNotExist)
			})
			It("does not return an error", func() {
				err = finalizer.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			It("has default values", func() {
				err = finalizer.LoadStaticfile()
				Expect(err).To(BeNil())
				Expect(finalizer.Config.RootDir).To(Equal(""))
				Expect(finalizer.Config.HostDotFiles).To(Equal(false))
				Expect(finalizer.Config.LocationInclude).To(Equal(""))
				Expect(finalizer.Config.DirectoryIndex).To(Equal(false))
				Expect(finalizer.Config.SSI).To(Equal(false))
				Expect(finalizer.Config.PushState).To(Equal(false))
				Expect(finalizer.Config.HSTS).To(Equal(false))
				Expect(finalizer.Config.ForceHTTPS).To(Equal(false))
				Expect(finalizer.Config.BasicAuth).To(Equal(false))
			})

			It("does not log enabling statements", func() {
				err = finalizer.LoadStaticfile()
				Expect(buffer.String()).To(Equal(""))
			})
		})
		Context("the staticfile exists", func() {
			JustBeforeEach(func() {
				err = finalizer.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			Context("and sets root", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["root"] = "root_test"
					})
				})
				It("sets RootDir", func() {
					Expect(finalizer.Config.RootDir).To(Equal("root_test"))
				})
			})

			Context("and sets host_dot_files", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Do(func(_ string, hash *map[string]string) {
						(*hash)["host_dot_files"] = "true"
					})
				})
				It("sets HostDotFiles", func() {
					Expect(finalizer.Config.HostDotFiles).To(Equal(true))
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
					Expect(finalizer.Config.LocationInclude).To(Equal("a/b/c"))
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
					Expect(finalizer.Config.DirectoryIndex).To(Equal(true))
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
					Expect(finalizer.Config.SSI).To(Equal(true))
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
					Expect(finalizer.Config.PushState).To(Equal(true))
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
					Expect(finalizer.Config.HSTS).To(Equal(true))
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
					Expect(finalizer.Config.ForceHTTPS).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(Equal("-----> Enabling HTTPS redirect\n"))
				})
			})
		})

		Context("Staticfile.auth is present", func() {
			BeforeEach(func() {
				err = ioutil.WriteFile(filepath.Join(buildDir, "Staticfile.auth"), []byte("some credentials"), 0644)
				Expect(err).To(BeNil())
			})
			JustBeforeEach(func() {
				err = finalizer.LoadStaticfile()
				Expect(err).To(BeNil())
			})

			Context("the staticfile exists", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(gomock.Any(), gomock.Any())
				})

				It("sets BasicAuth", func() {
					Expect(finalizer.Config.BasicAuth).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(ContainSubstring("-----> Enabling basic authentication using Staticfile.auth\n"))
				})
			})

			Context("the staticfile does not exist", func() {
				BeforeEach(func() {
					mockYaml.EXPECT().Load(gomock.Any(), gomock.Any()).Return(syscall.ENOENT)
				})

				It("sets BasicAuth", func() {
					Expect(finalizer.Config.BasicAuth).To(Equal(true))
				})
				It("Logs", func() {
					Expect(buffer.String()).To(ContainSubstring("-----> Enabling basic authentication using Staticfile.auth\n"))
				})
			})
		})

		Context("the staticfile exists and is not valid", func() {
			BeforeEach(func() {
				mockYaml.EXPECT().Load(filepath.Join(buildDir, "Staticfile"), gomock.Any()).Return(errors.New("a yaml parsing error"))
			})

			It("returns an error", func() {
				err = finalizer.LoadStaticfile()
				Expect(err).NotTo(BeNil())
			})
		})
	})

	Describe("GetAppRootDir", func() {
		var (
			returnDir string
		)

		JustBeforeEach(func() {
			returnDir, err = finalizer.GetAppRootDir()
		})

		Context("the staticfile has a root directory specified", func() {
			Context("the directory does not exist", func() {
				BeforeEach(func() {
					staticfile.RootDir = "not_exist"
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
					staticfile.RootDir = "actually_a_file"
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
					staticfile.RootDir = "a_directory"
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
				staticfile.RootDir = ""
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

	Describe("Warnings", func() {
		JustBeforeEach(func() {
			finalizer.Warnings()
		})

		Context("location_include is set", func() {
			BeforeEach(func() {
				staticfile.LocationInclude = "includes/*.conf"
			})

			Context("root is NOT set", func() {
				BeforeEach(func() {
					staticfile.RootDir = ""
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("**WARNING** The location_include directive only works in conjunction with root."))
					Expect(buffer.String()).To(ContainSubstring("Please specify root to use location_include"))
				})
			})

			Context("root IS set", func() {
				BeforeEach(func() {
					staticfile.RootDir = "public"
				})
				It("does not warn the user", func() {
					Expect(buffer.String()).ToNot(ContainSubstring("location_include directive only works in conjunction with root."))
					Expect(buffer.String()).ToNot(ContainSubstring("Please specify root to use location_include"))
				})
			})
		})

		Context("app dir has a nginx/conf directory", func() {
			BeforeEach(func() {
				os.MkdirAll(filepath.Join(buildDir, "nginx", "conf"), 0755)
			})

			Context("root is NOT set", func() {
				BeforeEach(func() {
					staticfile.RootDir = ""
				})
				It("warns the user", func() {
					Expect(buffer.String()).To(ContainSubstring("**WARNING** You have an nginx/conf directory, but have not set *root*."))
					Expect(buffer.String()).To(ContainSubstring("If you are using the nginx/conf directory for nginx configuration, you probably need to also set the *root* directive."))
				})
			})

			Context("root IS set", func() {
				BeforeEach(func() {
					staticfile.RootDir = "somedir"
				})
				It("does not warn the user", func() {
					Expect(buffer.String()).ToNot(ContainSubstring("**WARNING** You have an nginx/conf directory, but have not set *root*."))
					Expect(buffer.String()).ToNot(ContainSubstring("If you are using the nginx/conf directory for nginx configuration, you probably need to also set the *root* directive."))
				})
			})
		})
	})

	Describe("ConfigureNginx", func() {
		JustBeforeEach(func() {
			err = finalizer.ConfigureNginx()
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
					staticfile.HostDotFiles = true
				})
				It("allows dotfiles to be hosted", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(hostDotConf))
				})
			})

			Context("host_dot_files is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.HostDotFiles = false
				})
				It("allows dotfiles to be hosted", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(hostDotConf))
				})
			})

			Context("location_include is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.LocationInclude = "a/b/c"
				})
				It("includes the file", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("include a/b/c;"))
				})
			})

			Context("location_include is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.LocationInclude = ""
				})
				It("does not include the file", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("include ;"))
				})
			})

			Context("directory is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.DirectoryIndex = true
				})
				It("sets autoindex on", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("autoindex on;"))
				})
			})

			Context("directory is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.DirectoryIndex = false
				})
				It("does not set autoindex on", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("autoindex on;"))
				})
			})

			Context("ssi is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.SSI = true
				})
				It("enables SSI", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring("ssi on;"))
				})
			})

			Context("ssi is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.SSI = false
				})
				It("does not enable SSI", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring("ssi on;"))
				})
			})

			Context("pushstate is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.PushState = true
				})
				It("it adds the configuration", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(pushStateConf))
				})
			})

			Context("pushstate is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.PushState = false
				})
				It("it does not add the configuration", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(pushStateConf))
				})
			})

			Context("http_strict_transport_security is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.HSTS = true
				})
				It("it adds the HSTS header", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(`add_header Strict-Transport-Security "max-age=31536000";`))
				})
			})

			Context("http_strict_transport_security is NOT set in staticfile", func() {
				BeforeEach(func() {
					staticfile.HSTS = false
				})
				It("it does not add the HSTS header", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).NotTo(ContainSubstring(`add_header Strict-Transport-Security "max-age=31536000";`))
				})
			})

			Context("force_https is set in staticfile", func() {
				BeforeEach(func() {
					staticfile.ForceHTTPS = true
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
					staticfile.ForceHTTPS = false
				})
				It("the 301 redirect does depend on ENV['FORCE_HTTPS']", func() {
					data, err = ioutil.ReadFile(filepath.Join(buildDir, "nginx", "conf", "nginx.conf"))
					Expect(err).To(BeNil())
					Expect(string(data)).To(ContainSubstring(forceHTTPSErb))
				})
			})

			Context("there is a Staticfile.auth", func() {
				BeforeEach(func() {
					staticfile.BasicAuth = true
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
					staticfile.BasicAuth = false
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
				Expect(string(data)).To(Equal(finalize.MimeTypes))
			})
		})
	})

	Describe("CopyFilesToPublic", func() {
		var (
			appRootDir          string
			appRootFiles        []string
			buildDirFiles       []string
			buildDirDirectories []string
		)

		JustBeforeEach(func() {
			buildDirFiles = []string{"Staticfile", "Staticfile.auth", "manifest.yml", ".profile", "stackato.yml"}

			for _, file := range buildDirFiles {
				err = ioutil.WriteFile(filepath.Join(buildDir, file), []byte(file+"contents"), 0644)
				Expect(err).To(BeNil())
			}

			appRootFiles = []string{".hidden.html", "index.html"}

			for _, file := range appRootFiles {
				err = ioutil.WriteFile(filepath.Join(appRootDir, file), []byte(file+"contents"), 0644)
				Expect(err).To(BeNil())
			}

			buildDirDirectories = []string{".profile.d", ".cloudfoundry"}
			for _, dir := range buildDirDirectories {
				err = os.MkdirAll(filepath.Join(buildDir, dir), 0755)
				Expect(err).To(BeNil())
			}

			err = finalizer.CopyFilesToPublic(appRootDir)
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
					Expect(filepath.Join(buildDir, file)).To(BeAnExistingFile())
				}

				for _, dir := range buildDirDirectories {
					Expect(filepath.Join(buildDir, dir)).To(BeADirectory())
				}

				for _, file := range appRootFiles {
					Expect(filepath.Join(appRootDir, file)).To(BeAnExistingFile())
				}

				Expect(filepath.Join(appRootDir, "index2.html")).To(BeAnExistingFile())
			})
		})

		Context("The appRootDir is NOT <buildDir>/public", func() {
			Context("host dotfiles is set", func() {
				BeforeEach(func() {
					staticfile.HostDotFiles = true
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
					Expect(filepath.Join(buildDir, "Staticfile")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "Staticfile.auth")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "manifest.yml")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, ".profile")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "stackato.yml")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, ".profile.d")).To(BeADirectory())
					Expect(filepath.Join(buildDir, ".cloudfoundry")).To(BeADirectory())

					Expect(filepath.Join(buildDir, "public", "Staticfile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "Staticfile.auth")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "manifest.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "stackato.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile.d")).ToNot(BeADirectory())
					Expect(filepath.Join(buildDir, "public", ".cloudfoundry")).ToNot(BeADirectory())
				})

				Context("and <buildDir>/public exists", func() {
					BeforeEach(func() {
						Expect(os.Mkdir(filepath.Join(buildDir, "public"), 0755)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(buildDir, "public", "orig.html"), []byte("html contents"), 0644)).To(Succeed())
					})
					It("overrides <buildDir>/public", func() {
						Expect(filepath.Join(buildDir, "public", "orig.html")).ToNot(BeAnExistingFile())
						Expect(filepath.Join(buildDir, "public", "index.html")).To(BeAnExistingFile())
					})
				})
			})
			Context("host dotfiles is NOT set", func() {
				BeforeEach(func() {
					staticfile.HostDotFiles = false
					appRootDir = buildDir
				})

				It("does NOT move the dot files to public/", func() {
					Expect(filepath.Join(buildDir, ".hidden.html")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".hidden.html")).NotTo(BeAnExistingFile())
				})

				It("Moves the regular files to public/", func() {
					Expect(filepath.Join(buildDir, "public", "index.html")).To(BeAnExistingFile())
				})

				It("Does not move the blacklisted files to public/", func() {
					Expect(filepath.Join(buildDir, "Staticfile")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "Staticfile.auth")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "manifest.yml")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, ".profile")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "stackato.yml")).To(BeAnExistingFile())
					Expect(filepath.Join(buildDir, ".profile.d")).To(BeADirectory())
					Expect(filepath.Join(buildDir, ".cloudfoundry")).To(BeADirectory())

					Expect(filepath.Join(buildDir, "public", "Staticfile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "Staticfile.auth")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "manifest.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", "stackato.yml")).ToNot(BeAnExistingFile())
					Expect(filepath.Join(buildDir, "public", ".profile.d")).ToNot(BeADirectory())
					Expect(filepath.Join(buildDir, "public", ".cloudfoundry")).ToNot(BeADirectory())
				})
			})
		})
	})
})
