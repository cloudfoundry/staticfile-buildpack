require 'spec_helper'
require 'excon'

describe 'deploy a staticfile app' do
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  before do
    expect(app).to be_running
  end

  context 'Using ENV Variable' do
    let(:app) { Machete.deploy_app('with_https') }

    it 'receives a 301 redirect to https' do
      response = Excon.get("http://#{browser.base_url}/")
      expect(response.status).to eq(301)
      expect(response.headers['Location']).to eq("https://#{browser.base_url}/")
    end
  end

  context 'Using Staticfile' do
    let(:app) { Machete.deploy_app('with_https_in_staticfile', name: 'https_file') }

    it 'receives a 301 redirect to https' do
      response = Excon.get("http://#{browser.base_url}/")
      expect(response.status).to eq(301)
      expect(response.headers['Location']).to eq("https://#{browser.base_url}/")
    end
  end
end
