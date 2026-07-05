# frozen_string_literal: true

require 'shellwords'
require 'tmpdir'
require_relative 'lib/common'

# Tests for running fzf in a tmux floating pane (--popup on tmux 3.7 or above)
class TestTmux < TestInteractive
  def setup
    super
    # Cannot rely on the exit status; tmux versions before 3.7 exit
    # normally with empty output for an unknown command name
    supported = IO.popen(%w[tmux list-commands new-pane], err: File::NULL, &:read).include?('new-pane')
    skip('floating panes not supported') unless supported
  end

  def test_floating_pane
    tmux.send_keys "seq 100 | #{fzf('--popup center,80% --margin 0')}", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    # Border text is cleared when no label is given
    format = IO.popen(['tmux', 'show-options', '-p', '-t', floating_pane, 'pane-border-format'], &:read)
    assert_includes format, "''"
    tmux.send_keys '99'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    assert_equal '99', fzf_output
  end

  def test_floating_pane_killed
    tmux.send_keys "seq 100 | #{FZF} --popup bottom,50% --margin 0; echo code:$?", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    pane = floating_pane
    refute_nil pane
    assert system('tmux', 'kill-pane', '-t', pane)
    tmux.until { |lines| assert lines.any_include?('code:130') }
  end

  def test_floating_pane_border_label
    tmux.send_keys "seq 100 | #{fzf(%(--popup center,80% --margin 0 --border-label ' #fzf-label 100% '))}", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    pane = floating_pane
    refute_nil pane
    title = IO.popen(['tmux', 'display-message', '-p', '-t', pane, "\#{pane_title}"], &:read)
    assert_equal ' #fzf-label 100% ', title.chomp
    format = IO.popen(['tmux', 'show-options', '-p', '-t', pane, 'pane-border-format'], &:read)
    assert_includes format, "\#{pane_title}"
    tmux.send_keys :Enter
    assert_equal '1', fzf_output
  end

  def test_floating_pane_become
    tmux.send_keys "seq 100 | #{fzf(%(--popup center,80% --margin 0 --bind 'enter:become(echo became-{})'))}", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    tmux.send_keys :Enter
    assert_equal 'became-1', fzf_output
  end

  def test_explicit_border_falls_back_to_popup
    # display-popup requires an attached client, which the test environment
    # may not have; intercept it with a tmux shim on PATH
    dir = Dir.mktmpdir
    real = `command -v tmux`.chomp
    shim = File.join(dir, 'tmux')
    File.write(shim, <<~SH)
      #!/bin/sh
      if [ "$1" = display-popup ]; then
        echo popup-used >&2
        exit 0
      fi
      exec #{real.shellescape} "$@"
    SH
    FileUtils.chmod(0o755, shim)
    tmux.send_keys "seq 100 | PATH=#{dir.shellescape}:$PATH #{FZF} --popup center --border rounded", :Enter
    tmux.until { |lines| assert lines.any_include?('popup-used') }
    refute floating_pane
  ensure
    FileUtils.remove_entry(dir) if dir
  end

  private

  def floating_pane
    format = "\#{pane_id} \#{pane_floating_flag}"
    lines = IO.popen(['tmux', 'list-panes', '-t', tmux.win, '-F', format]) { |io| io.readlines(chomp: true) }
    lines.filter_map { |line| line.split.first if line.end_with?(' 1') }.first
  end
end
