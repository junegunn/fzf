#!/usr/bin/env ruby
# frozen_string_literal: true

require 'minitest/autorun'
require 'fileutils'
require 'English'
require 'shellwords'
require 'erb'
require 'tempfile'

TEMPLATE = DATA.read
UNSETS = %w[
  FZF_DEFAULT_COMMAND FZF_DEFAULT_OPTS
  FZF_CTRL_T_COMMAND FZF_CTRL_T_OPTS
  FZF_ALT_C_COMMAND
  FZF_ALT_C_OPTS FZF_CTRL_R_OPTS
  fish_history
].freeze
DEFAULT_TIMEOUT = 20

FILE = File.expand_path(__FILE__)
BASE = File.expand_path('..', __dir__)
FZF = "FZF_DEFAULT_OPTS= FZF_DEFAULT_COMMAND= #{BASE}/bin/fzf"

# For pane_dead_status
system('tmux', 'set-window-option', '-g', 'remain-on-exit', 'on')

def wait
  since = Time.now
  while Time.now - since < DEFAULT_TIMEOUT
    return if yield

    sleep(0.05)
  end
  raise('timeout')
end

class Tmux
  attr_reader :win

  def initialize(shell_command)
    @win = go(%W[new-window -d -P -F #I #{shell_command}]).first
    go(%W[set-window-option -t :#{win} pane-base-index 0])
  end

  def focus
    go(%W[select-window -t :#{win}])
  end

  def send_keys(*args)
    go(%W[send-keys -t :#{win}] + args.map(&:to_s))
  end

  def paste(str)
    system('tmux', 'setb', str, ';', 'pasteb', '-t', ":#{win}", ';', 'send-keys', '-t', ":#{win}", 'Enter')
  end

  def capture
    go(%W[capture-pane -p -J -S - -t :#{win}]).map(&:rstrip).drop_while(&:empty?).reverse.drop_while(&:empty?).reverse
  end

  def until(refresh = false)
    lines = nil
    begin
      wait do
        lines = capture
        class << lines
          def counts
            lazy
              .map { |l| l.scan(%r{^. ([0-9]+)\/([0-9]+)( \(([0-9]+)\))?}) }
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
            find { |line| line.send(method, val) }
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
    self.until { |lines| lines[-1] == '$' }
  end

  private

  def go(args)
    IO.popen(%w[tmux] + args) { |io| io.readlines(chomp: true) }
  end
end

class TestBase < Minitest::Test
  TEMPNAME = 'output'

  def setup
    super
    Dir.chdir(Dir.mktmpdir)
  end

  def teardown
    super
    FileUtils.remove_entry(Dir.pwd) if Dir.pwd.start_with?("#{Dir.tmpdir}/")
    system('tmux', 'kill-window', '-a')
  end

  def tempname
    @temp_suffix ||= 0
    File.expand_path([TEMPNAME,
                      caller_locations.map(&:label).find { |l| l.start_with?('test_') },
                      @temp_suffix].join('-'))
  end

  def readonce
    wait { File.exist?(tempname) }
    File.read(tempname)
  ensure
    File.unlink(tempname) while File.exist?(tempname)
    @temp_suffix += 1
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
    "#{FZF} #{opts.join(' ')}"
  end
end

class TestGoFZF < TestBase
  def test_vanilla
    tmux = Tmux.new("seq 1 100000 | #{fzf}")
    tmux.until do |lines|
      lines[-4..-1] == [
        '  2',
        '> 1',
        '  100000/100000',
        '>'
      ]
    end

    # Testing basic key bindings
    tmux.send_keys '99', 'C-a', '1', 'C-f', '3', 'C-b', 'C-h', 'C-u', 'C-e', 'C-y', 'C-k', 'Tab', 'BTab'
    tmux.until do |lines|
      lines[-4..-1] == [
        '> 3910',
        '  391',
        '  856/100000',
        '> 391'
      ]
    end

    tmux.send_keys :Enter
    assert_equal '3910', readonce.chomp
  end

  def test_fzf_default_command
    tmux = Tmux.new(fzf.sub('FZF_DEFAULT_COMMAND=', "FZF_DEFAULT_COMMAND='echo hello'"))
    tmux.until { |lines| lines[-3] == '> hello' }

    tmux.send_keys :Enter
    assert_equal 'hello', readonce.chomp
  end

  def test_fzf_default_command_failure
    tmux = Tmux.new(fzf.sub('FZF_DEFAULT_COMMAND=', 'FZF_DEFAULT_COMMAND=false'))
    tmux.until { |lines| lines[-2] == '  [Command failed: false]' }
  end

  def test_key_bindings
    tmux = Tmux.new("#{FZF} -q 'foo bar foo-bar'")
    tmux.until { |lines| lines.last == '> foo bar foo-bar' }

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
    tmux.until { |lines| lines[-1]&.start_with?('Pane is dead') }
  end

  def test_file_word
    tmux = Tmux.new("#{FZF} -q '--/foo bar/foo-bar/baz' --filepath-word")
    tmux.until { |lines| lines.last == '> --/foo bar/foo-bar/baz' }

    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :d
    tmux.send_keys :Escape, :f
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| lines.last == '> --///baz' }
  end

  def test_multi_order
    tmux = Tmux.new("seq 1 10 | #{fzf(:multi)}")
    tmux.until { |lines| lines.last == '>' }

    tmux.send_keys :Tab, :Up, :Up, :Tab, :Tab, :Tab, # 3, 2
                   'C-K', 'C-K', 'C-K', 'C-K', :BTab, :BTab, # 5, 6
                   :PgUp, 'C-J', :Down, :Tab, :Tab # 8, 7
    tmux.until { |lines| lines[-2] == '  10/10 (6)' }
    tmux.send_keys 'C-M'
    assert_equal %w[3 2 5 6 8 7], readonce.lines(chomp: true)
  end

  def test_multi_max
    tmux = Tmux.new("seq 1 10 | #{FZF} -m 3 --bind A:select-all,T:toggle-all --preview 'echo [{+}]/{}'")

    tmux.until { |lines| lines.item_count == 10 }

    tmux.send_keys '1'
    tmux.until do |lines|
      lines[1]&.include?(' [1]/1 ') && lines[-2]&.start_with?('  2/10 ')
    end

    tmux.send_keys 'A'
    tmux.until do |lines|
      lines[1]&.include?(' [1 10]/1 ') && lines[-2]&.start_with?('  2/10 (2/3) ')
    end

    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2]&.start_with?('  10/10 (2/3) ') }

    tmux.send_keys 'T'
    tmux.until do |lines|
      lines[1]&.include?(' [2 3 4]/1 ') && lines[-2]&.start_with?('  10/10 (3/3) ')
    end

    %w[T A].each do |key|
      tmux.send_keys key
      tmux.until do |lines|
        lines[1]&.include?(' [1 5 6]/1 ') && lines[-2]&.start_with?('  10/10 (3/3) ')
      end
    end

    tmux.send_keys :BTab
    tmux.until do |lines|
      lines[1]&.include?(' [5 6]/2 ') && lines[-2]&.start_with?('  10/10 (2/3) ')
    end

    [:BTab, :BTab, 'A'].each do |key|
      tmux.send_keys key
      tmux.until do |lines|
        lines[1]&.include?(' [5 6 2]/3 ') && lines[-2]&.start_with?('  10/10 (3/3) ')
      end
    end

    tmux.send_keys '2'
    tmux.until { |lines| lines[-2]&.start_with?('  1/10 (3/3) ') }

    tmux.send_keys 'T'
    tmux.until do |lines|
      lines[1]&.include?(' [5 6]/2 ') && lines[-2]&.start_with?('  1/10 (2/3) ')
    end

    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2]&.start_with?('  10/10 (2/3) ') }

    tmux.send_keys 'A'
    tmux.until do |lines|
      lines[1]&.include?(' [5 6 1]/1 ') && lines[-2]&.start_with?('  10/10 (3/3) ')
    end
  end

  def test_with_nth
    [true, false].each do |multi|
      tmux = Tmux.new("(echo '  1st 2nd 3rd/';
                       echo '  first second third/') |
                       #{fzf(multi && :multi, :x, :nth, 2, :with_nth, '2,-1,1')}")
      # Transformed list
      tmux.until do |lines|
        lines[-4..-2] == [
          '  second third/first',
          '> 2nd 3rd/1st',
          '  2/2'
        ]
      end

      # However, the output must not be transformed
      if multi
        tmux.send_keys :BTab, :BTab
        tmux.until { |lines| lines[-2] == '  2/2 (2)' }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/', '  first second third/'], readonce.lines(chomp: true)
      else
        tmux.send_keys '^', '3'
        tmux.until { |lines| lines[-2] == '  1/2' }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/'], readonce.lines(chomp: true)
      end
    end
  end

  def test_scroll
    [true, false].each do |rev|
      tmux = Tmux.new("seq 1 100 | #{fzf(rev && :reverse)}")
      tmux.until { |lines| lines.include?('  100/100') }
      tmux.send_keys(*Array.new(110) { rev ? :Down : :Up })
      tmux.until { |lines| lines.include?('> 100') }
      tmux.send_keys :Enter
      assert_equal '100', readonce.chomp
    end
  end

  def test_select_1
    Tmux.new("seq 1 100 | #{fzf(:with_nth, '..,..', :print_query, :q, 5555, :'1')}")
    assert_equal %w[5555 55], readonce.lines(chomp: true)
  end

  def test_exit_0
    Tmux.new("seq 1 100 | #{fzf(:with_nth, '..,..', :print_query, :q, 555_555, :'0')}")
    assert_equal %w[555555], readonce.lines(chomp: true)
  end

  def test_select_1_exit_0_fail
    [:'0', :'1', %i[1 0]].each do |opt|
      tmux = Tmux.new("seq 1 100 | #{fzf(:print_query, :multi, :q, 5, *opt)}")
      tmux.until { |lines| lines.last == '> 5' }
      tmux.send_keys :BTab, :BTab, :BTab
      tmux.until { |lines| lines[-2] == '  19/100 (3)' }
      tmux.send_keys :Enter
      assert_equal %w[5 5 50 51], readonce.lines(chomp: true)
    end
  end

  def test_query_unicode
    tmux = Tmux.new("(echo abc; echo $'\\352\\260\\200\\353\\202\\230\\353\\213\\244') | #{fzf(:query, "$'\\352\\260\\200\\353\\213\\244'")}")
    tmux.until { |lines| lines[-2] == '  1/2' }
    tmux.send_keys :Enter
    assert_equal %w[가나다], readonce.lines(chomp: true)
  end

  def test_sync
    tmux = Tmux.new("seq 1 100 | #{fzf!(:multi)} | awk '{print $1 $1}' | #{fzf(:sync)}")
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 9
    tmux.until { |lines| lines[-2] == '  19/100' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  19/100 (3)' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 'C-K', :Enter
    assert_equal %w[9090], readonce.lines(chomp: true)
  end

  def test_tac
    tmux = Tmux.new("seq 1 1000 | #{fzf(:tac, :multi)}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  1000/1000 (3)' }
    tmux.send_keys :Enter
    assert_equal %w[1000 999 998], readonce.lines(chomp: true)
  end

  def test_tac_sort
    tmux = Tmux.new("seq 1 1000 | #{fzf(:tac, :multi)}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys '99'
    tmux.until { |lines| lines[-2] == '  28/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  28/1000 (3)' }
    tmux.send_keys :Enter
    assert_equal %w[99 999 998], readonce.lines(chomp: true)
  end

  def test_tac_nosort
    tmux = Tmux.new("seq 1 1000 | #{fzf(:tac, :no_sort, :multi)}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys '00'
    tmux.until { |lines| lines[-2] == '  10/1000' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  10/1000 (3)' }
    tmux.send_keys :Enter
    assert_equal %w[1000 900 800], readonce.lines(chomp: true)
  end

  def test_expect
    test = lambda do |key, feed, expected = key|
      tmux = Tmux.new("seq 1 100 | #{fzf(:expect, key)}")
      tmux.until { |lines| lines[-2] == '  100/100' }
      tmux.send_keys '55'
      tmux.until { |lines| lines[-2] == '  1/100' }
      tmux.send_keys(*feed)
      assert_equal [expected, '55'], readonce.lines(chomp: true)
    end
    test.call('ctrl-t', 'C-T')
    test.call('ctrl-t', 'Enter', '')
    test.call('alt-c', %i[Escape c])
    test.call('f1', 'f1')
    test.call('f2', 'f2')
    test.call('f3', 'f3')
    test.call('f2,f4', 'f2', 'f2')
    test.call('f2,f4', 'f4', 'f4')
    test.call('alt-/', %i[Escape /])
    %w[f5 f6 f7 f8 f9 f10].each do |key|
      test.call('f5,f6,f7,f8,f9,f10', key, key)
    end
    test.call('@', '@')
  end

  def test_expect_print_query
    tmux = Tmux.new("seq 1 100 | #{fzf('--expect=alt-z', :print_query)}")
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys '55'
    tmux.until { |lines| lines[-2] == '  1/100' }
    tmux.send_keys :Escape, :z
    assert_equal %w[55 alt-z 55], readonce.lines(chomp: true)
  end

  def test_expect_printable_character_print_query
    tmux = Tmux.new("seq 1 100 | #{fzf('--expect=z --print-query')}")
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys '55'
    tmux.until { |lines| lines[-2] == '  1/100' }
    tmux.send_keys 'z'
    assert_equal %w[55 z 55], readonce.lines(chomp: true)
  end

  def test_expect_print_query_select_1
    Tmux.new("seq 1 100 | #{fzf('-q55 -1 --expect=alt-z --print-query')}")
    assert_equal ['55', '', '55'], readonce.lines(chomp: true)
  end

  def test_toggle_sort
    ['--toggle-sort=ctrl-r', '--bind=ctrl-r:toggle-sort'].each do |opt|
      tmux = Tmux.new("seq 1 111 | #{fzf("-m +s --tac #{opt} -q11")}")
      tmux.until { |lines| lines[-3] == '> 111' }
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-2] == '  4/111 -S (1)' }
      tmux.send_keys 'C-R'
      tmux.until { |lines| lines[-3] == '> 11' }
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-2] == '  4/111 +S (2)' }
      tmux.send_keys :Enter
      assert_equal %w[111 11], readonce.lines(chomp: true)
    end
  end

  def test_unicode_case
    File.open(tempname, 'w') { |f| f.puts %w[строКА1 СТРОКА2 строка3 Строка4] }
    assert_equal %w[СТРОКА2 Строка4], `#{FZF} -fС < #{tempname}`.lines(chomp: true)
    assert_equal %w[строКА1 СТРОКА2 строка3 Строка4], `#{FZF} -fс < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak
    input = %w[
      --foobar--------
      -----foobar---
      ----foobar--
      -------foobar-
    ]
    File.open(tempname, 'w') { |f| f.puts input }

    assert_equal input, `#{FZF} -ffoobar --tiebreak=index < #{tempname}`.lines(chomp: true)

    by_length = %w[
      ----foobar--
      -----foobar---
      -------foobar-
      --foobar--------
    ]
    assert_equal by_length, `#{FZF} -ffoobar < #{tempname}`.lines(chomp: true)
    assert_equal by_length, `#{FZF} -ffoobar --tiebreak=length < #{tempname}`.lines(chomp: true)

    by_begin = %w[
      --foobar--------
      ----foobar--
      -----foobar---
      -------foobar-
    ]
    assert_equal by_begin, `#{FZF} -ffoobar --tiebreak=begin < #{tempname}`.lines(chomp: true)
    assert_equal by_begin, `#{FZF} -f"!z foobar" -x --tiebreak begin < #{tempname}`.lines(chomp: true)

    assert_equal %w[
      -------foobar-
      ----foobar--
      -----foobar---
      --foobar--------
    ], `#{FZF} -ffoobar --tiebreak end < #{tempname}`.lines(chomp: true)

    assert_equal input, `#{FZF} -f"!z" -x --tiebreak end < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_index_begin
    File.open(tempname, 'w') do |f|
      f.puts [
        'xoxxxxxoxx',
        'xoxxxxxox',
        'xxoxxxoxx',
        'xxxoxoxxx',
        'xxxxoxox',
        '  xxoxoxxx'
      ]
    end

    assert_equal [
      'xxxxoxox',
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxoxxxoxx',
      'xoxxxxxox',
      'xoxxxxxoxx'
    ], `#{FZF} -foo < #{tempname}`.lines(chomp: true)

    assert_equal [
      'xxxoxoxxx',
      'xxxxoxox',
      '  xxoxoxxx',
      'xxoxxxoxx',
      'xoxxxxxoxx',
      'xoxxxxxox'
    ], `#{FZF} -foo --tiebreak=index < #{tempname}`.lines(chomp: true)

    # Note that --tiebreak=begin is now based on the first occurrence of the
    # first character on the pattern
    assert_equal [
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxxxoxox',
      'xxoxxxoxx',
      'xoxxxxxoxx',
      'xoxxxxxox'
    ], `#{FZF} -foo --tiebreak=begin < #{tempname}`.lines(chomp: true)

    assert_equal [
      '  xxoxoxxx',
      'xxxoxoxxx',
      'xxxxoxox',
      'xxoxxxoxx',
      'xoxxxxxox',
      'xoxxxxxoxx'
    ], `#{FZF} -foo --tiebreak=begin,length < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_begin_algo_v2
    File.open(tempname, 'w') do |f|
      f.puts [
        'baz foo bar',
        'foo bar baz'
      ]
    end
    assert_equal [
      'foo bar baz',
      'baz foo bar'
    ], `#{FZF} -fbar --tiebreak=begin --algo=v2 < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_end
    File.open(tempname, 'w') do |f|
      f.puts [
        'xoxxxxxxxx',
        'xxoxxxxxxx',
        'xxxoxxxxxx',
        'xxxxoxxxx',
        'xxxxxoxxx',
        '  xxxxoxxx'
      ]
    end

    assert_equal [
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxxxoxxx',
      'xoxxxxxxxx',
      'xxoxxxxxxx',
      'xxxoxxxxxx'
    ], `#{FZF} -fo < #{tempname}`.lines(chomp: true)

    assert_equal [
      'xxxxxoxxx',
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxoxxxxxx',
      'xxoxxxxxxx',
      'xoxxxxxxxx'
    ], `#{FZF} -fo --tiebreak=end < #{tempname}`.lines(chomp: true)

    assert_equal [
      'xxxxxoxxx',
      '  xxxxoxxx',
      'xxxxoxxxx',
      'xxxoxxxxxx',
      'xxoxxxxxxx',
      'xoxxxxxxxx'
    ], `#{FZF} -fo --tiebreak=end,length,begin < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_length_with_nth
    input = %w[
      1:hell
      123:hello
      12345:he
      1234567:h
    ]
    File.open(tempname, 'w') { |f| f.puts input }

    output = %w[
      1:hell
      12345:he
      123:hello
      1234567:h
    ]
    assert_equal output, `#{FZF} -fh < #{tempname}`.lines(chomp: true)

    # Since 0.16.8, --nth doesn't affect --tiebreak
    assert_equal output, `#{FZF} -fh -n2 -d: < #{tempname}`.lines(chomp: true)
  end

  def test_invalid_cache
    tmux = Tmux.new("(echo d; echo D; echo x) | #{fzf('-q d')}")
    tmux.until { |lines| lines[-2] == '  2/3' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2] == '  3/3' }
    tmux.send_keys :D
    tmux.until { |lines| lines[-2] == '  1/3' }
  end

  def test_invalid_cache_query_type
    command = %[(echo 'foo$bar'; echo 'barfoo'; echo 'foo^bar'; echo "foo'1-2"; seq 100) | #{fzf}]

    # Suffix match
    tmux = Tmux.new(command)
    tmux.until { |lines| lines.match_count == 104 }
    tmux.send_keys 'foo$'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'bar'
    tmux.until { |lines| lines.match_count == 1 }

    # Prefix match
    tmux = Tmux.new(command)
    tmux.until { |lines| lines.match_count == 104 }
    tmux.send_keys '^bar'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'C-a', 'foo'
    tmux.until { |lines| lines.match_count == 1 }

    # Exact match
    tmux = Tmux.new(command)
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
    tmux = Tmux.new("seq 1 1000 | #{fzf('-m --bind=ctrl-j:accept,u:up,T:toggle-up,t:toggle')}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 'uuu', 'TTT', 'tt', 'uu', 'ttt', 'C-j'
    assert_equal %w[4 5 6 9], readonce.lines(chomp: true)
  end

  def test_bind_print_query
    tmux = Tmux.new("seq 1 1000 | #{fzf('-m --bind=ctrl-j:print-query')}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 'print-my-query', 'C-j'
    assert_equal %w[print-my-query], readonce.lines(chomp: true)
  end

  def test_bind_replace_query
    tmux = Tmux.new("seq 1 1000 | #{fzf('--print-query --bind=ctrl-j:replace-query')}")
    tmux.send_keys '1'
    tmux.until { |lines| lines[-2] == '  272/1000' }
    tmux.send_keys 'C-k', 'C-j'
    tmux.until { |lines| lines[-2] == '  29/1000' }
    tmux.until { |lines| lines[-1] == '> 10' }
  end

  def test_long_line
    data = '.' * 256 * 1024
    File.open(tempname, 'w') do |f|
      f << data
    end
    assert_equal data, `#{FZF} -f . < #{tempname}`.chomp
  end

  def test_read0
    lines = `find .`.lines(chomp: true)
    assert_equal lines.last, `find . | #{FZF} -e -f "^#{lines.last}$"`.chomp
    assert_equal \
      lines.last,
      `find . -print0 | #{FZF} --read0 -e -f "^#{lines.last}$"`.chomp
  end

  def test_select_all_deselect_all_toggle_all
    tmux = Tmux.new("seq 100 | #{fzf('--bind ctrl-a:select-all,ctrl-d:deselect-all,ctrl-t:toggle-all --multi')}")
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  100/100 (3)' }
    tmux.send_keys 'C-t'
    tmux.until { |lines| lines[-2] == '  100/100 (97)' }
    tmux.send_keys 'C-a'
    tmux.until { |lines| lines[-2] == '  100/100 (100)' }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| lines[-2] == '  100/100 (98)' }
    tmux.send_keys '100'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys 'C-d'
    tmux.until { |lines| lines[-2] == '  1/100 (97)' }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines.match_count == 100 }
    tmux.send_keys 'C-d'
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys :BTab, :BTab
    tmux.until { |lines| lines[-2] == '  100/100 (2)' }
    tmux.send_keys 0
    tmux.until { |lines| lines[-2] == '  10/100 (2)' }
    tmux.send_keys 'C-a'
    tmux.until { |lines| lines[-2] == '  10/100 (12)' }
    tmux.send_keys :Enter
    assert_equal %w[1 2 10 20 30 40 50 60 70 80 90 100],
                 readonce.lines(chomp: true)
  end

  def test_history
    # History with limited number of entries
    opts = '--history=fzf-test-history --history-size=4'
    input = %w[00 11 22 33 44]
    input.each do |keys|
      tmux = Tmux.new("seq 100 | #{fzf(opts)}")
      tmux.until { |lines| lines[-2] == '  100/100' }
      tmux.send_keys keys
      tmux.until { |lines| lines[-2] == '  1/100' }
      tmux.send_keys :Enter
    end
    wait { File.exist?('fzf-test-history') && File.readlines('fzf-test-history', chomp: true) == input[1..-1] }

    # Update history entries (not changed on disk)
    tmux = Tmux.new("seq 100 | #{fzf(opts)}")
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1] == '> 44' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1] == '> 33' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-1] == '> 3' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-1] == '> 31' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| lines[-1] == '> 22' }
    tmux.send_keys 'C-n'
    tmux.until { |lines| lines[-1] == '> 31' }
    tmux.send_keys 0
    tmux.until { |lines| lines[-1] == '> 310' }
    tmux.send_keys :Enter
    wait { File.exist?('fzf-test-history') && File.readlines('fzf-test-history', chomp: true) == %w[22 33 44 310] }

    # Respect --bind option
    tmux = Tmux.new("seq 100 | #{fzf(opts + ' --bind ctrl-p:next-history,ctrl-n:previous-history')}")
    tmux.until { |lines| lines[-2] == '  100/100' }
    tmux.send_keys 'C-n', 'C-n', 'C-n', 'C-n', 'C-p'
    tmux.until { |lines| lines[-1] == '> 33' }
  end

  def test_execute
    opts = "--bind 'alt-a:execute(echo /{}/ >> output),alt-b:execute[echo /{}{}/ >> output],C:execute:echo /{}{}{}/ >> output'"
    File.open(tempname, 'w') { |f| f.puts %w[foo'bar foo"bar foo$bar] }
    tmux = Tmux.new("cat #{tempname} | #{fzf(opts)}")
    tmux.until { |lines| lines[-2] == '  3/3' }
    tmux.send_keys :Escape, :a
    tmux.send_keys :Escape, :a
    tmux.send_keys :Up
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Up
    tmux.send_keys :C
    tmux.send_keys 'barfoo'
    tmux.until { |lines| lines[-2] == '  0/3' }
    tmux.send_keys :Escape, :a
    tmux.send_keys :Escape, :b
    wait do
      File.exist?('output') && File.readlines('output', chomp: true) == %w[/foo'bar/ /foo'bar/
                                                                           /foo"barfoo"bar/ /foo"barfoo"bar/
                                                                           /foo$barfoo$barfoo$bar/]
    end
  end

  def test_execute_multi
    opts = "--multi --bind 'alt-a:execute-multi(echo {}/{+} >> output)'"
    File.open(tempname, 'w') { |f| f.puts %w[foo'bar foo"bar foo$bar foobar] }
    tmux = Tmux.new("cat #{tempname} | #{fzf(opts)}")
    tmux.until { |lines| lines[-2] == '  4/4' }
    tmux.send_keys :Escape, :a
    tmux.until { |lines| lines[-2] == '  4/4' }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.send_keys :Escape, :a
    tmux.until { |lines| lines[-2] == '  4/4 (3)' }
    tmux.send_keys :Tab, :Tab
    tmux.send_keys :Escape, :a
    wait do
      File.exist?('output') && File.readlines('output', chomp: true) == [%(foo'bar/foo'bar),
                                                                         %(foo'bar foo"bar foo$bar/foo'bar foo"bar foo$bar),
                                                                         %(foo'bar foo"bar foobar/foo'bar foo"bar foobar)]
    end
  end

  def test_execute_plus_flag
    File.open(tempname, 'w') { |f| f.puts ['foo bar', '123 456'] }

    tmux = Tmux.new("cat #{tempname} | #{FZF} --multi --bind 'x:execute-silent(echo {+}/{}/{+2}/{2} >> output)'")

    tmux.until { |lines| lines[-2] == '  2/2' }
    tmux.send_keys 'xy'
    tmux.until { |lines| lines[-2] == '  0/2' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2] == '  2/2' }

    tmux.send_keys :Up
    tmux.send_keys :Tab
    tmux.send_keys 'xy'
    tmux.until { |lines| lines[-2] == '  0/2 (1)' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2] == '  2/2 (1)' }

    tmux.send_keys :Tab
    tmux.send_keys 'xy'
    tmux.until { |lines| lines[-2] == '  0/2 (2)' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2] == '  2/2 (2)' }

    wait do
      File.exist?('output') && File.readlines('output', chomp: true) == [
        %(foo bar/foo bar/bar/bar),
        %(123 456/foo bar/456/bar),
        %(123 456 foo bar/foo bar/456 bar/bar)
      ]
    end
  end

  def test_execute_shell
    # Custom script to use as $SHELL
    File.open(tempname, 'w') do |f|
      f.puts \
        ['#!/usr/bin/env bash', 'echo $1 / $2 > output']
    end
    system("chmod +x #{tempname}")

    tmux = Tmux.new("echo foo | SHELL=#{tempname} #{FZF} --bind 'enter:execute:{}bar'")
    tmux.until { |lines| lines[-2] == '  1/1' }
    tmux.send_keys :Enter
    wait { File.exist?('output') && File.readlines('output', chomp: true) == ["-c / 'foo'bar"] }
  end

  def test_cycle
    tmux = Tmux.new("seq 8 | #{fzf(:cycle)}")
    tmux.until { |lines| lines[-2] == '  8/8' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-10] == '> 8' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-9] == '> 7' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-10] == '> 8' }
    tmux.send_keys :PgUp
    tmux.until { |lines| lines[-10] == '> 8' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-3] == '> 1' }
    tmux.send_keys :PgDn
    tmux.until { |lines| lines[-3] == '> 1' }
    tmux.send_keys :Down
    tmux.until { |lines| lines[-10] == '> 8' }
  end

  def test_header_lines
    tmux = Tmux.new("seq 100 | #{fzf('--header-lines=10 -q 5')}")
    2.times do
      tmux.until do |lines|
        lines[-2] == '  18/90' &&
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
    tmux = Tmux.new("seq 100 | #{fzf('--header-lines=10 -q 5 --reverse')}")
    2.times do
      tmux.until do |lines|
        lines[1] == '  18/90' &&
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
    tmux = Tmux.new("seq 100 | #{fzf('--header-lines=10 -q 5 --layout=reverse-list')}")
    2.times do
      tmux.until do |lines|
        lines[0]    == '> 50' &&
          lines[-4] == '  2' &&
          lines[-3] == '  1' &&
          lines[-2] == '  18/90'
      end
      tmux.send_keys :Up
    end
    tmux.send_keys :Enter
    assert_equal '50', readonce.chomp
  end

  def test_header_lines_overflow
    tmux = Tmux.new("seq 100 | #{fzf('--header-lines=200')}")
    tmux.until do |lines|
      lines[-2] == '  0/0' &&
        lines[-3] == '  1'
    end
    tmux.send_keys :Enter
    assert_equal '', readonce.chomp
  end

  def test_header_lines_with_nth
    tmux = Tmux.new("seq 100 | #{fzf('--header-lines 5 --with-nth 1,1,1,1,1')}")
    tmux.until do |lines|
      lines[-2] == '  95/95' &&
        lines[-3] == '  11111' &&
        lines[-7] == '  55555' &&
        lines[-8] == '> 66666'
    end
    tmux.send_keys :Enter
    assert_equal '6', readonce.chomp
  end

  def test_header
    tmux = Tmux.new("seq 100 | #{fzf("--header \"$(head -5 #{FILE})\"")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[-2] == '  100/100' &&
        lines[-7..-3] == header.map { |line| "  #{line}".rstrip } &&
        lines[-8] == '> 1'
    end
  end

  def test_header_reverse
    tmux = Tmux.new("seq 100 | #{fzf("--header \"$(head -5 #{FILE})\" --reverse")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[1] == '  100/100' &&
        lines[2..6] == header.map { |line| "  #{line}".rstrip } &&
        lines[7] == '> 1'
    end
  end

  def test_header_reverse_list
    tmux = Tmux.new("seq 100 | #{fzf("--header \"$(head -5 #{FILE})\" --layout=reverse-list")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[-2] == '  100/100' &&
        lines[-7..-3] == header.map { |line| "  #{line}".rstrip } &&
        lines[0] == '> 1'
    end
  end

  def test_header_and_header_lines
    tmux = Tmux.new("seq 100 | #{fzf("--header-lines 10 --header \"$(head -5 #{FILE})\"")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[-2] == '  90/90' &&
        lines[-7...-2] == header.map { |line| "  #{line}".rstrip } &&
        lines[-17...-7] == ('  1'..'  10').to_a.reverse
    end
  end

  def test_header_and_header_lines_reverse
    tmux = Tmux.new("seq 100 | #{fzf("--reverse --header-lines 10 --header \"$(head -5 #{FILE})\"")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[1] == '  90/90' &&
        lines[2...7] == header.map { |line| "  #{line}".rstrip } &&
        lines[7...17] == ('  1'..'  10').to_a
    end
  end

  def test_header_and_header_lines_reverse_list
    tmux = Tmux.new("seq 100 | #{fzf("--layout=reverse-list --header-lines 10 --header \"$(head -5 #{FILE})\"")}")
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      lines[-2] == '  90/90' &&
        lines[-7...-2] == header.map { |line| "  #{line}".rstrip } &&
        lines[-17...-7] == ('  1'..'  10').to_a.reverse
    end
  end

  def test_cancel
    tmux = Tmux.new("seq 10 | #{fzf('--bind 2:cancel')}")
    tmux.until { |lines| lines[-2] == '  10/10' }
    tmux.send_keys '123'
    tmux.until { |lines| lines[-1] == '> 3' && lines[-2] == '  1/10' }
    tmux.send_keys 'C-y', 'C-y'
    tmux.until { |lines| lines[-1] == '> 311' }
    tmux.send_keys 2
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 2
    tmux.until { |lines| lines[-1]&.start_with?('Pane is dead') }
  end

  def test_margin
    tmux = Tmux.new("yes | head -1000 | #{fzf('--margin 5,3')}")
    tmux.until { |lines| lines[0] == '     y' }
  end

  def test_margin_reverse
    tmux = Tmux.new("seq 1000 | #{fzf('--margin 7,5 --reverse')}")
    tmux.until { |lines| lines[1] == '       1000/1000' }
  end

  def test_margin_reverse_list
    tmux = Tmux.new("yes | head -1000 | #{fzf('--margin 5,3 --layout=reverse-list')}")
    tmux.until { |lines| lines[0] == '   > y' }
  end

  def test_tabstop
    File.open(tempname, 'w') { |f| f.puts %W[f\too\tba\tr\tbaz\tbarfooq\tux] }
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
      tmux = Tmux.new(%(cat #{tempname} | #{FZF} --tabstop=#{ts}))
      tmux.until do |lines|
        lines[-3] == exp
      end
    end
  end

  def test_with_nth_basic
    File.open(tempname, 'w') { |f| f.puts ['hello world ', 'byebye'] }
    assert_equal \
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 < #{tempname}`.chomp
  end

  def test_with_nth_ansi
    File.open(tempname, 'w') { |f| f.puts ["\x1b[33mhello \x1b[34;1mworld\x1b[m ", 'byebye'] }
    assert_equal \
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 --ansi < #{tempname}`.chomp
  end

  def test_with_nth_no_ansi
    src = "\x1b[33mhello \x1b[34;1mworld\x1b[m "
    File.open(tempname, 'w') { |f| f.puts [src, 'byebye'] }
    assert_equal \
      src,
      `#{FZF} -fhehe -x -n 2.. --with-nth 2,1,1 --no-ansi < #{tempname}`.chomp
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
      tmux = Tmux.new("seq 100 | #{FZF} -q #{query} > /dev/null; echo --$?--")
      tmux.until { |lines| lines[-2] =~ %r{ [10]/100} }
      tmux.send_keys :Enter
      tmux.until { |lines| lines[0] == "--#{status}--" }
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
    assert_equal %w[1 5 10], `seq 10 | #{FZF} -f "1 | 5"`.lines(chomp: true)
    assert_equal %w[1 10 2 3 4 5 6 7 8 9],
                 `seq 10 | #{FZF} -f '1 | !1'`.lines(chomp: true)
  end

  def test_hscroll_off
    File.open(tempname, 'w') { |f| f.puts '=' * 10_000 + '0123456789' }
    [0, 3, 6].each do |off|
      tmux = Tmux.new("#{FZF} --hscroll-off=#{off} -q 0 < #{tempname}")
      tmux.until { |lines| lines[-3]&.end_with?((0..off).to_a.join + '..') }
      tmux.send_keys '9'
      tmux.until { |lines| lines[-3]&.end_with?('789') }
    end
  end

  def test_partial_caching
    tmux = Tmux.new("seq 1000 | #{FZF} -e")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 11
    tmux.until { |lines| lines[-2] == '  19/1000' }
    tmux.send_keys 'C-a', "'"
    tmux.until { |lines| lines[-2] == '  28/1000' }
  end

  def test_jump
    tmux = Tmux.new("seq 1000 | #{fzf("--multi --jump-labels 12345 --bind 'ctrl-j:jump'")}")
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
    assert_equal %w[5 2 1], readonce.lines(chomp: true)
  end

  def test_jump_accept
    tmux = Tmux.new("seq 1000 | #{fzf("--multi --jump-labels 12345 --bind 'ctrl-j:jump-accept'")}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys 'C-j'
    tmux.until { |lines| lines[-7] == '5 5' }
    tmux.send_keys '3'
    assert_equal '3', readonce.chomp
  end

  def test_pointer
    tmux = Tmux.new("seq 10 | #{fzf("--pointer '>>'")}")
    # Assert that specified pointer is displayed
    tmux.until { |lines| lines[-3] == '>> 1' }
  end

  def test_pointer_with_jump
    tmux = Tmux.new("seq 10 | #{fzf("--multi --jump-labels 12345 --bind 'ctrl-j:jump' --pointer '>>'")}")
    tmux.until { |lines| lines[-2] == '  10/10' }
    tmux.send_keys 'C-j'
    # Correctly padded jump label should appear
    tmux.until { |lines| lines[-7] == '5  5' }
    tmux.until { |lines| lines[-8] == '   6' }
    tmux.send_keys '5'
    # Assert that specified pointer is displayed
    tmux.until { |lines| lines[-7] == '>> 5' }
  end

  def test_marker
    tmux = Tmux.new("seq 10 | #{fzf("--multi --marker '>>'")}")
    tmux.until { |lines| lines[-2] == '  10/10' }
    tmux.send_keys :BTab
    # Assert that specified marker is displayed
    tmux.until { |lines| lines[-3] == ' >>1' }
  end

  def test_preview
    tmux = Tmux.new(%(seq 1000 | sed s/^2$// | #{FZF} -m --preview 'sleep 0.2; echo {{}-{+}}' --bind ?:toggle-preview))
    tmux.until { |lines| lines[1]&.include?(' {1-1} ') }
    tmux.send_keys :Up
    tmux.until { |lines| lines[1]&.include?(' {-} ') }
    tmux.send_keys '555'
    tmux.until { |lines| lines[1]&.include?(' {555-555} ') }
    tmux.send_keys '?'
    tmux.until { |lines| lines[1]&.include?(' {555-555} ') == false }
    tmux.send_keys '?'
    tmux.until { |lines| lines[1]&.include?(' {555-555} ') }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-2]&.start_with?('  28/1000 ') }
    tmux.send_keys 'foobar'
    tmux.until { |lines| lines[1]&.include?(' {55-55} ') == false }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.until { |lines| lines[1]&.include?(' {1-1} ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {-1} ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {3-1 } ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {4-1  3} ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {5-1  3 4} ') }
  end

  def test_preview_hidden
    tmux = Tmux.new(%(seq 1000 | #{FZF} --preview 'echo {{}-{}-$FZF_PREVIEW_LINES-$FZF_PREVIEW_COLUMNS}' --preview-window down:1:hidden --bind ?:toggle-preview))
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys '?'
    tmux.until { |lines| lines[-2] =~ / {1-1-1-[0-9]+}/ }
    tmux.send_keys '555'
    tmux.until { |lines| lines[-2] =~ / {555-555-1-[0-9]+}/ }
    tmux.send_keys '?'
    tmux.until { |lines| lines[-1] == '> 555' }
  end

  def test_preview_size_0
    tmux = Tmux.new(%(seq 100 | #{FZF} --reverse --preview 'echo {} >> #{tempname}; echo ' --preview-window 0))
    tmux.until { |lines| lines.item_count == 100 && lines[1] == '  100/100' && lines[2] == '> 1' }
    wait { File.exist?(tempname) && File.readlines(tempname, chomp: true) == %w[1] }
    tmux.send_keys :Down
    tmux.until { |lines| lines[3] == '> 2' }
    wait { File.exist?(tempname) && File.readlines(tempname, chomp: true) == %w[1 2] }
    tmux.send_keys :Down
    tmux.until { |lines| lines[4] == '> 3' }
    wait { File.exist?(tempname) && File.readlines(tempname, chomp: true) == %w[1 2 3] }
  end

  def test_preview_flags
    tmux = Tmux.new(%(seq 10 | sed 's/^/:: /; s/$/  /' |
        #{FZF} --multi --preview 'echo {{2}/{s2}/{+2}/{+s2}/{q}/{n}/{+n}}'))
    tmux.until { |lines| lines[1]&.include?(' {1/1  /1/1  //0/0} ') }
    tmux.send_keys '123'
    tmux.until { |lines| lines[1]&.include?(' {////123//} ') }
    tmux.send_keys 'C-u', '1'
    tmux.until { |lines| lines.match_count == 2 }
    tmux.until { |lines| lines[1]&.include?(' {1/1  /1/1  /1/0/0} ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {10/10  /1/1  /1/9/0} ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' {10/10  /1 10/1   10  /1/9/0 9} ') }
    tmux.send_keys '2'
    tmux.until { |lines| lines[1]&.include?(' {//1 10/1   10  /12//0 9} ') }
    tmux.send_keys '3'
    tmux.until { |lines| lines[1]&.include?(' {//1 10/1   10  /123//0 9} ') }
  end

  def test_preview_file
    tmux = Tmux.new(%[(echo foo bar; echo bar foo) | #{FZF} --multi --preview 'cat {+f} {+f2} {+nf} {+fn}' --print0])
    tmux.until { |lines| lines[1]&.include?(' foo barbar00 ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' foo barbar00 ') }
    tmux.send_keys :BTab
    tmux.until { |lines| lines[1]&.include?(' foo barbar foobarfoo0101 ') }
  end

  def test_preview_q_no_match
    tmux = Tmux.new(%(: | #{FZF} --preview 'echo foo {q}'))
    tmux.until { |lines| lines.match_count == 0 }
    tmux.until { |lines| lines[1]&.include?(' foo ') == false }
    tmux.send_keys 'bar'
    tmux.until { |lines| lines[1]&.include?(' foo bar ') }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines[1]&.include?(' foo ') == false }
  end

  def test_preview_q_no_match_with_initial_query
    tmux = Tmux.new(%(: | #{FZF} --preview 'echo foo {q}{q}' --query foo))
    tmux.until { |lines| lines.match_count == 0 }
    tmux.until { |lines| lines[1]&.include?(' foofoo ') }
  end

  def test_no_clear
    tmux = Tmux.new("seq 10 | #{FZF} --no-clear --inline-info --height 5 > #{tempname}")
    prompt = '>   < 10/10'
    tmux.until { |lines| lines[-1] == prompt }
    tmux.send_keys :Enter
    wait { File.exist?(tempname) && File.readlines(tempname, chomp: true) == %w[1] }
    tmux.until { |lines| lines[5 - 1] == prompt }
  end

  def test_info_hidden
    tmux = Tmux.new("seq 10 | #{FZF} --info=hidden")
    tmux.until { |lines| lines[-2] == '> 1' }
  end

  def test_change_top
    tmux = Tmux.new(%(seq 1000 | #{FZF} --bind change:top))
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-4] == '> 2' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-3] == '> 1' }
    tmux.send_keys :Up
    tmux.until { |lines| lines[-4] == '> 10' }
    tmux.send_keys 1
    tmux.until { |lines| lines[-3] == '> 11' }
  end

  def test_accept_non_empty
    tmux = Tmux.new(%(seq 1000 | #{fzf('--print-query --bind enter:accept-non-empty')}))
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines[-2] == '  0/1000' }
    # fzf doesn't exit since there's no selection
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-2] == '  0/1000' }
    tmux.send_keys 'C-u'
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys '999'
    tmux.until { |lines| lines[-2] == '  1/1000' }
    tmux.send_keys :Enter
    assert_equal %w[999 999], readonce.lines(chomp: true)
  end

  def test_accept_non_empty_with_multi_selection
    tmux = Tmux.new(%(seq 1000 | #{fzf('-m --print-query --bind enter:accept-non-empty')}))
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-2] == '  1000/1000 (1)' }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines[-2] == '  0/1000 (1)' }
    # fzf will exit in this case even though there's no match for the current query
    tmux.send_keys :Enter
    assert_equal %w[foo 1], readonce.lines(chomp: true)
  end

  def test_accept_non_empty_with_empty_list
    tmux = Tmux.new(%(: | #{fzf('-q foo --print-query --bind enter:accept-non-empty')}))
    tmux.until { |lines| lines[-2] == '  0/0' }
    tmux.send_keys :Enter
    # fzf will exit anyway since input list is empty
    assert_equal %w[foo], readonce.lines(chomp: true)
  end

  def test_preview_update_on_select
    tmux = Tmux.new(%(seq 10 | #{FZF} -m --preview 'echo {+}' --bind a:toggle-all))
    tmux.until { |lines| lines.item_count == 10 }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.any? { |line| line.include?('1 2 3 4 5') } }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.none? { |line| line.include?('1 2 3 4 5') } }
  end

  def test_escaped_meta_characters
    input = [
      'foo^bar',
      'foo$bar',
      'foo!bar',
      "foo'bar",
      'foo bar',
      'bar foo'
    ]
    File.open(tempname, 'w') { |f| f.puts input }

    assert_equal input.length, `#{FZF} -f'foo bar' < #{tempname}`.lines.length
    assert_equal input.length - 1, `#{FZF} -f'^foo bar$' < #{tempname}`.lines.length
    assert_equal ['foo bar'], `#{FZF} -f'foo\\ bar' < #{tempname}`.lines(chomp: true)
    assert_equal ['foo bar'], `#{FZF} -f'^foo\\ bar$' < #{tempname}`.lines(chomp: true)
    assert_equal input.length - 1, `#{FZF} -f'!^foo\\ bar$' < #{tempname}`.lines.length
  end

  def test_inverse_only_search_should_not_sort_the_result
    # Filter
    assert_equal %w[aaaaa b ccc],
                 `printf '%s\n' aaaaa b ccc BAD | #{FZF} -f '!bad'`.lines(chomp: true)

    # Interactive
    tmux = Tmux.new(%(printf '%s\n' aaaaa b ccc BAD | #{FZF} -q '!bad'))
    tmux.until { |lines| lines.item_count == 4 && lines.match_count == 3 }
    tmux.until { |lines| lines[-3] == '> aaaaa' }
    tmux.until { |lines| lines[-4] == '  b' }
    tmux.until { |lines| lines[-5] == '  ccc' }
  end

  def test_preview_correct_tab_width_after_ansi_reset_code
    File.open(tempname, 'w') { |f| f.puts "\x1b[31m+\x1b[m\t\x1b[32mgreen" }
    tmux = Tmux.new("#{FZF} --preview 'cat #{tempname}'")
    tmux.until { |lines| lines[1]&.include?(' +       green ') }
  end

  def test_phony
    tmux = Tmux.new(%(seq 1000 | #{FZF} --query 333 --phony --preview 'echo {} {q}'))
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.until { |lines| lines[1]&.include?(' 1 333 ') }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines.match_count == 1000 }
    tmux.until { |lines| lines[1]&.include?(' 1 333foo ') }
  end

  def test_reload
    tmux = Tmux.new(%(seq 1000 | #{FZF} --bind 'change:reload(seq {q}),a:reload(seq 100),b:reload:seq 200' --header-lines 2 --multi 2))
    tmux.until { |lines| lines.match_count == 998 }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.item_count == 98 && lines.match_count == 98 }
    tmux.send_keys 'b'
    tmux.until { |lines| lines.item_count == 198 && lines.match_count == 198 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-2] == '  198/198 (1/2)' }
    tmux.send_keys '555'
    tmux.until { |lines| lines[-2] == '  1/553' }
  end

  def test_reload_even_when_theres_no_match
    tmux = Tmux.new(%(: | #{FZF} --bind 'space:reload(seq 10)'))
    tmux.until { |lines| lines.item_count == 0 }
    tmux.send_keys :Space
    tmux.until { |lines| lines.item_count == 10 }
  end

  def test_clear_list_when_header_lines_changed_due_to_reload
    tmux = Tmux.new(%(seq 10 | #{FZF} --header 0 --header-lines 3 --bind 'space:reload(seq 1)'))
    tmux.until { |lines| lines.any? { |line| line.include?('9') } }
    tmux.send_keys :Space
    tmux.until { |lines| lines.none? { |line| line.include?('9') } }
  end

  def test_clear_query
    tmux = Tmux.new(%(: | #{FZF} --query foo --bind space:clear-query))
    tmux.until { |lines| lines.item_count == 0 }
    tmux.until { |lines| lines.last == '> foo' }
    tmux.send_keys 'C-a', 'bar'
    tmux.until { |lines| lines.last == '> barfoo' }
    tmux.send_keys :Space
    tmux.until { |lines| lines.last == '>' }
  end

  def test_clear_selection
    tmux = Tmux.new(%(seq 100 | #{FZF} --multi --bind space:clear-selection))
    tmux.until { |lines| lines.match_count == 100 }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-2] == '  100/100 (1)' }
    tmux.send_keys 'foo'
    tmux.until { |lines| lines[-2] == '  0/100 (1)' }
    tmux.send_keys :Space
    tmux.until { |lines| lines[-2] == '  0/100' }
  end

  def test_backward_delete_char_eof
    tmux = Tmux.new("seq 1000 | #{fzf("--bind 'bs:backward-delete-char/eof'")}")
    tmux.until { |lines| lines[-2] == '  1000/1000' }
    tmux.send_keys '11'
    tmux.until { |lines| lines[-1] == '> 11' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-1] == '> 1' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines[-1]&.start_with?('Pane is dead') }
  end

  def test_strip_xterm_osc_sequence
    %W[\x07 \x1b\\].each do |esc|
      File.open(tempname, 'w') { |f| f.puts %(printf $1"\e]4;3;rgb:aa/bb/cc#{esc} "$2) }
      File.chmod(0o755, tempname)
      tmux = Tmux.new(
        %(echo foo bar | #{FZF} --preview '#{tempname} {2} {1}')
      )
      tmux.until { |lines| lines.any_include?('bar foo') }
    end
  end

  def test_keep_right
    tmux = Tmux.new("seq 10000 | #{FZF} --read0 --keep-right")
    tmux.until { |lines| lines.any_include?('9999 10000') }
  end
end

module TestShell
  attr_reader :tmux

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "export #{name}='#{val}'", :Enter
  end

  def unset_var(name)
    tmux.prepare
    tmux.send_keys "unset #{name}", :Enter
  end

  def test_ctrl_t
    set_var('FZF_CTRL_T_COMMAND', 'seq 100')

    tmux.prepare
    tmux.send_keys 'C-t'
    tmux.until { |lines| lines.item_count == 100 }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| lines.any_include?(' (3)') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines.any_include?('1 2 3') }
  end

  def test_ctrl_t_unicode
    File.open(tempname, 'w') { |f| f.puts ['fzf-unicode 테스트1', 'fzf-unicode 테스트2'] }
    set_var('FZF_CTRL_T_COMMAND', "cat #{tempname}")

    tmux.prepare
    tmux.send_keys 'echo ', 'C-t'
    tmux.until { |lines| lines.item_count == 2 }
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
    tmux.until { |lines| lines.join =~ /echo .*fzf-unicode.*1.* .*fzf-unicode.*2/ }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-2] == 'fzf-unicode 테스트1 fzf-unicode 테스트2' }
  end

  def test_alt_c
    Dir.mkdir('foo')
    tmux.prepare
    tmux.send_keys :Escape, :c
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys :Enter
    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| lines[-2]&.end_with?('/foo') }
  end

  def test_alt_c_command
    set_var('FZF_ALT_C_COMMAND', 'echo /tmp')

    tmux.prepare
    tmux.send_keys :Escape, :c
    tmux.until { |lines| lines.item_count == 1 }
    tmux.send_keys :Enter

    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| lines[-2] == '/tmp' }
  end

  def test_ctrl_r
    tmux.prepare
    tmux.send_keys 'echo 1st', :Enter
    tmux.prepare
    tmux.send_keys 'echo 2nd', :Enter
    tmux.prepare
    tmux.send_keys 'echo 3d', :Enter
    3.times do
      tmux.prepare
      tmux.send_keys 'echo 3rd', :Enter
    end
    tmux.prepare
    tmux.send_keys 'echo 4th', :Enter
    tmux.prepare
    tmux.send_keys 'C-r'
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys 'e3d'
    # Duplicates removed: 3d (1) + 3rd (1) => 2 matches
    tmux.until { |lines| lines.match_count == 2 }
    tmux.until { |lines| lines[-3]&.end_with?(' echo 3d') }
    tmux.send_keys 'C-r'
    tmux.until { |lines| lines[-3]&.end_with?(' echo 3rd') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ echo 3rd' }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-2] == '3rd' }
  end

  def test_ctrl_r_multiline
    tmux.prepare
    tmux.send_keys 'echo "foo', :Enter, 'bar"', :Enter
    tmux.until { |lines| lines[-3..-2] == %w[foo bar] }
    tmux.send_keys 'C-r'
    tmux.until { |lines| lines[-1] == '>' }
    tmux.send_keys 'foo bar'
    tmux.until { |lines| lines[-3]&.end_with?('bar"') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1]&.end_with?('bar"') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-3..-2] == %w[foo bar] }
  end

  def test_ctrl_r_abort
    skip("doesn't restore the original line when search is aborted pre Bash 4") if is_a?(TestBash) && `bash --version`[/(?<= version )\d+/].to_i < 4
    %w[foo ' "].each do |query|
      tmux.prepare
      tmux.send_keys query
      tmux.until { |lines| lines[-1]&.start_with?("$ #{query}") }
      tmux.send_keys 'C-r'
      tmux.until { |lines| lines[-1] == "> #{query}" }
      tmux.send_keys 'C-g'
      tmux.until { |lines| lines[-1]&.start_with?("$ #{query}") }
      tmux.send_keys 'C-u'
    end
  end
end

module CompletionTest
  include TestShell

  def test_file_completion
    FileUtils.touch(('1'..'100').to_a)
    FileUtils.touch(['no~such~user', 'foobar', File.expand_path('~/.fzf-home')])
    tmux.prepare
    tmux.send_keys 'cat 10**', :Tab
    tmux.until { |lines| lines.match_count == 2 }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| lines.select_count == 2 }
    tmux.send_keys :Enter
    tmux.until do |lines|
      lines[-1] == '$ cat 10 100'
    end

    # ~USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys "cat ~#{ENV['USER']}**", :Tab
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys "'.fzf-home"
    tmux.until { |lines| lines.count { |l| l.include?('.fzf-home') } > 1 }
    tmux.send_keys :Enter
    tmux.until do |lines|
      lines[-1] =~ %r{cat .*/\.fzf-home}
    end

    # ~INVALID_USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat ~such**', :Tab
    tmux.until { |lines| lines.any_include?('no~such~user') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ cat no~such~user' }

    # **<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat **', :Tab
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys 'foobar$'
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ cat foobar' }

    # Should include hidden files
    FileUtils.touch((1..100).map { |i| ".hidden-#{i}" })
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat hidden**', :Tab
    tmux.until { |lines| lines.match_count == 100 && lines.any_include?('.hidden-') }
  ensure
    File.unlink(File.expand_path('~/.fzf-home'))
  end

  def test_file_completion_root
    tmux.prepare
    tmux.send_keys 'ls /**', :Tab
    tmux.until { |lines| lines.match_count > 0 }
  end

  def test_dir_completion
    FileUtils.mkdir((1..100).map { |i| "d#{i}" })
    FileUtils.touch('d55/xxx')
    tmux.prepare
    tmux.send_keys 'cd **', :Tab
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys :Tab, :Tab # Tab does not work here
    tmux.send_keys 55
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ cd d55/' }
    tmux.send_keys :xx
    tmux.until { |lines| lines[-1] == '$ cd d55/xx' }

    # Should not match regular files (bash-only)
    if is_a?(TestBash)
      tmux.send_keys :Tab
      tmux.until { |lines| lines[-1] == '$ cd d55/xx' }
    end

    # Fail back to plusdirs
    tmux.send_keys :BSpace, :BSpace, :BSpace
    tmux.until { |lines| lines[-1] == '$ cd d55' }
    tmux.send_keys :Tab
    tmux.until { |lines| lines[-1] == '$ cd d55/' }
  end

  def test_process_completion
    tmux.prepare
    tmux.send_keys 'sleep 12345 &', :Enter
    pid = nil
    tmux.until { |lines| pid = lines[-2]&.[](/\d+$/) }
    tmux.send_keys 'kill ', :Tab
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys 'sleep12345'
    tmux.until { |lines| lines[3 + 2]&.end_with?(' sleep 12345') }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == "$ kill #{pid}" }
  ensure
    Process.kill('TERM', pid.to_i)
  end

  def test_custom_completion
    tmux.prepare
    tmux.send_keys '_fzf_compgen_path() { echo "$1"; seq 10; }', :Enter
    tmux.prepare
    tmux.send_keys 'ls /tmp/**', :Tab
    tmux.until { |lines| lines.match_count == 11 }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| lines.select_count == 3 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ ls /tmp 1 2' }
  end

  def test_unset_completion
    tmux.prepare
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    tmux.prepare

    # Using tmux
    tmux.send_keys 'unset FZFFOOBR**', :Tab
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ unset FZFFOOBAR' }
    tmux.send_keys 'C-c'

    # FZF_TMUX=1
    new_shell
    tmux.focus
    tmux.prepare
    tmux.send_keys 'unset FZFFOOBR**', :Tab
    tmux.until { |lines| lines.match_count == 1 }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-1] == '$ unset FZFFOOBAR' }
  end

  def test_file_completion_unicode
    File.open('fzf-unicode 테스트1', 'w') { |f| f.puts 'test3' }
    File.open('fzf-unicode 테스트2', 'w') { |f| f.puts 'test4' }
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
    tmux.until { |lines| lines[-1] =~ /cat .*fzf-unicode.*1.* .*fzf-unicode.*2/ }
    tmux.send_keys :Enter
    tmux.until { |lines| lines[-3..-2] == %w[test3 test4] }
  end

  def test_custom_completion_api
    tmux.prepare
    tmux.send_keys 'eval "_fzf$(declare -f _comprun)"', :Enter
    %w[f g].each do |command|
      tmux.prepare
      tmux.send_keys "#{command} b**", :Tab
      tmux.until do |lines|
        lines.item_count == 2 && lines.match_count == 1 &&
          lines.any_include?("prompt-#{command}") &&
          lines.any_include?("preview-#{command}-bar")
      end
      tmux.send_keys :Enter
      tmux.until { |lines| lines[-1] == "$ #{command} #{command}barbar" }
      tmux.send_keys 'C-u'
    end
  end
end

class TestBash < TestBase
  include CompletionTest

  BASHRC = '/tmp/fzf.bash'
  File.open(BASHRC, 'w') do |f|
    f.puts ERB.new(TEMPLATE).result(binding)
  end

  def setup
    super
    @tmux = Tmux.new("bash --rcfile #{BASHRC}")
  end

  def new_shell
    tmux.prepare
    tmux.send_keys "FZF_TMUX=1 bash --rcfile #{BASHRC}", :Enter
  end

  def test_dynamic_completion_loader
    FileUtils.touch('foo')
    tmux.prepare
    tmux.paste('_fzf_completion_loader=1')
    tmux.paste('_completion_loader() { complete -o default fake; }')
    tmux.paste('complete -F _fzf_path_completion -o default -o bashdefault fake')
    tmux.send_keys 'fake foo**', :Tab
    tmux.until { |lines| lines.match_count > 0 }
    tmux.send_keys 'C-c'

    tmux.until { |lines| lines[-1] == '$ fake foo**' }
    tmux.send_keys 'C-u'
    tmux.prepare
    tmux.send_keys 'fake foo'
    tmux.send_keys :Tab, 'C-u'

    tmux.prepare
    tmux.send_keys 'fake foo**', :Tab
    tmux.until { |lines| lines.match_count > 0 }
  end
end

class TestZsh < TestBase
  include CompletionTest

  ZDOTDIR = '/tmp/fzf-zsh'
  FileUtils.rm_rf(ZDOTDIR)
  FileUtils.mkdir_p(ZDOTDIR)
  File.open("#{ZDOTDIR}/.zshrc", 'w') do |f|
    f.puts ERB.new(TEMPLATE).result(binding)
  end

  def setup
    super
    @tmux = Tmux.new("ZDOTDIR=#{ZDOTDIR} zsh")
  end

  def new_shell
    tmux.prepare
    tmux.send_keys 'FZF_TMUX=1 zsh', :Enter
  end
end

class TestFish < TestBase
  include TestShell

  def setup
    super
    @tmux = Tmux.new(UNSETS.map { |v| v + '= ' }.join + 'fish')
    tmux.send_keys "function fish_prompt; echo '$ '; end", :Enter
  end

  def new_shell
    tmux.prepare
    tmux.send_keys 'env FZF_TMUX=1 fish', :Enter
    tmux.send_keys "function fish_prompt; echo '$ '; end", :Enter
  end

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "set -g #{name} '#{val}'", :Enter
  end
end

__END__
# Setup fzf
# ---------
if [[ ! "$PATH" == *<%= BASE %>/bin* ]]; then
  export PATH="${PATH:+${PATH}:}<%= BASE %>/bin"
fi

# Auto-completion
# ---------------
[[ $- == *i* ]] && source "<%= BASE %>/shell/completion.$0" 2> /dev/null

# Key bindings
# ------------
source "<%= BASE %>/shell/key-bindings.$0"

PS1='$ ' HISTFILE= HISTSIZE=100
unset <%= UNSETS.join(' ') %>

# Old API
_fzf_complete_f() {
  _fzf_complete "+m --multi --prompt \"prompt-f> \"" "$@" < <(
    echo foo
    echo bar
  )
}

# New API
_fzf_complete_g() {
  _fzf_complete +m --multi --prompt "prompt-g> " -- "$@" < <(
    echo foo
    echo bar
  )
}

_fzf_complete_f_post() {
  awk '{print "f" $0 $0}'
}

_fzf_complete_g_post() {
  awk '{print "g" $0 $0}'
}

[ -n "$BASH" ] && complete -F _fzf_complete_f -o default -o bashdefault f
[ -n "$BASH" ] && complete -F _fzf_complete_g -o default -o bashdefault g

_comprun() {
  local command=$1
  shift

  case "$command" in
    f) fzf "$@" --preview 'echo preview-f-{}' ;;
    g) fzf "$@" --preview 'echo preview-g-{}' ;;
    *) fzf "$@" ;;
  esac
}
