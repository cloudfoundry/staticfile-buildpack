package execution

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

type Options struct {
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

func (e Executable) Execute(options Options, args ...string) (string, string, error) {
	data := lager.Data{"options": options, "args": args, "path": e.name}
	session := e.logger.Session("execute", data)

	cmd := exec.Command(e.name, args...)

	if options.Dir != "" {
		cmd.Dir = options.Dir
	}

	if len(options.Env) > 0 {
		cmd.Env = options.Env
	}

	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	if options.Stdout != nil {
		cmd.Stdout = io.MultiWriter(stdout, options.Stdout)
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if options.Stderr != nil {
		cmd.Stderr = io.MultiWriter(stderr, options.Stderr)
	}

	session.Debug("running")
	err := cmd.Run()
	if err != nil {
		session.Error("errored", err)
	}

	session.Debug("done")
	return stdout.String(), stderr.String(), err
}
