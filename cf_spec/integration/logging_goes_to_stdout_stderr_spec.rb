require 'spec_helper'
require 'rspec/eventually'
require 'excon'

describe 'nginx logs go to stdout and stderr' do
  let(:app) { Machete.deploy_app('staticfile_app') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'writes regular logs to stdout and does not write to actual log files' do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')
    expect(lambda{app}).to eventually(have_logged(/OUT.*GET \/ HTTP\/1.1/)).within 30
    expect(lambda{`cf ssh staticfile_app -c "ls -l /app/nginx/logs/ | grep access.log" | awk '{print $5}'`.strip + 'B'}).to eventually(eq("0B")).within 30
  end

  it 'writes error logs to stderr and does not write to actual log files' do
    expect(app).to be_running

    browser.visit_path('/idontexist', allow_404: true)
    expect(browser).to have_body('404 Not Found')
    expect(lambda{app}).to eventually(have_logged(/ERR.*GET \/idontexist HTTP\/1.1/)).within 30
    expect(lambda{`cf ssh staticfile_app -c "ls -l /app/nginx/logs/ | grep error.log" | awk '{print $5}'`.strip + 'B'}).to eventually(eq("0B")).within 30
  end

end
