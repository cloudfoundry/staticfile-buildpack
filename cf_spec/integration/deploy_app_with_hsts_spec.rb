require 'spec_helper'

describe 'deploy an app using hsts' do
  let(:app_name) { 'with_hsts'}
  let(:app)      { Machete.deploy_app(app_name) }
  let(:browser)  { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'provides the Strict-Transport-Security header' do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_header('Strict-Transport-Security')
  end
end
