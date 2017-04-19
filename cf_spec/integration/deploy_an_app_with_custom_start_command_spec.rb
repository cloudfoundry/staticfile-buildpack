require 'spec_helper'

describe 'a staticfile app with a custom start command that uses boot.sh' do
  let(:app) { Machete.deploy_app('custom_start_command') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'starts using the custom command' do
    expect(app).to be_running
    expect(app).to have_logged("A custom start command")
    browser.visit_path('/')
    expect(browser).to have_body("This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.")
  end
end
