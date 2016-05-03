require 'spec_helper'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('pushstate') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running
  end

  context 'requesting the index file' do
    it 'returns the index file' do
      browser.visit_path('/')
      expect(browser).to have_body('This is the index file')
    end
  end

  context 'requesting a static file' do
    it 'returns the static file' do
      browser.visit_path('/static.html')
      expect(browser).to have_body('This is a static file')
    end
  end

  context 'requesting a inexistent file' do
    it 'returns the index file' do
      browser.visit_path('/inexistent')
      expect(browser).to have_body('This is the index file')
    end
  end
end
