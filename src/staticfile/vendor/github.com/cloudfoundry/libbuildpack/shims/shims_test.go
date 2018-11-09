package shims_test

import (
	"bytes"
	"github.com/cloudfoundry/libbuildpack/shims"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:generate mockgen -source=shims.go --destination=mocks_shims_test.go --package=shims_test

var _ = Describe("Shims", func() {
	Describe("Detect", func() {
		var (
			mockCtrl    *gomock.Controller
			mockShimmer *MockShimmer
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockShimmer = NewMockShimmer(mockCtrl)
		})

		It("runs with the correct arguments", func() {
			mockShimmer.
				EXPECT().
				RootDir().
				Return("buildpack-dir").
				Times(2)

			mockShimmer.
				EXPECT().
				Detect(
					filepath.Join("buildpack-dir", "cnbs"),
					filepath.Join("build-dir", "group.toml"),
					"build-dir",
					filepath.Join("buildpack-dir", "order.toml"),
					filepath.Join("build-dir", "plan.toml"),
				).
				Times(1)

			Expect(shims.Detect(mockShimmer, "build-dir")).To(Succeed())
		})
	})

	Describe("Supply", func() {
		var (
			mockCtrl                                   *gomock.Controller
			mockShimmer                                *MockShimmer
			buildDir, depsDir, launchDir, workspaceDir string
		)

		BeforeEach(func() {
			var err error

			mockCtrl = gomock.NewController(GinkgoT())
			mockShimmer = NewMockShimmer(mockCtrl)

			workspaceDir, err = ioutil.TempDir("", "workspace")
			Expect(err).NotTo(HaveOccurred())

			buildDir = filepath.Join(workspaceDir, "build")
			Expect(os.MkdirAll(buildDir, 0777)).To(Succeed())

			depsDir = filepath.Join(workspaceDir, "deps")
			Expect(os.MkdirAll(filepath.Join(depsDir, "0"), 0777)).To(Succeed())

			launchDir = filepath.Join(workspaceDir, "launch")
			Expect(os.MkdirAll(filepath.Join(launchDir, "config"), 0777)).To(Succeed())
		})

		AfterEach(func() {
			os.RemoveAll(workspaceDir)
		})

		Context("when detection has already run", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(workspaceDir, "group.toml"), []byte(""), 0666)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(workspaceDir, "plan.toml"), []byte(""), 0666)).To(Succeed())
			})

			It("runs with the correct arguments and moves things to the correct place", func() {
				mockShimmer.
					EXPECT().
					RootDir().
					Return("buildpack-dir").
					Times(2)

				mockShimmer.
					EXPECT().
					Supply(
						filepath.Join("buildpack-dir", "cnbs"),
						"cache-dir",
						filepath.Join(workspaceDir, "group.toml"),
						launchDir,
						filepath.Join(workspaceDir, "plan.toml"),
						workspaceDir,
					).
					Do(func(args ...string) {
						Expect(ioutil.WriteFile(filepath.Join(launchDir, "test.txt"), []byte("hello"), 0666)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(launchDir, "config", "metadata.toml"), []byte("howdy"), 0666)).To(Succeed())
					}).
					Times(1)

				Expect(shims.Supply(mockShimmer, buildDir, "cache-dir", depsDir, "0", workspaceDir, launchDir)).To(Succeed())
				Expect(filepath.Join(buildDir, "metadata.toml")).To(BeAnExistingFile())
				Expect(filepath.Join(depsDir, "0", "test.txt")).To(BeAnExistingFile())
			})
		})

		Context("when the group.toml and plan.toml do not exist", func() {
			It("runs the v3 detector", func() {
				mockShimmer.
					EXPECT().
					RootDir().
					Return("buildpack-dir").
					Times(2)

				mockShimmer.
					EXPECT().
					Detect(
						filepath.Join("buildpack-dir", "cnbs"),
						filepath.Join(workspaceDir, "group.toml"),
						workspaceDir,
						filepath.Join("buildpack-dir", "order.toml"),
						filepath.Join(workspaceDir, "plan.toml"),
					).Times(1)

				mockShimmer.
					EXPECT().
					RootDir().
					Return("buildpack-dir").
					Times(2)

				mockShimmer.
					EXPECT().
					Supply(
						filepath.Join("buildpack-dir", "cnbs"),
						"cache-dir",
						filepath.Join(workspaceDir, "group.toml"),
						launchDir,
						filepath.Join(workspaceDir, "plan.toml"),
						workspaceDir,
					).
					Do(func(args ...string) {
						Expect(ioutil.WriteFile(filepath.Join(launchDir, "test.txt"), []byte("hello"), 0666)).To(Succeed())
						Expect(ioutil.WriteFile(filepath.Join(launchDir, "config", "metadata.toml"), []byte("howdy"), 0666)).To(Succeed())
					}).
					Times(1)

				Expect(shims.Supply(mockShimmer, buildDir, "cache-dir", depsDir, "0", workspaceDir, launchDir)).To(Succeed())
				Expect(filepath.Join(buildDir, "metadata.toml")).To(BeAnExistingFile())
				Expect(filepath.Join(depsDir, "0", "test.txt")).To(BeAnExistingFile())
			})
		})
	})

	Describe("Finalize", func() {
		var (
			depsDir, profileDir string
		)

		BeforeEach(func() {
			var err error

			depsDir, err = ioutil.TempDir("", "deps")
			Expect(err).NotTo(HaveOccurred())

			tempProfileDir := filepath.Join(depsDir, "0", "some-buildpack", "some-dep", "profile.d")
			Expect(os.MkdirAll(tempProfileDir, 0777)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tempProfileDir, "some_script.sh"), []byte(""), 0666)).To(Succeed())

			otherTempProfileDir := filepath.Join(depsDir, "0", "some-other-buildpack", "some-other-dep", "profile.d")
			Expect(os.MkdirAll(otherTempProfileDir, 0777)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(otherTempProfileDir, "some_other_script.sh"), []byte(""), 0666)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(depsDir, "0", "some-buildpack", "some-dep", "bin"), 0777)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(depsDir, "0", "some-other-buildpack", "some-other-dep", "bin"), 0777)).To(Succeed())

			profileDir, err = ioutil.TempDir("", "profile")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(depsDir)
			os.RemoveAll(profileDir)
		})

		It("runs with the correct arguments and moves things to the correct place", func() {
			Expect(shims.Finalize(depsDir, "0", profileDir)).To(Succeed())

			Expect(filepath.Join(profileDir, "some_script.sh")).To(BeAnExistingFile())
			Expect(filepath.Join(profileDir, "some_other_script.sh")).To(BeAnExistingFile())

			Expect(filepath.Join(profileDir, "0.sh")).To(BeAnExistingFile())
			Expect(ioutil.ReadFile(filepath.Join(profileDir, "0.sh"))).To(Equal([]byte(
				`export PATH=$DEPS_DIR/0/some-buildpack/some-dep/bin:$DEPS_DIR/0/some-other-buildpack/some-other-dep/bin:$PATH`,
			)))
		})
	})

	Describe("Release", func() {
		var (
			buildDir string
		)

		BeforeEach(func() {
			var err error

			buildDir, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())
			contents := `
buildpacks = ["some.buildpacks", "some.other.buildpack"]
[[processes]]
type = "web"
command = "npm start"
`
			Expect(ioutil.WriteFile(filepath.Join(buildDir, "metadata.toml"), []byte(contents), 0666)).To(Succeed())
		})

		AfterEach(func() {
			os.RemoveAll(buildDir)
		})

		It("runs with the correct arguments and moves things to the correct place", func() {
			buf := &bytes.Buffer{}
			Expect(shims.Release(buildDir, buf)).To(Succeed())
			Expect(buf.Bytes()).To(Equal([]byte("default_process_types:\n  web: npm start\n")))
			Expect(filepath.Join(buildDir, "metadata.toml")).NotTo(BeAnExistingFile())
		})
	})
})
