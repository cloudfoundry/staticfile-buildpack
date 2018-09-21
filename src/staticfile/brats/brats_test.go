package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Staticfile buildpack", func() {
	bratshelper.UnbuiltBuildpack("nginx", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("nginx", CopyBrats)
	bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)
})
