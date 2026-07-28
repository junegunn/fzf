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
    # Border text is empty when no label is given
    pane = floating_pane
    refute_nil pane
    assert_equal "\#{@fzf-border-label}", pane_option(pane, 'pane-border-format')
    assert_equal '', pane_option(pane, '@fzf-border-label')
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
    # The label is held in a user option, not in the pane title, which is
    # left to the user. '#' is doubled so that the label is drawn as it is
    # written, instead of '#[...]' being taken as a style directive
    assert_equal ' ##fzf-label 100% ', pane_option(pane, '@fzf-border-label')
    assert_equal "\#{@fzf-border-label}", pane_option(pane, 'pane-border-format')
    title = IO.popen(['tmux', 'display-message', '-p', '-t', pane, "\#{pane_title}"], &:read)
    refute_equal ' #fzf-label 100% ', title.chomp
    tmux.send_keys :Enter
    assert_equal '1', fzf_output
  end

  def test_floating_pane_change_border_label
    tmux.send_keys "seq 100 | #{fzf(%(--popup center,80% --margin 0 --bind 'space:change-border-label( #[fg=red] 100% )'))}", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    pane = floating_pane
    refute_nil pane
    assert_equal '', pane_option(pane, '@fzf-border-label')
    tmux.send_keys :Space
    wait { assert_equal ' ##[fg=red] 100% ', pane_option(pane, '@fzf-border-label') }
    tmux.send_keys :Enter
    assert_equal '1', fzf_output
  end

  # A program running in the pane owns the pane title, but not the label
  def test_floating_pane_border_label_not_affected_by_title
    tmux.send_keys "seq 100 | #{fzf(%(--popup center,80% --margin 0 --border-label ' label ' --bind 'space:execute-silent(printf "\\033]2;hijacked\\033\\\\" > /dev/tty)'))}", :Enter
    tmux.until { |lines| assert_equal 100, lines.item_count }
    pane = floating_pane
    refute_nil pane
    tmux.send_keys :Space
    wait do
      title = IO.popen(['tmux', 'display-message', '-p', '-t', pane, "\#{pane_title}"], &:read)
      assert_equal 'hijacked', title.chomp
    end
    assert_equal ' label ', pane_option(pane, '@fzf-border-label')
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

  # stderr is merged in so that an unset option, which prints 'invalid
  # option' to stderr and nothing to stdout, is not read as an empty value
  def pane_option(pane, name)
    IO.popen(['tmux', 'show-options', '-p', '-t', pane, '-v', name], err: %i[child out], &:read).chomp
  end

  def floating_pane
    format = "\#{pane_id} \#{pane_floating_flag}"
    lines = IO.popen(['tmux', 'list-panes', '-t', tmux.win, '-F', format]) { |io| io.readlines(chomp: true) }
    lines.filter_map { |line| line.split.first if line.end_with?(' 1') }.first
  end
end
