#!/usr/bin/env ruby
# encoding: utf-8
# frozen_string_literal: true

# rubocop:disable Metrics/LineLength
# rubocop:disable Metrics/MethodLength

require 'minitest/autorun'
require 'fileutils'
require 'English'
require 'shellwords'

DEFAULT_TIMEOUT = 20

FILE = File.expand_path(__FILE__)
base = File.expand_path('../../', __FILE__)
Dir.chdir base
FZF = "FZF_DEFAULT_OPTS= FZF_DEFAULT_COMMAND= #{base}/bin/fzf"

class NilClass
  def include?(_str)
    false
  end

  def start_with?(_str)
    false
  end

  def end_with?(_str)
    false
  end
end

def wait
  since = Time.now
  while Time.now - since < DEFAULT_TIMEOUT
    return if yield
    sleep 0.05
  end
  raise 'timeout'
end

class Shell
  class << self
    def unsets
      'unset FZF_DEFAULT_COMMAND FZF_DEFAULT_OPTS FZF_CTRL_T_COMMAND FZF_CTRL_T_OPTS FZF_ALT_C_COMMAND FZF_ALT_C_OPTS FZF_CTRL_R_OPTS;'
    end

    def bash
      'PS1= PROMPT_COMMAND= bash --rcfile ~/.fzf.bash'
    end

    def zsh
      FileUtils.mkdir_p '/tmp/fzf-zsh'
      FileUtils.cp File.expand_path('~/.fzf.zsh'), '/tmp/fzf-zsh/.zshrc'
      'PS1= PROMPT_COMMAND= HISTSIZE=100 ZDOTDIR=/tmp/fzf-zsh zsh'
    end

    def fish
      'fish'
    end
  end
end

