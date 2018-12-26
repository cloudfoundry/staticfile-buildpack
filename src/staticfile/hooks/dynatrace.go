package hooks

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type DynatraceCredentials struct {
	ServiceName   string
	EnvironmentId string
	ApiToken      string
	ApiURL        string
	SkipErrors    bool
}

type DynatraceHook struct {
	libbuildpack.DefaultHook
	Log             *libbuildpack.Logger
	Command         Command
	DownloadRetries float64
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	command := &libbuildpack.Command{}

	libbuildpack.AddHook(DynatraceHook{
		Log:             logger,
		Command:         command,
		DownloadRetries: 3.0,
	})
}

func (h DynatraceHook) AfterCompile(stager *libbuildpack.Stager) error {
	h.Log.Debug("Checking for enabled dynatrace service...")

	credentials, found := h.dtCredentials()
	if !found {
		h.Log.Debug("Dynatrace service credentials not found!")
		return nil
	}

	h.Log.Info("Dynatrace service credentials found. Setting up Dynatrace PaaS agent.")

	apiurl := credentials.ApiURL
	if apiurl == "" {
		apiurl = "https://" + credentials.EnvironmentId + ".live.dynatrace.com/api"
	}

	url := apiurl + "/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&include=process&bitness=64&Api-Token=" + credentials.ApiToken
	installerPath := filepath.Join(os.TempDir(), "paasInstaller.sh")

	h.Log.Debug("Downloading '%s' to '%s'", url, installerPath)
	err := h.downloadFile(url, installerPath)
	if err != nil {
		if credentials.SkipErrors {
			h.Log.Warning("Error during installer download, skipping installation")
			return nil
		}
		return err
	}

	h.Log.Debug("Making %s executable...", installerPath)
	os.Chmod(installerPath, 0755)

	h.Log.BeginStep("Starting Dynatrace PaaS agent installer")

	if os.Getenv("BP_DEBUG") != "" {
		err = h.Command.Execute("", os.Stdout, os.Stderr, installerPath, stager.BuildDir())
	} else {
		err = h.Command.Execute("", ioutil.Discard, ioutil.Discard, installerPath, stager.BuildDir())
	}
	if err != nil {
		return err
	}

	h.Log.Info("Dynatrace PaaS agent installed.")

	dynatraceEnvName := "dynatrace-env.sh"
	installDir := filepath.Join("dynatrace", "oneagent")
	dynatraceEnvPath := filepath.Join(stager.DepDir(), "profile.d", dynatraceEnvName)
	agentLibPath, err := h.agentPath(filepath.Join(stager.BuildDir(), installDir))
	if err != nil {
		h.Log.Error("Manifest handling failed!")
		return err
	}

	agentLibPath = filepath.Join(installDir, agentLibPath)
	agentBuilderLibPath := filepath.Join(stager.BuildDir(), agentLibPath)

	_, err = os.Stat(agentBuilderLibPath)
	if os.IsNotExist(err) {
		h.Log.Error("Agent library (%s) not found!", agentBuilderLibPath)
		return err
	}

	h.Log.BeginStep("Setting up Dynatrace PaaS agent injection...")
	h.Log.Debug("Copy %s to %s", dynatraceEnvName, dynatraceEnvPath)
	err = libbuildpack.CopyFile(filepath.Join(stager.BuildDir(), installDir, dynatraceEnvName), dynatraceEnvPath)
	if err != nil {
		return err
	}

	h.Log.Debug("Open %s for modification...", dynatraceEnvPath)
	f, err := os.OpenFile(dynatraceEnvPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	defer f.Close()

	h.Log.Debug("Setting LD_PRELOAD ...")
	_, err = f.WriteString("\nexport LD_PRELOAD=${HOME}/" + agentLibPath)
	if err != nil {
		return err
	}

	h.Log.Debug("Setting DT_HOST_ID...")
	_, err = f.WriteString("\nexport DT_HOST_ID=" + h.appName() + "_${CF_INSTANCE_INDEX}")
	if err != nil {
		return err
	}

	if os.Getenv("DT_LOGSTREAM") == "" {
		h.Log.Debug("Setting DT_LOGSTREAM to stdout...")
		_, err = f.WriteString("\nexport DT_LOGSTREAM=stdout")
		if err != nil {
			return err
		}
	}

	h.Log.Info("Dynatrace PaaS agent injection is set up.")

	return nil
}

func (h DynatraceHook) getCredentialString(credentials map[string]interface{}, key string) string {
	value, isString := credentials[key].(string)

	if isString {
		return value
	}
	return ""
}

func (h DynatraceHook) dtCredentials() (DynatraceCredentials, bool) {
	type Service struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}

	var vcapServices map[string][]Service

	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		h.Log.Debug("Failed to unmarshal VCAP_SERVICES: %s", err)
		return DynatraceCredentials{}, false
	}

	var detectedCredentials []DynatraceCredentials

	for _, services := range vcapServices {
		for _, service := range services {
			if strings.Contains(service.Name, "dynatrace") {
				credentials := DynatraceCredentials{
					ServiceName:   service.Name,
					EnvironmentId: h.getCredentialString(service.Credentials, "environmentid"),
					ApiToken:      h.getCredentialString(service.Credentials, "apitoken"),
					ApiURL:        h.getCredentialString(service.Credentials, "apiurl"),
					SkipErrors:    h.getCredentialString(service.Credentials, "skiperrors") == "true",
				}

				if credentials.EnvironmentId != "" && credentials.ApiToken != "" {
					detectedCredentials = append(detectedCredentials, credentials)
				}
			}
		}
	}

	if len(detectedCredentials) == 1 {
		h.Log.Debug("Found one matching service: %s", detectedCredentials[0].ServiceName)
		return detectedCredentials[0], true
	} else if len(detectedCredentials) > 1 {
		h.Log.Warning("More than one matching service found!")
	}

	return DynatraceCredentials{}, false
}

