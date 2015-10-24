require "spec_helper"

describe "running ./bin/compile" do
  context "on a stack that is" do
    attr :output, :exitcode

    context "unsupported" do
      before(:all) do
        @output = `env CF_STACK='unsupported' ./bin/compile #{Dir.mktmpdir} #{Dir.mktmpdir} 2>&1`
        @exitcode = $?
      end

      describe "output" do
        it { expect(output).to include("not supported by this buildpack") }
      end

      describe "exit status" do
        it { expect(exitcode.exitstatus).to eq 44 }
      end
    end

    context "supported" do
      before(:all) do
        @output = `env CF_STACK='cflinuxfs2' ./bin/compile #{Dir.mktmpdir} #{Dir.mktmpdir} 2>&1`
        @exitcode = $?
      end

      describe "output" do
        it { expect(output).to_not include("not supported by this buildpack") }
      end

      describe "exit status" do
        it { expect(exitcode).to be_success }
      end
    end
  end
end

