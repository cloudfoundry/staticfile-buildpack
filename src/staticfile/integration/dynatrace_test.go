package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDynatrace(platform switchblade.Platform, fixtures, uri string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
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

		it("builds the app with a Dynatrace agent", func() {
			_, logs, err := platform.Deploy.
				WithEnv(map[string]string{"BP_DEBUG": "true"}).
				WithServices(map[string]switchblade.Service{
					"some-dynatrace": {
						"apitoken":      "secretpaastoken",
						"apiurl":        uri,
						"environmentid": "envid",
					},
				}).
				Execute(name, filepath.Join(fixtures, "default", "simple"))
			Expect(err).NotTo(HaveOccurred())

			Expect(logs.String()).To(SatisfyAll(
				ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace OneAgent."),
				ContainSubstring("Starting Dynatrace OneAgent installer"),
				ContainSubstring("Copy dynatrace-env.sh"),
				ContainSubstring("Dynatrace OneAgent installed."),
				ContainSubstring("Dynatrace OneAgent injection is set up."),
			))
		})

		context("when a network zone is configured", func() {
			it("builds the app with a Dynatrace agent", func() {
				_, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					WithServices(map[string]switchblade.Service{
						"some-dynatrace": {
							"apitoken":      "secretpaastoken",
							"apiurl":        uri,
							"environmentid": "envid",
							"networkzone":   "testzone",
						},
					}).
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace OneAgent."),
					ContainSubstring("Starting Dynatrace OneAgent installer"),
					ContainSubstring("Copy dynatrace-env.sh"),
					ContainSubstring("Setting DT_NETWORK_ZONE..."),
					ContainSubstring("Dynatrace OneAgent installed."),
					ContainSubstring("Dynatrace OneAgent injection is set up."),
				))
			})
		})

		context("when there is more than one matching service binding", func() {
			it("builds the app with a Dynatrace agent", func() {
				_, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					WithServices(map[string]switchblade.Service{
						"dynatrace-service-1": {
							"apitoken":      "secretpaastoken",
							"apiurl":        uri,
							"environmentid": "envid",
						},
						"dynatrace-service-2": {
							"apitoken":      "secretpaastoken",
							"apiurl":        uri,
							"environmentid": "envid",
						},
					}).
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("More than one matching service found!"),
					ContainSubstring("Dynatrace service credentials not found!"),
					Not(ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace OneAgent.")),
				))
			})
		})

		context("when the agent download fails", func() {
			it("checks if retrying downloads works", func() {
				_, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					WithServices(map[string]switchblade.Service{
						"dynatrace-service": {
							"apitoken":      "secretpaastoken",
							"apiurl":        fmt.Sprintf("%s/no-such-endpoint", uri),
							"environmentid": "envid",
						},
					}).
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).To(HaveOccurred()) // NOTE: the build intentionally fails here

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Error during installer download, retrying in 4s"),
					ContainSubstring("Error during installer download, retrying in 5s"),
					ContainSubstring("Error during installer download, retrying in 7s"),
					ContainSubstring("Download returned with status 404"),
					ContainSubstring("Failed to compile droplet"),
				))
			})

			context("when the service is configured to skip errors", func() {
				it("skips errors and builds successfully", func() {
					_, logs, err := platform.Deploy.
						WithEnv(map[string]string{"BP_DEBUG": "true"}).
						WithServices(map[string]switchblade.Service{
							"dynatrace-service": {
								"apitoken":      "secretpaastoken",
								"apiurl":        fmt.Sprintf("%s/no-such-endpoint", uri),
								"environmentid": "envid",
								"skiperrors":    "true",
							},
						}).
						Execute(name, filepath.Join(fixtures, "default", "simple"))
					Expect(err).NotTo(HaveOccurred())

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Download returned with status 404"),
						ContainSubstring("Error during installer download, skipping installation"),
					))
				})
			})
		})

		context("when there is a service bindings providing tags", func() {
			it("builds the app with a Dynatrace agent", func() {
				_, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "true"}).
					WithServices(map[string]switchblade.Service{
						"dynatrace-service-1": {
							"apitoken":      "secretpaastoken",
							"apiurl":        uri,
							"environmentid": "envid",
						},
						"dynatrace-service-2": {
							"tag:dttest": "dynatrace_test",
						},
					}).
					Execute(name, filepath.Join(fixtures, "default", "simple"))
				Expect(err).NotTo(HaveOccurred())

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace OneAgent."),
					ContainSubstring("Starting Dynatrace OneAgent installer"),
					ContainSubstring("Copy dynatrace-env.sh"),
					ContainSubstring("Dynatrace OneAgent installed."),
					ContainSubstring("Dynatrace OneAgent injection is set up."),
				))
			})
		})
	}
}
