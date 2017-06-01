require 'spec_helper'

describe 'deploy has nginx/conf directory' do
  let(:app_name) { 'nginx_conf' }
  let(:app)      { Machete.deploy_app(app_name) }
  let(:browser)  { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'warns user to set root' do
    expect(app).to be_running
    expect(app).to have_logged("You have an nginx/conf directory, but have not set *root*.")

    browser.visit_path('/')
    expect(browser).to have_body('Test warnings')
  end
end
