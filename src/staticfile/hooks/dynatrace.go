package hooks

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type DynatraceHook struct {
	libbuildpack.DefaultHook
}

func init() {
	libbuildpack.AddHook(DynatraceHook{})
}

func (h DynatraceHook) AfterCompile(stager *libbuildpack.Stager) error {
	stager.Log.Debug("Checking for enabled dynatrace service...")

	credentials := h.dtCredentials()
	if credentials == nil {
		stager.Log.Debug("Dynatrace service not found!")
		return nil
	}

	stager.Log.Info("Dynatrace service found. Setting up Dynatrace PaaS agent.")

	apiurl, present := credentials["apiurl"]
	if !present && credentials["environmentid"] != "" {
		apiurl = "https://" + credentials["environmentid"] + ".live.dynatrace.com/api"
	}

	if apiurl == "" {
		return errors.New("'environmentid' or 'apiurl' has to be specified in the service credentials!")
	}

	if credentials["apitoken"] == "" {
		return errors.New("'apitoken' has to be specified in the service credentials!")
	}

	url := apiurl + "/v1/deployment/installer/agent/unix/paas-sh/latest?include=nginx&bitness=64&Api-Token=" + credentials["apitoken"]
	installerPath := filepath.Join(os.TempDir(), "paasInstaller.sh")

	stager.Log.Debug("Downloading '%s' to '%s'", url, installerPath)
	err := h.downloadFile(url, installerPath)
	if err != nil {
		return err
	}

	stager.Log.Debug("Making %s executable...", installerPath)
	os.Chmod(installerPath, 0755)

	stager.Log.BeginStep("Starting Dynatrace PaaS agent installer")

	if os.Getenv("BP_DEBUG") != "" {
		err = stager.Command.Run(installerPath, stager.BuildDir)
	} else {
		_, err = stager.Command.CaptureOutput(installerPath, stager.BuildDir)
	}
	if err != nil {
		return err
	}

	stager.Log.Info("Dynatrace PaaS agent installed.")

	dynatraceEnvName := "dynatrace-env.sh"
	installDir := filepath.Join(stager.BuildDir, "dynatrace", "oneagent")
	dynatraceEnvPath := filepath.Join(stager.DepDir(), "profile.d", dynatraceEnvName)
	agentLibPath := "dynatrace/oneagent/agent/lib64/liboneagentproc.so"

	_, err = os.Stat(filepath.Join(stager.BuildDir, agentLibPath))
	if os.IsNotExist(err) {
		stager.Log.Error("Agent library (%s) not found!", agentLibPath)
		return err
	}

	stager.Log.BeginStep("Setting up Dynatrace PaaS agent injection...")
	stager.Log.Debug("Copy %s to %s", dynatraceEnvName, dynatraceEnvPath)
	err = libbuildpack.CopyFile(filepath.Join(installDir, dynatraceEnvName), dynatraceEnvPath)
	if err != nil {
		return err
	}

	stager.Log.Debug("Open %s for modification...", dynatraceEnvPath)
	f, err := os.OpenFile(dynatraceEnvPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	defer f.Close()

	stager.Log.Debug("Write LD_PRELOAD...")
	_, err = f.WriteString("\nexport LD_PRELOAD=${HOME}/" + agentLibPath)
	if err != nil {
		return err
	}

	stager.Log.Debug("Write DT_HOST_ID...")
	_, err = f.WriteString("\nexport DT_HOST_ID=" + h.appName() + "_${CF_INSTANCE_INDEX}")
	if err != nil {
		return err
	}

	stager.Log.Info("Dynatrace PaaS agent injection is set up.")

	return nil
}

func (h DynatraceHook) dtCredentials() map[string]string {
	var vcapServices map[string][]struct {
		Name        string            `json:"name"`
		Credentials map[string]string `json:"credentials"`
	}
	err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
	if err != nil {
		return nil
	}

	for _, services := range vcapServices {
		for _, service := range services {
			if strings.Contains(service.Name, "dynatrace") {
				return service.Credentials
			}
		}
	}

	return nil
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

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("Download returned with status " + resp.Status)
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
