package docker

import "github.com/cloudfoundry/packit"

const ExecutableName = "docker"

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(packit.Execution) (stdout, stderr string, err error)
}

type CLI struct {
	executable Executable
}

func NewCLI(executable Executable) CLI {
	return CLI{
		executable: executable,
	}
}

type BuildOptions struct {
	Remove  bool
	NoCache bool
	Tag     string
	File    string
	Context string
}

func (c CLI) Build(options BuildOptions) (string, string, error) {
	execution := packit.Execution{
		Args: []string{"build"},
	}

	if options.Remove {
		execution.Args = append(execution.Args, "--rm")
	}

	if options.NoCache {
		execution.Args = append(execution.Args, "--no-cache")
	}

	if options.Tag != "" {
		execution.Args = append(execution.Args, "--tag", options.Tag)
	}

	if options.File != "" {
		execution.Args = append(execution.Args, "--file", options.File)
	}

	if options.Context == "" {
		options.Context = "."
	} else {
		execution.Dir = options.Context
	}

	execution.Args = append(execution.Args, options.Context)

	stdout, stderr, err := c.executable.Execute(execution)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

type RunOptions struct {
	Network string
	Remove  bool
	TTY     bool
	Command string
}

func (c CLI) Run(image string, options RunOptions) (string, string, error) {
	execution := packit.Execution{
		Args: []string{"run"},
	}

	if options.Network != "" {
		execution.Args = append(execution.Args, "--network", options.Network)
	}

	if options.Remove {
		execution.Args = append(execution.Args, "--rm")
	}

	if options.TTY {
		execution.Args = append(execution.Args, "--tty")
	}

	execution.Args = append(execution.Args, image)

	if options.Command != "" {
		execution.Args = append(execution.Args, "bash", "-c", options.Command)
	}

	stdout, stderr, err := c.executable.Execute(execution)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

type RemoveImageOptions struct {
	Force bool
}

func (c CLI) RemoveImage(image string, options RemoveImageOptions) (string, string, error) {
	execution := packit.Execution{
		Args: []string{"image", "rm"},
	}

	if options.Force {
		execution.Args = append(execution.Args, "--force")
	}

	execution.Args = append(execution.Args, image)

	stdout, stderr, err := c.executable.Execute(execution)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}
