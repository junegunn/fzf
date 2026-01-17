# frozen_string_literal: true

require_relative 'lib/common'

# Test cases for API server
class TestServer < TestInteractive
  def test_listen
    { '--listen 6266' => -> { URI('http://localhost:6266') },
      "--listen --sync --bind 'start:execute-silent:echo $FZF_PORT > /tmp/fzf-port'" =>
        -> { URI("http://localhost:#{File.read('/tmp/fzf-port').chomp}") } }.each do |opts, fn|
      tmux.send_keys "seq 10 | fzf #{opts}", :Enter
      tmux.until { |lines| assert_equal 10, lines.match_count }
      state = JSON.parse(Net::HTTP.get(fn.call), symbolize_names: true)
      assert_equal 10, state[:totalCount]
      assert_equal 10, state[:matchCount]
      assert_empty state[:query]
      assert_equal({ index: 0, text: '1' }, state[:current])

      Net::HTTP.post(fn.call, 'change-query(yo)+reload(seq 100)+change-prompt:hundred> ')
      tmux.until { |lines| assert_equal 100, lines.item_count }
      tmux.until { |lines| assert_equal 'hundred> yo', lines[-1] }
      state = JSON.parse(Net::HTTP.get(fn.call), symbolize_names: true)
      assert_equal 100, state[:totalCount]
      assert_equal 0, state[:matchCount]
      assert_equal 'yo', state[:query]
      assert_nil state[:current]

      teardown
      setup
    end
  end

  def test_listen_with_api_key
    uri = URI('http://localhost:6266')
    tmux.send_keys 'seq 10 | FZF_API_KEY=123abc fzf --listen 6266', :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    # Incorrect API Key
    [nil, { 'x-api-key' => '' }, { 'x-api-key' => '124abc' }].each do |headers|
      res = Net::HTTP.post(uri, 'change-query(yo)+reload(seq 100)+change-prompt:hundred> ', headers)
      assert_equal '401', res.code
      assert_equal 'Unauthorized', res.message
      assert_equal "invalid api key\n", res.body

      res = Net::HTTP.get_response(uri, headers)
      assert_equal '401', res.code
      assert_equal 'Unauthorized', res.message
      assert_equal "invalid api key\n", res.body
    end

    # Valid API Key
    [{ 'x-api-key' => '123abc' }, { 'X-API-Key' => '123abc' }].each do |headers|
      res = Net::HTTP.post(uri, 'change-query(yo)+reload(seq 100)+change-prompt:hundred> ', headers)
      assert_equal '200', res.code
      tmux.until { |lines| assert_equal 100, lines.item_count }
      tmux.until { |lines| assert_equal 'hundred> yo', lines[-1] }

      res = Net::HTTP.get_response(uri, headers)
      assert_equal '200', res.code
      assert_equal 'yo', JSON.parse(res.body, symbolize_names: true)[:query]
    end
  end
end
