require 'spec_helper'
require 'excon'
require 'yaml'

describe 'Nginx default_versions' do
  let(:app) { Machete.deploy_app('staticfile_app') }
  let(:manifest_path) { File.expand_path(File.join(File.dirname(__FILE__), "..", "..", "manifest.yml")) }
  let(:loaded_yaml) { YAML.load_file(manifest_path) }
  let(:default_versions) { loaded_yaml["default_versions"].select { |element| element["name"] == 'nginx' }.first }
  let(:default_nginx_version) { default_versions["version"] }

  after do
    Machete::CF::DeleteApp.new.execute(app)
  end

  it 'shows the default version was selected' do
    expect(app).to have_logged("Using Nginx version #{default_nginx_version}")
  end
end
