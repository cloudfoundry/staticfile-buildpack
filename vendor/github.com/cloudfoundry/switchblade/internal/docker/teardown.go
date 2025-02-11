package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type TeardownPhase interface {
	Run(ctx context.Context, name string) error
}

//go:generate faux --interface TeardownClient --output fakes/teardown_client.go
type TeardownClient interface {
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
}

//go:generate faux --interface TeardownNetworkManager --output fakes/teardown_network_manager.go
type TeardownNetworkManager interface {
	Delete(ctx context.Context, name string) error
}

type Teardown struct {
	client    TeardownClient
	networks  TeardownNetworkManager
	workspace string
}

func NewTeardown(client TeardownClient, networks TeardownNetworkManager, workspace string) Teardown {
	return Teardown{
		client:    client,
		networks:  networks,
		workspace: workspace,
	}
}

func (t Teardown) Run(ctx context.Context, name string) error {
	err := t.client.ContainerRemove(ctx, name, types.ContainerRemoveOptions{Force: true})
	if err != nil && !client.IsErrNotFound(err) {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	err = t.networks.Delete(ctx, InternalNetworkName)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	err = os.Remove(filepath.Join(t.workspace, "droplets", fmt.Sprintf("%s.tar.gz", name)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete droplet tarball: %w", err)
	}

	err = os.Remove(filepath.Join(t.workspace, "source", fmt.Sprintf("%s.tar.gz", name)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete source tarball: %w", err)
	}

	err = os.Remove(filepath.Join(t.workspace, "buildpacks", fmt.Sprintf("%s.tar.gz", name)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete buildpack tarball: %w", err)
	}

	err = os.RemoveAll(filepath.Join(t.workspace, "buildpacks", name))
	if err != nil {
		return fmt.Errorf("failed to delete buildpacks: %w", err)
	}

	err = os.Remove(filepath.Join(t.workspace, "build-cache", fmt.Sprintf("%s.tar.gz", name)))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete build-cache tarball: %w", err)
	}

	return nil
}
