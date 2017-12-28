package hooks_test

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"

	"staticfile/hooks"

	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=dynatrace.go --destination=mocks_test.go --package=hooks_test

var _ = Describe("dynatraceHook", func() {
	var (
		err          error
		buildDir     string
		depsDir      string
		depsIdx      string
		logger       *libbuildpack.Logger
		stager       *libbuildpack.Stager
		mockCtrl     *gomock.Controller
		mockCommand  *MockCommand
		buffer       *bytes.Buffer
		dynatrace    hooks.DynatraceHook
		runInstaller func(string, io.Writer, io.Writer, string, string)
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "staticfile-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "staticfile-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "07"
		err = os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)
		dynatrace = hooks.DynatraceHook{
			Command:         mockCommand,
			Log:             logger,
			DownloadRetries: 0.0,
		}

		os.Setenv("DT_LOGSTREAM", "")

		httpmock.Reset()

		runInstaller = func(_ string, _, _ io.Writer, file string, _ string) {
			contents, err := ioutil.ReadFile(file)
			Expect(err).To(BeNil())

			Expect(string(contents)).To(Equal("echo Install Dynatrace"))

			err = os.MkdirAll(filepath.Join(buildDir, "dynatrace/oneagent/agent/lib64"), 0755)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(buildDir, "dynatrace/oneagent/agent/lib64/liboneagentproc.so"), []byte("library"), 0644)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(buildDir, "dynatrace/oneagent/dynatrace-env.sh"), []byte("echo running dynatrace-env.sh"), 0644)
			Expect(err).To(BeNil())

			manifestJson := `
			{
				"version" : "1.130.0.20170914-153344",
				"technologies" : {
					"process" : {
						"linux-x86-64" : [ {
							"path" : "agent/conf/runtime/default/process/binary_linux-x86-64",
							"md5" : "e086f9c70b53cd456988ff5c4d414f36",
							"version" : "1.130.0.20170914-125024"
						  }, {
							"path" : "agent/lib64/liboneagentproc.so",
							"md5" : "2bf4ba9e90e2589428f6f6f3a964cba2",
							"version" : "1.130.0.20170914-125024",
							"binarytype" : "primary"}]
					}
				}
			}`
			err = ioutil.WriteFile(filepath.Join(buildDir, "dynatrace/oneagent/manifest.json"), []byte(manifestJson), 0664)
			Expect(err).To(BeNil())
		}
	})

	JustBeforeEach(func() {
		args := []string{buildDir, "", depsDir, depsIdx}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})

	Describe("AfterCompile", func() {
		var (
			oldVcapApplication string
			oldVcapServices    string
			oldBpDebug         string
		)
		BeforeEach(func() {
			oldVcapApplication = os.Getenv("VCAP_APPLICATION")
			oldVcapServices = os.Getenv("VCAP_SERVICES")
			oldBpDebug = os.Getenv("BP_DEBUG")

		})
		AfterEach(func() {
			os.Setenv("VCAP_APPLICATION", oldVcapApplication)
			os.Setenv("VCAP_SERVICES", oldVcapServices)
			os.Setenv("BP_DEBUG", oldBpDebug)
		})

		Context("VCAP_SERVICES is empty", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", "{}")
			})

			It("does nothing and succeeds", func() {
				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("VCAP_SERVICES has non dynatrace services", func() {
			BeforeEach(func() {
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"redis"}]
				}`)
			})

			It("does nothing and succeeds", func() {
				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("VCAP_SERVICES has incomplete dynatrace service", func() {
			BeforeEach(func() {
				environmentid := "123456"
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"dynatrace","credentials":{"apiurl":"https://example.com","environmentid":"`+environmentid+`"}}],
				}`)
			})

			It("does nothing and succeeds", func() {
				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("VCAP_SERVICES contains dynatrace service using apiurl", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"dynatrace","credentials":{"apiurl":"https://example.com","apitoken":"`+apiToken+`","environmentid":"`+environmentid+`"}}],
					"2": [{"name":"redis"}]
				}`)

				httpmock.RegisterResponder("GET", "https://example.com/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token="+apiToken,
					httpmock.NewStringResponder(200, "echo Install Dynatrace"))
			})

			It("installs dynatrace", func() {
				mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), gomock.Any(), buildDir).Do(runInstaller)

				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "dynatrace-env.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("echo running dynatrace-env.sh\n" +
					"export LD_PRELOAD=${HOME}/dynatrace/oneagent/agent/lib64/liboneagentproc.so\n" +
					"export DT_HOST_ID=JimBob_${CF_INSTANCE_INDEX}\n" +
					"export DT_LOGSTREAM=stdout"))
			})
		})

		Context("VCAP_SERVICES contains dynatrace service using apiurl and has DT_LOGSTREAM set to stderr", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("DT_LOGSTREAM", "stderr")
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"dynatrace","credentials":{"apiurl":"https://example.com","apitoken":"`+apiToken+`","environmentid":"`+environmentid+`"}}],
					"2": [{"name":"redis"}]
				}`)

				httpmock.RegisterResponder("GET", "https://example.com/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token="+apiToken,
					httpmock.NewStringResponder(200, "echo Install Dynatrace"))
			})

			It("installs dynatrace", func() {
				mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), gomock.Any(), buildDir).Do(runInstaller)

				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "dynatrace-env.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("echo running dynatrace-env.sh\n" +
					"export LD_PRELOAD=${HOME}/dynatrace/oneagent/agent/lib64/liboneagentproc.so\n" +
					"export DT_HOST_ID=JimBob_${CF_INSTANCE_INDEX}"))
			})
		})

		Context("VCAP_SERVICES contains dynatrace service using apiurl and has DT_LOGSTREAM not set", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"dynatrace","credentials":{"apiurl":"https://example.com","apitoken":"`+apiToken+`","environmentid":"`+environmentid+`"}}],
					"2": [{"name":"redis"}]
				}`)

				httpmock.RegisterResponder("GET", "https://example.com/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token="+apiToken,
					httpmock.NewStringResponder(200, "echo Install Dynatrace"))
			})

			It("installs dynatrace", func() {
				mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), gomock.Any(), buildDir).Do(runInstaller)

				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "dynatrace-env.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("echo running dynatrace-env.sh\n" +
					"export LD_PRELOAD=${HOME}/dynatrace/oneagent/agent/lib64/liboneagentproc.so\n" +
					"export DT_HOST_ID=JimBob_${CF_INSTANCE_INDEX}\n" +
					"export DT_LOGSTREAM=stdout"))
			})
		})
		Context("VCAP_SERVICES contains dynatrace service using environmentid", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"mysql"}],
					"1": [{"name":"dynatrace","credentials":{"environmentid":"`+environmentid+`","apitoken":"`+apiToken+`"}}],
					"2": [{"name":"redis"}]
				}`)

				httpmock.RegisterResponder("GET", "https://123456.live.dynatrace.com/api/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token="+apiToken,
					httpmock.NewStringResponder(200, "echo Install Dynatrace"))
			})

			It("installs dyntatrace", func() {
				mockCommand.EXPECT().Execute("", gomock.Any(), gomock.Any(), gomock.Any(), buildDir).Do(runInstaller)

				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				// Sets up profile.d
				contents, err := ioutil.ReadFile(filepath.Join(depsDir, depsIdx, "profile.d", "dynatrace-env.sh"))
				Expect(err).To(BeNil())

				Expect(string(contents)).To(Equal("echo running dynatrace-env.sh\n" +
					"export LD_PRELOAD=${HOME}/dynatrace/oneagent/agent/lib64/liboneagentproc.so\n" +
					"export DT_HOST_ID=JimBob_${CF_INSTANCE_INDEX}\n" +
					"export DT_LOGSTREAM=stdout"))
			})
		})

		Context("VCAP_SERVICES contains second dynatrace service with credentials", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"dynatrace","credentials":{"environmentid":"`+environmentid+`","apitoken":"`+apiToken+`"}}],
					"1": [{"name":"dynatrace-dupe","credentials":{"environmentid":"`+environmentid+`","apitoken":"`+apiToken+`"}}]
				}`)
			})

			It("does nothing and succeeds", func() {
				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("More than one matching service found!"))
			})
		})

		Context("VCAP_SERVICES contains skiperrors flag", func() {
			BeforeEach(func() {
				environmentid := "123456"
				apiToken := "ExcitingToken28"
				os.Setenv("BP_DEBUG", "true")
				os.Setenv("VCAP_APPLICATION", `{"name":"JimBob"}`)
				os.Setenv("VCAP_SERVICES", `{
					"0": [{"name":"dynatrace","credentials":{"environmentid":"`+environmentid+`","apitoken":"`+apiToken+`","skiperrors":"true"}}]
				}`)

				httpmock.RegisterResponder("GET", "https://123456.live.dynatrace.com/api/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token="+apiToken,
					httpmock.NewStringResponder(404, "echo agent not found"))
			})

			It("does nothing and succeeds", func() {
				err = dynatrace.AfterCompile(stager)
				Expect(err).To(BeNil())

				Expect(buffer.String()).To(ContainSubstring("Download returned with status 404"))
				Expect(buffer.String()).To(ContainSubstring("Error during installer download, skipping installation"))
			})
		})
	})
})
