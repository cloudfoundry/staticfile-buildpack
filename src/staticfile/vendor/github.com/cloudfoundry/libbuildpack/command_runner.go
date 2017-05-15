package libbuildpack

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

type CommandRunner interface {
	SetOutput(io.Writer)
	SetStdout(io.Writer)
	SetStderr(io.Writer)
	SetDir(string)
	Reset()
	ResetOutput()
	Run(program string, args ...string) error
	CaptureOutput(program string, args ...string) (string, error)
	CaptureStdout(program string, args ...string) (string, error)
	CaptureStderr(program string, args ...string) (string, error)
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
}

type commandRunner struct {
	dir    string
	stdout io.Writer
	stderr io.Writer
}

func NewCommandRunner() CommandRunner {
	return &commandRunner{
		dir:    "",
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func (c *commandRunner) SetOutput(output io.Writer) {
	c.SetStderr(output)
	c.SetStdout(output)
}

func (c *commandRunner) SetStderr(output io.Writer) {
	c.stderr = output
}
func (c *commandRunner) SetStdout(output io.Writer) {
	c.stdout = output
}
func (c *commandRunner) SetDir(dir string) {
	c.dir = dir
}

func (c *commandRunner) Run(program string, args ...string) error {
	cmd := exec.Command(program, args...)
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	cmd.Dir = c.dir

	return cmd.Run()
}

func (c *commandRunner) CaptureOutput(program string, args ...string) (string, error) {
	output := new(bytes.Buffer)
	c.SetOutput(output)
	defer c.ResetOutput()

	err := c.Run(program, args...)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

func (c *commandRunner) CaptureStdout(program string, args ...string) (string, error) {
	output := new(bytes.Buffer)
	c.SetStdout(output)
	c.SetStderr(ioutil.Discard)
	defer c.ResetOutput()

	err := c.Run(program, args...)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

func (c *commandRunner) CaptureStderr(program string, args ...string) (string, error) {
	output := new(bytes.Buffer)
	c.SetStdout(ioutil.Discard)
	c.SetStderr(output)
	defer c.ResetOutput()

	err := c.Run(program, args...)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

func (c *commandRunner) Reset() {
	c.dir = ""
	c.stdout = os.Stdout
	c.stderr = os.Stderr
}

func (c *commandRunner) ResetOutput() {
	c.stdout = os.Stdout
	c.stderr = os.Stderr
}

func (c *commandRunner) Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error {
	cmd := exec.Command(program, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = dir

	return cmd.Run()
}