func (h DynatraceHook) appName() string {
	var application struct {
		Name string `json:"name"`
	}
	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &application)
	if err != nil {
		return ""
	}

	return application.Name
}

func (h DynatraceHook) downloadFile(url, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}

	defer out.Close()

	baseWaitTime := 3.0
	for retries := 0.0; retries <= 3.0; retries++ {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		_, err = io.Copy(out, resp.Body)

		if resp.StatusCode < 400 && err == nil {
			break
		} else if retries < h.DownloadRetries {
			waitTime := time.Duration(baseWaitTime + math.Pow(2, retries))
			h.Log.Warning("Error during installer download, retrying in %d seconds", waitTime)
			time.Sleep(waitTime * time.Second)
			continue
		} else {
			responseError := "Download returned with status " + resp.Status
			h.Log.Debug(responseError)
			return errors.New(responseError)
		}
	}
	return nil
}

func (h DynatraceHook) agentPath(installDir string) (string, error) {
	manifestPath := filepath.Join(installDir, "manifest.json")

	type Binary struct {
		Path       string `json:"path"`
		Md5        string `json:"md5"`
		Version    string `json:"version"`
		Binarytype string `json:"binarytype,omitempty"`
	}

	type Architecture map[string][]Binary
	type Technologies map[string]Architecture

	type Manifest struct {
		Tech Technologies `json:"technologies"`
		Ver  string       `json:"version"`
	}

	var manifest Manifest

	_, err := os.Stat(manifestPath)
	if !os.IsNotExist(err) {
		raw, err := ioutil.ReadFile(manifestPath)
		if err != nil {
			return "", err
		}

		err = json.Unmarshal(raw, &manifest)
		if err != nil {
			return "", err
		}

		var bin_type string
		for _, binary := range manifest.Tech["process"]["linux-x86-64"] {
			if binary.Binarytype == "primary" {
				return binary.Path, nil
			} else {
				bin_type = binary.Binarytype
			}
		}

		if bin_type != "primary" {
			// using fallback path
			h.Log.Warning("Agent path not found in manifest.json, using fallback!")
			fallbackPath := filepath.Join("agent", "lib64", "liboneagentproc.so")
			return fallbackPath, nil
		}
		return "", errors.New("No primary binary for process agent found!")
	} else {
		h.Log.Info("manifest.json not found, using fallback!")
		fallbackPath := filepath.Join("agent", "lib64", "liboneagentproc.so")
		return fallbackPath, nil
	}
}
