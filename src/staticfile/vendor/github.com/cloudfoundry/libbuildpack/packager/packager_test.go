package packager_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/packager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("Packager", func() {
	var (
		buildpackDir string
		version      string
		cacheDir     string
	)

	BeforeEach(func() {
		var err error
		buildpackDir = "./fixtures/good"
		cacheDir, err = ioutil.TempDir("", "packager-cachedir")
		Expect(err).To(BeNil())
		version = fmt.Sprintf("1.23.45.%s", time.Now().Format("20060102150405"))

		httpmock.Reset()
	})

	Describe("Package", func() {
		var zipFile string
		var cached bool
		// AfterEach(func() { os.Remove(zipFile) })

		Context("uncached", func() {
			BeforeEach(func() { cached = false })
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).To(BeNil())
			})

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-v%s.zip", version))))
			})

			It("includes files listed in manifest.yml", func() {
				Expect(ZipContents(zipFile, "bin/filename")).To(Equal("awesome content"))
			})

			It("overrides VERSION", func() {
				Expect(ZipContents(zipFile, "VERSION")).To(Equal(version))
			})

			It("runs pre-package script", func() {
				Expect(ZipContents(zipFile, "hi.txt")).To(Equal("hi mom\n"))
			})

			It("does not include files not in list", func() {
				_, err := ZipContents(zipFile, "ignoredfile")
				Expect(err.Error()).To(HavePrefix("ignoredfile not found in"))
			})

			It("does not include dependencies", func() {
				_, err := ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")
				Expect(err.Error()).To(HavePrefix("dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt not found in"))
			})
		})

		Context("cached", func() {
			BeforeEach(func() { cached = true })
			JustBeforeEach(func() {
				var err error
				zipFile, err = packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).To(BeNil())
			})

			It("generates a zipfile with name", func() {
				dir, err := filepath.Abs(buildpackDir)
				Expect(err).To(BeNil())
				Expect(zipFile).To(Equal(filepath.Join(dir, fmt.Sprintf("ruby_buildpack-cached-v%s.zip", version))))
			})

			It("includes files listed in manifest.yml", func() {
				Expect(ZipContents(zipFile, "bin/filename")).To(Equal("awesome content"))
			})

			It("overrides VERSION", func() {
				Expect(ZipContents(zipFile, "VERSION")).To(Equal(version))
			})

			It("runs pre-package script", func() {
				Expect(ZipContents(zipFile, "hi.txt")).To(Equal("hi mom\n"))
			})

			It("does not include files not in list", func() {
				_, err := ZipContents(zipFile, "ignoredfile")
				Expect(err.Error()).To(HavePrefix("ignoredfile not found in"))
			})

			It("includes dependencies", func() {
				Expect(ZipContents(zipFile, "dependencies/d39cae561ec1f485d1a4a58304e87105/rfc2324.txt")).To(ContainSubstring("Hyper Text Coffee Pot Control Protocol"))
			})
		})

		Context("cached dependency has wrong md5", func() {
			BeforeEach(func() {
				cached = true
				buildpackDir = "./fixtures/bad"
			})
			It("includes dependencies", func() {
				_, err := packager.Package(buildpackDir, cacheDir, version, cached)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("dependency md5 mismatch: expected md5 fffffff, actual md5 84fc21c1adb2f0441c357a943ac464bc"))
			})
		})
	})
})
