require 'spec_helper'

describe 'deploy a staticfile app' do
  let(:browser) { Machete::Browser.new(app) }
  let(:timestamp) { Time.now.to_s }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'ssi expose is toggled on' do
    let(:app) { Machete.deploy_app('ssi_expose_enabled_app') }

    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to_not have_body('<!--#echo var="vcap_services" -->')
    end
  end

  context 'ssi expose is toggled off' do
    let(:app) { Machete.deploy_app('ssi_expose_disabled_app') }

    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('<!--#echo var="vcap_services" -->')
    end
  end
end
