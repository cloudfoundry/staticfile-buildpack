require 'spec_helper'
require 'fileutils'

describe 'deploy a an app with dot files' do
  let(:app_name)            { 'override'}
  let(:staticfile_contents) { 'override' }
  let(:app)                 { Machete.deploy_app(app_name) }
  let(:browser)             { Machete::Browser.new(app) }

  before do
    @staticfile_file = File.join(File.dirname(__FILE__), '..', 'fixtures', app_name, 'Staticfile')
    File.write(@staticfile_file, staticfile_contents)
  end


  after do
    FileUtils.rm(@staticfile_file)
    Machete::CF::DeleteApp.new.execute(app)
  end

  context 'host_dot_files: true is present in Staticfile' do
    context 'the app uses the default root location' do
      let(:app_name)            { 'with_dotfile'}
      let(:staticfile_contents) { 'host_dot_files: true' }

      it 'hosts the dotfiles' do
        expect(app).to be_running
        expect(app).to have_logged /Enabling hosting of dotfiles/

        browser.visit_path('/.hidden.html')
        expect(browser).to have_body 'Hello from a hidden file'
      end
    end

    context 'the app specifies /public as the root location' do
      let(:app_name) { 'dotfile_public'}
      let(:staticfile_contents) { "host_dot_files: true\nroot: public" }

      it 'hosts the dotfiles' do
        expect(app).to be_running
        expect(app).to have_logged /Enabling hosting of dotfiles/

        browser.visit_path('/.hidden.html')
        expect(browser).to have_body 'Hello from a hidden file'
      end
    end
  end

  context 'host_dot_files: true not present in Staticfile' do
    context 'the app uses the default root location' do
      let(:app_name) { 'with_dotfile'}
      let(:staticfile_contents) { 'host_dot_files: false' }

      it 'does not host the dotfiles' do
        expect(app).to be_running
        expect(app).not_to have_logged /Enabling hosting of dotfiles/

        browser.visit_path('/.hidden.html', allow_404: true)
        expect(browser).to have_body '404 Not Found'
      end
    end

    context 'the app specifies /public as the root location' do
      let(:app_name) { 'dotfile_public'}
      let(:staticfile_contents) { "host_dot_files: false\nroot: public" }

      it 'does not host the dotfiles' do
        expect(app).to be_running
        expect(app).not_to have_logged /Enabling hosting of dotfiles/

        browser.visit_path('/.hidden.html', allow_404: true)
        expect(browser).to have_body '404 Not Found'
      end
    end
  end
end
