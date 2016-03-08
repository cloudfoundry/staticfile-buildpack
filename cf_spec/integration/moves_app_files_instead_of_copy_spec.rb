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

    has_diego = `cf has-diego-enabled public_unspecified`.strip
    public_files = ''
    if has_diego == "true"
      public_files = `cf ssh public_unspecified -c "ls /app/public"`
    else
      public_files = `cf files public_unspecified /app/public`
    end

    expect(public_files).to match(/dummy_file/)

    root_files = ''
    has_diego = `cf has-diego-enabled public_unspecified`.strip
    if has_diego == "true"
      root_files = `cf ssh public_unspecified -c "ls /app"`
    else
      root_files = `cf files public_unspecified /app`
    end

    expect(root_files).to_not match(/dummy_file/)
  end

end
