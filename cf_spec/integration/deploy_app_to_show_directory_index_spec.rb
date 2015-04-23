require 'spec_helper'

describe 'deploy an app that shows the directory index' do
  let(:app) { Machete.deploy_app('directory_index') }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to be_running

    browser.visit_path('/')
    expect(browser).to have_body('find-me-too.html')
    expect(browser).to have_body('find-me.html')

    browser.visit_path('/subdir')
    expect(browser).to have_body('This index file should still load normally when viewing a directory; and not a directory index.')
  end
end
