package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

const (
	Regexp         = `\[.*/nginx-static\_[\d+\.]+\_linux\_x64\_(cflinuxfs.*_)?[\da-f]+\.tgz\]`
	DownloadRegexp = "Download " + Regexp
	CopyRegexp     = "Copy " + Regexp
)

func testCache(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = switchblade.Source(filepath.Join(fixtures, "default", "simple"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			if t.Failed() && name != "" {
				t.Logf("❌ FAILED TEST - App/Container: %s", name)
				t.Logf("   Platform: %s", settings.Platform)
			}
			if name != "" && (!settings.KeepFailedContainers || !t.Failed()) {
				Expect(platform.Delete.Execute(name)).To(Succeed())
			}
		})

		it("uses the cache for manifest dependencies", func() {
			deploy := platform.Deploy

			_, logs, err := deploy.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(MatchRegexp(DownloadRegexp)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(CopyRegexp)))

			_, logs, err = deploy.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).NotTo(ContainLines(MatchRegexp(DownloadRegexp)))
			Expect(logs).To(ContainLines(MatchRegexp(CopyRegexp)))
		})
	}
}
