package shims_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/shims"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supplier.go --destination=mocks_shims_test.go --package=shims_test

var _ = Describe("Shims", func() {
	Describe("Supplier", func() {
		var (
			supplier                                                                                                                                        shims.Supplier
			mockCtrl                                                                                                                                        *gomock.Controller
			mockDetector                                                                                                                                    *MockDetector
			binDir, v2BuildDir, cnbAppDir, buildpacksDir, cacheDir, depsDir, depsIndex, groupMetadata, launchDir, orderMetadata, planMetadata, workspaceDir string
		)

		BeforeEach(func() {
			var err error

			mockCtrl = gomock.NewController(GinkgoT())
			mockDetector = NewMockDetector(mockCtrl)

			workspaceDir, err = ioutil.TempDir("", "workspace")
			Expect(err).NotTo(HaveOccurred())

			v2BuildDir = filepath.Join(workspaceDir, "build")
			Expect(os.MkdirAll(v2BuildDir, 0777)).To(Succeed())

			cnbAppDir = filepath.Join(workspaceDir, "cnb-app")

			binDir = filepath.Join(workspaceDir, "bin")
			Expect(os.MkdirAll(binDir, 0777)).To(Succeed())

			cacheDir = filepath.Join(workspaceDir, "cache")
			Expect(os.MkdirAll(cacheDir, 0777)).To(Succeed())

			buildpacksDir = filepath.Join(workspaceDir, "cnbs")
			Expect(os.MkdirAll(buildpacksDir, 0777)).To(Succeed())

			depsDir = filepath.Join(workspaceDir, "deps")
			depsIndex = "0"

			Expect(os.MkdirAll(filepath.Join(depsDir, depsIndex), 0777)).To(Succeed())

			launchDir = filepath.Join(workspaceDir, "launch")
			Expect(os.MkdirAll(filepath.Join(launchDir, "config"), 0777)).To(Succeed())

			orderMetadata = filepath.Join(workspaceDir, "order.toml")
			Expect(ioutil.WriteFile(orderMetadata, []byte(""), 0666)).To(Succeed())

			groupMetadata = filepath.Join(workspaceDir, "group.toml")

			planMetadata = filepath.Join(workspaceDir, "plan.toml")

			supplier = shims.Supplier{
				Detector:      mockDetector,
				BinDir:        binDir,
				V2BuildDir:    v2BuildDir,
				CNBAppDir:     cnbAppDir,
				BuildpacksDir: buildpacksDir,
				CacheDir:      cacheDir,
				DepsDir:       depsDir,
				DepsIndex:     depsIndex,
				LaunchDir:     launchDir,
				OrderMetadata: orderMetadata,
				GroupMetadata: groupMetadata,
				PlanMetadata:  planMetadata,
				WorkspaceDir:  workspaceDir,
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
			Expect(os.RemoveAll(workspaceDir)).To(Succeed())
		})

		Context("GetBuildPlan", func() {
			It("runs detection when group or plan metadata does not exist", func() {
				mockDetector.
					EXPECT().
					Detect()
				Expect(supplier.GetBuildPlan()).To(Succeed())
			})

			It("does NOT run detection when group and plan metadata exists", func() {
				Expect(ioutil.WriteFile(groupMetadata, []byte(""), 0666)).To(Succeed())
				Expect(ioutil.WriteFile(planMetadata, []byte(""), 0666)).To(Succeed())

				mockDetector.
					EXPECT().
					Detect().
					Times(0)
				Expect(supplier.GetBuildPlan()).To(Succeed())
			})
		})

		Context("MoveLayers", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(launchDir, "config"), 0777)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(launchDir, "config", "metadata.toml"), []byte(""), 0666)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(launchDir, "layer"), 0777)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(launchDir, "anotherLayer"), 0777)).To(Succeed())
			})

			It("moves the layers to deps dir and metadata to build dir", func() {
				Expect(supplier.MoveLayers()).To(Succeed())
				Expect(filepath.Join(v2BuildDir, ".cloudfoundry", "metadata.toml")).To(BeAnExistingFile())
				Expect(filepath.Join(depsDir, depsIndex, "layer")).To(BeAnExistingFile())
				Expect(filepath.Join(depsDir, depsIndex, "anotherLayer")).To(BeAnExistingFile())
			})
		})
	})

	Describe("Finalizer", func() {
		var (
			finalizer             shims.Finalizer
			profileDir, depsIndex string
		)

		BeforeEach(func() {
			var err error

			depsIndex = "0"

			profileDir, err = ioutil.TempDir("", "profile")
			Expect(err).NotTo(HaveOccurred())

			finalizer = shims.Finalizer{DepsIndex: depsIndex, ProfileDir: profileDir}
		})

		AfterEach(func() {
			os.RemoveAll(profileDir)
		})

		It("runs with the correct arguments and moves things to the correct place", func() {
			Expect(finalizer.Finalize()).To(Succeed())

			contents, err := ioutil.ReadFile(filepath.Join(profileDir, "0_shim.sh"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(contents)).To(Equal(`export PACK_STACK_ID="org.cloudfoundry.stacks."
export PACK_LAUNCH_DIR="$DEPS_DIR/0"
export PACK_APP_DIR="$HOME"
exec $DEPS_DIR/v3-launcher "$2"
`))
		})
	})

	Describe("Releaser", func() {
		var (
			releaser   shims.Releaser
			v2BuildDir string
			buf        *bytes.Buffer
		)

		BeforeEach(func() {
			var err error

			v2BuildDir, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())

			contents := `
			buildpacks = ["some.buildpacks", "some.other.buildpack"]
			[[processes]]
			type = "web"
			command = "npm start"
			`
			os.MkdirAll(filepath.Join(v2BuildDir, ".cloudfoundry"), 0777)
			Expect(ioutil.WriteFile(filepath.Join(v2BuildDir, ".cloudfoundry", "metadata.toml"), []byte(contents), 0666)).To(Succeed())

			buf = &bytes.Buffer{}

			releaser = shims.Releaser{
				MetadataPath: filepath.Join(v2BuildDir, ".cloudfoundry", "metadata.toml"),
				Writer:       buf,
			}
		})

		AfterEach(func() {
			os.RemoveAll(v2BuildDir)
		})

		It("runs with the correct arguments and moves things to the correct place", func() {
			Expect(releaser.Release()).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte("default_process_types:\n  web: npm start\n")))
			Expect(filepath.Join(v2BuildDir, ".cloudfoundry", "metadata.toml")).NotTo(BeAnExistingFile())
		})
	})
})