class Tmux
  TEMPNAME = '/tmp/fzf-test.txt'

  attr_reader :win

  def initialize(shell = :bash)
    @win =
      case shell
      when :bash
        go("new-window -d -P -F '#I' '#{Shell.unsets + Shell.bash}'").first
      when :zsh
        go("new-window -d -P -F '#I' '#{Shell.unsets + Shell.zsh}'").first
      when :fish
        go("new-window -d -P -F '#I' '#{Shell.unsets + Shell.fish}'").first
      else
        raise "Unknown shell: #{shell}"
      end
    go("set-window-option -t #{@win} pane-base-index 0")
    @lines = `tput lines`.chomp.to_i

    return unless shell == :fish
    send_keys('function fish_prompt; end; clear', :Enter)
    self.until(&:empty?)
  end

  def kill
    go("kill-window -t #{win} 2> /dev/null")
  end

  def send_keys(*args)
    target =
      if args.last.is_a?(Hash)
        hash = args.pop
        go("select-window -t #{win}")
        "#{win}.#{hash[:pane]}"
      else
        win
      end
    enum = (args + [nil]).each_cons(2)
    loop do
      pair = enum.next
      if pair.first == :Escape
        arg = pair.compact.map { |key| %("#{key}") }.join(' ')
        go(%(send-keys -t #{target} #{arg}))
        enum.next if pair.last
      else
        go(%(send-keys -t #{target} "#{pair.first}"))
      end
      break unless pair.last
    end
  end

  def paste(str)
    `tmux setb '#{str.gsub("'", "'\\''")}' \\; pasteb -t #{win} \\; send-keys -t #{win} Enter`
  end

  def capture(pane = 0)
    File.unlink TEMPNAME while File.exist? TEMPNAME
    wait do
      go("capture-pane -t #{win}.#{pane} \\; save-buffer #{TEMPNAME} 2> /dev/null")
      $CHILD_STATUS.exitstatus.zero?
    end
    File.read(TEMPNAME).split($INPUT_RECORD_SEPARATOR)[0, @lines].reverse.drop_while(&:empty?).reverse
  end

  def until(refresh = false, pane = 0)
    lines = nil
    begin
      wait do
        lines = capture(pane)
        class << lines
          def counts
            lazy
              .map { |l| l.scan %r{^. ([0-9]+)\/([0-9]+)( \(([0-9]+)\))?} }
              .reject(&:empty?)
              .first&.first&.map(&:to_i)&.values_at(0, 1, 3) || [0, 0, 0]
          end

          def match_count
            counts[0]
          end

          def item_count
            counts[1]
          end

          def select_count
            counts[2]
          end

          def any_include?(val)
            method = val.is_a?(Regexp) ? :match : :include?
            select { |line| line.send method, val }.first
          end
        end
        yield(lines).tap do |ok|
          send_keys 'C-l' if refresh && !ok
        end
      end
    rescue StandardError
      puts $ERROR_INFO.backtrace
      puts '>' * 80
      puts lines
      puts '<' * 80
      raise
    end
    lines
  end

  def prepare
    tries = 0
    begin
      self.until do |lines|
        send_keys 'C-u', 'hello'
        lines[-1].end_with?('hello')
      end
    rescue StandardError
      (tries += 1) < 5 ? retry : raise
    end
    send_keys 'C-u'
  end

  private

  def go(*args)
    `tmux #{args.join ' '}`.split($INPUT_RECORD_SEPARATOR)
  end
end

class TestBase < Minitest::Test
  TEMPNAME = '/tmp/output'

  attr_reader :tmux

  def tempname
    @temp_suffix ||= 0
    [TEMPNAME,
     caller_locations.map(&:label).find { |l| l =~ /^test_/ },
     @temp_suffix].join '-'
  end

  def writelines(path, lines)
    File.unlink path while File.exist? path
    File.open(path, 'w') { |f| f << lines.join($INPUT_RECORD_SEPARATOR) + $INPUT_RECORD_SEPARATOR }
  end

  def readonce
    wait { File.exist?(tempname) }
    File.read(tempname)
  ensure
    File.unlink tempname while File.exist?(tempname)
    @temp_suffix += 1
    tmux.prepare
  end

  def fzf(*opts)
    fzf!(*opts) + " > #{tempname}.tmp; mv #{tempname}.tmp #{tempname}"
  end

  def fzf!(*opts)
    opts = opts.map do |o|
      case o
      when Symbol
        o = o.to_s
        o.length > 1 ? "--#{o.tr('_', '-')}" : "-#{o}"
      when String, Numeric
        o.to_s
      end
    end.compact
    "#{FZF} #{opts.join ' '}"
  end
end

class TestGoFZF < TestBase
  def setup
    super
    @tmux = Tmux.new
  end

  def teardown
    @tmux.kill
  end

  def test_vanilla
    tmux.send_keys "seq 1 100000 | #{fzf}", :Enter
    tmux.until { |lines| lines.last =~ /^>/ && lines[-2] =~ /^  100000/ }
    lines = tmux.capture
    assert_equal '  2',             lines[-4]
    assert_equal '> 1',             lines[-3]
    assert_equal '  100000/100000', lines[-2]
    assert_equal '>',               lines[-1]

    # Testing basic key bindings
    tmux.send_keys '99', 'C-a', '1', 'C-f', '3', 'C-b', 'C-h', 'C-u', 'C-e', 'C-y', 'C-k', 'Tab', 'BTab'
    tmux.until { |lines| lines[-2] == '  856/100000' }
    lines = tmux.capture
    assert_equal '> 3910',       lines[-4]
    assert_equal '  391',        lines[-3]
    assert_equal '  856/100000', lines[-2]
    assert_equal '> 391',        lines[-1]

    tmux.send_keys :Enter
    assert_equal '3910', readonce.chomp
  end

  def test_fzf_default_command
    tmux.send_keys fzf.sub('FZF_DEFAULT_COMMAND=', "FZF_DEFAULT_COMMAND='echo hello'"), :Enter
    tmux.until { |lines| lines.last =~ /^>/ }

    tmux.send_keys :Enter
    assert_equal 'hello', readonce.chomp
  end

  def test_fzf_default_command_failure
    tmux.send_keys fzf.sub('FZF_DEFAULT_COMMAND=', 'FZF_DEFAULT_COMMAND=false'), :Enter
    tmux.until { |lines| lines[-2].include?('FZF_DEFAULT_COMMAND failed') }
    tmux.send_keys :Enter
  end

  def test_key_bindings
    tmux.send_keys "#{FZF} -q 'foo bar foo-bar'", :Enter
    tmux.until { |lines| lines.last =~ /^>/ }

    # CTRL-A
    tmux.send_keys 'C-A', '('
    tmux.until { |lines| lines.last == '> (foo bar foo-bar' }

    # META-F
    tmux.send_keys :Escape, :f, ')'
    tmux.until { |lines| lines.last == '> (foo) bar foo-bar' }

    # CTRL-B
    tmux.send_keys 'C-B', 'var'
    tmux.until { |lines| lines.last == '> (foovar) bar foo-bar' }

    # Left, CTRL-D
    tmux.send_keys :Left, :Left, 'C-D'
    tmux.until { |lines| lines.last == '> (foovr) bar foo-bar' }

    # META-BS
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| lines.last == '> (r) bar foo-bar' }

    # CTRL-Y
    tmux.send_keys 'C-Y', 'C-Y'
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
    tmux.send_keys 'C-E', 'baz'
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar foo-barbaz' }

    # CTRL-U
    tmux.send_keys 'C-U'
    tmux.until { |lines| lines.last == '>' }

    # CTRL-Y
    tmux.send_keys 'C-Y'
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar foo-barbaz' }

    # CTRL-W
    tmux.send_keys 'C-W', 'bar-foo'
    tmux.until { |lines| lines.last == '> (  fovfoovr) bar bar-foo' }

    # META-D
    tmux.send_keys :Escape, :b, :Escape, :b, :Escape, :d, 'C-A', 'C-Y'
    tmux.until { |lines| lines.last == '> bar(  fovfoovr) bar -foo' }

    # CTRL-M
    tmux.send_keys 'C-M'
    tmux.until { |lines| lines.last !~ /^>/ }
  end

  def test_file_word
    tmux.send_keys "#{FZF} -q '--/foo bar/foo-bar/baz' --filepath-word", :Enter
    tmux.until { |lines| lines.last =~ /^>/ }

    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :d
    tmux.send_keys :Escape, :f
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| lines.last == '> --///baz' }
  end

  def test_multi_order
    tmux.send_keys "seq 1 10 | #{fzf :multi}", :Enter
    tmux.until { |lines| lines.last =~ /^>/ }

    tmux.send_keys :Tab, :Up, :Up, :Tab, :Tab, :Tab, # 3, 2
                   'C-K', 'C-K', 'C-K', 'C-K', :BTab, :BTab, # 5, 6
                   :PgUp, 'C-J', :Down, :Tab, :Tab # 8, 7
    tmux.until { |lines| lines[-2].include? '(6)' }
    tmux.send_keys 'C-M'
    assert_equal %w[3 2 5 6 8 7], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_with_nth
    [true, false].each do |multi|
      tmux.send_keys "(echo '  1st 2nd 3rd/';
                       echo '  first second third/') |
                       #{fzf multi && :multi, :x, :nth, 2, :with_nth, '2,-1,1'}",
                     :Enter
      tmux.until { |lines| lines[-2].include?('2/2') }

      # Transformed list
      lines = tmux.capture
      assert_equal '  second third/first', lines[-4]
      assert_equal '> 2nd 3rd/1st',        lines[-3]

      # However, the output must not be transformed
      if multi
        tmux.send_keys :BTab, :BTab
        tmux.until { |lines| lines[-2].include?('(2)') }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/', '  first second third/'], readonce.split($INPUT_RECORD_SEPARATOR)
      else
        tmux.send_keys '^', '3'
        tmux.until { |lines| lines[-2].include?('1/2') }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/'], readonce.split($INPUT_RECORD_SEPARATOR)
      end
    end
  end

  def test_scroll
    [true, false].each do |rev|
      tmux.send_keys "seq 1 100 | #{fzf rev && :reverse}", :Enter
      tmux.until { |lines| lines.include? '  100/100' }
      tmux.send_keys(*Array.new(110) { rev ? :Down : :Up })
      tmux.until { |lines| lines.include? '> 100' }
      tmux.send_keys :Enter
      assert_equal '100', readonce.chomp
    end
  end

  def test_select_1
    tmux.send_keys "seq 1 100 | #{fzf :with_nth, '..,..', :print_query, :q, 5555, :'1'}", :Enter
    assert_equal %w[5555 55], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_exit_0
    tmux.send_keys "seq 1 100 | #{fzf :with_nth, '..,..', :print_query, :q, 555_555, :'0'}", :Enter
    assert_equal ['555555'], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_select_1_exit_0_fail
    [:'0', :'1', %i[1 0]].each do |opt|
      tmux.send_keys "seq 1 100 | #{fzf :print_query, :multi, :q, 5, *opt}", :Enter
      tmux.until { |lines| lines.last =~ /^> 5/ }
      tmux.send_keys :BTab, :BTab, :BTab
      tmux.until { |lines| lines[-2].include?('(3)') }
      tmux.send_keys :Enter
      assert_equal %w[5 5 50 51], readonce.split($INPUT_RECORD_SEPARATOR)
    end
  end

  def test_query_unicode
    tmux.paste "(echo abc; echo 가나다) | #{fzf :query, '가다'}"
    tmux.until { |lines| lines[-2].include? '1/2' }
    tmux.send_keys :Enter
    assert_equal ['가나다'], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_sync
    tmux.send_keys "seq 1 100 | #{fzf! :multi} | awk '{print \\$1 \\$1}' | #{fzf :sync}", :Enter
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 9
    tmux.until { |lines| lines[-2] == '  19/100' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2].include?('(3)') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 'C-K', :Enter
    assert_equal ['9090'], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tac
    tmux.send_keys "seq 1 1000 | #{fzf :tac, :multi}", :Enter
    tmux.until { |lines| lines[-2].include? '1000/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2].include?('(3)') }
    tmux.send_keys :Enter
    assert_equal %w[1000 999 998], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tac_sort
    tmux.send_keys "seq 1 1000 | #{fzf :tac, :multi}", :Enter
    tmux.until { |lines| lines[-2].include? '1000/1000' }
    tmux.send_keys '99'
    tmux.until { |lines| lines[-2].include? '28/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2].include?('(3)') }
    tmux.send_keys :Enter
    assert_equal %w[99 999 998], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tac_nosort
    tmux.send_keys "seq 1 1000 | #{fzf :tac, :no_sort, :multi}", :Enter
    tmux.until { |lines| lines[-2].include? '1000/1000' }
    tmux.send_keys '00'
    tmux.until { |lines| lines[-2].include? '10/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2].include?('(3)') }
    tmux.send_keys :Enter
    assert_equal %w[1000 900 800], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_expect
    test = lambda do |key, feed, expected = key|
      tmux.send_keys "seq 1 100 | #{fzf :expect, key}; sync", :Enter
      tmux.until { |lines| lines[-2].include? '100/100' }
      tmux.send_keys '55'
      tmux.until { |lines| lines[-2].include? '1/100' }
      tmux.send_keys(*feed)
      tmux.prepare
      assert_equal [expected, '55'], readonce.split($INPUT_RECORD_SEPARATOR)
    end
    test.call 'ctrl-t', 'C-T'
    test.call 'ctrl-t', 'Enter', ''
    test.call 'alt-c', %i[Escape c]
    test.call 'f1', 'f1'
    test.call 'f2', 'f2'
    test.call 'f3', 'f3'
    test.call 'f2,f4', 'f2', 'f2'
    test.call 'f2,f4', 'f4', 'f4'
    test.call 'alt-/', %i[Escape /]
    %w[f5 f6 f7 f8 f9 f10].each do |key|
      test.call 'f5,f6,f7,f8,f9,f10', key, key
    end
    test.call '@', '@'
  end

  def test_expect_print_query
    tmux.send_keys "seq 1 100 | #{fzf '--expect=alt-z', :print_query}", :Enter
    tmux.until { |lines| lines[-2].include? '100/100' }
    tmux.send_keys '55'
    tmux.until { |lines| lines[-2].include? '1/100' }
    tmux.send_keys :Escape, :z
    assert_equal ['55', 'alt-z', '55'], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_expect_printable_character_print_query
    tmux.send_keys "seq 1 100 | #{fzf '--expect=z --print-query'}", :Enter
    tmux.until { |lines| lines[-2].include? '100/100' }
    tmux.send_keys '55'
    tmux.until { |lines| lines[-2].include? '1/100' }
    tmux.send_keys 'z'
    assert_equal %w[55 z 55], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_expect_print_query_select_1
    tmux.send_keys "seq 1 100 | #{fzf '-q55 -1 --expect=alt-z --print-query'}", :Enter
    assert_equal ['55', '', '55'], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_toggle_sort
    ['--toggle-sort=ctrl-r', '--bind=ctrl-r:toggle-sort'].each do |opt|
      tmux.send_keys "seq 1 111 | #{fzf "-m +s --tac #{opt} -q11"}", :Enter
      tmux.until { |lines| lines[-3].include? '> 111' }
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-2].include? '4/111 -S (1)' }
      tmux.send_keys 'C-R'
      tmux.until { |lines| lines[-3].include? '> 11' }
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-2].include? '4/111 +S (2)' }
      tmux.send_keys :Enter
      assert_equal %w[111 11], readonce.split($INPUT_RECORD_SEPARATOR)
    end
  end

  def test_unicode_case
    writelines tempname, %w[строКА1 СТРОКА2 строка3 Строка4]
    assert_equal %w[СТРОКА2 Строка4], `#{FZF} -fС < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
    assert_equal %w[строКА1 СТРОКА2 строка3 Строка4], `#{FZF} -fс < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tiebreak
    input = %w[
      --foobar--------
      -----foobar---
      ----foobar--
      -------foobar-
    ]
    writelines tempname, input

    assert_equal input, `#{FZF} -ffoobar --tiebreak=index < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    by_length = %w[
      ----foobar--
      -----foobar---
      -------foobar-
      --foobar--------
    ]
    assert_equal by_length, `#{FZF} -ffoobar < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
    assert_equal by_length, `#{FZF} -ffoobar --tiebreak=length < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    by_begin = %w[
      --foobar--------
      ----foobar--
      -----foobar---
      -------foobar-
    ]
    assert_equal by_begin, `#{FZF} -ffoobar --tiebreak=begin < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
    assert_equal by_begin, `#{FZF} -f"!z foobar" -x --tiebreak begin < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal %w[
      -------foobar-
      ----foobar--
      -----foobar---
      --foobar--------
    ], `#{FZF} -ffoobar --tiebreak end < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal input, `#{FZF} -f"!z" -x --tiebreak end < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tiebreak_index_begin
    writelines tempname, [
      'xoxxxxxoxx',
      'xoxxxxxox',
      'xxoxxxoxx',
      'xxxoxoxxx',
      'xxxxoxox',
      '  xxoxoxxx'
    ]

    assert_equal [
      'xxxxoxox',
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxoxxxoxx',
      'xoxxxxxox',
      'xoxxxxxoxx'
    ], `#{FZF} -foo < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal [
      'xxxoxoxxx',
      'xxxxoxox',
      '  xxoxoxxx',
      'xxoxxxoxx',
      'xoxxxxxoxx',
      'xoxxxxxox'
    ], `#{FZF} -foo --tiebreak=index < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    # Note that --tiebreak=begin is now based on the first occurrence of the
    # first character on the pattern
    assert_equal [
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxxxoxox',
      'xxoxxxoxx',
      'xoxxxxxoxx',
      'xoxxxxxox'
    ], `#{FZF} -foo --tiebreak=begin < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal [
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxxxoxox',
      'xxoxxxoxx',
      'xoxxxxxox',
      'xoxxxxxoxx'
    ], `#{FZF} -foo --tiebreak=begin,length < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tiebreak_begin_algo_v2
    writelines tempname, [
      'baz foo bar',
      'foo bar baz'
    ]
    assert_equal [
      'foo bar baz',
      'baz foo bar'
    ], `#{FZF} -fbar --tiebreak=begin --algo=v2 < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tiebreak_end
    writelines tempname, [
      'xoxxxxxxxx',
      'xxoxxxxxxx',
      'xxxoxxxxxx',
      'xxxxoxxxx',
      'xxxxxoxxx',
      '  xxxxoxxx'
    ]

    assert_equal [
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxxxoxxx',
      'xoxxxxxxxx',
      'xxoxxxxxxx',
      'xxxoxxxxxx'
    ], `#{FZF} -fo < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal [
      'xxxxxoxxx',
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxoxxxxxx',
      'xxoxxxxxxx',
      'xoxxxxxxxx'
    ], `#{FZF} -fo --tiebreak=end < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    assert_equal [
      'xxxxxoxxx',
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxoxxxxxx',
      'xxoxxxxxxx',
      'xoxxxxxxxx'
    ], `#{FZF} -fo --tiebreak=end,length,begin < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_tiebreak_length_with_nth
    input = %w[
      1:hell
      123:hello
      12345:he
      1234567:h
    ]
    writelines tempname, input

    output = %w[
      1:hell
      12345:he
      123:hello
      1234567:h
    ]
    assert_equal output, `#{FZF} -fh < #{tempname}`.split($INPUT_RECORD_SEPARATOR)

    # Since 0.16.8, --nth doesn't affect --tiebreak
    assert_equal output, `#{FZF} -fh -n2 -d: < #{tempname}`.split($INPUT_RECORD_SEPARATOR)
  end

  def test_invalid_cache
    tmux.send_keys "(echo d; echo D; echo x) | #{fzf '-q d'}", :Enter
    tmux.until { |lines| lines[-2].include? '2/3' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2].include? '3/3' }
    tmux.send_keys :D
    tmux.until { |lines| lines[-2].include? '1/3' }
    tmux.send_keys :Enter
  end

  def test_invalid_cache_query_type
    command = %[(echo 'foo\\$bar'; echo 'barfoo'; echo 'foo^bar'; echo \\"foo'1-2\\"; seq 100) | #{fzf}]

    # Suffix match
    tmux.send_keys command, :Enter
    tmux.until { |lines| lines.match_count == 104 }
    tmux.send_keys 'foo$'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'bar'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter

    # Prefix match
    tmux.prepare
    tmux.send_keys command, :Enter
    tmux.until { |lines| lines.match_count == 104 }
    tmux.send_keys '^bar'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'C-a', 'foo'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter

    # Exact match
    tmux.prepare
    tmux.send_keys command, :Enter
    tmux.until { |lines| lines.match_count == 104 }
    tmux.send_keys "'12"
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'C-a', 'foo'
    tmux.until { |lines| lines.match_count == 1 }
  end

  def test_smart_case_for_each_term
    assert_equal 1, `echo Foo bar | #{FZF} -x -f "foo Fbar" | wc -l`.to_i
  end

  def test_bind
    tmux.send_keys "seq 1 1000 | #{fzf '-m --bind=ctrl-j:accept,u:up,T:toggle-up,t:toggle'}", :Enter
    tmux.until { |lines| lines[-2].end_with? '/1000' }
    tmux.send_keys 'uuu', 'TTT', 'tt', 'uu', 'ttt', 'C-j'
    assert_equal %w[4 5 6 9], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_bind_print_query
    tmux.send_keys "seq 1 1000 | #{fzf '-m --bind=ctrl-j:print-query'}", :Enter
    tmux.until { |lines| lines[-2].end_with? '/1000' }
    tmux.send_keys 'print-my-query', 'C-j'
    assert_equal %w[print-my-query], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_bind_replace_query
    tmux.send_keys "seq 1 1000 | #{fzf '--print-query --bind=ctrl-j:replace-query'}", :Enter
    tmux.send_keys '1'
    tmux.until { |lines| lines[-2].end_with? '272/1000' }
    tmux.send_keys 'C-k', 'C-j'
    tmux.until { |lines| lines[-2].end_with? '29/1000' }
    tmux.until { |lines| lines[-1].end_with? '> 10' }
  end

  def test_long_line
    data = '.' * 256 * 1024
    File.open(tempname, 'w') do |f|
      f << data
    end
    assert_equal data, `#{FZF} -f . < #{tempname}`.chomp
  end

  def test_read0
    lines = `find .`.split($INPUT_RECORD_SEPARATOR)
    assert_equal lines.last, `find . | #{FZF} -e -f "^#{lines.last}$"`.chomp
    assert_equal(
      lines.last,
      `find . -print0 | #{FZF} --read0 -e -f "^#{lines.last}$"`.chomp
    )
  end

  def test_select_all_deselect_all_toggle_all
    tmux.send_keys "seq 100 | #{fzf '--bind ctrl-a:select-all,ctrl-d:deselect-all,ctrl-t:toggle-all --multi'}", :Enter
    tmux.until { |lines| lines[-2].include? '100/100' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2].include? '(3)' }
    tmux.send_keys 'C-t'
    tmux.until { |lines| lines[-2].include? '(97)' }
    tmux.send_keys 'C-a'
    tmux.until { |lines| lines[-2].include? '(100)' }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| lines[-2].include? '(98)' }
    tmux.send_keys 'C-d'
    tmux.until { |lines| !lines[-2].include? '(' }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| lines[-2].include? '(2)' }
    tmux.send_keys 0
    tmux.until { |lines| lines[-2].include? '10/100' }
    tmux.send_keys 'C-a'
    tmux.until { |lines| lines[-2].include? '(12)' }
    tmux.send_keys :Enter
    assert_equal %w[2 1 10 20 30 40 50 60 70 80 90 100],
                 readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_history
    history_file = '/tmp/fzf-test-history'

    # History with limited number of entries
    begin
      File.unlink history_file
    rescue
      nil
    end
    opts = "--history=#{history_file} --history-size=4"
    input = %w[00 11 22 33 44].map { |e| e + $INPUT_RECORD_SEPARATOR }
    input.each do |keys|
      tmux.send_keys "seq 100 | #{fzf opts}", :Enter
      tmux.until { |lines| lines[-2].include? '100/100' }
      tmux.send_keys keys
      tmux.until { |lines| lines[-2].include? '1/100' }
      tmux.send_keys :Enter
      readonce
    end
    assert_equal input[1..-1], File.readlines(history_file)

    # Update history entries (not changed on disk)
    tmux.send_keys "seq 100 | #{fzf opts}", :Enter
    tmux.until { |lines| lines[-2].include? '100/100' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1].end_with? '> 44' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1].end_with? '> 33' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-1].end_with? '> 3' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-1].end_with? '> 31' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1].end_with? '> 22' }
    tmux.send_keys 'C-n'
    tmux.until { |lines| lines[-1].end_with? '> 31' }
    tmux.send_keys 0
    tmux.until { |lines| lines[-1].end_with? '> 310' }
    tmux.send_keys :Enter
    readonce
    assert_equal %w[22 33 44 310].map { |e| e + $INPUT_RECORD_SEPARATOR }, File.readlines(history_file)

    # Respect --bind option
    tmux.send_keys "seq 100 | #{fzf opts + ' --bind ctrl-p:next-history,ctrl-n:previous-history'}", :Enter
    tmux.until { |lines| lines[-2].include? '100/100' }
    tmux.send_keys 'C-n', 'C-n', 'C-n', 'C-n', 'C-p'
    tmux.until { |lines| lines[-1].end_with?('33') }
    tmux.send_keys :Enter
  ensure
    File.unlink history_file
  end

  def test_execute
    output = '/tmp/fzf-test-execute'
    opts = %[--bind \\"alt-a:execute(echo [{}] >> #{output}),alt-b:execute[echo /{}{}/ >> #{output}],C:execute:echo /{}{}{}/ >> #{output}\\"]
    wait = ->(exp) { tmux.until { |lines| lines[-2].include? exp } }
    writelines tempname, %w[foo'bar foo"bar foo$bar]
    tmux.send_keys "cat #{tempname} | #{fzf opts}; sync", :Enter
    wait['3/3']
    tmux.send_keys :Escape, :a
    wait['/3']
    tmux.send_keys :Escape, :a
    wait['/3']
    tmux.send_keys :Up
    tmux.send_keys :Escape, :b
    wait['/3']
    tmux.send_keys :Escape, :b
    wait['/3']
    tmux.send_keys :Up
    tmux.send_keys :C
    wait['3/3']
    tmux.send_keys 'barfoo'
    wait['0/3']
    tmux.send_keys :Escape, :a
    wait['/3']
    tmux.send_keys :Escape, :b
    wait['/3']
    tmux.send_keys :Enter
    readonce
    assert_equal %w[[foo'bar] [foo'bar]
                    /foo"barfoo"bar/ /foo"barfoo"bar/
                    /foo$barfoo$barfoo$bar/],
                 File.readlines(output).map(&:chomp)
  ensure
    begin
      File.unlink output
    rescue
      nil
    end
  end

  def test_execute_multi
    output = '/tmp/fzf-test-execute-multi'
    opts = %[--multi --bind \\"alt-a:execute-multi(echo {}/{+} >> #{output}; sync)\\"]
    writelines tempname, %w[foo'bar foo"bar foo$bar foobar]
    tmux.send_keys "cat #{tempname} | #{fzf opts}", :Enter
    tmux.until { |lines| lines[-2].include? '4/4' }
    tmux.send_keys :Escape, :a
    tmux.until { |lines| lines[-2].include? '/4' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.send_keys :Escape, :a
    tmux.until { |lines| lines[-2].include? '/4' }
    tmux.send_keys :Tab, :Tab
    tmux.send_keys :Escape, :a
    tmux.until { |lines| lines[-2].include? '/4' }
    tmux.send_keys :Enter
    tmux.prepare
    readonce
    assert_equal [%(foo'bar/foo'bar),
                  %(foo'bar foo"bar foo$bar/foo'bar foo"bar foo$bar),
                  %(foo'bar foo"bar foobar/foo'bar foo"bar foobar)],
                 File.readlines(output).map(&:chomp)
  ensure
    begin
      File.unlink output
    rescue
      nil
    end
  end

  def test_execute_plus_flag
    output = tempname + '.tmp'
    begin
      File.unlink output
    rescue
      nil
    end
    writelines tempname, ['foo bar', '123 456']

    tmux.send_keys "cat #{tempname} | #{FZF} --multi --bind 'x:execute-silent(echo {+}/{}/{+2}/{2} >> #{output})'", :Enter

    execute = lambda do
      tmux.send_keys 'x', 'y'
      tmux.until { |lines| lines[-2].include? '0/2' }
      tmux.send_keys :BSpace
      tmux.until { |lines| lines[-2].include? '2/2' }
    end

    tmux.until { |lines| lines[-2].include? '2/2' }
    execute.call

    tmux.send_keys :Up
    tmux.send_keys :Tab
    execute.call

    tmux.send_keys :Tab
    execute.call

    tmux.send_keys :Enter
    tmux.prepare
    readonce

    assert_equal [
      %(foo bar/foo bar/bar/bar),
      %(123 456/foo bar/456/bar),
      %(123 456 foo bar/foo bar/456 bar/bar)
    ], File.readlines(output).map(&:chomp)
  rescue
    begin
      File.unlink output
    rescue
      nil
    end
  end

  def test_execute_shell
    # Custom script to use as $SHELL
    output = tempname + '.out'
    begin
      File.unlink output
    rescue
      nil
    end
    writelines tempname,
               ['#!/usr/bin/env bash', "echo $1 / $2 > #{output}", 'sync']
    system "chmod +x #{tempname}"

    tmux.send_keys "echo foo | SHELL=#{tempname} fzf --bind 'enter:execute:{}bar'", :Enter
    tmux.until { |lines| lines[-2].include? '1/1' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-2].include? '1/1' }
    tmux.send_keys 'C-c'
    tmux.prepare
    assert_equal ["-c / 'foo'bar"], File.readlines(output).map(&:chomp)
  ensure
    begin
      File.unlink output
    rescue
      nil
    end
  end

  def test_cycle
    tmux.send_keys "seq 8 | #{fzf :cycle}", :Enter
    tmux.until { |lines| lines[-2].include? '8/8' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-10].start_with? '>' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-9].start_with? '>' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-10].start_with? '>' }
    tmux.send_keys :PgUp
    tmux.until { |lines| lines[-10].start_with? '>' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-3].start_with? '>' }
    tmux.send_keys :PgDn
    tmux.until { |lines| lines[-3].start_with? '>' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-10].start_with? '>' }
  end

  def test_header_lines
    tmux.send_keys "seq 100 | #{fzf '--header-lines=10 -q 5'}", :Enter
    2.times do
      tmux.until do |lines|
        lines[-2].include?('/90') &&
          lines[-3]  == '  1' &&
          lines[-4]  == '  2' &&
          lines[-13] == '> 50'
      end
      tmux.send_keys :Down
    end
    tmux.send_keys :Enter
    assert_equal '50', readonce.chomp
  end

  def test_header_lines_reverse
    tmux.send_keys "seq 100 | #{fzf '--header-lines=10 -q 5 --reverse'}", :Enter
    2.times do
      tmux.until do |lines|
        lines[1].include?('/90') &&
          lines[2]  == '  1' &&
          lines[3]  == '  2' &&
          lines[12] == '> 50'
      end
      tmux.send_keys :Up
    end
    tmux.send_keys :Enter
    assert_equal '50', readonce.chomp
  end

  def test_header_lines_reverse_list
    tmux.send_keys "seq 100 | #{fzf '--header-lines=10 -q 5 --layout=reverse-list'}", :Enter
    2.times do
      tmux.until do |lines|
        lines[0]    == '> 50' &&
          lines[-4] == '  2' &&
          lines[-3] == '  1' &&
          lines[-2].include?('/90')
      end
      tmux.send_keys :Up
    end
    tmux.send_keys :Enter
    assert_equal '50', readonce.chomp
  end

  def test_header_lines_overflow
    tmux.send_keys "seq 100 | #{fzf '--header-lines=200'}", :Enter
    tmux.until do |lines|
      lines[-2].include?('0/0') &&
        lines[-3].include?('  1')
    end
    tmux.send_keys :Enter
    assert_equal '', readonce.chomp
  end

  def test_header_lines_with_nth
    tmux.send_keys "seq 100 | #{fzf '--header-lines 5 --with-nth 1,1,1,1,1'}", :Enter
    tmux.until do |lines|
      lines[-2].include?('95/95') &&
        lines[-3] == '  11111' &&
        lines[-7] == '  55555' &&
        lines[-8] == '> 66666'
    end
    tmux.send_keys :Enter
    assert_equal '6', readonce.chomp
  end

  def test_header
    tmux.send_keys "seq 100 | #{fzf "--header \\\"\\$(head -5 #{FILE})\\\""}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[-2].include?('100/100') &&
        lines[-7..-3].map(&:strip) == header &&
        lines[-8] == '> 1'
    end
  end

  def test_header_reverse
    tmux.send_keys "seq 100 | #{fzf "--header=\\\"\\$(head -5 #{FILE})\\\" --reverse"}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[1].include?('100/100') &&
        lines[2..6].map(&:strip) == header &&
        lines[7] == '> 1'
    end
  end

  def test_header_reverse_list
    tmux.send_keys "seq 100 | #{fzf "--header=\\\"\\$(head -5 #{FILE})\\\" --layout=reverse-list"}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[-2].include?('100/100') &&
        lines[-7..-3].map(&:strip) == header &&
        lines[0] == '> 1'
    end
  end

  def test_header_and_header_lines
    tmux.send_keys "seq 100 | #{fzf "--header-lines 10 --header \\\"\\$(head -5 #{FILE})\\\""}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[-2].include?('90/90') &&
        lines[-7...-2].map(&:strip) == header &&
        lines[-17...-7].map(&:strip) == (1..10).map(&:to_s).reverse
    end
  end

  def test_header_and_header_lines_reverse
    tmux.send_keys "seq 100 | #{fzf "--reverse --header-lines 10 --header \\\"\\$(head -5 #{FILE})\\\""}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[1].include?('90/90') &&
        lines[2...7].map(&:strip) == header &&
        lines[7...17].map(&:strip) == (1..10).map(&:to_s)
    end
  end

  def test_header_and_header_lines_reverse_list
    tmux.send_keys "seq 100 | #{fzf "--layout=reverse-list --header-lines 10 --header \\\"\\$(head -5 #{FILE})\\\""}", :Enter
    header = File.readlines(FILE).take(5).map(&:strip)
    tmux.until do |lines|
      lines[-2].include?('90/90') &&
        lines[-7...-2].map(&:strip) == header &&
        lines[-17...-7].map(&:strip) == (1..10).map(&:to_s).reverse
    end
  end

  def test_cancel
    tmux.send_keys "seq 10 | #{fzf '--bind 2:cancel'}", :Enter
    tmux.until { |lines| lines[-2].include?('10/10') }
    tmux.send_keys '123'
    tmux.until { |lines| lines[-1] == '> 3' && lines[-2].include?('1/10') }
    tmux.send_keys 'C-y', 'C-y'
    tmux.until { |lines| lines[-1] == '> 311' }
    tmux.send_keys 2
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 2
    tmux.prepare
  end

  def test_margin
    tmux.send_keys "yes | head -1000 | #{fzf '--margin 5,3'}", :Enter
    tmux.until { |lines| lines[4] == '' && lines[5] == '     y' }
    tmux.send_keys :Enter
  end

  def test_margin_reverse
    tmux.send_keys "seq 1000 | #{fzf '--margin 7,5 --reverse'}", :Enter
    tmux.until { |lines| lines[1 + 7] == '       1000/1000' }
    tmux.send_keys :Enter
  end

  def test_margin_reverse_list
    tmux.send_keys "yes | head -1000 | #{fzf '--margin 5,3 --layout=reverse-list'}", :Enter
    tmux.until { |lines| lines[4] == '' && lines[5] == '   > y' }
    tmux.send_keys :Enter
  end

  def test_tabstop
    writelines tempname, ["f\too\tba\tr\tbaz\tbarfooq\tux"]
    {
      1 => '> f oo ba r baz barfooq ux',
      2 => '> f oo  ba  r baz barfooq ux',
      3 => '> f  oo ba r  baz   barfooq  ux',
      4 => '> f   oo  ba  r   baz barfooq ux',
      5 => '> f    oo   ba   r    baz  barfooq   ux',
      6 => '> f     oo    ba    r     baz   barfooq     ux',
      7 => '> f      oo     ba     r      baz    barfooq       ux',
      8 => '> f       oo      ba      r       baz     barfooq ux',
      9 => '> f        oo       ba       r        baz      barfooq  ux'
    }.each do |ts, exp|
      tmux.prepare
      tmux.send_keys %(cat #{tempname} | fzf --tabstop=#{ts}), :Enter
      tmux.until(true) do |lines|
        exp.start_with? lines[-3].to_s.strip.sub(/\.\.$/, '')
      end
      tmux.send_keys :Enter
    end
  end

  def test_with_nth_basic
    writelines tempname, ['hello world ', 'byebye']
    assert_equal(
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 < #{tempname}`.chomp
    )
  end

  def test_with_nth_ansi
    writelines tempname, ["\x1b[33mhello \x1b[34;1mworld\x1b[m ", 'byebye']
    assert_equal(
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 --ansi < #{tempname}`.chomp
    )
  end

  def test_with_nth_no_ansi
    src = "\x1b[33mhello \x1b[34;1mworld\x1b[m "
    writelines tempname, [src, 'byebye']
    assert_equal(
      src,
      `#{FZF} -fhehe -x -n 2.. --with-nth 2,1,1 --no-ansi < #{tempname}`.chomp
    )
  end

  def test_exit_0_exit_code
    `echo foo | #{FZF} -q bar -0`
    assert_equal 1, $CHILD_STATUS.exitstatus
  end

  def test_invalid_option
    lines = `#{FZF} --foobar 2>&1`
    assert_equal 2, $CHILD_STATUS.exitstatus
    assert lines.include?('unknown option: --foobar'), lines
  end

  def test_filter_exitstatus
    # filter / streaming filter
    ['', '--no-sort'].each do |opts|
      assert `echo foo | #{FZF} -f foo #{opts}`.include?('foo')
      assert_equal 0, $CHILD_STATUS.exitstatus

      assert `echo foo | #{FZF} -f bar #{opts}`.empty?
      assert_equal 1, $CHILD_STATUS.exitstatus
    end
  end

  def test_exitstatus_empty
    { '99' => '0', '999' => '1' }.each do |query, status|
      tmux.send_keys "seq 100 | #{FZF} -q #{query}; echo --\\$?--", :Enter
      tmux.until { |lines| lines[-2] =~ %r{ [10]/100} }
      tmux.send_keys :Enter
      tmux.until { |lines| lines.last.include? "--#{status}--" }
    end
  end

  def test_default_extended
    assert_equal '100', `seq 100 | #{FZF} -f "1 00$"`.chomp
    assert_equal '', `seq 100 | #{FZF} -f "1 00$" +x`.chomp
  end

  def test_exact
    assert_equal 4, `seq 123 | #{FZF} -f 13`.lines.length
    assert_equal 2, `seq 123 | #{FZF} -f 13 -e`.lines.length
    assert_equal 4, `seq 123 | #{FZF} -f 13 +e`.lines.length
  end

  def test_or_operator
    assert_equal %w[1 5 10], `seq 10 | #{FZF} -f "1 | 5"`.lines.map(&:chomp)
    assert_equal %w[1 10 2 3 4 5 6 7 8 9],
                 `seq 10 | #{FZF} -f '1 | !1'`.lines.map(&:chomp)
  end

  def test_hscroll_off
    writelines tempname, ['=' * 10_000 + '0123456789']
    [0, 3, 6].each do |off|
      tmux.prepare
      tmux.send_keys "#{FZF} --hscroll-off=#{off} -q 0 < #{tempname}", :Enter
      tmux.until { |lines| lines[-3].end_with?((0..off).to_a.join + '..') }
      tmux.send_keys '9'
      tmux.until { |lines| lines[-3].end_with? '789' }
      tmux.send_keys :Enter
    end
  end

  def test_partial_caching
    tmux.send_keys 'seq 1000 | fzf -e', :Enter
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 11
    tmux.until { |lines| lines[-2] == '  19/1000' }
    tmux.send_keys 'C-a', "'"
    tmux.until { |lines| lines[-2] == '  28/1000' }
    tmux.send_keys :Enter
  end

  def test_jump
    tmux.send_keys "seq 1000 | #{fzf "--multi --jump-labels 12345 --bind 'ctrl-j:jump'"}", :Enter
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 'C-j'
    tmux.until { |lines| lines[-7] == '5 5' }
    tmux.until { |lines| lines[-8] == '  6' }
    tmux.send_keys '5'
    tmux.until { |lines| lines[-7] == '> 5' }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-7] == ' >5' }
    tmux.send_keys 'C-j'
    tmux.until { |lines| lines[-7] == '5>5' }
    tmux.send_keys '2'
    tmux.until { |lines| lines[-4] == '> 2' }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-4] == ' >2' }
    tmux.send_keys 'C-j'
    tmux.until { |lines| lines[-7] == '5>5' }

    # Press any key other than jump labels to cancel jump
    tmux.send_keys '6'
    tmux.until { |lines| lines[-3] == '> 1' }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-3] == '>>1' }
    tmux.send_keys :Enter
    assert_equal %w[5 2 1], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_jump_accept
    tmux.send_keys "seq 1000 | #{fzf "--multi --jump-labels 12345 --bind 'ctrl-j:jump-accept'"}", :Enter
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 'C-j'
    tmux.until { |lines| lines[-7] == '5 5' }
    tmux.send_keys '3'
    assert_equal '3', readonce.chomp
  end

  def test_preview
    tmux.send_keys %(seq 1000 | sed s/^2$// | #{FZF} -m --preview 'sleep 0.2; echo {{}-{+}}' --bind ?:toggle-preview), :Enter
    tmux.until { |lines| lines[1].include?(' {1-1}') }
    tmux.send_keys :Up
    tmux.until { |lines| lines[1].include?(' {-}') }
    tmux.send_keys '555'
    tmux.until { |lines| lines[1].include?(' {555-555}') }
    tmux.send_keys '?'
    tmux.until { |lines| !lines[1].include?(' {555-555}') }
    tmux.send_keys '?'
    tmux.until { |lines| lines[1].include?(' {555-555}') }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2].start_with? '  28/1000' }
    tmux.send_keys 'foobar'
    tmux.until { |lines| !lines[1].include?('{') }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.until { |lines| lines[1].include?(' {1-1}') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?(' {-1}') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?(' {3-1 }') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?(' {4-1  3}') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?(' {5-1  3 4}') }
  end

  def test_preview_hidden
    tmux.send_keys %(seq 1000 | #{FZF} --preview 'echo {{}-{}-\\$FZF_PREVIEW_LINES-\\$FZF_PREVIEW_COLUMNS}' --preview-window down:1:hidden --bind ?:toggle-preview), :Enter
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys '?'
    tmux.until { |lines| lines[-2] =~ / {1-1-1-[0-9]+}/ }
    tmux.send_keys '555'
    tmux.until { |lines| lines[-2] =~ / {555-555-1-[0-9]+}/ }
    tmux.send_keys '?'
    tmux.until { |lines| lines[-1] == '> 555' }
  end

  def test_preview_size_0
    begin
      File.unlink tempname
    rescue
      nil
    end
    tmux.send_keys %(seq 100 | #{FZF} --reverse --preview 'echo {} >> #{tempname}; echo ' --preview-window 0), :Enter
    tmux.until { |lines| lines.item_count == 100 && lines[1] == '  100/100' && lines[2] == '> 1' }
    tmux.until { |_| %w[1] == File.readlines(tempname).map(&:chomp) }
    tmux.send_keys :Down
    tmux.until { |lines| lines[3] == '> 2' }
    tmux.until { |_| %w[1 2] == File.readlines(tempname).map(&:chomp) }
    tmux.send_keys :Down
    tmux.until { |lines| lines[4] == '> 3' }
    tmux.until { |_| %w[1 2 3] == File.readlines(tempname).map(&:chomp) }
  end

  def test_preview_flags
    tmux.send_keys %(seq 10 | sed 's/^/:: /; s/$/  /' |
        #{FZF} --multi --preview 'echo {{2}/{s2}/{+2}/{+s2}/{q}/{n}/{+n}}'), :Enter
    tmux.until { |lines| lines[1].include?('{1/1  /1/1  //0/0}') }
    tmux.send_keys '123'
    tmux.until { |lines| lines[1].include?('{////123//}') }
    tmux.send_keys 'C-u', '1'
    tmux.until { |lines| lines.match_count == 2 }
    tmux.until { |lines| lines[1].include?('{1/1  /1/1  /1/0/0}') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?('{10/10  /1/1  /1/9/0}') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1].include?('{10/10  /1 10/1   10  /1/9/0 9}') }
    tmux.send_keys '2'
    tmux.until { |lines| lines[1].include?('{//1 10/1   10  /12//0 9}') }
    tmux.send_keys '3'
    tmux.until { |lines| lines[1].include?('{//1 10/1   10  /123//0 9}') }
  end

  def test_preview_q_no_match
    tmux.send_keys %(: | #{FZF} --preview 'echo foo {q}'), :Enter
    tmux.until { |lines| lines.match_count == 0 }
    tmux.until { |lines| !lines[1].include?('foo') }
    tmux.send_keys 'bar'
    tmux.until { |lines| lines[1].include?('foo bar') }
    tmux.send_keys 'C-u'
    tmux.until { |lines| !lines[1].include?('foo') }
  end

  def test_preview_q_no_match_with_initial_query
    tmux.send_keys %(: | #{FZF} --preview 'echo foo {q}{q}' --query foo), :Enter
    tmux.until { |lines| lines.match_count == 0 }
    tmux.until { |lines| lines[1].include?('foofoo') }
  end

  def test_no_clear
    tmux.send_keys "seq 10 | fzf --no-clear --inline-info --height 5 > #{tempname}", :Enter
    prompt = '>   < 10/10'
    tmux.until { |lines| lines[-1] == prompt }
    tmux.send_keys :Enter
    tmux.until { |_| %w[1] == File.readlines(tempname).map(&:chomp) }
    tmux.until { |lines| lines[-1] == prompt }
  end

  def test_change_top
    tmux.send_keys %(seq 1000 | #{FZF} --bind change:top), :Enter
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-4] == '> 2' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-3] == '> 1' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-4] == '> 10' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-3] == '> 11' }
    tmux.send_keys :Enter
  end

  def test_accept_non_empty
    tmux.send_keys %(seq 1000 | #{fzf '--print-query --bind enter:accept-non-empty'}), :Enter
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines[-2].include? '0/1000' }
    # fzf doesn't exit since there's no selection
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-2].include? '0/1000' }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines[-2].include? '1000/1000' }
    tmux.send_keys '999'
    tmux.until { |lines| lines[-2].include? '1/1000' }
    tmux.send_keys :Enter
    assert_equal %w[999 999], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_accept_non_empty_with_multi_selection
    tmux.send_keys %(seq 1000 | #{fzf '-m --print-query --bind enter:accept-non-empty'}), :Enter
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-2].include? '1000/1000 (1)' }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines[-2].include? '0/1000' }
    # fzf will exit in this case even though there's no match for the current query
    tmux.send_keys :Enter
    assert_equal %w[foo 1], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_accept_non_empty_with_empty_list
    tmux.send_keys %(: | #{fzf '-q foo --print-query --bind enter:accept-non-empty'}), :Enter
    tmux.until { |lines| lines[-2].strip == '0/0' }
    tmux.send_keys :Enter
    # fzf will exit anyway since input list is empty
    assert_equal %w[foo], readonce.split($INPUT_RECORD_SEPARATOR)
  end

  def test_preview_update_on_select
    tmux.send_keys(%(seq 10 | fzf -m --preview 'echo {+}' --bind a:toggle-all),
                   :Enter)
    tmux.until { |lines| lines.item_count == 10 }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.any? { |line| line.include? '1 2 3 4 5' } }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.none? { |line| line.include? '1 2 3 4 5' } }
  end

  def test_escaped_meta_characters
    input = <<~EOF
      foo^bar
      foo$bar
      foo!bar
      foo'bar
      foo bar
      bar foo
    EOF
    writelines tempname, input.lines.map(&:chomp)

    assert_equal input.lines.count, `#{FZF} -f'foo bar' < #{tempname}`.lines.count
    assert_equal input.lines.count - 1, `#{FZF} -f'^foo bar$' < #{tempname}`.lines.count
    assert_equal ['foo bar'], `#{FZF} -f'foo\\ bar' < #{tempname}`.lines.map(&:chomp)
    assert_equal ['foo bar'], `#{FZF} -f'^foo\\ bar$' < #{tempname}`.lines.map(&:chomp)
    assert_equal input.lines.count - 1, `#{FZF} -f'!^foo\\ bar$' < #{tempname}`.lines.count
  end

  def test_inverse_only_search_should_not_sort_the_result
    # Filter
    assert_equal(%w[aaaaa b ccc],
      `printf '%s\n' aaaaa b ccc BAD | #{FZF} -f '!bad'`.lines.map(&:chomp))

    # Interactive
    tmux.send_keys(%[printf '%s\n' aaaaa b ccc BAD | #{FZF} -q '!bad'], :Enter)
    tmux.until { |lines| lines.item_count == 4 && lines.match_count == 3 }
    tmux.until { |lines| lines[-3] == '> aaaaa' }
    tmux.until { |lines| lines[-4] == '  b' }
    tmux.until { |lines| lines[-5] == '  ccc' }
  end

  def test_preview_correct_tab_width_after_ansi_reset_code
    writelines tempname, ["\x1b[31m+\x1b[m\t\x1b[32mgreen"]
    tmux.send_keys "#{FZF} --preview 'cat #{tempname}'", :Enter
    tmux.until { |lines| lines[1].include?('+       green') }
  end
end

module TestShell
  def setup
    super
  end

  def teardown
    @tmux.kill
  end

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "export #{name}='#{val}'", :Enter
    tmux.prepare
  end

  def unset_var(name)
    tmux.prepare
    tmux.send_keys "unset #{name}", :Enter
    tmux.prepare
  end

  def test_ctrl_t
    set_var 'FZF_CTRL_T_COMMAND', 'seq 100'

    retries do
      tmux.prepare
      tmux.send_keys 'C-t'
      tmux.until { |lines| lines.item_count == 100 }
    end
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| lines.any_include? ' (3)' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines.any_include? '1 2 3' }
    tmux.send_keys 'C-c'
  end

  def test_ctrl_t_unicode
    writelines tempname, ['fzf-unicode 테스트1', 'fzf-unicode 테스트2']
    set_var 'FZF_CTRL_T_COMMAND', "cat #{tempname}"

    retries do
      tmux.prepare
      tmux.send_keys 'echo ', 'C-t'
      tmux.until { |lines| lines.item_count == 2 }
    end
    tmux.send_keys 'fzf-unicode'
    tmux.until { |lines| lines.match_count == 2 }

    tmux.send_keys '1'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines.select_count == 1 }

    tmux.send_keys :BSpace
    tmux.until { |lines| lines.match_count == 2 }

    tmux.send_keys '2'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines.select_count == 2 }

    tmux.send_keys :Enter
    tmux.until { |lines| lines.any_include?(/echo.*fzf-unicode.*1.*fzf-unicode.*2/) }
    tmux.send_keys :Enter
    tmux.until { |lines| lines.any_include?(/^fzf-unicode.*1.*fzf-unicode.*2/) }
  end

  def test_alt_c
    lines = retries do
      tmux.prepare
      tmux.send_keys :Escape, :c
      tmux.until { |lines| lines.match_count.positive? }
    end
    expected = lines.reverse.select { |l| l.start_with? '>' }.first[2..-1]
    tmux.send_keys :Enter
    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| lines[-1].end_with?(expected) }
  end

  def test_alt_c_command
    set_var 'FZF_ALT_C_COMMAND', 'echo /tmp'

    tmux.prepare
    tmux.send_keys 'cd /', :Enter

    retries do
      tmux.prepare
      tmux.send_keys :Escape, :c
      tmux.until { |lines| lines.item_count == 1 }
    end
    tmux.send_keys :Enter

    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| lines[-1].end_with? '/tmp' }
  end

  def test_ctrl_r
    tmux.prepare
    tmux.send_keys 'echo 1st', :Enter; tmux.prepare
    tmux.send_keys 'echo 2nd', :Enter; tmux.prepare
    tmux.send_keys 'echo 3d',  :Enter; tmux.prepare
    tmux.send_keys 'echo 3rd', :Enter; tmux.prepare
    tmux.send_keys 'echo 4th', :Enter
    retries do
      tmux.prepare
      tmux.send_keys 'C-r'
      tmux.until { |lines| lines.match_count.positive? }
    end
    tmux.send_keys 'C-r'
    tmux.send_keys '3d'
    tmux.until { |lines| lines[-3].end_with? 'echo 3rd' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == 'echo 3rd' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '3rd' }
  end

  def retries(times = 3)
    (times - 1).times do
      begin
        return yield
      rescue RuntimeError
      end
    end
    yield
  end
