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
