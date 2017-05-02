package main

import (
	"os"
	_ "staticfile/hooks"
	"staticfile/supply"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	stager, err := libbuildpack.NewStager(os.Args[1:], libbuildpack.NewLogger())
	if err != nil {
		os.Exit(10)
	}

	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	err = libbuildpack.RunBeforeCompile(stager)
	if err != nil {
		stager.Log.Error("Before Compile: %s", err.Error())
		os.Exit(12)
	}

	err = libbuildpack.SetStagingEnvironment(stager.DepsDir)
	if err != nil {
		stager.Log.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(13)
	}

	ss := supply.Supplier{
		Stager: stager,
	}

	err = supply.Run(&ss)
	if err != nil {
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		stager.Log.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}
}
