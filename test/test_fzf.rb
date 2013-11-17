#!/usr/bin/env ruby
# encoding: utf-8

require 'minitest/autorun'
$LOAD_PATH.unshift File.expand_path('../..', __FILE__)
ENV['FZF_EXECUTABLE'] = '0'
load 'fzf'

class TestFZF < MiniTest::Unit::TestCase
  def test_default_options
    fzf = FZF.new []
    assert_equal 1000, fzf.sort
    assert_equal false, fzf.multi
    assert_equal true, fzf.color
    assert_equal Regexp::IGNORECASE, fzf.rxflag

    begin
      ENV['FZF_DEFAULT_SORT'] = '1500'
      fzf = FZF.new []
      assert_equal 1500, fzf.sort
    ensure
      ENV.delete 'FZF_DEFAULT_SORT'
    end
  end

  def test_option_parser
    # Long opts
    fzf = FZF.new %w[--sort=2000 --no-color --multi +i]
    assert_equal 2000, fzf.sort
    assert_equal true, fzf.multi
    assert_equal false, fzf.color
    assert_equal 0, fzf.rxflag

    # Short opts
    fzf = FZF.new %w[-s 2000 +c -m +i]
    assert_equal 2000, fzf.sort
    assert_equal true, fzf.multi
    assert_equal false, fzf.color
    assert_equal 0, fzf.rxflag
  end

  def test_invalid_option
    [%w[-s 2000 +s], %w[yo dawg]].each do |argv|
      assert_raises(SystemExit) do
        fzf = FZF.new argv
      end
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

  def test_fuzzy_matcher_case_sensitive
    assert_equal [['Fruit', [[0, 5]]]],
      FZF::FuzzyMatcher.new(0).match(%w[Fruit Grapefruit], 'Fruit', '', '').sort

    assert_equal [["Fruit", [[0, 5]]], ["Grapefruit", [[5, 10]]]],
      FZF::FuzzyMatcher.new(Regexp::IGNORECASE).
      match(%w[Fruit Grapefruit], 'Fruit', '', '').sort
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
      FZF.new([]).sort_by_rank(matcher.match(list, '01', '', '')))

    assert_equal(
      [["01",       [[0, 1], [1, 2]]],
       ["01_",      [[0, 1], [1, 2]]],
       ["_01_",     [[1, 2], [2, 3]]],
       ["0____1",   [[0, 1], [5, 6]]],
       ["0_____1",  [[0, 1], [6, 7]]],
       ["____0_1",  [[4, 5], [6, 7]]],
       ["0______1", [[0, 1], [7, 8]]],
       ["___01___", [[3, 4], [4, 5]]]],
      FZF.new([]).sort_by_rank(xmatcher.match(list, '0 1', '', '')))

    assert_equal(
      [["_01_",     [[1, 3], [0, 4]]],
       ["0____1",   [[0, 6], [1, 3]]],
       ["0_____1",  [[0, 7], [1, 3]]],
       ["0______1", [[0, 8], [1, 3]]],
       ["___01___", [[3, 5], [0, 2]]],
       ["____0_1",  [[4, 7], [0, 2]]]],
      FZF.new([]).sort_by_rank(xmatcher.match(list, '01 __', '', '')))
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
end

