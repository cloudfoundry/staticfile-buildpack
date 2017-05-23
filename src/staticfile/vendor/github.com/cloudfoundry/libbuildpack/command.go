package libbuildpack

import (
	"io"
	"os/exec"
)

type Command struct {
}

func (c *Command) Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error {
	cmd := exec.Command(program, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = dir

	return cmd.Run()
}
