#!/usr/bin/env ruby
# encoding: utf-8

require 'minitest/autorun'
$LOAD_PATH.unshift File.expand_path('../..', __FILE__)
load 'fzf'

class TestFZF < MiniTest::Unit::TestCase
  def test_default_options
    fzf = FZF.new []
    assert_equal 500, fzf.sort
    assert_equal false, fzf.multi
    assert_equal true, fzf.color
    assert_equal Regexp::IGNORECASE, fzf.rxflag

    begin
      ENV['FZF_DEFAULT_SORT'] = '1000'
      fzf = FZF.new []
      assert_equal 1000, fzf.sort
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
    assert_equal ['사.',   6], fzf.trim('가나다라마바사.', 4, true)
    assert_equal ['바사.', 5], fzf.trim('가나다라마바사.', 5, true)
    assert_equal ['가나',  6], fzf.trim('가나다라마바사.', 4, false)
    assert_equal ['가나',  6], fzf.trim('가나다라마바사.', 5, false)
    assert_equal ['가나a', 6], fzf.trim('가나ab라마바사.', 5, false)
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
    assert matcher.cache.empty?
    assert_equal(
      [["juice",     [[0, 1]]],
       ["juiceful",  [[0, 1]]],
       ["juiceless", [[0, 1]]],
       ["juicily",   [[0, 1]]],
       ["juiciness", [[0, 1]]],
       ["juicy",     [[0, 1]]]], matcher.match(list, 'j', '', '').sort)
    assert !matcher.cache.empty?
    assert_equal [list.object_id], matcher.cache.keys
    assert_equal 1, matcher.cache[list.object_id].length
    assert_equal 6, matcher.cache[list.object_id]['j'].length

    assert_equal(
      [["juicily",   [[0, 5]]],
       ["juiciness", [[0, 5]]]], matcher.match(list, 'jii', '', '').sort)

    assert_equal(
      [["juicily",   [[2, 5]]],
       ["juiciness", [[2, 5]]]], matcher.match(list, 'ii', '', '').sort)

    assert_equal 3, matcher.cache[list.object_id].length
    assert_equal 2, matcher.cache[list.object_id]['ii'].length

    # TODO : partial_cache
  end

  def test_sort_by_rank
    matcher = FZF::FuzzyMatcher.new Regexp::IGNORECASE
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
    assert_equal %w[01 01_ _01_ ___01___ ____0_1 0____1 0_____1 0______1],
        FZF.new([]).sort_by_rank(matcher.match(list, '01', '', '')).map(&:first)
  end
end
