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
end
