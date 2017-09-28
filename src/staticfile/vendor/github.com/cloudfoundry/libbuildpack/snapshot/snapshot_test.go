package snapshot_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/snapshot"
	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=snapshot.go --destination=mocks_test.go --package=snapshot_test

var _ = Describe("Snapshot", func() {
	var (
		origBPDebug string
		tmpDir      string
		mockCtrl    *gomock.Controller
		mockLogger  *MockLogger
		err         error
	)

	BeforeEach(func() {
		origBPDebug = os.Getenv("BP_DEBUG")

		tmpDir, err = ioutil.TempDir("", "libbuildpack.snapshot.build.")
		Expect(err).To(BeNil())

		Expect(ioutil.WriteFile(filepath.Join(tmpDir, "Gemfile"), []byte("source \"https://rubygems.org\"\r\ngem \"rack\"\r\n"), 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(tmpDir, "other"), []byte("other"), 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(tmpDir, "dir"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(tmpDir, "dir", "other"), []byte("other"), 0644)).To(Succeed())
		Expect(os.Symlink(filepath.Join(tmpDir, "Gemfile"), filepath.Join(tmpDir, "mySymLink"))).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(tmpDir, "myEmptyDir"), 0755)).To(Succeed())

		mockCtrl = gomock.NewController(GinkgoT())
		mockLogger = NewMockLogger(mockCtrl)
	})

	AfterEach(func() {
		os.Setenv("BP_DEBUG", origBPDebug)

		mockCtrl.Finish()

		err = os.RemoveAll(tmpDir)
		Expect(err).To(BeNil())
	})

	Context("BP_DEBUG is set", func() {
		BeforeEach(func() {
			os.Setenv("BP_DEBUG", "1")
		})

		It("Initially logs an MD5 of the full contents", func() {
			mockLogger.EXPECT().Debug("Initial dir checksum %s", "6fac581b4f82368eb477e733400db4b7")
			snapshot.Dir(tmpDir, mockLogger)
		})

		Context(".cloudfoundry directory", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, ".cloudfoundry", "dir"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, ".cloudfoundry", "other"), []byte("other"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, ".cloudfoundry", "dir", "other"), []byte("other"), 0644)).To(Succeed())
			})

			It("excludes .cloudfoundry directory in the checksum calcuation", func() {
				mockLogger.EXPECT().Debug("Initial dir checksum %s", "6fac581b4f82368eb477e733400db4b7")
				snapshot.Dir(tmpDir, mockLogger)
			})
		})

		Describe("Diff()", func() {
			var dirSnapshot *snapshot.DirSnapshot
			BeforeEach(func() {
				mockLogger.EXPECT().Debug("Initial dir checksum %s", "6fac581b4f82368eb477e733400db4b7")
				dirSnapshot = snapshot.Dir(tmpDir, mockLogger)
				time.Sleep(1 * time.Second) //TODO: don't rely on `find`
			})

			Context("when a directory is added", func() {
				BeforeEach(func() {
					Expect(os.MkdirAll(filepath.Join(tmpDir, "myNewDir"), 0755)).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "2c5c0c34c2311ddc82a6f10a241f0bfb")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./myNewDir")
					dirSnapshot.Diff()
				})
			})

			Context("when a directory is removed", func() {
				BeforeEach(func() {
					Expect(os.Remove(filepath.Join(tmpDir, "myEmptyDir"))).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "23015917a95569c120e6daf29e2d49d6")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./myEmptyDir")
					dirSnapshot.Diff()
				})
			})

			Context("when a symlink is added", func() {
				BeforeEach(func() {
					Expect(os.Symlink(filepath.Join(tmpDir, "Gemfile"), filepath.Join(tmpDir, "myNewSymLink"))).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "50e11a9949ef9345b8786bfbbcee836e")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./myNewSymLink")
					dirSnapshot.Diff()
				})
			})

			Context("when a symlink is removed", func() {
				BeforeEach(func() {
					Expect(os.Remove(filepath.Join(tmpDir, "mySymLink"))).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "7fda572908dba44c22a193b46be0b792")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./mySymLink")
					dirSnapshot.Diff()
				})
			})

			Context("when a symlink is changed to point elsewhere", func() {
				BeforeEach(func() {
					Expect(os.Remove(filepath.Join(tmpDir, "mySymLink"))).To(Succeed())
					Expect(os.Symlink(filepath.Join(tmpDir, "other"), filepath.Join(tmpDir, "mySymLink"))).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "a640f5e0955c19ab97e6e13311d233c7")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./mySymLink")
					dirSnapshot.Diff()
				})
			})

			Context("when a file is added", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(tmpDir, "extrafile"), []byte("extrafile"), 0644)).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "b80ec5942c0a305c6c00c61468986526")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./extrafile")
					dirSnapshot.Diff()
				})
			})

			Context("when a file is removed", func() {
				BeforeEach(func() {
					Expect(os.Remove(filepath.Join(tmpDir, "other"))).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "4343e8dc600a5771210ac833699258fc")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./other")
					dirSnapshot.Diff()
				})
			})

			Context("when a files contents have changed", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(tmpDir, "other"), []byte("other other"), 0644)).To(Succeed())
				})

				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("New dir checksum is %s", "1545570abe7d2068ae20085e63f89bc5")
					mockLogger.EXPECT().Debug("paths changed:")
					mockLogger.EXPECT().Debug("./other")
					dirSnapshot.Diff()
				})
			})

			Context("when no changes have been made", func() {
				It("logs the new MD5 sum as well as changed paths", func() {
					mockLogger.EXPECT().Debug("Dir checksum unchanged")
					dirSnapshot.Diff()
				})
			})
		})
	})

	Context("BP_DEBUG is not set", func() {
		BeforeEach(func() {
			os.Setenv("BP_DEBUG", "")
		})

		It("Does not log anything", func() {
			snapshot.Dir(tmpDir, mockLogger)
		})

		Describe("Diff()", func() {
			var dirSnapshot *snapshot.DirSnapshot
			BeforeEach(func() {
				dirSnapshot = snapshot.Dir(tmpDir, mockLogger)
				Expect(ioutil.WriteFile(filepath.Join(tmpDir, "other"), []byte("other other"), 0644)).To(Succeed())
			})
			It("Does not log anything", func() {
				dirSnapshot.Diff()
			})
		})
	})

})
