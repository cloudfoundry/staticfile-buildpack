package cutlass

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass/docker"
	"github.com/cloudfoundry/packit"
	"github.com/pkg/errors"
)

func EnsureUsesProxy(fixturePath, buildpackPath string) error {
	proxyNetworkName, err := CreateProxyNetwork()
	if err != nil {
		return err
	}
	defer DeleteProxyNetwork(proxyNetworkName)

	proxyName, proxyPort, err := CreateProxyServer(proxyNetworkName)
	if err != nil {
		return err
	}
	defer DeleteContainer(proxyName)

	proxyEndpoint := proxyName + ":" + proxyPort
	traffic, built, logs, err := InternetTrafficForNetwork(proxyNetworkName, fixturePath, buildpackPath, []string{
		"http_proxy=" + "http://" + proxyEndpoint,
		"https_proxy=" + "http://" + proxyEndpoint, //Set it to http to avoid npm/yarn proxy issue
	}, true)
	if err != nil {
		return err
	} else if !built {
		return fmt.Errorf("failed to run buildpack lifecycle\n%s", strings.Join(logs, "\n"))
	}

	networkTrafficMatch := proxyName + "." + proxyNetworkName + "." + proxyPort
	return UniqueDestination(traffic, networkTrafficMatch)
}

func InternetTrafficForNetwork(networkName, fixturePath, buildpackPath string, envs []string, resolve bool) ([]string, bool, []string, error) {
	data := lager.Data{"network": networkName, "fixture": fixturePath, "buildpack": buildpackPath, "envs": envs, "resolve": resolve}
	session := DefaultLogger.Session("internet-traffic-for-network", data)

	session.Debug("preparing-docker-build-context")
	tmpDir, err := ioutil.TempDir("", "docker-context")
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to create docker context directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	buildpackCopy, err := os.Create(filepath.Join(tmpDir, "buildpack"))
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to create buildpack copy: %s", err)
	}

	buildpackOriginal, err := os.Open(buildpackPath)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to open buildpack original: %s", err)
	}

	_, err = io.Copy(buildpackCopy, buildpackOriginal)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to copy buildpack: %s", err)
	}

	fixtureDir := filepath.Join(tmpDir, "fixture")
	err = os.Mkdir(fixtureDir, 0755)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to create fixture directory: %s", err)
	}

	err = libbuildpack.CopyDirectory(fixturePath, fixtureDir)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to copy fixture directory: %s", err)
	}

	dockerfile := docker.BuildStagingDockerfile(session, "fixture", "buildpack", envs)

	session.Debug("creating-dockerfile")
	file, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf("itf.Dockerfile.%d", rand.Int())))
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to create dockerfile: %s", err)
	}
	defer file.Close()

	session.Debug("writing-dockerfile")
	_, err = io.Copy(file, dockerfile)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to create dockerfile: %s", err)
	}

	dockerfilePath := file.Name()

	// TODO: Take this out, after refactor all proxy tests to use EnsureUsesProxy
	networkCommands := []string{
		"(sudo tcpdump -i %s eth0 not udp port 53 and not udp port 1900 and not udp port 5353 and ip -t -Uw /tmp/dumplog &)",
		"/buildpack/bin/detect /tmp/staged && echo 'Detect completed'",
		"/buildpack/bin/supply /tmp/staged /tmp/cache /buildpack 0 && echo 'Supply completed'",
		"/buildpack/bin/finalize /tmp/staged /tmp/cache /buildpack 0 /tmp && echo 'Finalize completed'",
		"/buildpack/bin/release /tmp/staged && echo 'Release completed'",
		"sleep 1",
		"pkill tcpdump; tcpdump -r %s /tmp/dumplog | sed -e 's/^/internet traffic: /' 2>&1 || true",
	}

	var flags string
	if !resolve {
		flags = "-n"
	}
	networkCommand := fmt.Sprintf(strings.Join(networkCommands, " && "), flags, flags)

	output, err := ExecuteDockerFile(dockerfilePath, networkName, networkCommand)
	if err != nil {
		return nil, false, nil, errors.Wrapf(err, "failed to build and run docker image: %s\n%s", err, output)
	}

	return ParseTrafficAndLogs(output)
}

