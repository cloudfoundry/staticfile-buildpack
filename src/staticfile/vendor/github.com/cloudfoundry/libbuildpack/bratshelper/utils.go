package bratshelper

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/gomega"
)

func PushApp(app *cutlass.App) {
	Expect(app.Push()).To(Succeed(), "Failed to push %s", app.Name)
	Eventually(app.InstanceStates, 20*time.Second).Should(Equal([]string{"RUNNING"}))
}

func DestroyApp(app *cutlass.App) {
	if app != nil {
		app.Destroy()
	}
}

func AddDotProfileScriptToApp(dir string) {
	profilePath := filepath.Join(dir, ".profile")
	Expect(ioutil.WriteFile(profilePath, []byte(`#!/usr/bin/env bash
echo PROFILE_SCRIPT_IS_PRESENT_AND_RAN
`), 0755)).To(Succeed())
}
