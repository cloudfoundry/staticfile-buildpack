package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("override yml", func() {
	var app *cutlass.App
	var buildpackName string
	AfterEach(func() {
		if buildpackName != "" {
			cutlass.DeleteBuildpack(buildpackName)
		}

		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	BeforeEach(func() {
		if !ApiHasMultiBuildpack() {
			Skip("Multi buildpack support is required")
		}

		buildpackName = "override_yml_" + cutlass.RandStringRunes(5)
		Expect(cutlass.CreateOrUpdateBuildpack(buildpackName, filepath.Join(bpDir, "fixtures", "overrideyml_bp"), "")).To(Succeed())

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "staticfile_app"))
		app.Buildpacks = []string{buildpackName + "_buildpack", "staticfile_buildpack"}
	})

	It("Forces nginx from override buildpack", func() {
		Expect(app.Push()).ToNot(Succeed())
		Expect(app.Stdout.String()).To(ContainSubstring("-----> OverrideYML Buildpack"))
		Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

		Eventually(app.Stdout.String).Should(ContainSubstring("-----> Installing nginx"))
		Eventually(app.Stdout.String).Should(MatchRegexp("Copy .*/nginx.tgz"))
		Eventually(app.Stdout.String).Should(ContainSubstring("Unable to install nginx: dependency sha256 mismatch: expected sha256 062d906c87839d03b243e2821e10653c89b4c92878bfe2bf995dec231e117bfc, actual sha256 b56b58ac21f9f42d032e1e4b8bf8b8823e69af5411caa15aee2b140bc756962f"))
	})
})
