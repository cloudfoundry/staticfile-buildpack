package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"bytes"

	bp "github.com/cloudfoundry/libbuildpack"
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

type StaticfileCompiler struct {
	Compiler *bp.Compiler
	Config   Staticfile
	YAML     bp.YAML
}

var skipCopyFile = map[string]bool{
	"Staticfile":      true,
	"Staticfile.auth": true,
	"manifest.yml":    true,
	".profile":        true,
	".profile.d":      true,
	"stackato.yml":    true,
}

func main() {
	buildDir := os.Args[1]
	cacheDir := os.Args[2]

	compiler, err := bp.NewCompiler(buildDir, cacheDir, bp.NewLogger())
	err = compiler.CheckBuildpackValid()
	if err != nil {
		panic(err)
	}

	sc := StaticfileCompiler{Compiler: compiler, Config: Staticfile{}, YAML: bp.NewYAML()}
	err = sc.Compile()
	if err != nil {
		panic(err)
	}

	compiler.StagingComplete()
}

func (sc *StaticfileCompiler) Compile() error {
	var err error

	err = sc.LoadStaticfile()
	if err != nil {
		sc.Compiler.Log.Error("Unable to load Staticfile: %s", err.Error())
		return err
	}

	appRootDir, err := sc.GetAppRootDir()
	if err != nil {
		sc.Compiler.Log.Error("Invalid root directory: %s", err.Error())
		return err
	}

	err = sc.CopyFilesToPublic(appRootDir)
	if err != nil {
		sc.Compiler.Log.Error("Unable to copy project files: %s", err.Error())
		return err
	}

	err = sc.InstallNginx()
	if err != nil {
		sc.Compiler.Log.Error("Unable to install nginx: %s", err.Error())
		return err
	}

	err = sc.ConfigureNginx()
	if err != nil {
		sc.Compiler.Log.Error("Unable to configure nginx: %s", err.Error())
		return err
	}

	err = sc.WriteProfileD()
	if err != nil {
		sc.Compiler.Log.Error("Could not write .profile.d script: %s", err.Error())
		return err
	}

	return nil
}

func (sc *StaticfileCompiler) LoadStaticfile() error {
	var hash = make(map[string]string)
	conf := &sc.Config

	err := sc.YAML.Load(filepath.Join(sc.Compiler.BuildDir, "Staticfile"), &hash)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for key, value := range hash {
		isEnabled := (value == "enabled" || value == "true")
		switch key {
		case "root":
			conf.RootDir = value
		case "host_dot_files":
			if isEnabled {
				sc.Compiler.Log.BeginStep("Enabling hosting of dotfiles")
				conf.HostDotFiles = true
			}
		case "location_include":
			conf.LocationInclude = value
			if conf.LocationInclude != "" {
				sc.Compiler.Log.BeginStep("Enabling location include file %s", conf.LocationInclude)
			}
		case "directory":
			if value != "" {
				sc.Compiler.Log.BeginStep("Enabling directory index for folders without index.html files")
				conf.DirectoryIndex = true
			}
		case "ssi":
			if isEnabled {
				sc.Compiler.Log.BeginStep("Enabling SSI")
				conf.SSI = true
			}
		case "pushstate":
			if isEnabled {
				sc.Compiler.Log.BeginStep("Enabling pushstate")
				conf.PushState = true
			}
		case "http_strict_transport_security":
			if isEnabled {
				sc.Compiler.Log.BeginStep("Enabling HSTS")
				conf.HSTS = true
			}
		case "force_https":
			if isEnabled {
				sc.Compiler.Log.BeginStep("Enabling HTTPS redirect")
				conf.ForceHTTPS = true
			}
		}
	}

	authFile := filepath.Join(sc.Compiler.BuildDir, "Staticfile.auth")
	_, err = os.Stat(authFile)
	if err == nil {
		conf.BasicAuth = true
		sc.Compiler.Log.BeginStep("Enabling basic authentication using Staticfile.auth")
		sc.Compiler.Log.Protip("Learn about basic authentication", "http://docs.cloudfoundry.org/buildpacks/staticfile/index.html#authentication")
	}

	return nil
}

