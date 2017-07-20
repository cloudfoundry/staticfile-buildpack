package cutlass

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var DefaultMemory string = ""
var DefaultDisk string = ""
var Cached bool = false
var DefaultStdoutStderr io.Writer = nil

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
	Name      string
	Path      string
	Buildpack string
	Stdout    *bytes.Buffer
	appGUID   string
	env       map[string]string
	logCmd    *exec.Cmd
}

func New(fixture string) *App {
	return &App{
		Name:      filepath.Base(fixture) + "-" + randStringRunes(20),
		Path:      fixture,
		Buildpack: "",
		appGUID:   "",
		env:       map[string]string{},
		logCmd:    nil,
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

func DeleteOrphanedRoutes() error {
	command := exec.Command("cf", "delete-orphaned-routes", "-f")
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func UpdateBuildpack(language, file string) error {
	command := exec.Command("cf", "update-buildpack", fmt.Sprintf("%s_buildpack", language), "-p", file, "--enable")
	if data, err := command.CombinedOutput(); err != nil {
		fmt.Println(string(data))
		return err
	}
	return nil
}

func (a *App) ConfirmBuildpack(version string) error {
	if !strings.Contains(a.Stdout.String(), fmt.Sprintf("Buildpack version %s", version)) {
		var versionLine string
		for _, line := range strings.Split(a.Stdout.String(), "\n") {
			if versionLine == "" && strings.Contains(line, " Buildpack version ") {
				versionLine = line
			}
		}
		return fmt.Errorf("Wrong buildpack version(%s): %s", version, versionLine)
	}
	return nil
}

func (a *App) SetEnv(key, value string) {
	a.env[key] = value
}

func (a *App) SpaceGUID() (string, error) {
	bytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".cf", "config.json"))
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

func (a *App) Push() error {
	args := []string{"push", a.Name, "--no-start", "-p", a.Path}
	if a.Buildpack != "" {
		args = append(args, "-b", a.Buildpack)
	}
	if _, err := os.Stat(filepath.Join(a.Path, "manifest.yml")); err == nil {
		args = append(args, "-f", filepath.Join(a.Path, "manifest.yml"))
	}
	if DefaultMemory != "" {
		args = append(args, "-m", DefaultMemory)
	}
	if DefaultDisk != "" {
		args = append(args, "-k", DefaultDisk)
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

	a.logCmd = exec.Command("cf", "logs", a.Name)
	a.logCmd.Stderr = DefaultStdoutStderr
	a.Stdout = bytes.NewBuffer(nil)
	a.logCmd.Stdout = a.Stdout
	if err := a.logCmd.Start(); err != nil {
		return err
	}

	command = exec.Command("cf", "start", a.Name)
	command.Stdout = DefaultStdoutStderr
	command.Stderr = DefaultStdoutStderr
	if err := command.Run(); err != nil {
		return err
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
	host := gjson.Get(string(data), "routes.0.host").String()
	domain := gjson.Get(string(data), "routes.0.domain.name").String()
	return fmt.Sprintf("http://%s.%s%s", host, domain, path), nil
}

func (a *App) Get(path string, headers map[string]string) (string, map[string][]string, error) {
	url, err := a.GetUrl(path)
	if err != nil {
		return "", map[string][]string{}, err
	}
	client := &http.Client{}
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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
