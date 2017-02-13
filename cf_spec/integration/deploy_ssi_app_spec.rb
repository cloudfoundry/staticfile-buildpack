require 'spec_helper'

describe 'deploy a staticfile app' do
  let(:browser) { Machete::Browser.new(app) }
  let(:timestamp) { Time.now.to_s }
  let(:body) { "I feel included!" }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'ssi is toggled on' do
    let(:app) { Machete.deploy_app('ssi_enabled') }

    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body(body)
      expect(browser).to_not have_body('<!--# include file="ssi_body.html" -->')
    end
  end

  context 'ssi is toggled off' do
    let(:app) { Machete.deploy_app('ssi_disabled') }

    specify do
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to_not have_body(body)
      expect(browser).to have_body('<!--# include file="ssi_body.html" -->')
    end
  end
end
