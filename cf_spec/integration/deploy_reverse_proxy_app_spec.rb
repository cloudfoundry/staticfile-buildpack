require 'spec_helper'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('reverse_proxy') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    browser.visit_path('/intl/en/policies')
    expect(browser).to have_body('Google Product Privacy Guide')
  end
end
