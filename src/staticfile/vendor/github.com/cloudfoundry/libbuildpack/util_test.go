package libbuildpack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("ExtractZip", func() {
		var (
			tmpdir   string
			err      error
			fileInfo os.FileInfo
		)
		BeforeEach(func() {
			tmpdir, err = ioutil.TempDir("", "exploded")
			Expect(err).To(BeNil())
		})
		AfterEach(func() { err = os.RemoveAll(tmpdir); Expect(err).To(BeNil()) })

		Context("with a valid zip file", func() {
			It("extracts a file at the root", func() {
				err = libbuildpack.ExtractZip("fixtures/thing.zip", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "root.txt")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "root.txt"))).To(Equal([]byte("root\n")))
			})

			It("extracts a nested file", func() {
				err = libbuildpack.ExtractZip("fixtures/thing.zip", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
			})

			It("preserves the file mode", func() {
				err = libbuildpack.ExtractZip("fixtures/thing.zip", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
				fileInfo, err = os.Stat(filepath.Join(tmpdir, "thing", "bin", "file2.exe"))

				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0755)))
			})
		})

		Context("with a missing zip file", func() {
			It("returns an error", func() {
				err = libbuildpack.ExtractZip("fixtures/notexist.zip", tmpdir)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("with an invalid zip file", func() {
			It("returns an error", func() {
				err = libbuildpack.ExtractZip("fixtures/manifest.yml", tmpdir)
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("GetBuildpackDir", func() {
		var (
			parentDir string
			testBpDir string
			oldBpDir  string
			err       error
		)

		BeforeEach(func() {
			parentDir, err = filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), ".."))
			Expect(err).To(BeNil())
		})

		JustBeforeEach(func() {
			oldBpDir = os.Getenv("BUILDPACK_DIR")
			err = os.Setenv("BUILDPACK_DIR", testBpDir)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.Setenv("BUILDPACK_DIR", oldBpDir)
			Expect(err).To(BeNil())
		})

		Context("BUILDPACK_DIR is set", func() {
			BeforeEach(func() {
				testBpDir = "buildpack_root_directory"
			})
			It("returns the value for BUILDPACK_DIR", func() {
				dir, err := libbuildpack.GetBuildpackDir()
				Expect(err).To(BeNil())
				Expect(dir).To(Equal("buildpack_root_directory"))
			})
		})
		Context("BUILDPACK_DIR is not set", func() {
			BeforeEach(func() {
				testBpDir = ""
			})
			It("returns the parent of the directory containing the executable", func() {
				dir, err := libbuildpack.GetBuildpackDir()
				Expect(err).To(BeNil())
				Expect(dir).To(Equal(parentDir))
			})
		})
	})

	Describe("Untar", func() {
		var (
			tmpdir   string
			err      error
			fileInfo os.FileInfo
		)
		BeforeEach(func() {
			tmpdir, err = ioutil.TempDir("", "exploded")
			Expect(err).To(BeNil())
		})
		AfterEach(func() { err = os.RemoveAll(tmpdir); Expect(err).To(BeNil()) })

		Context("with a valid tar file", func() {
			It("extracts a file at the root", func() {
				err = libbuildpack.ExtractTarGz("fixtures/thing.tgz", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "root.txt")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "root.txt"))).To(Equal([]byte("root\n")))
			})
			It("extracts a nested file", func() {
				err = libbuildpack.ExtractTarGz("fixtures/thing.tgz", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "thing", "bin", "file2.exe"))).To(Equal([]byte("progam2\n")))
			})
			It("preserves the file mode", func() {
				err = libbuildpack.ExtractTarGz("fixtures/thing.tgz", tmpdir)
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "thing", "bin", "file2.exe")).To(BeAnExistingFile())
				fileInfo, err = os.Stat(filepath.Join(tmpdir, "thing", "bin", "file2.exe"))

				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0755)))
			})
			It("handles symlinks", func() {
				err = libbuildpack.ExtractTarGz("fixtures/symlink.tgz", tmpdir)
				Expect(err).To(BeNil())

				path := filepath.Join(tmpdir, "other", "file.txt")
				Expect(path).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(path)).To(Equal([]byte("content\n")))

				fi, err := os.Lstat(path)
				Expect(err).To(BeNil())
				Expect(fi.Mode() & os.ModeSymlink).ToNot(Equal(0))
			})
		})

		Context("with a missing tar file", func() {
			It("returns an error", func() {
				err = libbuildpack.ExtractTarGz("fixtures/notexist.tgz", tmpdir)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("with an invalid tar file", func() {
			It("returns an error", func() {
				err = libbuildpack.ExtractTarGz("fixtures/manifest.yml", tmpdir)
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("CopyFile", func() {
		var (
			tmpdir   string
			err      error
			fileInfo os.FileInfo
			oldMode  os.FileMode
			oldUmask int
		)
		BeforeEach(func() {
			var fh *os.File
			sourceFile := "fixtures/source.txt"

			tmpdir, err = ioutil.TempDir("", "copy")
			Expect(err).To(BeNil())

			fileInfo, err = os.Stat(sourceFile)
			Expect(err).To(BeNil())
			oldMode = fileInfo.Mode()

			fh, err = os.Open(sourceFile)
			Expect(err).To(BeNil())
			defer fh.Close()

			err = fh.Chmod(0742)
			Expect(err).To(BeNil())

			oldUmask = syscall.Umask(0000)

		})
		AfterEach(func() {
			var fh *os.File
			sourceFile := "fixtures/source.txt"

			fh, err = os.Open(sourceFile)
			Expect(err).To(BeNil())
			defer fh.Close()

			err = fh.Chmod(oldMode)
			Expect(err).To(BeNil())

			err = os.RemoveAll(tmpdir)
			Expect(err).To(BeNil())

			syscall.Umask(oldUmask)
		})

		Context("with a valid source file", func() {
			It("copies the file", func() {
				err = libbuildpack.CopyFile("fixtures/source.txt", filepath.Join(tmpdir, "out.txt"))
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "out.txt")).To(BeAnExistingFile())
				Expect(ioutil.ReadFile(filepath.Join(tmpdir, "out.txt"))).To(Equal([]byte("a file\n")))
			})

			It("preserves the file mode", func() {
				err = libbuildpack.CopyFile("fixtures/source.txt", filepath.Join(tmpdir, "out.txt"))
				Expect(err).To(BeNil())

				Expect(filepath.Join(tmpdir, "out.txt")).To(BeAnExistingFile())
				fileInfo, err = os.Stat(filepath.Join(tmpdir, "out.txt"))

				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0742)))
			})
		})

		Context("with a missing file", func() {
			It("returns an error", func() {
				err = libbuildpack.ExtractTarGz("fixtures/notexist.txt", tmpdir)
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("CopyDirectory", func() {
		var (
			destDir string
			err     error
		)

		BeforeEach(func() {
			destDir, err = ioutil.TempDir("", "destDir")
			Expect(err).To(BeNil())
		})

		It("copies source to destination", func() {
			srcDir := filepath.Join("fixtures", "copydir")
			err = libbuildpack.CopyDirectory(srcDir, destDir)
			Expect(err).To(BeNil())

			Expect(filepath.Join(srcDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "standard", "manifest.yml")).To(BeAnExistingFile())

			Expect(filepath.Join(destDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "standard", "manifest.yml")).To(BeAnExistingFile())
		})

		It("handles symlink to directory", func() {
			srcDir := filepath.Join("fixtures", "copydir_symlinks")
			err = libbuildpack.CopyDirectory(srcDir, destDir)
			Expect(err).To(BeNil())

			Expect(filepath.Join(srcDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "sym_source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "standard", "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(srcDir, "sym_standard", "manifest.yml")).To(BeAnExistingFile())

			Expect(filepath.Join(destDir, "source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "sym_source.txt")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "standard", "manifest.yml")).To(BeAnExistingFile())
			Expect(filepath.Join(destDir, "sym_standard", "manifest.yml")).To(BeAnExistingFile())
		})
	})

	Describe("FileExists", func() {
		Context("the file exists", func() {
			var (
				dir string
				err error
			)

			BeforeEach(func() {
				dir, err = ioutil.TempDir("", "file.exists")
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				err = os.RemoveAll(dir)
				Expect(err).To(BeNil())
			})

			It("returns true", func() {
				exists, err := libbuildpack.FileExists(dir)
				Expect(err).To(BeNil())
				Expect(exists).To(BeTrue())
			})
		})

		Context("the file does not exist", func() {
			It("returns false", func() {
				exists, err := libbuildpack.FileExists("not/exist")
				Expect(err).To(BeNil())
				Expect(exists).To(BeFalse())
			})
		})

	})
})
