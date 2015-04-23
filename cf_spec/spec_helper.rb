require 'open3'
require 'machete'
require 'machete/matchers'

`mkdir -p log`
Machete.logger = Machete::Logger.new("log/integration.log")
