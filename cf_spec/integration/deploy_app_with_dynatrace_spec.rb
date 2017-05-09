require 'spec_helper'
require 'excon'
require 'open3'
require 'timeout'

describe 'deploy a staticfile app with dynatrace agent' do
	let(:service_name) { "dynatrace-#{rand(100000)}-service"}
	let(:app) {
		app = Machete.deploy_app('logenv', env: {'BP_DEBUG' => '1'})
		Machete::SystemHelper.run_cmd(%(cf cups #{service_name} -p '{"apitoken":"secretpaastoken","apiurl":"https://s3.amazonaws.com/dt-paas","environmentid":"envid"}'))
		Machete::SystemHelper.run_cmd(%(cf bind-service #{app.name} #{service_name}))
		Machete::SystemHelper.run_cmd(%(cf restage #{app.name}))
		app
	}
	let(:browser) { Machete::Browser.new(app) }

	after do
		Machete::CF::DeleteApp.new.execute(app)
		Machete::SystemHelper.run_cmd(%(cf delete-service -f #{service_name}))
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

			## Test dynatrcae profile script has loaded
			expect(app).to have_logged('ProfileD: DT_HOST_ID=logenv_0')

			browser.visit_path('/')
			expect(browser).to have_body('Hello from dynatrace app')
		end
	end
end
