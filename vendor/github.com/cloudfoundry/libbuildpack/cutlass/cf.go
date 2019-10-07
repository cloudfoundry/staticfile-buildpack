package cutlass

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/tidwall/gjson"
)

type cfConfig struct {
	SpaceFields struct {
		GUID string
	}
}
type cfApps struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
	} `json:"resources"`
}
type cfInstance struct {
	State string `json:"state"`
}

type App struct {
	Name         string
	Path         string
	Stack        string
	Buildpacks   []string
	Memory       string
	Disk         string
	StartCommand string
	Stdout       *Buffer
	appGUID      string
	env          map[string]string
	logCmd       *exec.Cmd
	HealthCheck  string
}

func New(fixture string) *App {
	return &App{
		Name:         filepath.Base(fixture) + "-" + RandStringRunes(20),
		Path:         fixture,
		Stack:        os.Getenv("CF_STACK"),
		Buildpacks:   []string{},
		Memory:       DefaultMemory,
		Disk:         DefaultDisk,
		StartCommand: "",
		appGUID:      "",
		env:          map[string]string{},
		logCmd:       nil,
		HealthCheck:  "",
	}
}

func ApiVersion() (string, error) {
	cmd := exec.Command("cf", "curl", "/v2/info")
	cmd.Stderr = DefaultStdoutStderr
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var info struct {
		ApiVersion string `json:"api_version"`
	}
	if err := json.Unmarshal(bytes, &info); err != nil {
		return "", err
	}
	return info.ApiVersion, nil
}

func ApiGreaterThan(version string) (bool, error) {
	apiVersionString, err := ApiVersion()
	if err != nil {
		return false, err
	}
	apiVersion, err := semver.Make(apiVersionString)
	if err != nil {
		return false, err
	}
	reqVersion, err := semver.ParseRange(">= " + version)
	if err != nil {
		return false, err
	}
	return reqVersion(apiVersion), nil
}

