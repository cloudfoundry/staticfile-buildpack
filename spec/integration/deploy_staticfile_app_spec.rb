require 'spec_helper'

describe 'deploy a staticfile app' do
	after do
		`cf delete #{app_name} -f`
	end

	context 'standard staticfile app' do
		let(:app_name) { 'static-app' }

		specify do
			Open3.capture3("cf push #{app_name} -p ./test/fixtures/staticfile_app")
			contents, error, status = Open3.capture3("curl http://#{app_name}.10.244.0.34.xip.io/")
			expect(contents).to include('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')
		end
	end

	context 'app contents are in alternate root' do
		let(:app_name) { 'alternate-app' }

		specify do
			Open3.capture3("cf push #{app_name} -p ./test/fixtures/alternate_root")
			contents, error, status = Open3.capture3("curl http://#{app_name}.10.244.0.34.xip.io/")
			expect(contents).to include('This index file comes from an alternate root <code>dist/</code>.')
		end
	end

	context 'basic auth app' do
		let(:app_name) { 'basicauth-app' }

		specify do
			Open3.capture3("cf push #{app_name} -p ./test/fixtures/basic_auth")
			contents, error, status = Open3.capture3("curl http://bob:bob@#{app_name}.10.244.0.34.xip.io/")
			expect(contents).to include('This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>.')
			contents, error, status = Open3.capture3("curl http://#{app_name}.10.244.0.34.xip.io/")
			expect(contents).to include('401 Authorization Required')
			contents, error, status = Open3.capture3("curl http://bob:bob1@#{app_name}.10.244.0.34.xip.io/")
			expect(contents).to include('401 Authorization Required')
		end
	end
end