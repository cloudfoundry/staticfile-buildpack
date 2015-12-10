require 'spec_helper'
require 'excon'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('staticfile_https_app') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to have_logged(/Buildpack version [\d\.]{5,}/)
    expect(app).to be_running

  end

  context 'visiting http' do
    it 'receives a 301 redirect to https' do
      response = Excon.get("http://#{browser.base_url}/")
      expect(response.status).to eq(301)
      expect(response.headers['Location']).to eq("https://#{browser.base_url}/")
    end
  end

  context 'with a cached buildpack', :cached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Downloaded \[file:\/\/.*\]/)
    end
  end

  context 'with a uncached buildpack', :uncached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Downloaded \[https:\/\/.*\]/)
    end
  end
end
