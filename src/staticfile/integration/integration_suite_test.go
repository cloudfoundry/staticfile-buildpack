package integration_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var bpDir string
var buildpackVersion string
var buildpackZipFile string
var packagedBuildpack cutlass.VersionedBuildpackPackage

func init() {
	flag.StringVar(&buildpackVersion, "version", "", "version to use (builds if version and zipFile empty)")
	flag.BoolVar(&cutlass.Cached, "cached", true, "cached buildpack")
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "128M", "default disk for pushed apps")
	flag.StringVar(&buildpackZipFile, "zipFile", "", "buildpack zip file in buildpack root dir (builds if zipFile and version empty)")
	flag.Parse()
	fmt.Println("cutlass.Cached", cutlass.Cached)
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Run once
	if buildpackVersion == "" && buildpackZipFile == "" {
		packagedBuildpack, err := cutlass.PackageUniquelyVersionedBuildpack()
		Expect(err).NotTo(HaveOccurred())

		data, err := json.Marshal(packagedBuildpack)
		Expect(err).NotTo(HaveOccurred())
		return data
	}
	if buildpackZipFile != "" {
		Expect(buildpackVersion).NotTo(BeEmpty())

		rootDir, err := cutlass.FindRoot()
		Expect(err).NotTo(HaveOccurred())

		buildpackZipFilePath := filepath.Join(rootDir, buildpackZipFile)
		_, err = os.Stat(buildpackZipFilePath)
		Expect(err).NotTo(HaveOccurred())

		packagedBuildpack := cutlass.VersionedBuildpackPackage{
			Version: buildpackVersion,
			File: buildpackZipFilePath,
		}

		data, err := json.Marshal(packagedBuildpack)
		Expect(err).NotTo(HaveOccurred())
		return data
	}

	return []byte{}
}, func(data []byte) {
	// Run on all nodes
	var err error
	if len(data) > 0 {
		err = json.Unmarshal(data, &packagedBuildpack)
		Expect(err).NotTo(HaveOccurred())
		buildpackVersion = packagedBuildpack.Version
	}

	bpDir, err = cutlass.FindRoot()
	Expect(err).NotTo(HaveOccurred())

	Expect(cutlass.CopyCfHome()).To(Succeed())
	cutlass.SeedRandom()
	cutlass.DefaultStdoutStderr = GinkgoWriter
	SetDefaultEventuallyTimeout(10 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all nodes
}, func() {
	// Run once
	if buildpackZipFile == "" {
		Expect(cutlass.RemovePackagedBuildpack(packagedBuildpack)).To(Succeed())
	}
	Expect(cutlass.DeleteOrphanedRoutes()).To(Succeed())
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func PushAppAndConfirm(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(app.InstanceStates).Should(Equal([]string{"RUNNING"}))
	Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
}

func ApiGreaterThan(version string) bool {
	apiVersionString, err := cutlass.ApiVersion()
	Expect(err).To(BeNil())
	apiVersion, err := semver.Make(apiVersionString)
	Expect(err).To(BeNil())
	reqVersion, err := semver.ParseRange(">= " + version)
	Expect(err).To(BeNil())
	return reqVersion(apiVersion)
}

func ApiHasTask() bool {
	return ApiGreaterThan("2.75.0")
}

func ApiHasMultiBuildpack() bool {
	return ApiGreaterThan("2.90.0")
}
