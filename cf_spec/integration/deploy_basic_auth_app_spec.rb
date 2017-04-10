require 'spec_helper'

describe 'deploy a basic auth app' do
  let(:app_name) { 'override'}
  let(:app)      { Machete.deploy_app(app_name) }
  let(:browser)  { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'the app uses Staticfile.auth' do
    let(:app_name) { 'basic_auth' }

    it 'uses the provided credentials for authorization' do
      expect(app).to be_running

      browser.visit_path('/', username: 'bob', password: 'bob')
      expect(browser).to have_body('This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>.')

      browser.visit_path('/', allow_404: true)
      expect(browser).to have_body('401 Authorization Required')

      browser.visit_path('/', username: 'bob', password: 'bob1', allow_404: true)
      expect(browser).to have_body('401 Authorization Required')
    end

    it 'does not write the contents of .htpasswd to the logs' do
      expect(app).not_to have_logged('bob:$')
      expect(app).not_to have_logged('dave:$')
    end

    it 'logs the source of authentication credentials' do
      expect(app).to have_logged('-----> Enabling basic authentication using Staticfile.auth')
    end

    context 'and is missing Staticfile' do
      let(:buildpack) { ENV.fetch('SHARED_HOST')=='true' ? 'staticfile_buildpack' : 'staticfile-test-buildpack' }
      let(:app) { Machete.deploy_app(app_name, buildpack: buildpack, skip_verify_version: true) }
      before { FileUtils.rm("#{Machete::App::FIXTURES_DIR}/#{app_name}/Staticfile") }
      after { FileUtils.touch("#{Machete::App::FIXTURES_DIR}/#{app_name}/Staticfile") }

      it 'uses the provided credentials for authorization' do
        expect(app).to be_running

        browser.visit_path('/', username: 'bob', password: 'bob')
        expect(browser).to have_body('This site is protected by basic auth. User: <code>bob</code>; Password: <code>bob</code>.')

        browser.visit_path('/', allow_404: true)
        expect(browser).to have_body('401 Authorization Required')

        browser.visit_path('/', username: 'bob', password: 'bob1', allow_404: true)
        expect(browser).to have_body('401 Authorization Required')
      end
    end
  end
end
