require 'spec_helper'

describe 'When running ./bin/compile' do
  context 'and on an unsupported stack' do
    before(:all) do
      @output = `env CF_STACK='unsupported' ./bin/compile #{Dir.mktmpdir} #{Dir.mktmpdir} 2>&1`
    end

    it 'displays a helpful error message' do
      expect(@output).to include('not supported by this buildpack')
    end

    it 'exits with our error code' do
      expect($?.exitstatus).to eq 44
    end
  end

  context 'and on a supported stack' do
    before(:all) do
      @output = `env CF_STACK='cflinuxfs2' ./bin/compile #{Dir.mktmpdir} #{Dir.mktmpdir} 2>&1`
    end

    it 'displays a helpful error message' do
      expect(@output).to_not include('not supported by this buildpack')
    end

    it 'does not exit with our error code' do
      expect($?.exitstatus).to_not eq 44
    end
  end
end

