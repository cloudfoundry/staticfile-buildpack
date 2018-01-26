package finalize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
)

type Staticfile struct {
	RootDir               string `yaml:"root"`
	HostDotFiles          bool   `yaml:"host_dot_files"`
	LocationInclude       string `yaml:"location_include"`
	DirectoryIndex        bool   `yaml:"directory"`
	SSI                   bool   `yaml:"ssi"`
	PushState             bool   `yaml:"pushstate"`
	HSTS                  bool   `yaml:"http_strict_transport_security"`
	HSTSIncludeSubDomains bool   `yaml:"http_strict_transport_security_include_subdomains"`
	HSTSPreload           bool   `yaml:"http_strict_transport_security_preload"`
	ForceHTTPS            bool   `yaml:"force_https"`
	BasicAuth             bool
	StatusCodes           map[string]string `yaml:"status_codes"`
}

type YAML interface {
	Load(string, interface{}) error
}

type Finalizer struct {
	BuildDir string
	DepDir   string
	Log      *libbuildpack.Logger
	Config   Staticfile
	YAML     YAML
}
type StaticfileTemp struct {
	RootDir               string            `yaml:"root,omitempty"`
	HostDotFiles          string            `yaml:"host_dot_files,omitempty"`
	LocationInclude       string            `yaml:"location_include"`
	DirectoryIndex        string            `yaml:"directory"`
	SSI                   string            `yaml:"ssi"`
	PushState             string            `yaml:"pushstate"`
	HSTS                  string            `yaml:"http_strict_transport_security"`
	HSTSIncludeSubDomains string            `yaml:"http_strict_transport_security_include_subdomains"`
	HSTSPreload           string            `yaml:"http_strict_transport_security_preload"`
	ForceHTTPS            string            `yaml:"force_https"`
	StatusCodes           map[string]string `yaml:"status_codes"`
}

var skipCopyFile = map[string]bool{
	"Staticfile":      true,
	"Staticfile.auth": true,
	"manifest.yml":    true,
	".profile":        true,
	".profile.d":      true,
	"stackato.yml":    true,
	".cloudfoundry":   true,
}

func Run(sf *Finalizer) error {
	var err error

	err = sf.LoadStaticfile()
	if err != nil {
		sf.Log.Error("Unable to load Staticfile: %s", err.Error())
		return err
	}

	appRootDir, err := sf.GetAppRootDir()
	if err != nil {
		sf.Log.Error("Invalid root directory: %s", err.Error())
		return err
	}

	sf.Warnings()

	err = sf.CopyFilesToPublic(appRootDir)
	if err != nil {
		sf.Log.Error("Unable to copy project files: %s", err.Error())
		return err
	}

	err = sf.ConfigureNginx()
	if err != nil {
		sf.Log.Error("Unable to configure nginx: %s", err.Error())
		return err
	}

	err = sf.WriteStartupFiles()
	if err != nil {
		sf.Log.Error("Unable to write startup file: %s", err.Error())
		return err
	}
	return nil
}

func (sf *Finalizer) WriteStartupFiles() error {
	profiledDir := filepath.Join(sf.DepDir, "profile.d")
	err := os.MkdirAll(profiledDir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(profiledDir, "staticfile.sh"), []byte(initScript), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(sf.BuildDir, "start_logging.sh"), []byte(startLoggingScript), 0755)
	if err != nil {
		return err
	}

	bootScript := filepath.Join(sf.BuildDir, "boot.sh")
	return ioutil.WriteFile(bootScript, []byte(startCommand), 0755)
}

