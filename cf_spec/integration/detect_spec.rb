require 'spec_helper'

describe 'detect script' do
  context 'there is a Staticfile' do
    specify do
      stdout, stderr, status = Open3.capture3('bin/detect cf_spec/fixtures/staticfile_app')
      expect(status.exitstatus).to eq(0)
      expect(stdout.chomp).to include('staticfile')
    end
  end

  context 'there is no Staticfile' do
    specify do
      stdout, stderr, status = Open3.capture3('bin/detect cf_spec/fixtures/non_staticfile')
      expect(status.exitstatus).to eq(1)
      expect(stdout.chomp).to eq('no')
    end
  end
end

