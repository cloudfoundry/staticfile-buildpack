package libbuildpack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("YAML", func() {
	var (
		yaml   *libbuildpack.YAML
		tmpDir string
		err    error
	)

	BeforeEach(func() {
		tmpDir, err = ioutil.TempDir("", "yaml")
		Expect(err).To(BeNil())

		yaml = &libbuildpack.YAML{}
	})

	AfterEach(func() {
		err = os.RemoveAll(tmpDir)
		Expect(err).To(BeNil())
	})

	Describe("Load", func() {
		Context("file is valid yaml", func() {
			BeforeEach(func() {
				ioutil.WriteFile(filepath.Join(tmpDir, "valid.yml"), []byte("key: value"), 0666)
			})
			It("returns an error", func() {
				obj := make(map[string]string)
				err = yaml.Load(filepath.Join(tmpDir, "valid.yml"), &obj)

				Expect(err).To(BeNil())
				Expect(obj["key"]).To(Equal("value"))
			})
		})

		Context("file is NOT valid yaml", func() {
			BeforeEach(func() {
				ioutil.WriteFile(filepath.Join(tmpDir, "invalid.yml"), []byte("not valid yml"), 0666)
			})
			It("returns an error", func() {
				obj := make(map[string]string)
				err = yaml.Load(filepath.Join(tmpDir, "invalid.yml"), &obj)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("file does not exist", func() {
			It("returns an error", func() {
				obj := make(map[string]string)
				err = yaml.Load(filepath.Join(tmpDir, "does_not_exist.yml"), &obj)
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("Write", func() {
		Context("directory exists", func() {
			It("writes the yaml to a file ", func() {
				obj := map[string]string{
					"key": "val",
				}
				err = yaml.Write(filepath.Join(tmpDir, "file.yml"), obj)
				Expect(err).To(BeNil())

				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "file.yml"))).To(Equal([]byte("key: val\n")))
			})
		})

		Context("directory does not exist", func() {
			It("creates the directory ", func() {
				obj := map[string]string{
					"key": "val",
				}
				err = yaml.Write(filepath.Join(tmpDir, "extradir", "file.yml"), obj)
				Expect(err).To(BeNil())

				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "extradir", "file.yml"))).To(Equal([]byte("key: val\n")))
			})
		})
	})
})
