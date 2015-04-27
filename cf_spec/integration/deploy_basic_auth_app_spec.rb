require 'spec_helper'

describe 'deploy a basic auth app' do
  let(:app) { Machete.deploy_app('basic_auth') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    browser.visit_path('/', username: 'bob', password: 'bob')
    expect(browser).to have_body('This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>.')

    browser.visit_path('/')
    expect(browser).to have_body('401 Authorization Required')

    browser.visit_path('/', username: 'bob', password: 'bob1')
    expect(browser).to have_body('401 Authorization Required')
  end
end
