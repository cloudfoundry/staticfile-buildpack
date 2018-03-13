package checksum_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/checksum"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Checksum", func() {
	var (
		dir   string
		lines []string
	)

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "checksum")
		Expect(err).To(BeNil())

		Expect(os.MkdirAll(filepath.Join(dir, "a/b"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(dir, "a/b", "file"), []byte("hi"), 0644)).To(Succeed())

		lines = []string{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	debug := func(format string, args ...interface{}) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}

	Describe("Do", func() {
		Context("Directory is unchanged", func() {
			It("Reports the current directory checksum", func() {
				exec := func() error { return nil }
				Expect(checksum.Do(dir, debug, exec)).To(Succeed())
				Expect(lines).To(Equal([]string{
					"Checksum Before (" + dir + "): 3e673106d28d587c5c01b3582bf15a50",
					"Checksum After (" + dir + "): 3e673106d28d587c5c01b3582bf15a50",
				}))
			})
		})

		Context("a file is changed", func() {
			It("Reports the current directory checksum", func() {
				exec := func() error {
					time.Sleep(10 * time.Millisecond)
					return ioutil.WriteFile(filepath.Join(dir, "a/b", "file"), []byte("bye"), 0644)
				}
				Expect(checksum.Do(dir, debug, exec)).To(Succeed())
				Expect(lines).To(Equal([]string{
					"Checksum Before (" + dir + "): 3e673106d28d587c5c01b3582bf15a50",
					"Checksum After (" + dir + "): e01956670269656ae69872c0672592ae",
					"Below files changed:",
					"./a/b/file\n",
				}))
			})
		})

		Context("a file is added", func() {
			It("Reports the current directory checksum", func() {
				exec := func() error {
					time.Sleep(10 * time.Millisecond)
					return ioutil.WriteFile(filepath.Join(dir, "a", "file"), []byte("new file"), 0644)
				}
				Expect(checksum.Do(dir, debug, exec)).To(Succeed())
				Expect(lines).To(Equal([]string{
					"Checksum Before (" + dir + "): 3e673106d28d587c5c01b3582bf15a50",
					"Checksum After (" + dir + "): 9fc7505dc69734c5d40c38a35017e1dc",
					"Below files changed:",
					"./a\n./a/file\n",
				}))
			})
		})

		Context("when exec returns an error", func() {
			It("Returns an error", func() {
				exec := func() error {
					return errors.New("some error")
				}
				Expect(checksum.Do(dir, debug, exec)).To(MatchError("some error"))
			})
		})
	})
})
