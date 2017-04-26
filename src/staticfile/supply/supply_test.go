package supply_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"staticfile/supply"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=../vendor/github.com/cloudfoundry/libbuildpack/manifest.go --destination=mocks_manifest_test.go --package=supply_test --imports=.=github.com/cloudfoundry/libbuildpack

var _ = Describe("Supply", func() {
	var (
		err          error
		depsDir      string
		depsIdx      string
		depDir       string
		supplier     *supply.Supplier
		logger       libbuildpack.Logger
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		buffer       *bytes.Buffer
	)

	BeforeEach(func() {
		depsDir, err = ioutil.TempDir("", "staticfile-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "32"
		depDir = filepath.Join(depsDir, depsIdx)

		err = os.MkdirAll(depDir, 0755)
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger()
		logger.SetOutput(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
	})

	JustBeforeEach(func() {
		bps := &libbuildpack.Stager{
			DepsDir:  depsDir,
			DepsIdx:  depsIdx,
			Manifest: mockManifest,
			Log:      logger,
		}

		supplier = &supply.Supplier{
			Stager: bps,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})

	Describe("InstallNginx", func() {
		BeforeEach(func() {
			dep := libbuildpack.Dependency{Name: "nginx", Version: "99.99"}

			mockManifest.EXPECT().DefaultVersion("nginx").Return(dep, nil)
			mockManifest.EXPECT().InstallDependency(dep, depDir)
		})

		It("Installs nginx to the depDir, creating a symlink in <depDir>/bin", func() {
			supplier.InstallNginx()
			Expect(err).To(BeNil())
			Expect(buffer.String()).To(ContainSubstring("-----> Installing nginx"))
			Expect(buffer.String()).To(ContainSubstring("       Using nginx version 99.99"))

			link, err := os.Readlink(filepath.Join(depDir, "bin", "nginx"))
			Expect(err).To(BeNil())

			Expect(link).To(Equal("../nginx/sbin/nginx"))
		})
	})
})
