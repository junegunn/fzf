# frozen_string_literal: true

Dir[File.join(__dir__, 'test_*.rb')].each { require it }

require 'minitest/autorun'
