package integration_test

import (
	"os/exec"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Staticfile Buildpack", func() {
	var app *cutlass.App
	var createdServices []string

	BeforeEach(func() {
		app = cutlass.New(Fixtures("logenv"))
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(app)

		createdServices = make([]string, 0)
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
			app = nil
		}

		for _, service := range createdServices {
			command := exec.Command("cf", "delete-service", "-f", service)
			_, err := command.Output()
			Expect(err).To(BeNil())
		}
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
			credentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", credentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, credentialsServiceName)

			duplicateCredentialsServiceName := "dynatrace-dupe-" + cutlass.RandStringRunes(20) + "-service"
			command = exec.Command("cf", "cups", duplicateCredentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, duplicateCredentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, credentialsServiceName)
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
			credentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", credentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paasFAILING/manifest\",\"environmentid\":\"envid\",\"skiperrors\":\"true\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, credentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, credentialsServiceName)
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
			credentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", credentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paas/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, credentialsServiceName)

			tagsServiceName := "dynatrace-tags-" + cutlass.RandStringRunes(20) + "-service"
			command = exec.Command("cf", "cups", tagsServiceName, "-p", "'{\"tag:dttest\":\"dynatrace_test\"}'")
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, tagsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, credentialsServiceName)
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
			credentialsServiceName := "dynatrace-" + cutlass.RandStringRunes(20) + "-service"
			command := exec.Command("cf", "cups", credentialsServiceName, "-p", "'{\"apitoken\":\"secretpaastoken\",\"apiurl\":\"https://s3.amazonaws.com/dt-paasFAILING/manifest\",\"environmentid\":\"envid\"}'")
			_, err := command.CombinedOutput()
			Expect(err).To(BeNil())
			createdServices = append(createdServices, credentialsServiceName)

			command = exec.Command("cf", "bind-service", app.Name, credentialsServiceName)
			_, err = command.CombinedOutput()
			Expect(err).To(BeNil())

			command = exec.Command("cf", "restage", app.Name)
			_, err = command.CombinedOutput()

			Eventually(app.Stdout.String).Should(ContainSubstring("Error during installer download, retrying in 4s"))
			Eventually(app.Stdout.String).Should(ContainSubstring("Error during installer download, retrying in 5s"))
			Eventually(app.Stdout.String).Should(ContainSubstring("Error during installer download, retrying in 7s"))
			Eventually(app.Stdout.String).Should(ContainSubstring("Download returned with status 404"))

			Eventually(app.Stdout.String).Should(ContainSubstring("Failed to compile droplet"))
		})
	})

	Context("deploying a staticfile app with Dynatrace agent with single credentials service and a redis service", func() {
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
			createdServices = append(createdServices, redisServiceName)
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
