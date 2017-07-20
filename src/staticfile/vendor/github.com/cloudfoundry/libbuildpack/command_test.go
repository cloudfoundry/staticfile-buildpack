package libbuildpack_test

import (
	"bytes"
	"os/exec"

	bp "github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {
	var (
		buffer *bytes.Buffer
		exe    string
		args   []string
		cmd    bp.Command
	)

	BeforeEach(func() {
		buffer = new(bytes.Buffer)
	})

	Context("valid command", func() {
		BeforeEach(func() {
			exe = "ls"
			args = []string{"-l", "fixtures"}
		})

		It("runs the command with the output in the right location", func() {
			err := cmd.Execute("", buffer, buffer, exe, args...)
			Expect(err).To(BeNil())

			Expect(buffer.String()).To(ContainSubstring("thing.tgz"))
		})
	})
	Context("changing directory", func() {
		BeforeEach(func() {
			exe = "pwd"
			args = []string{}
		})

		It("runs the command with the output in the right location", func() {
			err := cmd.Execute("fixtures", buffer, buffer, exe, args...)
			Expect(err).To(BeNil())

			Expect(buffer.String()).To(ContainSubstring("libbuildpack/fixtures"))
		})
	})

	Context("invalid command", func() {
		BeforeEach(func() {
			exe = "ls"
			args = []string{"-l", "not/a/dir"}

		})

		It("runs the command and returns an eror", func() {
			err := cmd.Execute("", buffer, buffer, exe, args...)
			Expect(err).NotTo(BeNil())
			_, ok := err.(*exec.ExitError)
			Expect(ok).To(BeTrue())

			Expect(buffer.String()).To(ContainSubstring("No such file or directory"))
		})
	})
})