end

module CompletionTest
  def test_file_completion
    FileUtils.mkdir_p '/tmp/fzf-test'
    FileUtils.mkdir_p '/tmp/fzf test'
    (1..100).each { |i| FileUtils.touch "/tmp/fzf-test/#{i}" }
    ['no~such~user', '/tmp/fzf test/foobar', '~/.fzf-home'].each do |f|
      FileUtils.touch File.expand_path(f)
    end
    tmux.prepare
    tmux.send_keys 'cat /tmp/fzf-test/10**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys ' !d'
    tmux.until { |lines| lines.match_count == 2 }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| lines.select_count == 2 }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      lines[-1].include?('/tmp/fzf-test/10') &&
        lines[-1].include?('/tmp/fzf-test/100')
    end

    # ~USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys "cat ~#{ENV['USER']}**", :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys "'.fzf-home"
    tmux.until { |lines| lines.select { |l| l.include? '.fzf-home' }.count > 1 }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      lines[-1].end_with?('.fzf-home')
    end

    # ~INVALID_USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat ~such**', :Tab
    tmux.until(true) { |lines| lines.any_include? 'no~such~user' }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines[-1].end_with?('no~such~user') }

    # /tmp/fzf\ test**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat /tmp/fzf\ test/**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys 'foobar$'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines[-1].end_with?('/tmp/fzf\ test/foobar') }

    # Should include hidden files
    (1..100).each { |i| FileUtils.touch "/tmp/fzf-test/.hidden-#{i}" }
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat /tmp/fzf-test/hidden**', :Tab
    tmux.until(true) { |lines| lines.match_count == 100 && lines.any_include?('/tmp/fzf-test/.hidden-') }
    tmux.send_keys :Enter
  ensure
    ['/tmp/fzf-test', '/tmp/fzf test', '~/.fzf-home', 'no~such~user'].each do |f|
      FileUtils.rm_rf File.expand_path(f)
    end
  end

  def test_file_completion_root
    tmux.send_keys 'ls /**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys :Enter
  end

  def test_dir_completion
    (1..100).each do |idx|
      FileUtils.mkdir_p "/tmp/fzf-test/d#{idx}"
    end
    FileUtils.touch '/tmp/fzf-test/d55/xxx'
    tmux.prepare
    tmux.send_keys 'cd /tmp/fzf-test/**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys :Tab, :Tab # Tab does not work here
    tmux.send_keys 55
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines[-1] == 'cd /tmp/fzf-test/d55/' }
    tmux.send_keys :xx
    tmux.until { |lines| lines[-1] == 'cd /tmp/fzf-test/d55/xx' }

    # Should not match regular files (bash-only)
    if self.class == TestBash
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-1] == 'cd /tmp/fzf-test/d55/xx' }
    end

    # Fail back to plusdirs
    tmux.send_keys :BSpace, :BSpace, :BSpace
    tmux.until { |lines| lines[-1] == 'cd /tmp/fzf-test/d55' }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-1] == 'cd /tmp/fzf-test/d55/' }
  end

  def test_process_completion
    tmux.send_keys 'sleep 12345 &', :Enter
    lines = tmux.until { |lines| lines[-1].start_with? '[1]' }
    pid = lines[-1].split.last
    tmux.prepare
    tmux.send_keys 'C-L'
    tmux.send_keys 'kill ', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys 'sleep12345'
    tmux.until { |lines| lines.any_include? 'sleep 12345' }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines[-1].include? "kill #{pid}" }
  ensure
    if pid
      begin
        Process.kill 'KILL', pid.to_i
      rescue
        nil
      end
    end
  end

  def test_custom_completion
    tmux.send_keys '_fzf_compgen_path() { echo "\$1"; seq 10; }', :Enter
    tmux.prepare
    tmux.send_keys 'ls /tmp/**', :Tab
    tmux.until { |lines| lines.match_count == 11 }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| lines.select_count == 3 }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines[-1] == 'ls /tmp 1 2' }
  end

  def test_unset_completion
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    tmux.prepare

    # Using tmux
    tmux.send_keys 'unset FZFFOOBR**', :Tab
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1].include? 'unset FZFFOOBAR' }
    tmux.send_keys 'C-c'

    # FZF_TMUX=1
    new_shell
    tmux.send_keys 'unset FZFFOOBR**', :Tab, pane: 0
    tmux.until(false, 1) { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1].include? 'unset FZFFOOBAR' }
  end

  def test_file_completion_unicode
    FileUtils.mkdir_p '/tmp/fzf-test'
    tmux.paste 'cd /tmp/fzf-test; echo -n test3 > "fzf-unicode 테스트1"; echo -n test4 > "fzf-unicode 테스트2"'
    tmux.prepare
    tmux.send_keys 'cat fzf-unicode**', :Tab
    tmux.until { |lines| lines.match_count == 2 }

    tmux.send_keys '1'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines.select_count == 1 }

    tmux.send_keys :BSpace
    tmux.until { |lines| lines.match_count == 2 }

    tmux.send_keys '2'
    tmux.until { |lines| lines.select_count == 1 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines.select_count == 2 }

    tmux.send_keys :Enter
    tmux.until(true) { |lines| lines.any_include? 'cat' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1].include? 'test3test4' }
  end