func (sc *StaticfileCompiler) GetAppRootDir() (string, error) {
	var rootDirRelative string

	if sc.Config.RootDir != "" {
		rootDirRelative = sc.Config.RootDir
	} else {
		rootDirRelative = "."
	}

	rootDirAbs, err := filepath.Abs(filepath.Join(sc.Compiler.BuildDir, rootDirRelative))
	if err != nil {
		return "", err
	}

	sc.Compiler.Log.BeginStep("Root folder %s", rootDirAbs)

	dirInfo, err := os.Stat(rootDirAbs)
	if err != nil {
		return "", fmt.Errorf("the application Staticfile specifies a root directory %s that does not exist", rootDirRelative)
	}

	if !dirInfo.IsDir() {
		return "", fmt.Errorf("the application Staticfile specifies a root directory %s that is a plain file, but was expected to be a directory", rootDirRelative)
	}

	return rootDirAbs, nil
}

func (sc *StaticfileCompiler) CopyFilesToPublic(appRootDir string) error {
	sc.Compiler.Log.BeginStep("Copying project files into public")

	publicDir := filepath.Join(sc.Compiler.BuildDir, "public")

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

		if strings.HasPrefix(file.Name(), ".") && !sc.Config.HostDotFiles {
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

func (sc *StaticfileCompiler) InstallNginx() error {
	sc.Compiler.Log.BeginStep("Installing nginx")

	nginx, err := sc.Compiler.Manifest.DefaultVersion("nginx")
	if err != nil {
		return err
	}
	sc.Compiler.Log.Info("Using nginx version %s", nginx.Version)

	return sc.Compiler.Manifest.InstallDependency(nginx, sc.Compiler.BuildDir)
}

func (sc *StaticfileCompiler) ConfigureNginx() error {
	var err error

	sc.Compiler.Log.BeginStep("Configuring nginx")

	nginxConf, err := sc.generateNginxConf()
	if err != nil {
		sc.Compiler.Log.Error("Unable to generate nginx.conf: %s", err.Error())
		return err
	}

	confFiles := map[string]string{
		"nginx.conf": nginxConf,
		"mime.types": MimeTypes}

	for file, contents := range confFiles {
		confDest := filepath.Join(sc.Compiler.BuildDir, "nginx", "conf", file)
		customConfFile := filepath.Join(sc.Compiler.BuildDir, "public", file)

		_, err = os.Stat(customConfFile)
		if err == nil {
			err = bp.CopyFile(customConfFile, confDest)
		} else {
			err = ioutil.WriteFile(confDest, []byte(contents), 0644)
		}

		if err != nil {
			return err
		}
	}

	if sc.Config.BasicAuth {
		authFile := filepath.Join(sc.Compiler.BuildDir, "Staticfile.auth")
		err = bp.CopyFile(authFile, filepath.Join(sc.Compiler.BuildDir, "nginx", "conf", ".htpasswd"))
		if err != nil {
			return err
		}
	}

	return nil
}

func (sc *StaticfileCompiler) generateNginxConf() (string, error) {
	buffer := new(bytes.Buffer)

	t := template.Must(template.New("nginx.conf").Parse(NginxConfTemplate))

	err := t.Execute(buffer, sc.Config)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (sc *StaticfileCompiler) WriteProfileD() error {
	err := os.MkdirAll(filepath.Join(sc.Compiler.BuildDir, ".profile.d"), 0755)
	if err != nil {
		return err
	}

	script := filepath.Join(sc.Compiler.BuildDir, ".profile.d", "staticfile.sh")

	err = ioutil.WriteFile(script, []byte(InitScript), 0755)
	if err != nil {
		return err
	}

	return nil
}
