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
    has_diego = `cf has-diego-enabled staticfile_app`.strip
    if has_diego == "true"
      accesslog_size = `cf ssh staticfile_app -c "ls -l /app/nginx/logs/ | grep access.log" | awk '{print $5}'`.strip + 'B'
    else
      accesslog_size = `cf files staticfile_app /app/nginx/logs/ | grep access.log | tr -d "access.log" | tr -d " "`.strip
    end
    expect(accesslog_size).to eq("0B")
  end

  it 'writes error logs to stderr' do
    expect(app).to be_running

    browser.visit_path('/idontexist')
    expect(browser).to have_body('404 Not Found')
    expect(lambda{app}).to eventually(have_logged(/ERR.*GET \/idontexist HTTP\/1.1/)).within 30
    has_diego = `cf has-diego-enabled staticfile_app`.strip
    if has_diego == "true"
      errorlog_size = `cf ssh staticfile_app -c "ls -l /app/nginx/logs/ | grep error.log" | awk '{print $5}'`.strip + 'B'
    else
      errorlog_size = `cf files staticfile_app /app/nginx/logs/ | grep error.log | tr -d "error.log" | tr -d " "`.strip
    end
    expect(errorlog_size).to eq("0B")
  end

end
