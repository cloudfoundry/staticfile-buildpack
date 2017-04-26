require 'spec_helper'
require 'excon'
require 'open3'
require 'timeout'

describe 'deploy a staticfile app' do
  let(:app) { Machete.deploy_app('staticfile_app', env: {'BP_DEBUG' => '1'}) }
  let(:browser) { Machete::Browser.new(app) }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  specify do
    expect(app).to have_logged(/Buildpack version [\d\.]{5,}/)
    expect(app).to have_logged(/HOOKS 1: BeforeCompile/)
    expect(app).to have_logged(/HOOKS 2: AfterCompile/)
    expect(app).to be_running
    expect(app).to have_logged(%r{nginx -p .*/nginx -c .*/nginx/conf/nginx.conf})

    browser.visit_path('/')
    expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')

    browser.visit_path('/fixture.json')
    expect(browser.content_type).to eq('application/json')

    response = Excon.get("http://#{browser.base_url}/lots_of.js", headers: {
      'Accept-Encoding' => 'gzip'
    })
    expect(response.headers['Content-Encoding']).to eq('gzip')
  end

  context 'requesting a non-compressed version of a compressed file' do
    context 'with a client that can handle receiving compressed content' do
      let(:compressed_flag) { '--compressed' }

      it 'returns and handles the file' do
        expect(app).to be_running

        content = `curl -s #{compressed_flag} http://#{browser.base_url}/war_and_peace.txt | grep --text 'Leo Tolstoy'`
        expect(content).to include("Leo Tolstoy")
      end
    end

    context 'with a client that cannot handle receiving compressed content' do
      let(:compressed_flag) { '' }

      it 'returns and handles the file' do
        expect(app).to be_running

        content = `curl -s #{compressed_flag} http://#{browser.base_url}/war_and_peace.txt | grep --text 'Leo Tolstoy'`
        expect(content).to include("Leo Tolstoy")
      end
    end
  end

  context 'with a cached buildpack', :cached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Copy \[\/.*\]/)
    end

    it 'does not call out over the internet' do
      expect(app).to_not have_internet_traffic
    end
  end

  context 'with a uncached buildpack', :uncached do
    it 'logs the files it downloads' do
      expect(app).to have_logged(/Download \[https:\/\/.*\]/)
    end

    it "uses a proxy during staging if present" do
      expect(app).to use_proxy_during_staging
    end
  end

  context 'unpackaged buildpack eg. from github' do
    let(:buildpack) { "staticfile-unpackaged-buildpack-#{rand(1000)}" }
    let(:app) { Machete.deploy_app('staticfile_app', buildpack: buildpack, skip_verify_version: true) }
    before do
      buildpack_file = "/tmp/#{buildpack}.zip"
      Open3.capture2e('zip','-r',buildpack_file,'bin/','src/', 'scripts/', 'manifest.yml','VERSION')[1].success? or raise 'Could not create unpackaged buildpack zip file'
      Open3.capture2e('cf', 'create-buildpack', buildpack, buildpack_file, '100', '--enable')[1].success? or raise 'Could not upload buildpack'
      FileUtils.rm buildpack_file
    end
    after do
      Open3.capture2e('cf', 'delete-buildpack', '-f', buildpack)
    end

    it 'runs' do
      expect(app).to be_running
      expect(app).to have_logged(/Running go build supply/)
      expect(app).to have_logged(/Running go build finalize/)

      browser.visit_path('/')
      expect(browser).to have_body('This is an example app for Cloud Foundry that is only static HTML/JS/CSS assets.')
    end
  end

  context 'running a task' do
    before { skip_if_no_run_task_support_on_targeted_cf }

    it 'exits' do
      expect(app).to be_running

      Open3.capture2e('cf','run-task','staticfile_app','wc -l public/index.html')[1].success? or raise 'Could not create run task'
      wait_until(60) do
        stdout, _ = Open3.capture2e('cf','tasks','staticfile_app')
        stdout =~ /SUCCEEDED.*wc.*index.html/
      end
      stdout, _ = Open3.capture2e('cf','tasks','staticfile_app')
      expect(stdout).to match(/SUCCEEDED.*wc.*index.html/)
    end
  end
end
