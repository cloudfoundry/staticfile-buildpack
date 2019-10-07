package docker

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
)

func BuildStagingDockerfile(logger lager.Logger, fixturePath, buildpackPath string, envs []string) Dockerfile {
	data := lager.Data{"fixture": fixturePath, "buildpack": buildpackPath, "envs": envs}
	session := logger.Session("build-staging-dockerfile", data)
	session.Debug("building")

	stack := os.Getenv("CF_STACK")
	baseImage := os.Getenv("CF_STACK_DOCKER_IMAGE")

	if stack == "" {
		stack = "cflinuxfs3"
	}

	if baseImage == "" {
		baseImage = fmt.Sprintf("cloudfoundry/%s", stack)
	}

	instructions := []DockerfileInstruction{
		NewDockerfileENV(fmt.Sprintf("CF_STACK %s", stack)),
		NewDockerfileENV("VCAP_APPLICATION {}"),
	}

	for _, env := range envs {
		instructions = append(instructions, NewDockerfileENV(env))
	}

	instructions = append(instructions, NewDockerfileADD(fmt.Sprintf("%s /tmp/staged/", fixturePath)))
	instructions = append(instructions, NewDockerfileADD(fmt.Sprintf("%s /tmp/", buildpackPath)))
	instructions = append(instructions, NewDockerfileRUN("mkdir -p /buildpack/0"))
	instructions = append(instructions, NewDockerfileRUN("mkdir -p /tmp/cache"))
	instructions = append(instructions, NewDockerfileRUN(fmt.Sprintf("unzip /tmp/%s -d /buildpack", filepath.Base(buildpackPath))))

	// HACK around https://github.com/dotcloud/docker/issues/5490
	instructions = append(instructions, NewDockerfileRUN("mv /usr/sbin/tcpdump /usr/bin/tcpdump"))

	return NewDockerfile(baseImage, instructions...)
}

type DockerfileInstructionType string

const (
	DockerfileInstructionTypeFROM DockerfileInstructionType = "FROM"
	DockerfileInstructionTypeADD  DockerfileInstructionType = "ADD"
	DockerfileInstructionTypeRUN  DockerfileInstructionType = "RUN"
	DockerfileInstructionTypeENV  DockerfileInstructionType = "ENV"
)

type DockerfileInstruction struct {
	Type    DockerfileInstructionType
	Content string
}

func NewDockerfileFROM(content string) DockerfileInstruction {
	return DockerfileInstruction{
		Type:    DockerfileInstructionTypeFROM,
		Content: content,
	}
}

func NewDockerfileENV(content string) DockerfileInstruction {
	return DockerfileInstruction{
		Type:    DockerfileInstructionTypeENV,
		Content: content,
	}
}

func NewDockerfileADD(content string) DockerfileInstruction {
	return DockerfileInstruction{
		Type:    DockerfileInstructionTypeADD,
		Content: content,
	}
}

func NewDockerfileRUN(content string) DockerfileInstruction {
	return DockerfileInstruction{
		Type:    DockerfileInstructionTypeRUN,
		Content: content,
	}
}

func (di DockerfileInstruction) String() string {
	return fmt.Sprintf("%s %s", di.Type, di.Content)
}

type Dockerfile struct {
	*bytes.Buffer
}

func NewDockerfile(baseImage string, instructions ...DockerfileInstruction) Dockerfile {
	instructions = append([]DockerfileInstruction{NewDockerfileFROM(baseImage)}, instructions...)

	buffer := bytes.NewBuffer(nil)
	for _, instruction := range instructions {
		buffer.WriteString(instruction.String())
		buffer.WriteRune('\n')
	}

	return Dockerfile{
		Buffer: buffer,
	}
}
