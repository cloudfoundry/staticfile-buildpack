require 'spec_helper'
require 'excon'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('staticfile_app') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to have_logged(/Buildpack version [\d\.]{5,}/)
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')

    browser.visit_path('/fixture.json')
    expect(browser.content_type).to eq('application/json')

    response = Excon.get("http://#{browser.base_url}/lots_of.js", headers: {
      'Accept-Encoding' => 'gzip'
    })
    expect(response.headers['Content-Encoding']).to eq('gzip')
  end

  context 'requesting a non-compressed version of a compressed file' do
    context 'with a client that can handle receiving compressed content' do
      let(:compressed_flag) { '--compressed' }

      it 'returns and handles the file' do
        expect(app).to be_running

        content = `curl #{compressed_flag} http://#{browser.base_url}/war_and_peace.txt`
        expect(content).to include("Leo Tolstoy")
      end
    end

    context 'with a client that cannot handle receiving compressed content' do
      let(:compressed_flag) { '' }

      it 'returns and handles the file' do
        expect(app).to be_running

        content = `curl #{compressed_flag} http://#{browser.base_url}/war_and_peace.txt`
        expect(content).to include("Leo Tolstoy")
      end
    end
  end

  context 'with a cached buildpack', :cached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Downloaded \[file:\/\/.*\]/)
    end

    it 'does not call out over the internet' do
      expect(app).to_not have_internet_traffic
    end
  end

  context 'with a uncached buildpack', :uncached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Downloaded \[https:\/\/.*\]/)
    end

    it "uses a proxy during staging if present" do
      expect(app).to use_proxy_during_staging
    end
  end
end
