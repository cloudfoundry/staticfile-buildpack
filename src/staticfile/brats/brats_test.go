package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Staticfile buildpack", func() {
	bratshelper.UnbuiltBuildpack("nginx", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(`nginx\-[\d\.]+\-linux\-x64\-(cflinuxfs.*-)?[\da-f]+\.tgz`, CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("nginx", CopyBrats)
	bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)
})
