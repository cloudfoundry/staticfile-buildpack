package glow

import "github.com/cloudfoundry/libbuildpack/cutlass/execution"

const ExecutableName = "cnb2cf"

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(options execution.Options, args ...string) (stdout, stderr string, err error)
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
	args := []string{"package", "-stack", stack}

	if options.Cached {
		args = append(args, "-cached")
	}

	if options.Dev {
		args = append(args, "-dev")
	}

	if options.ManifestPath != "" {
		args = append(args, "-manifestpath", options.ManifestPath)
	}

	if options.Version != "" {
		args = append(args, "-version", options.Version)
	}

	stdout, stderr, err := c.executable.Execute(execution.Options{Dir: dir}, args...)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}
