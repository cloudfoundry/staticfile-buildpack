package switchblade

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/switchblade/internal/cloudfoundry"
	"github.com/cloudfoundry/switchblade/internal/docker"
	"github.com/docker/docker/client"
	"github.com/paketo-buildpacks/packit/v2/pexec"
)

type Buildpack struct {
	Name string
	URI  string
}

type Service map[string]interface{}

type Platform struct {
	initialize initializeProcess

	Deploy DeployProcess
	Delete DeleteProcess
}

type DeployProcess interface {
	WithBuildpacks(buildpacks ...string) DeployProcess
	WithStack(stack string) DeployProcess
	WithEnv(env map[string]string) DeployProcess
	WithoutInternetAccess() DeployProcess
	WithServices(map[string]Service) DeployProcess
	WithStartCommand(command string) DeployProcess

	Execute(name, path string) (Deployment, fmt.Stringer, error)
}

type DeleteProcess interface {
	Execute(name string) error
}

type initializeProcess interface {
	Execute(buildpacks ...Buildpack) error
}

const (
	CloudFoundry = "cf"
	Docker       = "docker"
)

func NewPlatform(platformType, token, stack string) (Platform, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Platform{}, err
	}

	switch platformType {
	case CloudFoundry:
		cli := pexec.NewExecutable("cf")

		initialize := cloudfoundry.NewInitialize(cli, stack)
		setup := cloudfoundry.NewSetup(cli, filepath.Join(home, ".cf"), stack)
		stage := cloudfoundry.NewStage(cli)
		teardown := cloudfoundry.NewTeardown(cli)

		return NewCloudFoundry(initialize, setup, stage, teardown, os.TempDir()), nil
	case Docker:
		client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return Platform{}, err
		}

		workspace := filepath.Join(home, ".switchblade")

		golang := pexec.NewExecutable("go")
		archiver := docker.NewTGZArchiver()
		lifecycleManager := docker.NewLifecycleManager(golang, archiver)
		buildpacksCache := docker.NewBuildpacksCache(filepath.Join(workspace, "buildpacks-cache"))
		buildpacksRegistry := docker.NewBuildpacksRegistry("https://api.github.com", token)
		buildpacksManager := docker.NewBuildpacksManager(archiver, buildpacksCache, buildpacksRegistry)
		networkManager := docker.NewNetworkManager(client)

		initialize := docker.NewInitialize(buildpacksRegistry)
		setup := docker.NewSetup(client, lifecycleManager, buildpacksManager, archiver, networkManager, workspace, stack)
		stage := docker.NewStage(client, archiver, workspace)
		start := docker.NewStart(client, networkManager, workspace, stack)
		teardown := docker.NewTeardown(client, networkManager, workspace)

		return NewDocker(initialize, setup, stage, start, teardown), nil
	}

	return Platform{}, fmt.Errorf("unknown platform type: %q", platformType)
}

func (p Platform) Initialize(buildpacks ...Buildpack) error {
	return p.initialize.Execute(buildpacks...)
}
