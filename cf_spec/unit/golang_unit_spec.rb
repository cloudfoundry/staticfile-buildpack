require 'open3'
require 'spec_helper'

describe 'golang unit tests' do
  before do
    @old_gopath = ENV['GOPATH']
    ENV['GOPATH'] = Dir.pwd
  end

  after do
    ENV['GOPATH'] = @old_gopath
  end

  it 'passes all the unit tests' do
    unit_test_dirs = ['src/staticfile/supply', 'src/staticfile/finalize', 'src/staticfile/hooks']

    unit_test_dirs.each do |dir|
      Dir.chdir(dir) do
        _, stdout, stderr, wait_thr = Open3.popen3('go test')
        exit_status = wait_thr.value
        puts "stdout:"
        puts stdout.read

        unless exit_status.success?
          puts "stderr:"
          puts stderr.read
        end

        expect(wait_thr.value).to eq(0)
      end
    end
  end
end
