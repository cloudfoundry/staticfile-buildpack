package docker

import (
	"bytes"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(options ExecuteOptions, args ...string) (stdout, stderr string, err error)
}

type DockerExecutable struct {
	name   string
	logger lager.Logger
}

func NewDockerExecutable(logger lager.Logger) DockerExecutable {
	logger = logger.Session("docker.executable")

	return DockerExecutable{
		name:   "docker",
		logger: logger,
	}
}

type ExecuteOptions struct {
	Dir string
}

func (de DockerExecutable) Execute(options ExecuteOptions, args ...string) (string, string, error) {
	data := lager.Data{"options": options, "args": args}
	session := de.logger.Session("execute", data)

	stdout := bytes.NewBuffer([]byte{})
	stderr := bytes.NewBuffer([]byte{})

	command := exec.Command(de.name, args...)
	command.Stdout = stdout
	command.Stderr = stderr

	if options.Dir != "" {
		command.Dir = options.Dir
	}

	session.Debug("running", lager.Data{"path": de.name})
	err := command.Run()
	if err != nil {
		return "", "", err
	}

	return stdout.String(), stderr.String(), nil
}
