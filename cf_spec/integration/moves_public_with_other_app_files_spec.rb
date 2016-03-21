require 'spec_helper'
require 'rspec/eventually'
require 'excon'

describe 'pushing a static app with dummy file in root' do
  let(:app) { Machete.deploy_app('recursive_public') }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'should have a copy of the original public dir in the new public dir' do
    expect(app).to be_running

    has_diego = `cf has-diego-enabled recursive_public`.strip
    public_files = ''
    if has_diego == "true"
      public_files = `cf ssh recursive_public -c "ls /app/public/public"`
    else
      public_files = `cf files recursive_public /app/public/public`
    end

    expect(public_files).to match(/file_in_public/)
  end

end
