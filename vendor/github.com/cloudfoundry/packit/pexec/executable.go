package pexec

import (
	"io"
	"os/exec"
)

type Executable struct {
	name string
}

type Execution struct {
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

func NewExecutable(name string) Executable {
	return Executable{
		name: name,
	}
}

func (e Executable) Execute(execution Execution) error {
	path, err := exec.LookPath(e.name)
	if err != nil {
		return err
	}

	cmd := exec.Command(path, execution.Args...)

	if execution.Dir != "" {
		cmd.Dir = execution.Dir
	}

	if len(execution.Env) > 0 {
		cmd.Env = execution.Env
	}

	cmd.Stdout = execution.Stdout
	cmd.Stderr = execution.Stderr

	return cmd.Run()
}
