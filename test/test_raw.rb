# frozen_string_literal: true

require_relative 'lib/common'

# Testing raw mode
class TestRaw < TestInteractive
  def test_raw_mode
    tmux.send_keys %(seq 1000 | #{FZF} --raw --bind ctrl-x:toggle-raw --gutter '▌' --multi), :Enter
    tmux.until { assert_equal 1000, it.match_count }

    tmux.send_keys 1
    tmux.until { assert_equal 272, it.match_count }

    tmux.send_keys :Up
    tmux.until { assert_includes it, '> 2' }

    tmux.send_keys 'C-p'
    tmux.until do
      assert_includes it, '> 10'
      assert_includes it, '▖ 9'
    end

    tmux.send_keys 'C-x'
    tmux.until do
      assert_includes it, '> 10'
      assert_includes it, '▌ 1'
    end

    tmux.send_keys :Up, 'C-x'
    tmux.until do
      assert_includes it, '> 11'
      assert_includes it, '▖ 10'
    end

    tmux.send_keys 1
    tmux.until { assert_equal 28, it.match_count }

    tmux.send_keys 'C-p'
    tmux.until do
      assert_includes it, '> 101'
      assert_includes it, '▖ 100'
    end

    tmux.send_keys 'C-n'
    tmux.until do
      assert_includes it, '> 11'
      assert_includes it, '▖ 10'
    end

    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { assert_equal 3, it.select_count }

    tmux.send_keys 'C-x'
    tmux.until do
      assert_equal 1, it.select_count
      assert_includes it, '▌ 110'
      assert_includes it, '>>11'
    end
  end
end
