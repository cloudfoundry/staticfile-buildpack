package cutlass

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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
	// TODO: Take this out, after refactor all proxy tests to use EnsureUsesProxy
	var flags string
	if !resolve {
		flags = "-n"
	}

	networkCommand := "(sudo tcpdump -i %s eth0 not udp port 53 and not udp port 1900 and not udp port 5353 and ip -t -Uw /tmp/dumplog &) " +
		"&& /buildpack/bin/detect /tmp/staged && echo 'Detect completed' " +
		"&& /buildpack/bin/supply /tmp/staged /tmp/cache /buildpack 0 && echo 'Supply completed' " +
		"&& /buildpack/bin/finalize /tmp/staged /tmp/cache /buildpack 0 /tmp && echo 'Finalize completed' " +
		"&& /buildpack/bin/release /tmp/staged && echo 'Release completed' " +
		"&& sleep 1 && pkill tcpdump; tcpdump -r %s /tmp/dumplog | sed -e 's/^/internet traffic: /' 2>&1 || true"
	networkCommand = fmt.Sprintf(networkCommand, flags, flags)

	dockerfilePath, err := createDockerfile(fixturePath, buildpackPath, envs)
	if err != nil {
		return nil, false, nil, errors.Wrapf(err, "failed to create dockerfile: %s", dockerfilePath)
	}
	defer os.Remove(dockerfilePath)

	output, err := ExecuteDockerFile(dockerfilePath, networkName, networkCommand)
	if err != nil {
		return nil, false, nil, errors.Wrapf(err, "failed to build and run docker image: %s", output)
	}

	return ParseTrafficAndLogs(output)
}

// TODO: Delete after all buildpacks use EnsureUsesProxy
func InternetTraffic(bpDir, fixturePath, buildpackPath string, envs []string) ([]string, bool, []string, error) {
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

func ExecuteDockerFile(dockerfilePath, networkName, networkCommand string) (string, error) {
	dockerImageName := "internet_traffic_test" + RandStringRunes(8)
	defer exec.Command("docker", "rmi", "-f", dockerImageName).Output()

	dockerfileName := filepath.Base(dockerfilePath)
	dockerfileDir := filepath.Dir(dockerfilePath)

	cmd := exec.Command("docker", "build", "--rm", "--no-cache", "-t", dockerImageName, "-f", dockerfileName, ".")
	cmd.Dir = dockerfileDir
	cmd.Stderr = DefaultStdoutStderr
	if output, err := cmd.Output(); err != nil {
		return "", errors.Wrapf(err, "failed to docker build: %s", string(output))

	}

	cmd = exec.Command("docker", "run", "--network", networkName, "--rm", "-t", dockerImageName, "bash", "-c", networkCommand)
	cmd.Dir = dockerfileDir
	cmd.Stderr = DefaultStdoutStderr
	output, err := cmd.Output()

	return string(output), err
}

func ParseTrafficAndLogs(output string) ([]string, bool, []string, error) {
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

func createDockerfile(fixturePath, buildpackPath string, envs []string) (string, error) {
	bpDir := filepath.Dir(buildpackPath)
	bpName := filepath.Base(buildpackPath)

	dockerfileContents := dockerfile(fixturePath, bpName, envs)
	dockerfileName := fmt.Sprintf("itf.Dockerfile.%v", rand.Int())
	dockerfilePath := filepath.Join(bpDir, dockerfileName)
	err := ioutil.WriteFile(dockerfilePath, []byte(dockerfileContents), 0755)
	return dockerfilePath, err
}

func dockerfile(fixturePath, buildpackPath string, envs []string) string {
	cfStack := os.Getenv("CF_STACK")
	if cfStack == "" {
		cfStack = "cflinuxfs3"
	}

	stackDockerImage := os.Getenv("CF_STACK_DOCKER_IMAGE")
	if stackDockerImage == "" {
		stackDockerImage = fmt.Sprintf("cloudfoundry/%s", cfStack)
	}

	out := fmt.Sprintf("FROM %s\n"+
		"ENV CF_STACK %s\n"+
		"ENV VCAP_APPLICATION {}\n", stackDockerImage, cfStack)
	for _, env := range envs {
		out = out + "ENV " + env + "\n"
	}
	out = out +
		"ADD " + fixturePath + " /tmp/staged/\n" +
		"ADD " + buildpackPath + " /tmp/\n" +
		"RUN mkdir -p /buildpack/0\n" +
		"RUN mkdir -p /tmp/cache\n" +
		"RUN unzip /tmp/" + filepath.Base(buildpackPath) + " -d /buildpack\n" +
		"# HACK around https://github.com/dotcloud/docker/issues/5490\n" +
		"RUN mv /usr/sbin/tcpdump /usr/bin/tcpdump\n"
	return out
}
