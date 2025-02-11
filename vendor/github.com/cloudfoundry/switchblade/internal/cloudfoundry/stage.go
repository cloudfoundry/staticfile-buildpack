package cloudfoundry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
)

type StagePhase interface {
	Run(logs io.Writer, home, name string) (url string, err error)
}

type Stage struct {
	cli Executable
}

func NewStage(cli Executable) Stage {
	return Stage{
		cli: cli,
	}
}

func (s Stage) Run(logs io.Writer, home, name string) (string, error) {
	env := append(os.Environ(), fmt.Sprintf("CF_HOME=%s", home))

	err := s.cli.Execute(pexec.Execution{
		Args:   []string{"start", name},
		Stdout: logs,
		Stderr: logs,
		Env:    env,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start: %w\n\nOutput:\n%s", err, logs)
	}

	buffer := bytes.NewBuffer(nil)
	err = s.cli.Execute(pexec.Execution{
		Args:   []string{"app", name, "--guid"},
		Stdout: buffer,
		Env:    env,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch guid: %w\n\nOutput:\n%s", err, buffer)
	}

	guid := strings.TrimSpace(buffer.String())
	buffer = bytes.NewBuffer(nil)
	err = s.cli.Execute(pexec.Execution{
		Args:   []string{"curl", path.Join("/v2", "apps", guid, "routes")},
		Stdout: buffer,
		Env:    env,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch routes: %w\n\nOutput:\n%s", err, buffer)
	}

	var routes struct {
		Resources []struct {
			Entity struct {
				DomainURL string `json:"domain_url"`
				Host      string `json:"host"`
				Path      string `json:"path"`
			} `json:"entity"`
		} `json:"resources"`
	}
	err = json.NewDecoder(buffer).Decode(&routes)
	if err != nil {
		return "", fmt.Errorf("failed to parse routes: %w\n\nOutput:\n%s", err, buffer)
	}

	var url string
	if len(routes.Resources) > 0 {
		route := routes.Resources[0].Entity
		buffer = bytes.NewBuffer(nil)
		err = s.cli.Execute(pexec.Execution{
			Args:   []string{"curl", route.DomainURL},
			Stdout: buffer,
			Env:    env,
		})
		if err != nil {
			return "", fmt.Errorf("failed to fetch domain: %w\n\nOutput:\n%s", err, buffer)
		}

		var domain struct {
			Entity struct {
				Name string `json:"name"`
			} `json:"entity"`
		}
		err = json.NewDecoder(buffer).Decode(&domain)
		if err != nil {
			return "", fmt.Errorf("failed to parse domain: %w\n\nOutput:\n%s", err, buffer)
		}

		url = fmt.Sprintf("http://%s.%s%s", route.Host, domain.Entity.Name, route.Path)
	}

	return url, nil
}
