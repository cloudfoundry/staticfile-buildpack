package packager_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Scaffold", func() {
	var (
		buildpackDir string
		version      string
		cacheDir     string
	)

	BeforeEach(func() {
		var err error
		fmt.Print("\n\n **Remember**: If you have changed files in scaffold directory, be sure to run go generate.\n\n")
		buildpackDir = "./fixtures/good"
		cacheDir, err = ioutil.TempDir("", "packager-cachedir")
		Expect(err).To(BeNil())
		version = fmt.Sprintf("1.23.45.%s", time.Now().Format("20060102150405"))

		httpmock.Reset()
	})

	Describe("Init", func() {
		var baseDir string
		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "scaffold-basedir")
			Expect(err).To(BeNil())

			// run the code under test
			Expect(packager.Scaffold(filepath.Join(baseDir, "bpdir"), "mylanguage")).To(Succeed())
		})
		AfterEach(func() {
			os.RemoveAll(baseDir)
		})

		checkfileexists := func(path string) func() {
			return func() {
				Expect(libbuildpack.FileExists(filepath.Join(baseDir, path))).To(BeTrue())
			}
		}

		It("Creates all of the files", func() {
			// top-level directories
			By("creates a named directory", checkfileexists("bpdir"))
			By("creates a bin directory", checkfileexists("bpdir/bin"))
			By("creates a scripts directory", checkfileexists("bpdir/scripts"))
			By("creates a src directory", checkfileexists("bpdir/src"))
			By("creates a fixtures directory", checkfileexists("bpdir/fixtures"))

			// top-level files
			By("creates a .envrc file", checkfileexists("bpdir/.envrc"))
			By("creates a .envrc file", checkfileexists("bpdir/.gitignore"))
			By("creates a manifest.yml file", checkfileexists("bpdir/manifest.yml"))
			By("creates a VERSION file", checkfileexists("bpdir/VERSION"))
			By("creates a README file", checkfileexists("bpdir/README.md"))

			// bin directory files
			By("creates a detect script", checkfileexists("bpdir/bin/detect"))
			By("creates a compile script", checkfileexists("bpdir/bin/compile"))
			By("creates a supply script", checkfileexists("bpdir/bin/supply"))
			By("creates a finalize script", checkfileexists("bpdir/bin/finalize"))
			By("creates a release script", checkfileexists("bpdir/bin/release"))

			// scripts directory files
			By("creates a brats test script", checkfileexists("bpdir/scripts/brats.sh"))
			By("creates a build script", checkfileexists("bpdir/scripts/build.sh"))
			By("creates a install_go script", checkfileexists("bpdir/scripts/install_go.sh"))
			By("creates a install_tools script", checkfileexists("bpdir/scripts/install_tools.sh"))
			By("creates a integration test script", checkfileexists("bpdir/scripts/integration.sh"))
			By("creates a unit test script", checkfileexists("bpdir/scripts/unit.sh"))

			By("creates a Gopkg.toml", checkfileexists("bpdir/src/mylanguage/Gopkg.toml"))

			// src/supply files
			By("creates a supply src directory", checkfileexists("bpdir/src/mylanguage/supply"))
			By("creates a supply src file", checkfileexists("bpdir/src/mylanguage/supply/supply.go"))
			By("creates a supply test file", checkfileexists("bpdir/src/mylanguage/supply/supply_test.go"))
			By("creates a supply cli src file", checkfileexists("bpdir/src/mylanguage/supply/cli/main.go"))

			// src/finalize files
			By("creates a finalize src directory", checkfileexists("bpdir/src/mylanguage/finalize"))
			By("creates a finalize src file", checkfileexists("bpdir/src/mylanguage/finalize/finalize.go"))
			By("creates a finalize test file", checkfileexists("bpdir/src/mylanguage/finalize/finalize.go"))
			By("creates a finalize cli src file", checkfileexists("bpdir/src/mylanguage/finalize/cli/main.go"))

			By("creating unit tests that pass", func() {
				//TODO: this is an integration test; move this into integration
				command := exec.Command("./scripts/unit.sh")
				command.Dir = filepath.Join(baseDir, "bpdir")
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session, 1*time.Minute).Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring("Supply Suite"))
				Expect(string(session.Out.Contents())).To(ContainSubstring("Finalize Suite"))
			})
		})
	})

	Describe("Upgrade", func() {
		var baseDir string
		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "scaffold-basedir")
			Expect(err).To(BeNil())

			Expect(libbuildpack.CopyDirectory("fixtures/modified", baseDir)).To(Succeed())
		})
		AfterEach(func() {
			os.RemoveAll(baseDir)
		})

		Context("Force flag not set", func() {
			BeforeEach(func() {
				Expect(packager.Upgrade(baseDir, false)).To(Succeed())
			})
			It("updates files user has NOT modified", func() {
				file := "src/mylanguage/supply/supply_test.go"
				current, err := ioutil.ReadFile(filepath.Join(baseDir, file))
				Expect(err).ToNot(HaveOccurred())
				previous, err := ioutil.ReadFile(filepath.Join("fixtures", "modified", file))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(current)).ToNot(Equal(string(previous)))
			})
			It("leaves files user HAS modified", func() {
				file := "src/mylanguage/supply/supply.go"
				current, err := ioutil.ReadFile(filepath.Join(baseDir, file))
				Expect(err).ToNot(HaveOccurred())
				previous, err := ioutil.ReadFile(filepath.Join("fixtures", "modified", file))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(current)).To(Equal(string(previous)))
			})
		})

		Context("force flag is set", func() {
			BeforeEach(func() {
				Expect(packager.Upgrade(baseDir, true)).To(Succeed())
			})
			It("updates files user has NOT modified", func() {
				file := "src/mylanguage/supply/supply_test.go"
				current, err := ioutil.ReadFile(filepath.Join(baseDir, file))
				Expect(err).ToNot(HaveOccurred())
				previous, err := ioutil.ReadFile(filepath.Join("fixtures", "modified", file))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(current)).ToNot(Equal(string(previous)))
			})
			It("updates files user HAS modified", func() {
				file := "src/mylanguage/supply/supply.go"
				current, err := ioutil.ReadFile(filepath.Join(baseDir, file))
				Expect(err).ToNot(HaveOccurred())
				previous, err := ioutil.ReadFile(filepath.Join("fixtures", "modified", file))
				Expect(err).ToNot(HaveOccurred())

				Expect(string(current)).ToNot(Equal(string(previous)))
			})
		})

	})
})
