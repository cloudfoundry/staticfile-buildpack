require 'spec_helper'

describe 'a staticfile app with no staticfile' do
  let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'staticfile_buildpack' : 'staticfile-test-buildpack' }
  let(:app) { Machete.deploy_app('without_staticfile', buildpack: buildpack, skip_verify_version: true) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running
    expect(app).to_not have_logged("grep: Staticfile: No such file or directory")
  end
end
