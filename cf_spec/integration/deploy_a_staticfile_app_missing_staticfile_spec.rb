require 'spec_helper'

describe 'deploy a non staticfile app' do
  let(:app) { Machete.deploy_app('staticfile_app_without_staticfile') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running
    expect(app).to_not have_logged("grep: Staticfile: No such file or directory")
  end
end
