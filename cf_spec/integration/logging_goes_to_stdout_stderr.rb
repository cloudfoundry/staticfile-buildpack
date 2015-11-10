require 'spec_helper'
require 'rspec/eventually'
require 'excon'

describe 'nginx logs go to stdout and stderr' do
  let(:app) { Machete.deploy_app('staticfile_app') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'writes regular logs to stdout' do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')
    expect(lambda{app}).to eventually(have_logged(/OUT.*GET \/ HTTP\/1.1/)).within 30
    accesslog_size = `cf files staticfile_app app/nginx/logs/ | grep access.log | tr -d "access.log" | tr -d " "`.strip
    expect(accesslog_size).to eq("0B")
  end

  it 'writes error logs to stderr' do
    expect(app).to be_running

    browser.visit_path('/idontexist')
    expect(browser).to have_body('404 Not Found')
    expect(lambda{app}).to eventually(have_logged(/ERR.*GET \/idontexist HTTP\/1.1/)).within 30
    errorlog_size = `cf files staticfile_app app/nginx/logs/ | grep error.log | tr -d "error.log" | tr -d " "`.strip
    expect(errorlog_size).to eq("0B")
  end

end
