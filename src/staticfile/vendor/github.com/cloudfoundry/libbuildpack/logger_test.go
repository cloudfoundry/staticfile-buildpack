package libbuildpack_test

import (
	"bytes"
	"os"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {

	var (
		err    error
		logger *libbuildpack.Logger
		buffer *bytes.Buffer
	)

	BeforeEach(func() {
		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(buffer)
	})

	Describe("Debug", func() {
		var (
			bpDebug    string
			oldBpDebug string
		)

		JustBeforeEach(func() {
			oldBpDebug = os.Getenv("BP_DEBUG")
			err = os.Setenv("BP_DEBUG", bpDebug)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			err = os.Setenv("BP_DEBUG", oldBpDebug)
			Expect(err).To(BeNil())
		})

		Context("BP_DEBUG is set", func() {
			BeforeEach(func() {
				bpDebug = "true"
			})

			It("Logs the message", func() {
				logger.Debug("detailed info")
				Expect(buffer.String()).To(ContainSubstring("\033[34;1mDEBUG:\033[0m detailed info"))
			})
		})

		Context("BP_DEBUG is not set", func() {
			BeforeEach(func() {
				bpDebug = ""
			})

			It("Does not log the message", func() {
				logger.Debug("detailed info")
				Expect(buffer.String()).To(Equal(""))
			})
		})
	})
})
