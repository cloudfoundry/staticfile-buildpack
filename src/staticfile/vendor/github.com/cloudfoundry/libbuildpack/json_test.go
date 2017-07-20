package libbuildpack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JSON", func() {
	var (
		json   *libbuildpack.JSON
		tmpDir string
		err    error
	)

	BeforeEach(func() {
		tmpDir, err = ioutil.TempDir("", "json")
		Expect(err).To(BeNil())

		json = &libbuildpack.JSON{}
	})

	AfterEach(func() {
		err = os.RemoveAll(tmpDir)
		Expect(err).To(BeNil())
	})

	Describe("Load", func() {
		Context("file is valid json", func() {
			BeforeEach(func() {
				ioutil.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte(`{"key": "value"}`), 0666)
			})
			It("returns an error", func() {
				obj := make(map[string]string)
				err = json.Load(filepath.Join(tmpDir, "valid.json"), &obj)

				Expect(err).To(BeNil())
				Expect(obj["key"]).To(Equal("value"))
			})
		})

		Context("file is NOT valid json", func() {
			BeforeEach(func() {
				ioutil.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("not valid json"), 0666)
			})
			It("returns an error", func() {
				obj := make(map[string]string)
				err = json.Load(filepath.Join(tmpDir, "invalid.json"), &obj)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("file does not exist", func() {
			It("returns an error", func() {
				obj := make(map[string]string)
				err = json.Load(filepath.Join(tmpDir, "does_not_exist.json"), &obj)
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("Write", func() {
		Context("directory exists", func() {
			It("writes the json to a file ", func() {
				obj := map[string]string{
					"key": "val",
				}
				err = json.Write(filepath.Join(tmpDir, "file.json"), obj)
				Expect(err).To(BeNil())

				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "file.json"))).To(Equal([]byte(`{"key":"val"}`)))
			})
		})

		Context("directory does not exist", func() {
			It("creates the directory", func() {
				obj := map[string]string{
					"key": "val",
				}
				err = json.Write(filepath.Join(tmpDir, "extradir", "file.json"), obj)
				Expect(err).To(BeNil())

				Expect(ioutil.ReadFile(filepath.Join(tmpDir, "extradir", "file.json"))).To(Equal([]byte(`{"key":"val"}`)))
			})
		})
	})
})
