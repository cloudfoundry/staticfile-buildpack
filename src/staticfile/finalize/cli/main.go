package main

import (
	"os"
	"staticfile/finalize"
	_ "staticfile/hooks"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	stager, err := libbuildpack.NewStager(os.Args[1:], libbuildpack.NewLogger())
	if err != nil {
		os.Exit(10)
	}

	if err := libbuildpack.SetStagingEnvironment(stager.DepsDir); err != nil {
		stager.Log.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(11)
	}

	sf := finalize.Finalizer{
		Stager: stager,
		YAML:   libbuildpack.NewYAML(),
	}

	if err := finalize.Run(&sf); err != nil {
		os.Exit(12)
	}

	if err := libbuildpack.RunAfterCompile(stager); err != nil {
		stager.Log.Error("After Compile: %s", err.Error())
		os.Exit(13)
	}

	if err := libbuildpack.SetLaunchEnvironment(stager.DepsDir, stager.BuildDir); err != nil {
		stager.Log.Error("Unable to setup launch environment: %s", err.Error())
		os.Exit(14)
	}

	stager.StagingComplete()
}
