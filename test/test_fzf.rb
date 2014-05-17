#!/usr/bin/env ruby
# encoding: utf-8

require 'curses'
require 'timeout'
require 'stringio'
require 'minitest/autorun'
$LOAD_PATH.unshift File.expand_path('../..', __FILE__)
ENV['FZF_EXECUTABLE'] = '0'
load 'fzf'

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
    assert_equal true,  fzf.color
    assert_equal false, fzf.black
    assert_equal true,  fzf.ansi256
    assert_equal '',    fzf.query.get
    assert_equal false, fzf.select1
    assert_equal false, fzf.exit0
    assert_equal nil,   fzf.filter
    assert_equal nil,   fzf.extended
    assert_equal false, fzf.reverse
  end

  def test_environment_variables
    # Deprecated
    ENV['FZF_DEFAULT_SORT'] = '20000'
    fzf = FZF.new []
    assert_equal 20000, fzf.sort
    assert_equal nil,   fzf.nth

    ENV['FZF_DEFAULT_OPTS'] =
      '-x -m -s 10000 -q "  hello  world  " +c +2 --select-1 -0 ' +
      '--no-mouse -f "goodbye world" --black --nth=3,-1,2 --reverse'
    fzf = FZF.new []
    assert_equal 10000,   fzf.sort
    assert_equal '  hello  world  ',
                          fzf.query.get
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
    assert_equal [3, -1, 2], fzf.nth
  end

  def test_option_parser
    # Long opts
    fzf = FZF.new %w[--sort=2000 --no-color --multi +i --query hello --select-1
                     --exit-0 --filter=howdy --extended-exact
                     --no-mouse --no-256 --nth=1 --reverse]
    assert_equal 2000,    fzf.sort
    assert_equal true,    fzf.multi
    assert_equal false,   fzf.color
    assert_equal false,   fzf.ansi256
    assert_equal false,   fzf.black
    assert_equal false,   fzf.mouse
    assert_equal 0,       fzf.rxflag
    assert_equal 'hello', fzf.query.get
    assert_equal true,    fzf.select1
    assert_equal true,    fzf.exit0
    assert_equal 'howdy', fzf.filter
    assert_equal :exact,  fzf.extended
    assert_equal [1],     fzf.nth
    assert_equal true,    fzf.reverse

    # Long opts (left-to-right)
    fzf = FZF.new %w[--sort=2000 --no-color --multi +i --query=hello
                     --filter a --filter b --no-256 --black --nth -1 --nth -2
                     --select-1 --exit-0 --no-select-1 --no-exit-0
                     --no-sort -i --color --no-multi --256
                     --reverse --no-reverse]
    assert_equal nil,     fzf.sort
    assert_equal false,   fzf.multi
    assert_equal true,    fzf.color
    assert_equal true,    fzf.ansi256
    assert_equal true,    fzf.black
    assert_equal true,    fzf.mouse
    assert_equal 1,       fzf.rxflag
    assert_equal 'b',     fzf.filter
    assert_equal 'hello', fzf.query.get
    assert_equal false,   fzf.select1
    assert_equal false,   fzf.exit0
    assert_equal nil,     fzf.extended
    assert_equal [-2],    fzf.nth
    assert_equal false,   fzf.reverse

    # Short opts
    fzf = FZF.new %w[-s2000 +c -m +i -qhello -x -fhowdy +2 -n3 -1 -0]
    assert_equal 2000,    fzf.sort
    assert_equal true,    fzf.multi
    assert_equal false,   fzf.color
    assert_equal false,   fzf.ansi256
    assert_equal 0,       fzf.rxflag
    assert_equal 'hello', fzf.query.get
    assert_equal 'howdy', fzf.filter
    assert_equal :fuzzy,  fzf.extended
    assert_equal [3],     fzf.nth
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
    assert_equal 'world', fzf.query.get
    assert_equal false,   fzf.select1
    assert_equal false,   fzf.exit0
    assert_equal 'world', fzf.filter
    assert_equal nil,     fzf.extended
    assert_equal [4, 5],  fzf.nth
  rescue SystemExit => e
    assert false, "Exited"
  end

  def test_invalid_option
    [%w[--unknown], %w[yo dawg]].each do |argv|
      assert_raises(SystemExit) do
        fzf = FZF.new argv
      end
    end
    assert_raises(SystemExit) do
      fzf = FZF.new %w[--nth=0]
    end
    assert_raises(SystemExit) do
      fzf = FZF.new %w[-n 0]
    end
  end

  # FIXME Only on 1.9 or above
  def test_width
    fzf = FZF.new []
    assert_equal 5, fzf.width('abcde')
    assert_equal 4, fzf.width('한글')
    assert_equal 5, fzf.width('한글.')
  end

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
  end

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

  if RUBY_PLATFORM =~ /darwin/
    NFD = '한글'
    def test_nfc
      assert_equal 6, NFD.length
      assert_equal ["한글", [[0, 1], [1, 2]]],
        FZF::UConv.nfc(NFD, [[0, 3], [3, 6]])

      nfd2 = 'before' + NFD + 'after'
      assert_equal 6 + 6 + 5, nfd2.length

      nfc, offsets = FZF::UConv.nfc(nfd2, [[4, 14], [9, 13]])
      o1, o2 = offsets
      assert_equal 'before한글after', nfc
      assert_equal 're한글af',        nfc[(o1.first...o1.last)]
      assert_equal '글a',             nfc[(o2.first...o2.last)]
    end

    def test_nfd
      nfc = '한글'
      nfd = FZF::UConv.nfd(nfc)
      assert_equal 2, nfd.length
      assert_equal 6, nfd.join.length
      assert_equal NFD, nfd.join
    end

    def test_nfd_fuzzy_matcher
      matcher = FZF::FuzzyMatcher.new 0
      assert_equal [], matcher.match([NFD + NFD], '할', '', '')
      match   = matcher.match([NFD + NFD], '글글', '', '')
      assert_equal [[NFD + NFD, [[3, 12]]]], match
      assert_equal ['한글한글', [[1, 4]]], FZF::UConv.nfc(*match.first)
    end

    def test_nfd_extended_fuzzy_matcher
      matcher = FZF::ExtendedFuzzyMatcher.new 0
      assert_equal [], matcher.match([NFD], "'글글", '', '')
      match   = matcher.match([NFD], "'한글", '', '')
      assert_equal [[NFD, [[0, 6]]]], match
      assert_equal ['한글', [[0, 2]]], FZF::UConv.nfc(*match.first)
    end
  end

  def test_split
    assert_equal ["a", "b", "c", "\xFF", "d", "e", "f"],
      FZF::UConv.split("abc\xFFdef")
  end

  # ^$ -> matches empty item
  def test_format_empty_item
    fzf = FZF.new []
    item = ['', [[0, 0]]]
    line, offsets = fzf.convert_item item
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

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [2]
    assert_equal [[list[1], [[8, 9]]]], matcher.match(list, 'f', '', '')
    assert_equal [[list[0], [[8, 9]]]], matcher.match(list, 's', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [3]
    assert_equal [[list[0], [[19, 20]]]], matcher.match(list, 'r', '', '')

    # Comma-separated
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [3, 1]
    assert_equal [[list[0], [[19, 20]]], [list[1], [[3, 4]]]], matcher.match(list, 'r', '', '')

    # Ordered
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1, 3]
    assert_equal [[list[0], [[3, 4]]], [list[1], [[3, 4]]]], matcher.match(list, 'r', '', '')

    regex = FZF.build_delim_regex "\t"
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1], regex
    assert_equal [[list[0], [[3, 10]]]], matcher.match(list, 're', '', '')

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [2], regex
    assert_equal [], matcher.match(list, 'r', '', '')
    assert_equal [[list[1], [[9, 17]]]], matcher.match(list, 'is', '', '')

    # Negative indexing
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [-1], regex
    assert_equal [[list[0], [[3, 6]]]], matcher.match(list, 'rt', '', '')
    assert_equal [[list[0], [[2, 5]]], [list[1], [[9, 17]]]], matcher.match(list, 'is', '', '')

    # Regex delimiter
    regex = FZF.build_delim_regex "[ \t]+"
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [1], regex
    assert_equal [list[1]], matcher.match(list, 'f', '', '').map(&:first)

    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE, [2], regex
    assert_equal [[list[0], [[1, 2]]], [list[1], [[8, 9]]]], matcher.match(list, 'f', '', '')
  end

  def stream_for str
    StringIO.new(str).tap do |sio|
      sio.instance_eval do
        alias org_gets gets

        def gets
          org_gets.tap { |e| sleep 0.5 unless e.nil? }
        end
      end
    end
  end

  def test_select_1
    stream = stream_for "Hello\nWorld"
    output = StringIO.new

    begin
      $stdout = output
      FZF.new(%w[--query=ol --select-1], stream).start
    rescue SystemExit => e
      assert_equal 0, e.status
      assert_equal 'World', output.string.chomp
    ensure
      $stdout = STDOUT
    end
  end

  def test_select_1_without_query
    stream = stream_for "Hello World"
    output = StringIO.new

    begin
      $stdout = output
      FZF.new(%w[--select-1], stream).start
    rescue SystemExit => e
      assert_equal 0, e.status
      assert_equal 'Hello World', output.string.chomp
    ensure
      $stdout = STDOUT
    end
  end

  def test_select_1_ambiguity
    stream = stream_for "Hello\nWorld"
    begin
      Timeout::timeout(3) do
        FZF.new(%w[--query=o --select-1], stream).start
      end
      flunk 'Should not reach here'
    rescue Exception => e
      Curses.close_screen
      assert_instance_of Timeout::Error, e
    end
  end

  def test_exit_0
    stream = stream_for "Hello\nWorld"
    output = StringIO.new

    begin
      $stdout = output
      FZF.new(%w[--query=zz --exit-0], stream).start
    rescue SystemExit => e
      assert_equal 0, e.status
      assert_equal '', output.string
    ensure
      $stdout = STDOUT
    end
  end

  def test_exit_0_without_query
    stream = stream_for ""
    output = StringIO.new

    begin
      $stdout = output
      FZF.new(%w[--exit-0], stream).start
    rescue SystemExit => e
      assert_equal 0, e.status
      assert_equal '', output.string
    ensure
      $stdout = STDOUT
    end
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
end

