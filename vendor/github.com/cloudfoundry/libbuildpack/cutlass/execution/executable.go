package execution

import (
	"bytes"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

type Executable struct {
	name   string
	logger lager.Logger
}

type Options struct {
	Dir string
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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := exec.Command(e.name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if options.Dir != "" {
		cmd.Dir = options.Dir
	}

	session.Debug("running")
	err := cmd.Run()
	if err != nil {
		session.Error("errored", err)
	}

	session.Debug("done")
	return stdout.String(), stderr.String(), err
}