end

class TestBash < TestBase
  include TestShell
  include CompletionTest

  def new_shell
    tmux.prepare
    tmux.send_keys "FZF_TMUX=1 #{Shell.bash}", :Enter
    tmux.prepare
  end

  def setup
    super
    @tmux = Tmux.new :bash
  end

  def test_dynamic_completion_loader
    tmux.paste 'touch /tmp/foo; _fzf_completion_loader=1'
    tmux.paste '_completion_loader() { complete -o default fake; }'
    tmux.paste 'complete -F _fzf_path_completion -o default -o bashdefault fake'
    tmux.send_keys 'fake /tmp/foo**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
    tmux.send_keys 'C-c'

    tmux.prepare
    tmux.send_keys 'fake /tmp/foo'
    tmux.send_keys :Tab , 'C-u'

    tmux.prepare
    tmux.send_keys 'fake /tmp/foo**', :Tab
    tmux.until { |lines| lines.match_count.positive? }
  end
end

class TestZsh < TestBase
  include TestShell
  include CompletionTest

  def new_shell
    tmux.send_keys "FZF_TMUX=1 #{Shell.zsh}", :Enter
    tmux.prepare
  end

  def setup
    super
    @tmux = Tmux.new :zsh
  end
end

class TestFish < TestBase
  include TestShell

  def new_shell
    tmux.send_keys 'env FZF_TMUX=1 fish', :Enter
    tmux.send_keys 'function fish_prompt; end; clear', :Enter
    tmux.until(&:empty?)
  end

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "set -g #{name} '#{val}'", :Enter
    tmux.prepare
  end

  def setup
    super
    @tmux = Tmux.new :fish
  end
end
