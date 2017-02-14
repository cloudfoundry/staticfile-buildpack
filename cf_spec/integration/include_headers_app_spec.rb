require 'spec_helper'

describe 'deploy includes headers' do
  let(:app_name) { 'include_headers' }
  let(:app)      { Machete.deploy_app(app_name) }
  let(:browser)  { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'adds headers' do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('Test add headers')
    expect(browser).to have_header('X-Superspecial')
  end
end
