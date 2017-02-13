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
    Dir.chdir('src/compile') do
      _, stdout, stderr, wait_thr = Open3.popen3('go test')
      exit_status = wait_thr.value
      puts "Go Compile Suite stdout:"
      puts stdout.read

      unless exit_status.success?
        puts "Go Compile Suite stderr:"
        puts stderr.read
      end

      expect(wait_thr.value).to eq(0)
    end
  end
end
