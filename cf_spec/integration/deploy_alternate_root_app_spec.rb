require 'spec_helper'

describe 'deploy an app with contents in an alternate root' do
  let(:app) { Machete.deploy_app('alternate_root') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('This index file comes from an alternate root <code>dist/</code>.')
  end
end
