package glow

import (
	"bytes"

	"github.com/cloudfoundry/packit/pexec"
)

const ExecutableName = "cnb2cf"

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

type CLI struct {
	executable Executable
}

func NewCLI(executable Executable) CLI {
	return CLI{
		executable: executable,
	}
}

type PackageOptions struct {
	Cached       bool
	Dev          bool
	ManifestPath string
	Version      string
}

func (c CLI) Package(dir, stack string, options PackageOptions) (string, string, error) {
	execution := pexec.Execution{
		Args: []string{"package", "-stack", stack},
		Dir:  dir,
	}

	if options.Cached {
		execution.Args = append(execution.Args, "-cached")
	}

	if options.Dev {
		execution.Args = append(execution.Args, "-dev")
	}

	if options.ManifestPath != "" {
		execution.Args = append(execution.Args, "-manifestpath", options.ManifestPath)
	}

	if options.Version != "" {
		execution.Args = append(execution.Args, "-version", options.Version)
	}

	stdout := bytes.NewBuffer(nil)
	execution.Stdout = stdout

	stderr := bytes.NewBuffer(nil)
	execution.Stderr = stderr

	err := c.executable.Execute(execution)
	if err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}
