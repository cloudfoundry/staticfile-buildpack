require 'spec_helper'
require 'excon'

describe 'deploy a an app with a large page' do
  let(:app_name) { 'large_page'}
  let(:app)      { Machete.deploy_app(app_name) }
  let(:browser)  { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'responds with the Vary: Accept-Encoding header' do
    expect(app).to be_running

    response = Excon.get("http://#{browser.base_url}/")
    expect(response.status).to eq(200)
    expect(response.headers['Vary']).to eq('Accept-Encoding')
  end
end
