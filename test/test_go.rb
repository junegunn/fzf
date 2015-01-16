#!/usr/bin/env ruby
# encoding: utf-8

require 'minitest/autorun'

class Tmux
  TEMPNAME = '/tmp/fzf-test.txt'

  attr_reader :win

  def initialize shell = 'bash'
    @win = go("new-window -P -F '#I' 'bash --rcfile ~/.fzf.#{shell}'").first
  end

  def self.current
    `tmux display-message -p '#I'`.split($/).first
  end

  def self.select id
    system "tmux select-window -t #{id}"
  end

  def closed?
    !go("list-window -F '#I'").include?(win)
  end

  def close timeout = 1
    send_keys 'C-c', 'C-u', 'C-d'
    wait(timeout) { closed? }
  end

  def kill
    go("kill-window -t #{win} 2> /dev/null")
  end

  def send_keys *args
    args = args.map { |a| %{"#{a}"} }.join ' '
    go("send-keys -t #{win} #{args}")
  end

  def capture
    go("capture-pane -t #{win} \\; save-buffer #{TEMPNAME}")
    raise "Window not found" if $?.exitstatus != 0
    File.read(TEMPNAME).split($/)
  end

  def until timeout = 1
    wait(timeout) { yield capture }
  end

private
  def wait timeout = 1
    waited = 0
    until yield
      waited += 0.1
      sleep 0.1
      raise "timeout" if waited > timeout
    end
  end

  def go *args
    %x[tmux #{args.join ' '}].split($/)
  end
end

class TestGoFZF < MiniTest::Unit::TestCase
  attr_reader :tmux

  def tempname
    '/tmp/output'
  end

  def setup
    ENV.delete 'FZF_DEFAULT_OPTS'
    ENV.delete 'FZF_DEFAULT_COMMAND'
    @prev = Tmux.current
    @tmux = Tmux.new
    File.unlink tempname rescue nil
  end

  def teardown
    @tmux.kill
    Tmux.select @prev
  end

  def test_vanilla
    tmux.send_keys "seq 1 100000 | fzf > #{tempname}", :Enter
    tmux.until { |lines| lines.last =~ /^>/ && lines[-2] =~ /^  100000/ }
    lines = tmux.capture
    assert_equal '  2',             lines[-4]
    assert_equal '> 1',             lines[-3]
    assert_equal '  100000/100000', lines[-2]
    assert_equal '>',               lines[-1]

    # Testing basic key bindings
    tmux.send_keys '99', 'C-a', '1', 'C-f', '3', 'C-b', 'C-h', 'C-u', 'C-e', 'C-y', 'C-k', 'Tab', 'BTab'
    tmux.until { |lines| lines.last == '> 391' }
    lines = tmux.capture
    assert_equal '> 1391',       lines[-4]
    assert_equal '  391',        lines[-3]
    assert_equal '  856/100000', lines[-2]
    assert_equal '> 391',        lines[-1]

    tmux.send_keys :Enter
    tmux.close
    assert_equal '1391', File.read(tempname).chomp
  end

  def test_fzf_default_command
    tmux.send_keys "FZF_DEFAULT_COMMAND='echo hello' fzf > #{tempname}", :Enter
    tmux.until { |lines| lines.last =~ /^>/ }

    tmux.send_keys :Enter
    tmux.close
    assert_equal 'hello', File.read(tempname).chomp
  end

  def test_fzf_prompt
    tmux.send_keys "fzf -q 'foo bar foo-bar'", :Enter
    tmux.until { |lines| lines.last =~ /foo-bar/ }

    # CTRL-A
    tmux.send_keys "C-A", "("
    tmux.until { |lines| lines.last == '> (foo bar foo-bar' }

    # META-F
    tmux.send_keys :Escape, :f, ")"
    tmux.until { |lines| lines.last == '> (foo) bar foo-bar' }

    # CTRL-B
    tmux.send_keys "C-B", "var"
    tmux.until { |lines| lines.last == '> (foovar) bar foo-bar' }

    # Left, CTRL-D
    tmux.send_keys :Left, :Left, "C-D"
    tmux.until { |lines| lines.last == '> (foovr) bar foo-bar' }

    # META-BS
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| lines.last == '> (r) bar foo-bar' }

    # CTRL-Y
    tmux.send_keys "C-Y", "C-Y"
    tmux.until { |lines| lines.last == '> (foovfoovr) bar foo-bar' }

    # META-B
    tmux.send_keys :Escape, :b, :Space, :Space
    tmux.until { |lines| lines.last == '> (  foovfoovr) bar foo-bar' }

    # CTRL-F / Right
    tmux.send_keys 'C-F', :Right, '/'
    tmux.until { |lines| lines.last == '> (  fo/ovfoovr) bar foo-bar' }

    # CTRL-H / BS
    tmux.send_keys 'C-H', :BSpace
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar foo-bar' }

    # CTRL-E
    tmux.send_keys "C-E", 'baz'
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar foo-barbaz' }

    # CTRL-U
    tmux.send_keys "C-U"
    tmux.until { |lines| lines.last == '>' }

    # CTRL-Y
    tmux.send_keys "C-Y"
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar foo-barbaz' }

    # CTRL-W
    tmux.send_keys "C-W", "bar-foo"
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar bar-foo' }

    # META-D
    tmux.send_keys :Escape, :b, :Escape, :b, :Escape, :d, "C-A", "C-Y"
    tmux.until { |lines| lines.last == '> bar(  fovfoovr) bar -foo' }

    # CTRL-M
    tmux.send_keys "C-M"
    tmux.until { |lines| lines.last !~ /^>/ }
    tmux.close
  end
end

