package packit

import (
	"bytes"
	"io"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

type Executable struct {
	name   string
	logger lager.Logger
}

type Execution struct {
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

func NewExecutable(name string, logger lager.Logger) Executable {
	return Executable{
		name:   name,
		logger: logger,
	}
}

func (e Executable) Execute(execution Execution) (string, string, error) {
	path, err := exec.LookPath(e.name)
	if err != nil {
		return "", "", err
	}

	cmd := exec.Command(path, execution.Args...)

	if execution.Dir != "" {
		cmd.Dir = execution.Dir
	}

	if len(execution.Env) > 0 {
		cmd.Env = execution.Env
	}

	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	if execution.Stdout != nil {
		cmd.Stdout = io.MultiWriter(stdout, execution.Stdout)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if execution.Stderr != nil {
		cmd.Stderr = io.MultiWriter(stderr, execution.Stderr)
	}

	data := lager.Data{"execution": execution, "path": path}
	session := e.logger.Session("execute", data)

	session.Debug("running")
	err = cmd.Run()
	if err != nil {
		session.Error("errored", err)
	}

	session.Debug("done")
	return stdout.String(), stderr.String(), err
}
