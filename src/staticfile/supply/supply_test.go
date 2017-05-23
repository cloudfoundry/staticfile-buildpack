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

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		err          error
		depsDir      string
		depsIdx      string
		depDir       string
		supplier     *supply.Supplier
		logger       *libbuildpack.Logger
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
		logger = libbuildpack.NewLogger(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
	})

	JustBeforeEach(func() {
		args := []string{"", "", depsDir, depsIdx}
		bps := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		supplier = &supply.Supplier{
			Stager:   bps,
			Manifest: mockManifest,
			Log:      logger,
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
			Expect(supplier.InstallNginx()).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("-----> Installing nginx"))
			Expect(buffer.String()).To(ContainSubstring("       Using nginx version 99.99"))

			link, err := os.Readlink(filepath.Join(depDir, "bin", "nginx"))
			Expect(err).To(BeNil())

			Expect(link).To(Equal("../nginx/sbin/nginx"))
		})
	})
})
