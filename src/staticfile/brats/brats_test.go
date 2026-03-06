package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Staticfile buildpack", func() {
	bratshelper.UnbuiltBuildpack("nginx", CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("nginx", CopyBrats)
})
