# frozen_string_literal: true

require_relative 'lib/common'

# Process execution: execute, become, reload
class TestExec < TestInteractive
  def test_execute
    output = '/tmp/fzf-test-execute'
    opts = %[--bind "alt-a:execute(echo /{}/ >> #{output})+change-header(alt-a),alt-b:execute[echo /{}{}/ >> #{output}]+change-header(alt-b),C:execute(echo /{}{}{}/ >> #{output})+change-header(C)"]
    writelines(%w[foo'bar foo"bar foo$bar])
    tmux.send_keys "cat #{tempname} | #{FZF} #{opts}", :Enter
    tmux.until { |lines| assert_equal 3, lines.match_count }

    ready = ->(s) { tmux.until { |lines| assert_includes lines[-3], s } }
    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    tmux.send_keys :Escape, :b
    ready.call('alt-b')

    tmux.send_keys :Up
    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    tmux.send_keys :Escape, :b
    ready.call('alt-b')

    tmux.send_keys :Up
    tmux.send_keys :C
    ready.call('C')

    tmux.send_keys 'barfoo'
    tmux.until { |lines| assert_equal '  0/3', lines[-2] }

    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    tmux.send_keys :Escape, :b
    ready.call('alt-b')

    wait do
      assert_path_exists output
      assert_equal %w[
        /foo'bar/ /foo'barfoo'bar/
        /foo"bar/ /foo"barfoo"bar/
        /foo$barfoo$barfoo$bar/
      ], File.readlines(output, chomp: true)
    end
  ensure
    FileUtils.rm_f(output)
  end

  def test_execute_multi
    output = '/tmp/fzf-test-execute-multi'
    opts = %[--multi --bind "alt-a:execute-multi(echo {}/{+} >> #{output})+change-header(alt-a),alt-b:change-header(alt-b)"]
    writelines(%w[foo'bar foo"bar foo$bar foobar])
    tmux.send_keys "cat #{tempname} | #{FZF} #{opts}", :Enter
    ready = ->(s) { tmux.until { |lines| assert_includes lines[-3], s } }

    tmux.until { |lines| assert_equal '  4/4 (0)', lines[-2] }
    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    tmux.send_keys :Escape, :b
    ready.call('alt-b')

    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  4/4 (3)', lines[-2] }
    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    tmux.send_keys :Escape, :b
    ready.call('alt-b')

    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| assert_equal '  4/4 (3)', lines[-2] }
    tmux.send_keys :Escape, :a
    ready.call('alt-a')
    wait do
      assert_path_exists output
      assert_equal [
        %(foo'bar/foo'bar),
        %(foo'bar foo"bar foo$bar/foo'bar foo"bar foo$bar),
        %(foo'bar foo"bar foobar/foo'bar foo"bar foobar)
      ], File.readlines(output, chomp: true)
    end
  ensure
    FileUtils.rm_f(output)
  end

  def test_execute_plus_flag
    output = tempname + '.tmp'
    FileUtils.rm_f(output)
    writelines(['foo bar', '123 456'])

    tmux.send_keys "cat #{tempname} | #{FZF} --multi --bind 'x:execute-silent(echo {+}/{}/{+2}/{2} >> #{output})'", :Enter

    tmux.until { |lines| assert_equal '  2/2 (0)', lines[-2] }
    tmux.send_keys 'xy'
    tmux.until { |lines| assert_equal '  0/2 (0)', lines[-2] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '  2/2 (0)', lines[-2] }

    tmux.send_keys :Up
    tmux.send_keys :Tab
    tmux.send_keys 'xy'
    tmux.until { |lines| assert_equal '  0/2 (1)', lines[-2] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '  2/2 (1)', lines[-2] }

    tmux.send_keys :Tab
    tmux.send_keys 'xy'
    tmux.until { |lines| assert_equal '  0/2 (2)', lines[-2] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '  2/2 (2)', lines[-2] }

    wait do
      assert_path_exists output
      assert_equal [
        %(foo bar/foo bar/bar/bar),
        %(123 456/foo bar/456/bar),
        %(123 456 foo bar/foo bar/456 bar/bar)
      ], File.readlines(output, chomp: true)
    end
  rescue StandardError
    FileUtils.rm_f(output)
  end

  def test_execute_shell
    # Custom script to use as $SHELL
    output = tempname + '.out'
    FileUtils.rm_f(output)
    writelines(['#!/usr/bin/env bash', "echo $1 / $2 > #{output}"])
    system("chmod +x #{tempname}")

    tmux.send_keys "echo foo | SHELL=#{tempname} fzf --bind 'enter:execute:{}bar'", :Enter
    tmux.until { |lines| assert_equal '  1/1', lines[-2] }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal '  1/1', lines[-2] }
    wait do
      assert_path_exists output
      assert_equal ["-c / 'foo'bar"], File.readlines(output, chomp: true)
    end
  ensure
    FileUtils.rm_f(output)
  end

  def test_interrupt_execute
    tmux.send_keys "seq 100 | #{FZF} --bind 'ctrl-l:execute:echo executing {}; sleep 100'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'C-l'
    tmux.until { |lines| assert lines.any_include?('executing 1') }
    tmux.send_keys 'C-c'
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 99
    tmux.until { |lines| assert_equal 1, lines.match_count }
  end

  def test_kill_default_command_on_abort
    writelines(['#!/usr/bin/env bash',
                "echo 'Started'",
                'while :; do sleep 1; done'])
    system("chmod +x #{tempname}")

    tmux.send_keys FZF.sub('FZF_DEFAULT_COMMAND=', "FZF_DEFAULT_COMMAND=#{tempname}"), :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'C-c'
    tmux.send_keys 'C-l', 'closed'
    tmux.until { |lines| assert_includes lines[0], 'closed' }
    wait { refute system("pgrep -f #{tempname}") }
  ensure
    system("pkill -9 -f #{tempname}")
  end

  def test_kill_default_command_on_accept
    writelines(['#!/usr/bin/env bash',
                "echo 'Started'",
                'while :; do sleep 1; done'])
    system("chmod +x #{tempname}")

    tmux.send_keys fzf.sub('FZF_DEFAULT_COMMAND=', "FZF_DEFAULT_COMMAND=#{tempname}"), :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    assert_equal 'Started', fzf_output
    wait { refute system("pgrep -f #{tempname}") }
  ensure
    system("pkill -9 -f #{tempname}")
  end

  def test_kill_reload_command_on_abort
    writelines(['#!/usr/bin/env bash',
                "echo 'Started'",
                'while :; do sleep 1; done'])
    system("chmod +x #{tempname}")

    tmux.send_keys "seq 1 3 | #{FZF} --bind 'ctrl-r:reload(#{tempname})'", :Enter
    tmux.until { |lines| assert_equal 3, lines.match_count }
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'C-c'
    tmux.send_keys 'C-l', 'closed'
    tmux.until { |lines| assert_includes lines[0], 'closed' }
    wait { refute system("pgrep -f #{tempname}") }
  ensure
    system("pkill -9 -f #{tempname}")
  end

  def test_kill_reload_command_on_accept
    writelines(['#!/usr/bin/env bash',
                "echo 'Started'",
                'while :; do sleep 1; done'])
    system("chmod +x #{tempname}")

    tmux.send_keys "seq 1 3 | #{fzf("--bind 'ctrl-r:reload(#{tempname})'")}", :Enter
    tmux.until { |lines| assert_equal 3, lines.match_count }
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    assert_equal 'Started', fzf_output
    wait { refute system("pgrep -f #{tempname}") }
  ensure
    system("pkill -9 -f #{tempname}")
  end

  def test_reload
    tmux.send_keys %(seq 1000 | #{FZF} --bind 'change:reload(seq $FZF_QUERY),a:reload(seq 100),b:reload:seq 200' --header-lines 2 --multi 2), :Enter
    tmux.until { |lines| assert_equal 998, lines.match_count }
    tmux.send_keys 'a'
    tmux.until do |lines|
      assert_equal 98, lines.item_count
      assert_equal 98, lines.match_count
    end
    tmux.send_keys 'b'
    tmux.until do |lines|
      assert_equal 198, lines.item_count
      assert_equal 198, lines.match_count
    end
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal '  198/198 (1/2)', lines[-2] }
    tmux.send_keys '555'
    tmux.until { |lines| assert_equal '  1/553 (0/2)', lines[-2] }
  end

  def test_reload_even_when_theres_no_match
    tmux.send_keys %(: | #{FZF} --bind 'space:reload(seq 10)'), :Enter
    tmux.until { |lines| assert_equal 0, lines.item_count }
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 10, lines.item_count }
  end

  def test_reload_should_terminate_standard_input_stream
    tmux.send_keys %(ruby -e "STDOUT.sync = true; loop { puts 1; sleep 0.1 }" | fzf --bind 'start:reload(seq 100)'), :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
  end

  def test_clear_list_when_header_lines_changed_due_to_reload
    tmux.send_keys %(seq 10 | #{FZF} --header 0 --header-lines 3 --bind 'space:reload(seq 1)'), :Enter
    tmux.until { |lines| assert_includes lines, '  9' }
    tmux.send_keys :Space
    tmux.until { |lines| refute_includes lines, '  9' }
  end

  def test_item_index_reset_on_reload
    tmux.send_keys "seq 10 | #{FZF} --preview 'echo [[{n}]]' --bind 'up:last,down:first,space:reload:seq 100'", :Enter
    tmux.until { |lines| assert_includes lines[1], '[[0]]' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[1], '[[9]]' }
    tmux.send_keys :Down
    tmux.until { |lines| assert_includes lines[1], '[[0]]' }
    tmux.send_keys :Space
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      assert_includes lines[1], '[[0]]'
    end
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[1], '[[99]]' }
  end

  def test_reload_should_update_preview
    tmux.send_keys "seq 3 | #{FZF} --bind 'ctrl-t:reload:echo 4' --preview 'echo {}' --preview-window 'nohidden'", :Enter
    tmux.until { |lines| assert_includes lines[1], '1' }
    tmux.send_keys 'C-t'
    tmux.until { |lines| assert_includes lines[1], '4' }
  end

  def test_reload_and_change_preview_should_update_preview
    tmux.send_keys "seq 3 | #{FZF} --bind 'ctrl-t:reload(echo 4)+change-preview(echo {})'", :Enter
    tmux.until { |lines| assert_equal 3, lines.match_count }
    tmux.until { |lines| refute_includes lines[1], '1' }
    tmux.send_keys 'C-t'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], '4' }
  end

  def test_reload_sync
    tmux.send_keys "seq 100 | #{FZF} --bind 'load:reload-sync(sleep 1; seq 1000)+unbind(load)'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys '00'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    # After 1 second
    tmux.until { |lines| assert_equal 10, lines.match_count }
  end

  def test_reload_disabled_case1
    tmux.send_keys "seq 100 | #{FZF} --query 99 --bind 'space:disable-search+reload(sleep 2; seq 1000)'", :Enter
    tmux.until do |lines|
      assert_equal 100, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.until { |lines| assert_equal 1000, lines.match_count }
  end

  def test_reload_disabled_case2
    tmux.send_keys "seq 100 | #{FZF} --query 99 --bind 'space:disable-search+reload-sync(sleep 2; seq 1000)'", :Enter
    tmux.until do |lines|
      assert_equal 100, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert_equal 1000, lines.match_count }
  end

  def test_reload_disabled_case3
    tmux.send_keys "seq 100 | #{FZF} --query 99 --bind 'space:disable-search+reload(sleep 2; seq 1000)+backward-delete-char'", :Enter
    tmux.until do |lines|
      assert_equal 100, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.until { |lines| assert_equal 1000, lines.match_count }
  end

  def test_reload_disabled_case4
    tmux.send_keys "seq 100 | #{FZF} --query 99 --bind 'space:disable-search+reload-sync(sleep 2; seq 1000)+backward-delete-char'", :Enter
    tmux.until do |lines|
      assert_equal 100, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert_equal 1000, lines.match_count }
  end

  def test_reload_disabled_case5
    tmux.send_keys "seq 100 | #{FZF} --query 99 --bind 'space:disable-search+reload(echo xx; sleep 2; seq 1000)'", :Enter
    tmux.until do |lines|
      assert_equal 100, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert_equal 1, lines.item_count
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 1001, lines.match_count }
  end

  def test_reload_disabled_case6
    tmux.send_keys "seq 1000 | #{FZF} --disabled --bind 'change:reload:sleep 0.5; seq {q}'", :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys '9'
    tmux.until { |lines| assert_equal 9, lines.match_count }
    tmux.send_keys '9'
    tmux.until { |lines| assert_equal 99, lines.match_count }

    # TODO: How do we verify if an intermediate empty list is not shown?
  end

  def test_reload_and_change
    tmux.send_keys "(echo foo; echo bar) | #{FZF} --bind 'load:reload-sync(sleep 60)+change-query(bar)'", :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
  end

  def test_become_tty
    tmux.send_keys "sleep 0.5 | #{FZF} --bind 'start:reload:ls' --bind 'load:become:tty'", :Enter
    tmux.until { |lines| assert_includes lines, '/dev/tty' }
  end

  def test_disabled_preview_update
    tmux.send_keys "echo bar | #{FZF} --disabled --bind 'change:reload:echo foo' --preview 'echo [{q}-{}]'", :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert(lines.any? { |line| line.include?('[-bar]') }) }
    tmux.send_keys :x
    tmux.until { |lines| assert(lines.any? { |line| line.include?('[x-foo]') }) }
  end

  def test_start_on_reload
    tmux.send_keys %(echo foo | #{FZF} --header Loading --header-lines 1 --bind 'start:reload:sleep 2; echo bar' --bind 'load:change-header:Loaded' --bind space:change-header:), :Enter
    tmux.until(timeout: 1) { |lines| assert_includes lines[-3], 'Loading' }
    tmux.until(timeout: 1) { |lines| refute_includes lines[-4], 'foo' }
    tmux.until { |lines| assert_includes lines[-3], 'Loaded' }
    tmux.until { |lines| assert_includes lines[-4], 'bar' }
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines[-3], 'bar' }
  end

  def test_become
    tmux.send_keys "seq 100 | fzf --bind 'enter:become:seq {} | fzf'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 999
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 99, lines.item_count }
  end
end
