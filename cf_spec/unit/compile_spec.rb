require "spec_helper"

describe "running ./bin/compile" do
  attr :output, :exitcode

  def run_bin_compile(args={})
    stack = args[:stack] || "cflinuxfs2"
    build_dir = args[:build_dir] || Dir.mktmpdir
    `env CF_STACK='#{stack}' ./bin/compile #{build_dir} #{Dir.mktmpdir} 2>&1`
  end

  context "on a stack that is" do
    context "unsupported" do
      before(:all) do
        @output = run_bin_compile :stack => "unsupported"
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
        @output = run_bin_compile
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

  context "alternate root directory" do
    context "does not exist" do
      before(:all) do
        @output = run_bin_compile :build_dir => "cf_spec/fixtures/alternate_root_does_not_exist"
        @exitcode = $?
      end

      describe "output" do
        it { expect(output).to include("the application Staticfile specifies a root directory `build` that does not exist") }
      end

      describe "exitcode" do
        it { expect(exitcode).to_not be_success }
      end
    end

    context "is a plain file" do
      before(:all) do
        @output = run_bin_compile :build_dir => "cf_spec/fixtures/alternate_root_is_a_file"
        @exitcode = $?
      end

      describe "output" do
        it { expect(output).to include("the application Staticfile specifies a root directory `build` that is a plain file, but was expected to be a directory") }
      end

      describe "exitcode" do
        it { expect(exitcode).to_not be_success }
      end
    end
  end
end