func Stacks() ([]string, error) {
	cmd := exec.Command("cf", "curl", "/v2/stacks")
	cmd.Stderr = DefaultStdoutStderr
	bytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var info struct {
		Resources []struct {
			Entity struct {
				Name string `json:"name"`
			} `json:"entity"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(bytes, &info); err != nil {
		return nil, err
	}
	var out []string
	for _, r := range info.Resources {
		out = append(out, r.Entity.Name)
	}
	return out, nil
}

func DeleteOrphanedRoutes() error {
	command := exec.Command("cf", "delete-orphaned-routes", "-f")
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func DeleteBuildpack(language string) error {
	command := exec.Command("cf", "delete-buildpack", "-f", fmt.Sprintf("%s_buildpack", language))
	if data, err := command.CombinedOutput(); err != nil {
		fmt.Println(string(data))
		return err
	}
	return nil
}

func UpdateBuildpack(language, file, stack string) error {
	updateBuildpackArgs := []string{"update-buildpack", fmt.Sprintf("%s_buildpack", language), "-p", file, "--enable"}

	stackAssociationSupported, err := ApiGreaterThan("2.113.0")
	if err != nil {
		return err
	}

	if stack != "" && stackAssociationSupported {
		updateBuildpackArgs = append(updateBuildpackArgs, "-s", stack)
	}

	command := exec.Command("cf", updateBuildpackArgs...)
	if data, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to update buildpack by running '%s':\n%s\n%v", strings.Join(command.Args, " "), string(data), err)
	}
	return nil
}

func createBuildpack(language, file string) error {
	command := exec.Command("cf", "create-buildpack", fmt.Sprintf("%s_buildpack", language), file, "100", "--enable")
	if data, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create buildpack by running '%s':\n%s\n%v", strings.Join(command.Args, " "), string(data), err)
	}
	return nil
}

func CountBuildpack(language string) (int, error) {
	command := exec.Command("cf", "buildpacks")
	targetBpname := fmt.Sprintf("%s_buildpack", language)
	matches := 0
	lines, err := command.CombinedOutput()
	if err != nil {
		return -1, err
	}
	for _, line := range strings.Split(string(lines), "\n") {
		bpname := strings.SplitN(line, " ", 2)[0]
		if bpname == targetBpname {
			matches++
		}
	}
	return matches, nil
}

func CreateOrUpdateBuildpack(language, file, stack string) error {
	createBuildpack(language, file)
	return UpdateBuildpack(language, file, stack)
}

func (a *App) ConfirmBuildpack(version string) error {
	if !strings.Contains(a.Stdout.String(), fmt.Sprintf("Buildpack version %s\n", version)) {
		var versionLine string
		for _, line := range strings.Split(a.Stdout.String(), "\n") {
			if versionLine == "" && strings.Contains(line, " Buildpack version ") {
				versionLine = line
			}
		}
		return fmt.Errorf("Wrong buildpack version. Expected '%s', but this was logged: %s", version, versionLine)
	}
	return nil
}

func (a *App) RunTask(command string) ([]byte, error) {
	cmd := exec.Command("cf", "run-task", a.Name, command)
	cmd.Stderr = DefaultStdoutStderr
	bytes, err := cmd.Output()
	if err != nil {
		return bytes, err
	}
	return bytes, nil
}

func (a *App) Stop() error {
	command := exec.Command("cf", "stop", a.Name)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func (a *App) Restart() error {
	command := exec.Command("cf", "restart", a.Name)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func (a *App) SetEnv(key, value string) {
	a.env[key] = value
}

func (a *App) SpaceGUID() (string, error) {
	cfHome := os.Getenv("CF_HOME")
	if cfHome == "" {
		cfHome = os.Getenv("HOME")
	}
	bytes, err := ioutil.ReadFile(filepath.Join(cfHome, ".cf", "config.json"))
	if err != nil {
		return "", err
	}
	var config cfConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return "", err
	}
	return config.SpaceFields.GUID, nil
}

func (a *App) AppGUID() (string, error) {
	if a.appGUID != "" {
		return a.appGUID, nil
	}
	guid, err := a.SpaceGUID()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps?q=space_guid:"+guid+"&q=name:"+a.Name)
	cmd.Stderr = DefaultStdoutStderr
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var apps cfApps
	if err := json.Unmarshal(bytes, &apps); err != nil {
		return "", err
	}
	if len(apps.Resources) != 1 {
		return "", fmt.Errorf("Expected one app, found %d", len(apps.Resources))
	}
	a.appGUID = apps.Resources[0].Metadata.GUID
	return a.appGUID, nil
}

func (a *App) InstanceStates() ([]string, error) {
	guid, err := a.AppGUID()
	if err != nil {
		return []string{}, err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps/"+guid+"/instances")
	cmd.Stderr = DefaultStdoutStderr
	bytes, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}
	var data map[string]cfInstance
	if err := json.Unmarshal(bytes, &data); err != nil {
		return []string{}, err
	}
	var states []string
	for _, value := range data {
		states = append(states, value.State)
	}
	return states, nil
}

func (a *App) PushNoStart() error {
	args := []string{"push", a.Name, "--no-start", "-p", a.Path}
	if a.Stack != "" {
		args = append(args, "-s", a.Stack)
	}
	for _, buildpack := range a.Buildpacks {
		args = append(args, "-b", buildpack)
	}
	if _, err := os.Stat(filepath.Join(a.Path, "manifest.yml")); err == nil {
		args = append(args, "-f", filepath.Join(a.Path, "manifest.yml"))
	}
	if a.Memory != "" {
		args = append(args, "-m", a.Memory)
	}
	if a.Disk != "" {
		args = append(args, "-k", a.Disk)
	}
	if a.StartCommand != "" {
		args = append(args, "-c", a.StartCommand)
	}
	if a.HealthCheck != "" {
		args = append(args, "-u", a.HealthCheck)
	}

	command := exec.Command("cf", args...)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}

	for k, v := range a.env {
		command := exec.Command("cf", "set-env", a.Name, k, v)
		command.Stdout = DefaultStdoutStderr
		command.Stderr = DefaultStdoutStderr
		if err := command.Run(); err != nil {
			return err
		}
	}

	if a.logCmd == nil {
		a.logCmd = exec.Command("cf", "logs", a.Name)
		a.logCmd.Stderr = DefaultStdoutStderr
		a.Stdout = &Buffer{}
		a.logCmd.Stdout = a.Stdout
		if err := a.logCmd.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) V3Push() error {
	if err := a.PushNoStart(); err != nil {
		return err
	}

	args := []string{"v3-push", a.Name, "-p", a.Path}
	if len(a.Buildpacks) > 1 {
		for _, buildpack := range a.Buildpacks {
			args = append(args, "-b", buildpack)
		}
	}
	command := exec.Command("cf", args...)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func (a *App) Push() error {
	if err := a.PushNoStart(); err != nil {
		return err
	}

	command := exec.Command("cf", "start", a.Name)
	buf := &bytes.Buffer{}
	command.Stdout = buf
	command.Stderr = buf
	if err := command.Run(); err != nil {
		return fmt.Errorf("err: %s\n\nlogs: %s", err, buf)
	}
	return nil
}

func (a *App) GetUrl(path string) (string, error) {
	guid, err := a.AppGUID()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps/"+guid+"/summary")
	cmd.Stderr = DefaultStdoutStderr
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	schema, found := os.LookupEnv("CUTLASS_SCHEMA")
	if !found {
		schema = "http"
	}
	host := gjson.Get(string(data), "routes.0.host").String()
	domain := gjson.Get(string(data), "routes.0.domain.name").String()
	return fmt.Sprintf("%s://%s.%s%s", schema, host, domain, path), nil
}

func (a *App) Get(path string, headers map[string]string) (string, map[string][]string, error) {
	url, err := a.GetUrl(path)
	if err != nil {
		return "", map[string][]string{}, err
	}
	insecureSkipVerify, _ := os.LookupEnv("CUTLASS_SKIP_TLS_VERIFY")
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify == "true"},
		},
	}
	if headers["NoFollow"] == "true" {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		delete(headers, "NoFollow")
	}
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	if headers["user"] != "" && headers["password"] != "" {
		req.SetBasicAuth(headers["user"], headers["password"])
		delete(headers, "user")
		delete(headers, "password")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", map[string][]string{}, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", map[string][]string{}, err
	}
	resp.Header["StatusCode"] = []string{strconv.Itoa(resp.StatusCode)}
	return string(data), resp.Header, err
}

func (a *App) GetBody(path string) (string, error) {
	body, _, err := a.Get(path, map[string]string{})
	// TODO: Non 200 ??
	// if !(len(headers["StatusCode"]) == 1 && headers["StatusCode"][0] == "200") {
	// 	return "", fmt.Errorf("non 200 status: %v", headers)
	// }
	return body, err
}

func (a *App) Files(path string) ([]string, error) {
	cmd := exec.Command("cf", "ssh", a.Name, "-c", "find "+path)
	cmd.Stderr = DefaultStdoutStderr
	output, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}
	return strings.Split(string(output), "\n"), nil
}

func (a *App) DownloadDroplet(path string) error {
	guid, err := a.AppGUID()
	if err != nil {
		return err
	}

	cmd := exec.Command("cf", "curl", "/v2/apps/"+guid+"/droplet/download", "--output", path)
	cmd.Stderr = DefaultStdoutStderr
	_, err = cmd.Output()
	return err
}

func (a *App) Destroy() error {
	if a.logCmd != nil && a.logCmd.Process != nil {
		if err := a.logCmd.Process.Kill(); err != nil {
			return err
		}
	}

	command := exec.Command("cf", "delete", "-f", a.Name)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	return command.Run()
}