// TODO: Delete after all buildpacks use EnsureUsesProxy
func InternetTraffic(fixturePath, buildpackPath string, envs []string) ([]string, bool, []string, error) {
	data := lager.Data{"fixture": fixturePath, "buildpack": buildpackPath, "envs": envs}
	DefaultLogger.Debug("internet-traffic", data)

	return InternetTrafficForNetwork("bridge", fixturePath, buildpackPath, envs, false)
}

func UniqueDestination(traffic []string, destination string) error {
	re := regexp.MustCompile("^[\\d\\.:]+ IP ([\\S\\.]+) > ([\\S\\.]+):")
	for _, line := range traffic {
		m := re.FindStringSubmatch(line)
		if len(m) != 3 || (m[1] != destination && m[2] != destination) {
			return fmt.Errorf("Outgoing traffic: %s", line)
		}
	}
	return nil
}

func CreateProxyNetwork() (string, error) {
	networkName := "proxy-network-" + RandStringRunes(6)
	cmd := exec.Command("docker", "network", "create", networkName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", errors.Wrapf(err, "failed to create docker network: %s", string(output))
	}

	return networkName, nil
}

func CreateProxyServer(networkName string) (proxyContainerName string, proxyPort string, err error) {
	containerName := "proxy-container" + RandStringRunes(6)

	cmd := exec.Command("docker", "run", "-d", "--name", containerName,
		"--network", networkName, "-t", "cfbuildpacks/proxy-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to docker run cfbuildpacks/proxy-server: %s", string(output))
	}

	cmd = exec.Command("docker", "exec", containerName, "bash", "-c", "cat server.log")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to docker exec: %s", string(output))
	}

	re := regexp.MustCompile("Listening on Port: (\\d+)")
	matches := re.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		return "", "", fmt.Errorf("failed to get proxy port from: %s", string(output))
	}

	return containerName, matches[1], nil
}

func DeleteProxyNetwork(networkName string) error {
	cmd := exec.Command("docker", "network", "rm", networkName)
	if output, err := cmd.Output(); err != nil {
		return errors.Wrapf(err, "failed to delete docker network: %s", string(output))
	}

	return nil
}

func DeleteContainer(containerName string) error {
	cmd := exec.Command("docker", "container", "rm", containerName, "-f")
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to delete container %s: %s", containerName, string(output))
	}

	return nil
}

func ExecuteDockerFile(path, network, command string) (string, error) {
	data := lager.Data{"path": path, "network": network, "command": command}
	session := DefaultLogger.Session("execute-dockerfile", data)
	image := "internet_traffic_test" + RandStringRunes(8)

	session.Debug("creating-cli")
	cli := docker.NewCLI(packit.NewExecutable(docker.ExecutableName, session))

	defer cli.RemoveImage(image, docker.RemoveImageOptions{Force: true})

	session.Debug("building-image-cli")
	stdout, _, err := cli.Build(docker.BuildOptions{
		Remove:  true,
		NoCache: true,
		Tag:     image,
		File:    path,
		Context: filepath.Dir(path),
	})
	if err != nil {
		return stdout, err
	}

	session.Debug("running-image-cli")
	stdout, _, err = cli.Run(image, docker.RunOptions{
		Network: network,
		Remove:  true,
		TTY:     true,
		Command: command,
	})
	if err != nil {
		return stdout, err
	}

	return stdout, nil
}

func ParseTrafficAndLogs(output string) ([]string, bool, []string, error) {
	DefaultLogger.Debug("parse-traffic-and-logs")
	var internetTraffic, logs []string
	var detected, released, supplied, finalized bool
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "internet traffic: "); idx >= 0 && idx < 10 {
			internetTraffic = append(internetTraffic, line[(idx+18):])
		} else {
			logs = append(logs, line)
			if strings.Contains(line, "Detect completed") {
				detected = true
			} else if strings.Contains(line, "Supply completed") {
				supplied = true
			} else if strings.Contains(line, "Finalize completed") {
				finalized = true
			} else if strings.Contains(line, "Release completed") {
				released = true
			}
		}
	}

	built := detected && supplied && finalized && released
	return internetTraffic, built, logs, nil
}
