package integration_test

import (
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Statifile Buildpack", func() {
	var app *cutlass.App
	var createdServices []string
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil

		for _, service := range createdServices {
			command := exec.Command("cf", "delete-service", "-f", service)
			_, err := command.Output()
			Expect(err).To(BeNil())
		}
	})

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "logenv"))
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(app)

		createdServices = make([]string, 0)
	})

	Context("deploying a staticfile app with Dynatrace agent with single credentials service", func() {
		It("checks if Dynatrace injection was successful", func() {

			serviceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", serviceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, serviceName)

			command = exec.Command("cf", "bind-service", app.Name, serviceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace PaaS agent."))
			Expect(app.Stdout.String()).To(ContainSubstring("Starting Dynatrace PaaS agent installer"))
			Expect(app.Stdout.String()).To(ContainSubstring("Copy dynatrace-env.sh"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent installed."))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent injection is set up."))
		})
	})

	Context("deploying a staticfile app with Dynatrace agent with two credentials services", func() {
		It("checks if detection of second service with credentials works", func() {

			CredentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", CredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, CredentialsServiceName)

			duplicateCredentialsServiceName := "dynatrace-dupe-" + cutlass.RandStringRunes(20) + "-service"
			command = exec.Command("cf", "cups", duplicateCredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, duplicateCredentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, CredentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			command = exec.Command("cf", "bind-service", app.Name, duplicateCredentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.Stdout.String()).To(ContainSubstring("More than one matching service found!"))
		})
	})

	Context("deploying a staticfile app with Dynatrace agent with failing agent download and ignoring errors", func() {
		It("checks if skipping download errors works", func() {

			CredentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", CredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paasFAILING/manifest\",\"environmentid\":\"envid\",\"skiperrors\":\"true\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, CredentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, CredentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.Stdout.String()).To(ContainSubstring("Download returned with status 404"))
			Expect(app.Stdout.String()).To(ContainSubstring("Error during installer download, skipping installation"))
		})
	})
	Context("deploying a staticfile app with Dynatrace agent with two dynatrace services", func() {
		It("check if service detection isn't disturbed by a service with tags", func() {

			CredentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", CredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, CredentialsServiceName)

			tagsServiceName := "dynatrace-tags-" + cutlass.RandStringRunes(20) + "-service"
			command = exec.Command("cf", "cups", tagsServiceName, "-p", "'{\"tag:dttest\":\"dynatrace_test\"}'")
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, tagsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, CredentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			command = exec.Command("cf", "bind-service", app.Name, tagsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace PaaS agent."))
			Expect(app.Stdout.String()).To(ContainSubstring("Starting Dynatrace PaaS agent installer"))
			Expect(app.Stdout.String()).To(ContainSubstring("Copy dynatrace-env.sh"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent installed."))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent injection is set up."))

		})
	})

	Context("deploying a staticfile app with Dynatrace agent with single credentials service and without manifest.json", func() {
		It("checks if Dynatrace injection was successful", func() {

			serviceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", serviceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, serviceName)

			command = exec.Command("cf", "bind-service", app.Name, serviceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace PaaS agent."))
			Expect(app.Stdout.String()).To(ContainSubstring("Starting Dynatrace PaaS agent installer"))
			Expect(app.Stdout.String()).To(ContainSubstring("Copy dynatrace-env.sh"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent installed."))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent injection is set up."))
		})
	})

	Context("deploying a staticfile app with Dynatrace agent with failing agent download and checking retry", func() {
		It("checks if retrying downloads works", func() {

			CredentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", CredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paasFAILING/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, CredentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, CredentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.CombinedOutput()

			Expect(app.Stdout.String()).To(ContainSubstring("Error during installer download, retrying in 4 seconds"))
			Expect(app.Stdout.String()).To(ContainSubstring("Error during installer download, retrying in 5 seconds"))
			Expect(app.Stdout.String()).To(ContainSubstring("Error during installer download, retrying in 7 seconds"))
			Expect(app.Stdout.String()).To(ContainSubstring("Download returned with status 404"))

			Expect(app.Stdout.String()).To(ContainSubstring("Failed to compile droplet"))
		})
	})

	Context("deploying a NodeJS app with Dynatrace agent with single credentials service and a redis service", func() {
		It("checks if Dynatrace injection was successful", func() {
			serviceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", serviceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, serviceName)
			command = exec.Command("cf", "bind-service", app.Name, serviceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			redisServiceName := "redis-" + cutlass.RandStringRunes(20) + "-service"
			command = exec.Command("cf", "cups", redisServiceName, "-p", "'{\"name\":\"redis\", \"credentials\":{\"db_type\":\"redis\", \"instance_administration_api\":{\"deployment_id\":\"12345asdf\", \"instance_id\":\"12345asdf\", \"root\":\"https://doesnotexi.st\"}}}'")
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, serviceName)
			command = exec.Command("cf", "bind-service", app.Name, redisServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.Output()
			Expect(err).To(BeNil())

			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace service credentials found. Setting up Dynatrace PaaS agent."))
			Expect(app.Stdout.String()).To(ContainSubstring("Starting Dynatrace PaaS agent installer"))
			Expect(app.Stdout.String()).To(ContainSubstring("Copy dynatrace-env.sh"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent installed."))
			Expect(app.Stdout.String()).To(ContainSubstring("Dynatrace PaaS agent injection is set up."))
		})
	})
})
