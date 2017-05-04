require 'spec_helper'
require 'excon'
require 'open3'
require 'timeout'

describe 'deploy a staticfile app with dynatrace agent' do
	let(:app) {
		app = Machete.deploy_app('staticfile_app', env: {'BP_DEBUG' => '1'})
		Machete::SystemHelper.run_cmd(%(cf cups dynatrace-test-service -p '{"apitoken":"secretpaastoken","apiurl":"https://s3.amazonaws.com/dt-paas","environmentid":"envid"}'))
		Machete::SystemHelper.run_cmd(%(cf bind-service #{app.name} dynatrace-test-service))
		Machete::SystemHelper.run_cmd(%(cf restage #{app.name}))
		app
	}
	let(:browser) { Machete::Browser.new(app) }

	after do
		Machete::CF::DeleteApp.new.execute(app)
	end

	context "" do
		it "" do
			expect(app).to have_logged(/Buildpack version [\d\.]{5,}/)
			expect(app).to have_logged('Dynatrace service found. Setting up Dynatrace PaaS agent.')
			expect(app).to have_logged('Starting Dynatrace PaaS agent installer')
			expect(app).to have_logged('Copy dynatrace-env.sh')
			expect(app).to have_logged('Dynatrace PaaS agent installed.')
			expect(app).to have_logged('Dynatrace PaaS agent injection is set up.')
			expect(app).to be_running

			browser.visit_path('/')
			expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')
		end
	end
end
