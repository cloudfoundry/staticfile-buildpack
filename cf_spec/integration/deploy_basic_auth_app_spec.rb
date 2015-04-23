require 'spec_helper'

describe 'deploy a basic auth app' do
  let(:app) { Machete.deploy_app('basic_auth') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    contents, error, status = Open3.capture3("curl http://bob:bob@basic-auth.10.244.0.34.xip.io/")
    expect(contents).to include('This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>.')
    contents, error, status = Open3.capture3("curl http://basic-auth.10.244.0.34.xip.io/")
    expect(contents).to include('401 Authorization Required')
    contents, error, status = Open3.capture3("curl http://bob:bob1@basic-auth.10.244.0.34.xip.io/")
    expect(contents).to include('401 Authorization Required')
  end
end
