require 'spec_helper'

describe 'deploy an app with contents in an alternate root' do
  let(:app) { Machete.deploy_app(app_name) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'default path' do
    let(:app_name) { 'alternate_root' }
    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('This index file comes from an alternate root <code>dist/</code>.')
    end
  end

  context 'not default path' do
    let(:app_name) { 'alternate_root_not_default' }
    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('This index file comes from an alternate root dist/public/index.html')
    end
  end
end
