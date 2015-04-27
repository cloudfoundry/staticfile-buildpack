require 'spec_helper'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('staticfile_app') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')

    browser.visit_path('/fixture.json')
    expect(browser.content_type).to eq('application/json')
  end
end
