# frozen_string_literal: true

Dir[File.join(__dir__, 'test_*.rb')].each { |f| require f }

require 'minitest/autorun'
