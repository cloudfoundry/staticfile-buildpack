package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO explain when to make not pending
var _ = PDescribe("{{LANGUAGE}} buildpack", func() {
	bratshelper.UnbuiltBuildpack("{{LANGUAGE}}", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithBuildpackThatSetsEOL("{{LANGUAGE}}", CopyBrats)
	bratshelper.StagingWithADepThatIsNotTheLatest("{{LANGUAGE}}", CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(`{{LANGUAGE}}\-[\d\.]+\-linux\-x64\-[\da-f]+\.tgz`, CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("{{LANGUAGE}}", CopyBrats)
	bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)
	bratshelper.ForAllSupportedVersions("{{LANGUAGE}}", CopyBrats, func(version string, app *cutlass.App) {
		PushApp(app)

		By("does a thing", func() {
			Expect(app).ToNot(BeNil())
		})
	})
})
