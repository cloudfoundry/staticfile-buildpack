package docker

type InitializePhase interface {
	Run([]Buildpack)
}

type Initialize struct {
	registry BPRegistry
}

func NewInitialize(registry BPRegistry) Initialize {
	return Initialize{
		registry: registry,
	}
}

func (i Initialize) Run(buildpacks []Buildpack) {
	i.registry.Override(buildpacks...)
}
