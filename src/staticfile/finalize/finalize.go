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
	RootDir         string `yaml:"root"`
	HostDotFiles    bool   `yaml:"host_dot_files"`
	LocationInclude string `yaml:"location_include"`
	DirectoryIndex  bool   `yaml:"directory"`
	SSI             bool   `yaml:"ssi"`
	PushState       bool   `yaml:"pushstate"`
	HSTS            bool   `yaml:"http_strict_transport_security"`
	ForceHTTPS      bool   `yaml:"force_https"`
	BasicAuth       bool
}

type Finalizer struct {
	Stager *libbuildpack.Stager
	Config Staticfile
	YAML   libbuildpack.YAML
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
		sf.Stager.Log.Error("Unable to load Staticfile: %s", err.Error())
		return err
	}

	appRootDir, err := sf.GetAppRootDir()
	if err != nil {
		sf.Stager.Log.Error("Invalid root directory: %s", err.Error())
		return err
	}

	err = sf.CopyFilesToPublic(appRootDir)
	if err != nil {
		sf.Stager.Log.Error("Unable to copy project files: %s", err.Error())
		return err
	}

	err = sf.ConfigureNginx()
	if err != nil {
		sf.Stager.Log.Error("Unable to configure nginx: %s", err.Error())
		return err
	}

	err = sf.WriteStartupFiles()
	if err != nil {
		sf.Stager.Log.Error("Unable to write startup file: %s", err.Error())
		return err
	}
	return nil
}

func (sf *Finalizer) WriteStartupFiles() error {
	profiledDir := filepath.Join(sf.Stager.DepDir(), "profile.d")
	err := os.MkdirAll(profiledDir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(profiledDir, "staticfile.sh"), []byte(initScript), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(sf.Stager.BuildDir, "start_logging.sh"), []byte(startLoggingScript), 0755)
	if err != nil {
		return err
	}

	bootScript := filepath.Join(sf.Stager.BuildDir, "boot.sh")
	return ioutil.WriteFile(bootScript, []byte(startCommand), 0755)
}

func (sf *Finalizer) LoadStaticfile() error {
	var hash = make(map[string]string)
	conf := &sf.Config

	err := sf.YAML.Load(filepath.Join(sf.Stager.BuildDir, "Staticfile"), &hash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for key, value := range hash {
		isEnabled := (value == "enabled" || value == "true")
		switch key {
		case "root":
			conf.RootDir = value
		case "host_dot_files":
			if isEnabled {
				sf.Stager.Log.BeginStep("Enabling hosting of dotfiles")
				conf.HostDotFiles = true
			}
		case "location_include":
			conf.LocationInclude = value
			if conf.LocationInclude != "" {
				sf.Stager.Log.BeginStep("Enabling location include file %s", conf.LocationInclude)
			}
		case "directory":
			if value != "" {
				sf.Stager.Log.BeginStep("Enabling directory index for folders without index.html files")
				conf.DirectoryIndex = true
			}
		case "ssi":
			if isEnabled {
				sf.Stager.Log.BeginStep("Enabling SSI")
				conf.SSI = true
			}
		case "pushstate":
			if isEnabled {
				sf.Stager.Log.BeginStep("Enabling pushstate")
				conf.PushState = true
			}
		case "http_strict_transport_security":
			if isEnabled {
				sf.Stager.Log.BeginStep("Enabling HSTS")
				conf.HSTS = true
			}
		case "force_https":
			if isEnabled {
				sf.Stager.Log.BeginStep("Enabling HTTPS redirect")
				conf.ForceHTTPS = true
			}
		}
	}

	authFile := filepath.Join(sf.Stager.BuildDir, "Staticfile.auth")
	_, err = os.Stat(authFile)
	if err == nil {
		conf.BasicAuth = true
		sf.Stager.Log.BeginStep("Enabling basic authentication using Staticfile.auth")
		sf.Stager.Log.Protip("Learn about basic authentication", "http://docs.cloudfoundry.org/buildpacks/staticfile/index.html#authentication")
	}

	return nil
}

func (sf *Finalizer) GetAppRootDir() (string, error) {
	var rootDirRelative string

	if sf.Config.RootDir != "" {
		rootDirRelative = sf.Config.RootDir
	} else {
		rootDirRelative = "."
	}

	rootDirAbs, err := filepath.Abs(filepath.Join(sf.Stager.BuildDir, rootDirRelative))
	if err != nil {
		return "", err
	}

	sf.Stager.Log.BeginStep("Root folder %s", rootDirAbs)

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
	sf.Stager.Log.BeginStep("Copying project files into public")

	publicDir := filepath.Join(sf.Stager.BuildDir, "public")

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

	err = os.Rename(tmpDir, publicDir)
	if err != nil {
		return err
	}

	return nil
}

func (sf *Finalizer) ConfigureNginx() error {
	var err error

	sf.Stager.Log.BeginStep("Configuring nginx")

	nginxConf, err := sf.generateNginxConf()
	if err != nil {
		sf.Stager.Log.Error("Unable to generate nginx.conf: %s", err.Error())
		return err
	}

	confDir := filepath.Join(sf.Stager.BuildDir, "nginx", "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return err
	}

	logsDir := filepath.Join(sf.Stager.BuildDir, "nginx", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	confFiles := map[string]string{
		"nginx.conf": nginxConf,
		"mime.types": MimeTypes}

	for file, contents := range confFiles {
		confDest := filepath.Join(confDir, file)
		customConfFile := filepath.Join(sf.Stager.BuildDir, "public", file)

		_, err = os.Stat(customConfFile)
		if err == nil {
			err = libbuildpack.CopyFile(customConfFile, confDest)
		} else {
			err = ioutil.WriteFile(confDest, []byte(contents), 0644)
		}

		if err != nil {
			return err
		}
	}

	if sf.Config.BasicAuth {
		authFile := filepath.Join(sf.Stager.BuildDir, "Staticfile.auth")
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
