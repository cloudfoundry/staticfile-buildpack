require 'open3'
require 'machete'
require 'machete/matchers'

`mkdir -p log`
Machete.logger = Machete::Logger.new("log/integration.log")

RSpec.configure do |config|
  config.color = true
  config.tty = true

  config.filter_run_excluding :cached => ENV['BUILDPACK_MODE'] == 'uncached'
  config.filter_run_excluding :uncached => ENV['BUILDPACK_MODE'] == 'cached'
end

def wait_until(timeout = 10)
  require "timeout"
  Timeout.timeout(timeout) do
    sleep(0.1) until value = yield
    value
  end
end

def skip_if_no_run_task_support_on_targeted_cf
  minimum_acceptable_cf_api_version = '2.75.0'
  skip_reason = "run task functionality not supported before CF API version #{minimum_acceptable_cf_api_version}"
  Machete::RSpecHelpers.skip_if_cf_api_below(version: minimum_acceptable_cf_api_version, reason: skip_reason)
end
