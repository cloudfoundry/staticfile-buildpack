package docker

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
	args := []string{"build"}
	var execOptions ExecuteOptions

	if options.Remove {
		args = append(args, "--rm")
	}

	if options.NoCache {
		args = append(args, "--no-cache")
	}

	if options.Tag != "" {
		args = append(args, "--tag", options.Tag)
	}

	if options.File != "" {
		args = append(args, "--file", options.File)
	}

	if options.Context == "" {
		options.Context = "."
	} else {
		execOptions.Dir = options.Context
	}

	args = append(args, options.Context)

	stdout, stderr, err := c.executable.Execute(execOptions, args...)
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
	args := []string{"run"}

	if options.Network != "" {
		args = append(args, "--network", options.Network)
	}

	if options.Remove {
		args = append(args, "--rm")
	}

	if options.TTY {
		args = append(args, "--tty")
	}

	args = append(args, image)

	if options.Command != "" {
		args = append(args, "bash", "-c", options.Command)
	}

	stdout, stderr, err := c.executable.Execute(ExecuteOptions{}, args...)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

type RemoveImageOptions struct {
	Force bool
}

func (c CLI) RemoveImage(image string, options RemoveImageOptions) (string, string, error) {
	args := []string{"image", "rm"}

	if options.Force {
		args = append(args, "--force")
	}

	args = append(args, image)

	stdout, stderr, err := c.executable.Execute(ExecuteOptions{}, args...)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}
