package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testProxy(platform switchblade.Platform, fixtures, uri string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		it("installs pip packages through a proxy", func() {
			deployment, logs, err := platform.Deploy.
				WithEnv(map[string]string{
					"HTTP_PROXY":  uri,
					"HTTPS_PROXY": uri,
				}).
				WithoutInternetAccess().
				Execute(name, filepath.Join(fixtures, "default", "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(MatchRegexp(`Installing nginx [\d\.]+`)), logs.String())

			Eventually(deployment).Should(Serve(ContainSubstring("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")))
		})
	}
}
