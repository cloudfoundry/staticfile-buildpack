package shims

import (
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

type inputMetadata struct {
	Processes []struct {
		Type    string
		Command string
	}
}

func (i *inputMetadata) findCommand(processType string) (string, error) {
	for _, p := range i.Processes {
		if p.Type == processType {
			return p.Command, nil
		}
	}
	return "", fmt.Errorf("unable to find process with type %s in launch metadata %v", processType, i.Processes)
}

type defaultProcessTypes struct {
	Web string `yaml:"web"`
}

type outputMetadata struct {
	DefaultProcessTypes defaultProcessTypes `yaml:"default_process_types"`
}

type Releaser struct {
	MetadataPath string
	Writer       io.Writer
}

func (r *Releaser) Release() error {
	metadataFile, input := r.MetadataPath, inputMetadata{}
	_, err := toml.DecodeFile(metadataFile, &input)
	defer os.Remove(metadataFile)

	webCommand, err := input.findCommand("web")
	if err != nil {
		return err
	}

	output := outputMetadata{DefaultProcessTypes: defaultProcessTypes{Web: webCommand}}
	return yaml.NewEncoder(r.Writer).Encode(output)
}
