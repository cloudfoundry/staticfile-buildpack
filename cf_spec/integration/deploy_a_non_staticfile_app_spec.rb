require 'spec_helper'

describe 'deploy a non staticfile app' do
  let(:app) { Machete.deploy_app('non_staticfile', skip_verify_version: true) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to_not be_running
  end
end
