require 'spec_helper'
require 'rspec/eventually'
require 'excon'

describe 'pushing a static app with dummy file in root' do
  let(:app) { Machete.deploy_app('public_unspecified') }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'should only have dummy file in public' do
    expect(app).to be_running

    files = `cf files public_unspecified app/public`
    expect(files).to match(/dummy_file/)

    files = `cf files public_unspecified app`
    expect(files).to_not match(/dummy_file/)
  end

end
