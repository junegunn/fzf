#!/usr/bin/env ruby
# encoding: utf-8

require 'rubygems'
require 'curses'
require 'timeout'
require 'stringio'
require 'minitest/autorun'
require 'tempfile'
$LOAD_PATH.unshift File.expand_path('../..', __FILE__)
ENV['FZF_EXECUTABLE'] = '0'
load 'fzf'

class MockTTY
  def initialize
    @buffer = ''
    @mutex = Mutex.new
    @condv = ConditionVariable.new
  end

  def read_nonblock sz
    @mutex.synchronize do
      take sz
    end
  end

  def take sz
    if @buffer.length >= sz
      ret = @buffer[0, sz]
      @buffer = @buffer[sz..-1]
      ret
    end
  end

  def getc
    sleep 0.1
    while true
      @mutex.synchronize do
        if char = take(1)
          return char
        else
          @condv.wait(@mutex)
        end
      end
    end
  end

  def << str
    @mutex.synchronize do
      @buffer << str
      @condv.broadcast
    end
    self
  end
end

class TestFZF < MiniTest::Unit::TestCase
  def setup
    ENV.delete 'FZF_DEFAULT_SORT'
    ENV.delete 'FZF_DEFAULT_OPTS'
    ENV.delete 'FZF_DEFAULT_COMMAND'
  end

  def test_default_options
    fzf = FZF.new []
    assert_equal 1000,  fzf.sort
    assert_equal false, fzf.multi
    assert_equal true,  fzf.color
    assert_equal nil,   fzf.rxflag
    assert_equal true,  fzf.mouse
    assert_equal nil,   fzf.nth
    assert_equal nil,   fzf.with_nth
    assert_equal true,  fzf.color
    assert_equal false, fzf.black
    assert_equal true,  fzf.ansi256
    assert_equal '',    fzf.query
    assert_equal false, fzf.select1
    assert_equal false, fzf.exit0
    assert_equal nil,   fzf.filter
    assert_equal nil,   fzf.extended
    assert_equal false, fzf.reverse
    assert_equal '> ',  fzf.prompt
    assert_equal false, fzf.print_query
  end

  def test_environment_variables
    # Deprecated
    ENV['FZF_DEFAULT_SORT'] = '20000'
    fzf = FZF.new []
    assert_equal 20000, fzf.sort
    assert_equal nil,   fzf.nth

    ENV['FZF_DEFAULT_OPTS'] =
      '-x -m -s 10000 -q "  hello  world  " +c +2 --select-1 -0 ' <<
      '--no-mouse -f "goodbye world" --black --with-nth=3,-3..,2 --nth=3,-1,2 --reverse --print-query'
    fzf = FZF.new []
    assert_equal 10000,   fzf.sort
    assert_equal '  hello  world  ',
                          fzf.query
    assert_equal 'goodbye world',
                          fzf.filter
    assert_equal :fuzzy,  fzf.extended
    assert_equal true,    fzf.multi
    assert_equal false,   fzf.color
    assert_equal false,   fzf.ansi256
    assert_equal true,    fzf.black
    assert_equal false,   fzf.mouse
    assert_equal true,    fzf.select1
    assert_equal true,    fzf.exit0
    assert_equal true,    fzf.reverse
    assert_equal true,    fzf.print_query
    assert_equal [2..2, -1..-1, 1..1], fzf.nth
    assert_equal [2..2, -3..-1, 1..1], fzf.with_nth
  end

  def test_option_parser
    # Long opts
    fzf = FZF.new %w[--sort=2000 --no-color --multi +i --query hello --select-1
                     --exit-0 --filter=howdy --extended-exact
                     --no-mouse --no-256 --nth=1 --with-nth=.. --reverse --prompt (hi)
                     --print-query]
    assert_equal 2000,    fzf.sort
    assert_equal true,    fzf.multi
    assert_equal false,   fzf.color
    assert_equal false,   fzf.ansi256
    assert_equal false,   fzf.black
    assert_equal false,   fzf.mouse
    assert_equal 0,       fzf.rxflag
    assert_equal 'hello', fzf.query
    assert_equal true,    fzf.select1
    assert_equal true,    fzf.exit0
    assert_equal 'howdy', fzf.filter
    assert_equal :exact,  fzf.extended
    assert_equal [0..0],  fzf.nth
    assert_equal nil,     fzf.with_nth
    assert_equal true,    fzf.reverse
    assert_equal '(hi)',  fzf.prompt
    assert_equal true,    fzf.print_query

    # Long opts (left-to-right)
    fzf = FZF.new %w[--sort=2000 --no-color --multi +i --query=hello
                     --filter a --filter b --no-256 --black --nth -1 --nth -2
                     --select-1 --exit-0 --no-select-1 --no-exit-0
                     --no-sort -i --color --no-multi --256
                     --reverse --no-reverse --prompt (hi) --prompt=(HI)
                     --print-query --no-print-query]
    assert_equal nil,     fzf.sort
    assert_equal false,   fzf.multi
    assert_equal true,    fzf.color
    assert_equal true,    fzf.ansi256
    assert_equal true,    fzf.black
    assert_equal true,    fzf.mouse
    assert_equal 1,       fzf.rxflag
    assert_equal 'b',     fzf.filter
    assert_equal 'hello', fzf.query
    assert_equal false,   fzf.select1
    assert_equal false,   fzf.exit0
    assert_equal nil,     fzf.extended
    assert_equal [-2..-2], fzf.nth
    assert_equal false,   fzf.reverse
    assert_equal '(HI)',  fzf.prompt
    assert_equal false,    fzf.print_query

    # Short opts
    fzf = FZF.new %w[-s2000 +c -m +i -qhello -x -fhowdy +2 -n3 -1 -0]
    assert_equal 2000,    fzf.sort
    assert_equal true,    fzf.multi
    assert_equal false,   fzf.color
    assert_equal false,   fzf.ansi256
    assert_equal 0,       fzf.rxflag
    assert_equal 'hello', fzf.query
    assert_equal 'howdy', fzf.filter
    assert_equal :fuzzy,  fzf.extended
    assert_equal [2..2],  fzf.nth
    assert_equal true,    fzf.select1
    assert_equal true,    fzf.exit0

    # Left-to-right
    fzf = FZF.new %w[-s 2000 +c -m +i -qhello -x -fgoodbye +2 -n3 -n4,5
                     -s 3000 -c +m -i -q world +x -fworld -2 --black --no-black
                     -1 -0 +1 +0
                    ]
    assert_equal 3000,    fzf.sort
    assert_equal false,   fzf.multi
    assert_equal true,    fzf.color
    assert_equal true,    fzf.ansi256
    assert_equal false,   fzf.black
    assert_equal 1,       fzf.rxflag
    assert_equal 'world', fzf.query
    assert_equal false,   fzf.select1
    assert_equal false,   fzf.exit0
    assert_equal 'world', fzf.filter
    assert_equal nil,     fzf.extended
    assert_equal [3..3, 4..4], fzf.nth
  rescue SystemExit => e
    assert false, "Exited"
  end

  def test_invalid_option
    [
      %w[--unknown],
      %w[yo dawg],
      %w[--nth=0],
      %w[-n 0],
      %w[-n 1..2..3],
      %w[-n 1....],
      %w[-n ....3],
      %w[-n 1....3],
      %w[-n 1..0],
      %w[--nth ..0],
    ].each do |argv|
      assert_raises(SystemExit) do
        fzf = FZF.new argv
      end
    end
  end

  def test_width
    fzf = FZF.new []
    assert_equal 5, fzf.width('abcde')
    assert_equal 4, fzf.width('한글')
    assert_equal 5, fzf.width('한글.')
  end if RUBY_VERSION >= '1.9'

  def test_trim
    fzf = FZF.new []
    assert_equal ['사.',     6], fzf.trim('가나다라마바사.', 4, true)
    assert_equal ['바사.',   5], fzf.trim('가나다라마바사.', 5, true)
    assert_equal ['바사.',   5], fzf.trim('가나다라마바사.', 6, true)
    assert_equal ['마바사.', 4], fzf.trim('가나다라마바사.', 7, true)
    assert_equal ['가나',    6], fzf.trim('가나다라마바사.', 4, false)
    assert_equal ['가나',    6], fzf.trim('가나다라마바사.', 5, false)
    assert_equal ['가나a',   6], fzf.trim('가나ab라마바사.', 5, false)
    assert_equal ['가나ab',  5], fzf.trim('가나ab라마바사.', 6, false)
    assert_equal ['가나ab',  5], fzf.trim('가나ab라마바사.', 7, false)
  end if RUBY_VERSION >= '1.9'

  def test_format
    fzf = FZF.new []
    assert_equal [['01234..', false]], fzf.format('0123456789', 7, [])
    assert_equal [['012', false], ['34', true], ['..', false]],
      fzf.format('0123456789', 7, [[3, 5]])
    assert_equal [['..56', false], ['789', true]],
      fzf.format('0123456789', 7, [[7, 10]])
    assert_equal [['..56', false], ['78', true], ['9', false]],
      fzf.format('0123456789', 7, [[7, 9]])

    (3..5).each do |i|
      assert_equal [['..', false], ['567', true], ['89', false]],
        fzf.format('0123456789', 7, [[i, 8]])
    end

    assert_equal [['..', false], ['345', true], ['..', false]],
      fzf.format('0123456789', 7, [[3, 6]])
    assert_equal [['012', false], ['34', true], ['..', false]],
      fzf.format('0123456789', 7, [[3, 5]])

    # Multi-region
    assert_equal [["0", true], ["1", false], ["2", true], ["34..", false]],
      fzf.format('0123456789', 7, [[0, 1], [2, 3]])

    assert_equal [["..", false], ["5", true], ["6", false], ["78", true], ["9", false]],
      fzf.format('0123456789', 7, [[3, 6], [7, 9]])

    assert_equal [["..", false], ["3", true], ["4", false], ["5", true], ["..", false]],
      fzf.format('0123456789', 7, [[3, 4], [5, 6]])

    # Multi-region Overlap
    assert_equal [["..", false], ["345", true], ["..", false]],
      fzf.format('0123456789', 7, [[4, 5], [3, 6]])
  end

  def test_fuzzy_matcher
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE
    list = %w[
      juice
      juiceful
      juiceless
      juicily
      juiciness
      juicy]
    assert matcher.caches.empty?
    assert_equal(
      [["juice",     [[0, 1]]],
       ["juiceful",  [[0, 1]]],
       ["juiceless", [[0, 1]]],
       ["juicily",   [[0, 1]]],
       ["juiciness", [[0, 1]]],
       ["juicy",     [[0, 1]]]], matcher.match(list, 'j', '', '').sort)
    assert !matcher.caches.empty?
    assert_equal [list.object_id], matcher.caches.keys
    assert_equal 1, matcher.caches[list.object_id].length
    assert_equal 6, matcher.caches[list.object_id]['j'].length

    assert_equal(
      [["juicily",   [[0, 5]]],
       ["juiciness", [[0, 5]]]], matcher.match(list, 'jii', '', '').sort)

    assert_equal(
      [["juicily",   [[2, 5]]],
       ["juiciness", [[2, 5]]]], matcher.match(list, 'ii', '', '').sort)

    assert_equal 3, matcher.caches[list.object_id].length
    assert_equal 2, matcher.caches[list.object_id]['ii'].length

    # TODO : partial_cache
  end

  def test_fuzzy_matcher_rxflag
    assert_equal nil, FZF::FuzzyMatcher.new(nil).rxflag
    assert_equal 0, FZF::FuzzyMatcher.new(0).rxflag
    assert_equal 1, FZF::FuzzyMatcher.new(1).rxflag

    assert_equal 1, FZF::FuzzyMatcher.new(nil).rxflag_for('abc')
    assert_equal 0, FZF::FuzzyMatcher.new(nil).rxflag_for('Abc')
    assert_equal 0, FZF::FuzzyMatcher.new(0).rxflag_for('abc')
    assert_equal 0, FZF::FuzzyMatcher.new(0).rxflag_for('Abc')
    assert_equal 1, FZF::FuzzyMatcher.new(1).rxflag_for('abc')
    assert_equal 1, FZF::FuzzyMatcher.new(1).rxflag_for('Abc')
  end

  def test_fuzzy_matcher_case_sensitive
    # Smart-case match (Uppercase found)
    assert_equal [['Fruit', [[0, 5]]]],
      FZF::FuzzyMatcher.new(nil).match(%w[Fruit Grapefruit], 'Fruit', '', '').sort

    # Smart-case match (Uppercase not-found)
    assert_equal [["Fruit", [[0, 5]]], ["Grapefruit", [[5, 10]]]],
      FZF::FuzzyMatcher.new(nil).match(%w[Fruit Grapefruit], 'fruit', '', '').sort

    # Case-sensitive match (-i)
    assert_equal [['Fruit', [[0, 5]]]],
      FZF::FuzzyMatcher.new(0).match(%w[Fruit Grapefruit], 'Fruit', '', '').sort

    # Case-insensitive match (+i)
    assert_equal [["Fruit", [[0, 5]]], ["Grapefruit", [[5, 10]]]],
      FZF::FuzzyMatcher.new(Regexp::IGNORECASE).
      match(%w[Fruit Grapefruit], 'Fruit', '', '').sort
  end

  def test_extended_fuzzy_matcher_case_sensitive
    %w['Fruit Fruit$].each do |q|
      # Smart-case match (Uppercase found)
      assert_equal [['Fruit', [[0, 5]]]],
        FZF::ExtendedFuzzyMatcher.new(nil).match(%w[Fruit Grapefruit], q, '', '').sort

      # Smart-case match (Uppercase not-found)
      assert_equal [["Fruit", [[0, 5]]], ["Grapefruit", [[5, 10]]]],
        FZF::ExtendedFuzzyMatcher.new(nil).match(%w[Fruit Grapefruit], q.downcase, '', '').sort

      # Case-sensitive match (-i)
      assert_equal [['Fruit', [[0, 5]]]],
        FZF::ExtendedFuzzyMatcher.new(0).match(%w[Fruit Grapefruit], q, '', '').sort

      # Case-insensitive match (+i)
      assert_equal [["Fruit", [[0, 5]]], ["Grapefruit", [[5, 10]]]],
        FZF::ExtendedFuzzyMatcher.new(Regexp::IGNORECASE).
        match(%w[Fruit Grapefruit], q, '', '').sort
    end
  end

  def test_extended_fuzzy_matcher
    matcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE
    list = %w[
      juice
      juiceful
      juiceless
      juicily
      juiciness
      juicy
      _juice]
    match = proc { |q, prefix|
      matcher.match(list, q, prefix, '').sort.map { |p| [p.first, p.last.sort] }
    }

    assert matcher.caches.empty?
    3.times do
      ['y j', 'j y'].each do |pat|
        (0..pat.length - 1).each do |prefix_length|
          prefix = pat[0, prefix_length]
          assert_equal(
            [["juicily", [[0, 1], [6, 7]]],
             ["juicy",   [[0, 1], [4, 5]]]],
            match.call(pat, prefix))
        end
      end

      # $
      assert_equal [["juiceful",  [[7, 8]]]], match.call('l$', '')
      assert_equal [["juiceful",  [[7, 8]]],
                    ["juiceless", [[5, 6]]],
                    ["juicily",   [[5, 6]]]], match.call('l', '')

      # ^
      assert_equal list.length,     match.call('j', '').length
      assert_equal list.length - 1, match.call('^j', '').length

      # ^ + $
      assert_equal 0, match.call('^juici$', '').length
      assert_equal 1, match.call('^juice$', '').length
      assert_equal 0, match.call('^.*$', '').length

      # !
      assert_equal 0, match.call('!j', '').length

      # ! + ^
      assert_equal [["_juice", []]], match.call('!^j', '')

      # ! + $
      assert_equal list.length - 1, match.call('!l$', '').length

      # ! + f
      assert_equal [["juicy", [[4, 5]]]], match.call('y !l', '')

      # '
      assert_equal %w[juiceful juiceless juicily],
        match.call('il', '').map { |e| e.first }
      assert_equal %w[juicily],
        match.call("'il", '').map { |e| e.first }
      assert_equal (list - %w[juicily]).sort,
        match.call("!'il", '').map { |e| e.first }.sort
    end
    assert !matcher.caches.empty?
  end

  def test_xfuzzy_matcher_prefix_cache
    matcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE
    list = %w[
      a.java
      b.java
      java.jive
      c.java$
      d.java
    ]
    2.times do
      assert_equal 5, matcher.match(list, 'java',   'java',   '').length
      assert_equal 3, matcher.match(list, 'java$',  'java$',  '').length
      assert_equal 1, matcher.match(list, 'java$$', 'java$$', '').length

      assert_equal 0, matcher.match(list, '!java',  '!java',  '').length
      assert_equal 4, matcher.match(list, '!^jav',  '!^jav',  '').length
      assert_equal 4, matcher.match(list, '!^java', '!^java', '').length
      assert_equal 2, matcher.match(list, '!^java !b !c', '!^java', '').length
    end
  end

  def test_sort_by_rank
    matcher  = FZF::FuzzyMatcher.new Regexp::IGNORECASE
    xmatcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE
    list = %w[
      0____1
      0_____1
      01
      ____0_1
      01_
      _01_
      0______1
      ___01___
    ]
    assert_equal(
      [["01",       [[0, 2]]],
       ["01_",      [[0, 2]]],
       ["_01_",     [[1, 3]]],
       ["___01___", [[3, 5]]],
       ["____0_1",  [[4, 7]]],
       ["0____1",   [[0, 6]]],
       ["0_____1",  [[0, 7]]],
       ["0______1", [[0, 8]]]],
      FZF.sort(matcher.match(list, '01', '', '')))

    assert_equal(
      [["01",       [[0, 1], [1, 2]]],
       ["01_",      [[0, 1], [1, 2]]],
       ["_01_",     [[1, 2], [2, 3]]],
       ["0____1",   [[0, 1], [5, 6]]],
       ["0_____1",  [[0, 1], [6, 7]]],
       ["____0_1",  [[4, 5], [6, 7]]],
       ["0______1", [[0, 1], [7, 8]]],
       ["___01___", [[3, 4], [4, 5]]]],
      FZF.sort(xmatcher.match(list, '0 1', '', '')))

    assert_equal(
      [["_01_",     [[1, 3], [0, 4]], [4, 4, "_01_"]],
       ["___01___", [[3, 5], [0, 2]], [4, 8, "___01___"]],
       ["____0_1",  [[4, 7], [0, 2]], [5, 7, "____0_1"]],
       ["0____1",   [[0, 6], [1, 3]], [6, 6, "0____1"]],
       ["0_____1",  [[0, 7], [1, 3]], [7, 7, "0_____1"]],
       ["0______1", [[0, 8], [1, 3]], [8, 8, "0______1"]]],
      FZF.sort(xmatcher.match(list, '01 __', '', '')).map { |tuple|
        tuple << FZF.rank(tuple)
      }
    )
  end

  def test_extended_exact_mode
    exact = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE, :exact
    fuzzy = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE, :fuzzy
    list = %w[
      extended-exact-mode-not-fuzzy
      extended'-fuzzy-mode
    ]
    assert_equal 2, fuzzy.match(list, 'extended', '', '').length
    assert_equal 2, fuzzy.match(list, 'mode extended', '', '').length
    assert_equal 2, fuzzy.match(list, 'xtndd', '', '').length
    assert_equal 2, fuzzy.match(list, "'-fuzzy", '', '').length

    assert_equal 2, exact.match(list, 'extended', '', '').length
    assert_equal 2, exact.match(list, 'mode extended', '', '').length
    assert_equal 0, exact.match(list, 'xtndd', '', '').length
    assert_equal 1, exact.match(list, "'-fuzzy", '', '').length
    assert_equal 2, exact.match(list, "-fuzzy", '', '').length
  end

  # ^$ -> matches empty item
  def test_format_empty_item
    fzf = FZF.new []
    item = ['', [[0, 0]]]
    line, offsets = item
    tokens        = fzf.format line, 80, offsets
    assert_equal [], tokens
  end

  def test_mouse_event
    interval = FZF::MouseEvent::DOUBLE_CLICK_INTERVAL
    me = FZF::MouseEvent.new nil
    me.v = 10
    assert_equal false, me.double?(10)
    assert_equal false, me.double?(20)
    me.v = 20
    assert_equal false, me.double?(10)
    assert_equal false, me.double?(20)
    me.v = 20
    assert_equal false, me.double?(10)
    assert_equal true,  me.double?(20)
    sleep interval
    assert_equal false,  me.double?(20)
  end

  def test_nth_match
    list = [
      ' first  second  third',
      'fourth	 fifth   sixth',
    ]

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE
    assert_equal list, matcher.match(list, 'f', '', '').map(&:first)
    assert_equal [
      [list[0], [[2,  5]]],
      [list[1], [[9, 17]]]], matcher.match(list, 'is', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1..1]
    assert_equal [[list[1], [[8, 9]]]], matcher.match(list, 'f', '', '')
    assert_equal [[list[0], [[8, 9]]]], matcher.match(list, 's', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [2..2]
    assert_equal [[list[0], [[19, 20]]]], matcher.match(list, 'r', '', '')

    # Comma-separated
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [2..2, 0..0]
    assert_equal [[list[0], [[19, 20]]], [list[1], [[3, 4]]]], matcher.match(list, 'r', '', '')

    # Ordered
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [0..0, 2..2]
    assert_equal [[list[0], [[3, 4]]], [list[1], [[3, 4]]]], matcher.match(list, 'r', '', '')

    regex = FZF.build_delim_regex "\t"
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [0..0], regex
    assert_equal [[list[0], [[3, 10]]]], matcher.match(list, 're', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1..1], regex
    assert_equal [], matcher.match(list, 'r', '', '')
    assert_equal [[list[1], [[9, 17]]]], matcher.match(list, 'is', '', '')

    # Negative indexing
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [-1..-1], regex
    assert_equal [[list[0], [[3, 6]]]], matcher.match(list, 'rt', '', '')
    assert_equal [[list[0], [[2, 5]]], [list[1], [[9, 17]]]], matcher.match(list, 'is', '', '')

    # Regex delimiter
    regex = FZF.build_delim_regex "[ \t]+"
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [0..0], regex
    assert_equal [list[1]], matcher.match(list, 'f', '', '').map(&:first)

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1..1], regex
    assert_equal [[list[0], [[1, 2]]], [list[1], [[8, 9]]]], matcher.match(list, 'f', '', '')
  end

  def test_nth_match_range
    list = [
      ' first  second  third',
      'fourth	 fifth   sixth',
    ]

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1..2]
    assert_equal [[list[0], [[8, 20]]]], matcher.match(list, 'sr', '', '')
    assert_equal [], matcher.match(list, 'fo', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1..-1, 0..0]
    assert_equal [[list[0], [[8, 20]]]], matcher.match(list, 'sr', '', '')
    assert_equal [[list[1], [[0, 2]]]], matcher.match(list, 'fo', '', '')

    matcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE, :fuzzy, [0..0, 1..2]
    assert_equal [], matcher.match(list, '^t', '', '')

    matcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE, :fuzzy, [0..1, 2..2]
    assert_equal [[list[0], [[16, 17]]]], matcher.match(list, '^t', '', '')

    matcher = FZF::ExtendedFuzzyMatcher.new Regexp::IGNORECASE, :fuzzy, [1..-1]
    assert_equal [[list[0], [[8, 9]]]], matcher.match(list, '^s', '', '')
  end

  def stream_for str, delay = 0
    StringIO.new(str).tap do |sio|
      sio.instance_eval do
        alias org_gets gets

        def gets
          org_gets.tap { |e| sleep(@delay) unless e.nil? }
        end

        def reopen _
        end
      end
      sio.instance_variable_set :@delay, delay
    end
  end

  def assert_fzf_output opts, given, expected
    stream = stream_for given
    output = stream_for ''

    def sorted_lines line
      line.split($/).sort
    end

    begin
      tty = MockTTY.new
      $stdout = output
      fzf = FZF.new(opts, stream)
      fzf.instance_variable_set :@tty, tty
      thr = block_given? && Thread.new { yield tty }
      fzf.start
      thr && thr.join
    rescue SystemExit => e
      assert_equal 0, e.status
      assert_equal sorted_lines(expected), sorted_lines(output.string)
    ensure
      $stdout = STDOUT
    end
  end

  def test_filter
    {
      %w[--filter=ol] => 'World',
      %w[--filter=ol --print-query] => "ol\nWorld",
    }.each do |opts, expected|
      assert_fzf_output opts, "Hello\nWorld", expected
    end
  end

  def test_select_1
    {
      %w[--query=ol --select-1] => 'World',
      %w[--query=ol --select-1 --print-query] => "ol\nWorld",
    }.each do |opts, expected|
      assert_fzf_output opts, "Hello\nWorld", expected
    end
  end

  def test_select_1_without_query
    assert_fzf_output %w[--select-1], 'Hello World', 'Hello World'
  end

  def test_select_1_ambiguity
    begin
      Timeout::timeout(0.5) do
        assert_fzf_output %w[--query=o --select-1], "hello\nworld", "should not match"
      end
    rescue Timeout::Error
      Curses.close_screen
    end
  end

  def test_exit_0
    {
      %w[--query=zz --exit-0] => '',
      %w[--query=zz --exit-0 --print-query] => 'zz',
    }.each do |opts, expected|
      assert_fzf_output opts, "Hello\nWorld", expected
    end
  end

  def test_exit_0_without_query
    assert_fzf_output %w[--exit-0], '', ''
  end

  def test_with_nth
    source = "hello world\nbatman"
    assert_fzf_output %w[-0 -1 --with-nth=2,1 -x -q ^worl],
      source, 'hello world'
    assert_fzf_output %w[-0 -1 --with-nth=2,1 -x -q llo$],
      source, 'hello world'
    assert_fzf_output %w[-0 -1 --with-nth=.. -x -q llo$],
      source, ''
    assert_fzf_output %w[-0 -1 --with-nth=2,2,2,..,1 -x -q worlworlworlhellworlhell],
      source, 'hello world'
    assert_fzf_output %w[-0 -1 --with-nth=1,1,-1,1 -x -q batbatbatbat],
      source, 'batman'
  end

  def test_with_nth_transform
    fzf = FZF.new %w[--with-nth 2..,1]
    assert_equal 'my world hello', fzf.transform('hello my world')
    assert_equal 'my   world hello', fzf.transform('hello   my   world')
    assert_equal 'my   world  hello', fzf.transform('hello   my   world  ')

    fzf = FZF.new %w[--with-nth 2,-1,2]
    assert_equal 'my world my', fzf.transform('hello my world')
    assert_equal 'world world world', fzf.transform('hello world')
    assert_equal 'world  world  world', fzf.transform('hello world  ')
  end

  def test_ranking_overlap_match_regions
    list = [
      '1           3   4      2',
      '1           2   3    4'
    ]
    assert_equal [
      ['1           2   3    4',   [[0, 13], [16, 22]]],
      ['1           3   4      2', [[0, 24], [12, 17]]],
    ], FZF.sort(FZF::ExtendedFuzzyMatcher.new(nil).match(list, '12 34', '', ''))
  end

  def test_constrain
    fzf = FZF.new []

    # [#****             ]
    assert_equal [false, 0, 0], fzf.constrain(0, 0, 5, 100)

    # *****[**#**  ...   ] => [**#*******  ... ]
    assert_equal [true, 0, 2], fzf.constrain(5, 7, 10, 100)

    # [**********]**#** => ***[*********#]**
    assert_equal [true, 3, 12], fzf.constrain(0, 12, 15, 10)

    # *****[**#**  ] => ***[**#****]
    assert_equal [true, 3, 5], fzf.constrain(5, 7, 10, 7)

    # *****[**#** ] => ****[**#***]
    assert_equal [true, 4, 6], fzf.constrain(5, 7, 10, 6)

    # *****  [#] => ****[#]
    assert_equal [true, 4, 4], fzf.constrain(10, 10, 5, 1)

    # [ ] #**** => [#]****
    assert_equal [true, 0, 0], fzf.constrain(-5, 0, 5, 1)

    # [ ] **#** => **[#]**
    assert_equal [true, 2, 2], fzf.constrain(-5, 2, 5, 1)

    # [*****  #] => [****#   ]
    assert_equal [true, 0, 4], fzf.constrain(0, 7, 5, 10)

    # **[*****  #] => [******# ]
    assert_equal [true, 0, 6], fzf.constrain(2, 10, 7, 10)
  end

  def test_invalid_utf8
    tmp = Tempfile.new('fzf')
    tmp << 'hello ' << [0xff].pack('C*') << ' world' << $/ << [0xff].pack('C*')
    tmp.close
    begin
      Timeout::timeout(0.5) do
        FZF.new(%w[-n..,1,2.. -q^ -x], File.open(tmp.path)).start
      end
    rescue Timeout::Error
      Curses.close_screen
    end
  ensure
    tmp.unlink
  end

  def test_with_nth_mock_tty
    # Manual selection with input
    assert_fzf_output ["--with-nth=2,1"], "hello world", "hello world" do |tty|
      tty << "world"
      tty << "hell"
      tty << "\r"
    end

    # Manual selection without input
    assert_fzf_output ["--with-nth=2,1"], "hello world", "hello world" do |tty|
      tty << "\r"
    end

    # Manual selection with input and --multi
    lines = "hello world\ngoodbye world"
    assert_fzf_output %w[-m --with-nth=2,1], lines, lines do |tty|
      tty << "o"
      tty << "\e[Z\e[Z"
      tty << "\r"
    end

    # Manual selection without input and --multi
    assert_fzf_output %w[-m --with-nth=2,1], lines, lines do |tty|
      tty << "\e[Z\e[Z"
      tty << "\r"
    end

    # ALT-D
    assert_fzf_output %w[--print-query], "", "hello  baby = world" do |tty|
      tty << "hello world baby"
      tty << alt(:b) << alt(:b) << alt(:d)
      tty << ctrl(:e) << " = " << ctrl(:y)
      tty << "\r"
    end

    # ALT-BACKSPACE
    assert_fzf_output %w[--print-query], "", "hello baby = world " do |tty|
      tty << "hello world baby"
      tty << alt(:b) << alt(127.chr)
      tty << ctrl(:e) << " = " << ctrl(:y)
      tty << "\r"
    end
  end

  def alt chr
    "\e#{chr}"
  end

  def ctrl char
    char.to_s.ord - 'a'.ord + 1
  end
end