func (sf *Finalizer) LoadStaticfile() error {
	var hash StaticfileTemp
	conf := &sf.Config

	err := sf.YAML.Load(filepath.Join(sf.BuildDir, "Staticfile"), &hash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	isEnabled := func(value string) bool {
		return (value == "enabled" || value == "true")
	}

	if hash.RootDir != "" {
		conf.RootDir = hash.RootDir
	}

	if isEnabled(hash.HostDotFiles) {
		sf.Log.BeginStep("Enabling hosting of dotfiles")
		conf.HostDotFiles = true
	}

	conf.LocationInclude = hash.LocationInclude
	if conf.LocationInclude != "" {
		sf.Log.BeginStep("Enabling location include file %s", conf.LocationInclude)
	}

	if hash.DirectoryIndex != "" {
		sf.Log.BeginStep("Enabling directory index for folders without index.html files")
		conf.DirectoryIndex = true
	}

	if isEnabled(hash.SSI) {
		sf.Log.BeginStep("Enabling SSI")
		conf.SSI = true
	}

	if isEnabled(hash.PushState) {
		sf.Log.BeginStep("Enabling pushstate")
		conf.PushState = true
	}

	if isEnabled(hash.HSTS) {
		sf.Log.BeginStep("Enabling HSTS")
		conf.HSTS = true
	}
	if isEnabled(hash.HSTSIncludeSubDomains) {
		sf.Log.BeginStep("Enabling HSTS includeSubDomains")
		conf.HSTSIncludeSubDomains = true
	}
	if isEnabled(hash.HSTSPreload) {
		sf.Log.BeginStep("Enabling HSTS Preload")
		conf.HSTSPreload = true
	}
	if isEnabled(hash.ForceHTTPS) {
		sf.Log.BeginStep("Enabling HTTPS redirect")
		conf.ForceHTTPS = true
	}
	if len(hash.StatusCodes) > 0 {
		sf.Log.BeginStep("Enabling custom pages for status_codes")
		conf.StatusCodes = sf.getStatusCodes(hash.StatusCodes)
	}

	if !conf.HSTS && (conf.HSTSIncludeSubDomains || conf.HSTSPreload) {
		sf.Log.Warning("http_strict_transport_security is not enabled while http_strict_transport_security_include_subdomains or http_strict_transport_security_preload have been enabled.")
		sf.Log.Protip("http_strict_transport_security_include_subdomains and http_strict_transport_security_preload do nothing without http_strict_transport_security enabled.", "https://docs.cloudfoundry.org/buildpacks/staticfile/index.html#strict-security")
	}

	authFile := filepath.Join(sf.BuildDir, "Staticfile.auth")
	_, err = os.Stat(authFile)
	if err == nil {
		conf.BasicAuth = true
		sf.Log.BeginStep("Enabling basic authentication using Staticfile.auth")
		sf.Log.Protip("Learn about basic authentication", "https://docs.cloudfoundry.org/buildpacks/staticfile/index.html#authentication")
	}

	return nil
}

func (sf *Finalizer) getStatusCodes(codes map[string]string) map[string]string {
	var versions map[string]string
	versions = make(map[string]string)
	for key, value := range codes {
		if strings.Contains(key, "4xx") {
			key = "400 401 402 403 404 405 406 407 408 409 410 411 412 413 414 415 416 417 418 421 422 423 424 426 428 429 431 451"
		} else if strings.Contains(key, "5xx") {
			key = "500 501 502 503 504 505 506 507 508 510 511"
		}
		versions[key] = value
	}
	return versions
}

func (sf *Finalizer) GetAppRootDir() (string, error) {
	var rootDirRelative string

	if sf.Config.RootDir != "" {
		rootDirRelative = sf.Config.RootDir
	} else {
		rootDirRelative = "."
	}

	rootDirAbs, err := filepath.Abs(filepath.Join(sf.BuildDir, rootDirRelative))
	if err != nil {
		return "", err
	}

	sf.Log.BeginStep("Root folder %s", rootDirAbs)

	dirInfo, err := os.Stat(rootDirAbs)
	if err != nil {
		return "", fmt.Errorf("the application Staticfile specifies a root directory %s that does not exist", rootDirRelative)
	}

	if !dirInfo.IsDir() {
		return "", fmt.Errorf("the application Staticfile specifies a root directory %s that is a plain file, but was expected to be a directory", rootDirRelative)
	}

	return rootDirAbs, nil
}

func (sf *Finalizer) CopyFilesToPublic(appRootDir string) error {
	sf.Log.BeginStep("Copying project files into public")

	publicDir := filepath.Join(sf.BuildDir, "public")

	if publicDir == appRootDir {
		return nil
	}

	tmpDir, err := ioutil.TempDir("", "staticfile-buildpack.approot.")
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(appRootDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if skipCopyFile[file.Name()] {
			continue
		}

		if strings.HasPrefix(file.Name(), ".") && !sf.Config.HostDotFiles {
			continue
		}

		err = os.Rename(filepath.Join(appRootDir, file.Name()), filepath.Join(tmpDir, file.Name()))
		if err != nil {
			return err
		}
	}

	if err := os.RemoveAll(publicDir); err != nil {
		return err
	}

	if err := os.Rename(tmpDir, publicDir); err != nil {
		return err
	}

	return nil
}

func (sf *Finalizer) Warnings() {
	if len(sf.Config.LocationInclude) > 0 && len(sf.Config.RootDir) == 0 {
		sf.Log.Warning("The location_include directive only works in conjunction with root.\nPlease specify root to use location_include")
	}

	if filepath.Clean(sf.Config.RootDir) == "." {
		found, _ := libbuildpack.FileExists(filepath.Join(sf.BuildDir, "nginx", "conf"))
		if found {
			sf.Log.Info("\n\n\n")
			sf.Log.Warning("You have an nginx/conf directory, but have not set *root*, or have set it to '.'.\nIf you are using the nginx/conf directory for nginx configuration, you probably need to also set the *root* directive.")
			sf.Log.Info("\n\n\n")
		}
	}
}

func (sf *Finalizer) ConfigureNginx() error {
	var err error

	sf.Log.BeginStep("Configuring nginx")

	nginxConf, err := sf.generateNginxConf()
	if err != nil {
		sf.Log.Error("Unable to generate nginx.conf: %s", err.Error())
		return err
	}

	confDir := filepath.Join(sf.BuildDir, "nginx", "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return err
	}

	logsDir := filepath.Join(sf.BuildDir, "nginx", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	confFiles := map[string]string{
		"nginx.conf": nginxConf,
		"mime.types": MimeTypes}

	for file, contents := range confFiles {
		confDest := filepath.Join(confDir, file)
		customConfFile := filepath.Join(sf.BuildDir, "public", file)

		_, err = os.Stat(customConfFile)
		if err == nil {
			err = os.Rename(customConfFile, confDest)
			if file == "nginx.conf" {
				sf.Log.Warning("overriding nginx.conf is deprecated and highly discouraged, as it breaks the functionality of the Staticfile and Staticfile.auth configuration directives. Please use the NGINX buildpack available at: https://github.com/cloudfoundry/nginx-buildpack")
			}
		} else {
			err = ioutil.WriteFile(confDest, []byte(contents), 0644)
		}

		if err != nil {
			return err
		}
	}

	if sf.Config.BasicAuth {
		authFile := filepath.Join(sf.BuildDir, "Staticfile.auth")
		err = libbuildpack.CopyFile(authFile, filepath.Join(confDir, ".htpasswd"))
		if err != nil {
			return err
		}
	}

	return nil
}

func (sf *Finalizer) generateNginxConf() (string, error) {
	buffer := new(bytes.Buffer)

	t := template.Must(template.New("nginx.conf").Parse(nginxConfTemplate))

	err := t.Execute(buffer, sf.Config)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
