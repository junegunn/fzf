# frozen_string_literal: true

require_relative 'lib/common'

# Non-interactive tests
class TestFilter < TestBase
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

  def test_smart_case_for_each_term
    assert_equal 1, `echo Foo bar | #{FZF} -x -f "foo Fbar" | wc -l`.to_i
  end

  def test_filter_exitstatus
    # filter / streaming filter
    ['', '--no-sort'].each do |opts|
      assert_includes `echo foo | #{FZF} -f foo #{opts}`, 'foo'
      assert_equal 0, $CHILD_STATUS.exitstatus

      assert_empty `echo foo | #{FZF} -f bar #{opts}`
      assert_equal 1, $CHILD_STATUS.exitstatus
    end
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

  def test_nth_suffix_match
    assert_equal \
      'foo,bar,baz',
      `echo foo,bar,baz | #{FZF} -d, -f'bar$' -n2`.chomp
  end

  def test_with_nth_basic
    writelines(['hello world ', 'byebye'])
    assert_equal \
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 < #{tempname}`.chomp
  end

  def test_with_nth_template
    writelines(['hello world ', 'byebye'])
    assert_equal \
      'hello world ',
      `#{FZF} -f"^he he.he." -x -n 2.. --with-nth '{2} {1}. {1}.' < #{tempname}`.chomp
  end

  def test_with_nth_ansi
    writelines(["\x1b[33mhello \x1b[34;1mworld\x1b[m ", 'byebye'])
    assert_equal \
      'hello world ',
      `#{FZF} -f"^he hehe" -x -n 2.. --with-nth 2,1,1 --ansi < #{tempname}`.chomp
  end

  def test_with_nth_no_ansi
    src = "\x1b[33mhello \x1b[34;1mworld\x1b[m "
    writelines([src, 'byebye'])
    assert_equal \
      src,
      `#{FZF} -fhehe -x -n 2.. --with-nth 2,1,1 --no-ansi < #{tempname}`.chomp
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
    writelines(input)

    assert_equal input.length, `#{FZF} -f'foo bar' < #{tempname}`.lines.length
    assert_equal input.length - 1, `#{FZF} -f'^foo bar$' < #{tempname}`.lines.length
    assert_equal ['foo bar'], `#{FZF} -f'foo\\ bar' < #{tempname}`.lines(chomp: true)
    assert_equal ['foo bar'], `#{FZF} -f'^foo\\ bar$' < #{tempname}`.lines(chomp: true)
    assert_equal input.length - 1, `#{FZF} -f'!^foo\\ bar$' < #{tempname}`.lines.length
  end

  def test_normalized_match
    echoes = '(echo a; echo á; echo A; echo Á;)'
    assert_equal %w[a á A Á], `#{echoes} | #{FZF} -f a`.lines.map(&:chomp)
    assert_equal %w[á Á], `#{echoes} | #{FZF} -f á`.lines.map(&:chomp)
    assert_equal %w[A Á], `#{echoes} | #{FZF} -f A`.lines.map(&:chomp)
    assert_equal %w[Á], `#{echoes} | #{FZF} -f Á`.lines.map(&:chomp)
  end

  def test_unicode_case
    writelines(%w[строКА1 СТРОКА2 строка3 Строка4])
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
    writelines(input)

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
    writelines([
                 'xoxxxxxoxx',
                 'xoxxxxxox',
                 'xxoxxxoxx',
                 'xxxoxoxxx',
                 'xxxxoxox',
                 '  xxoxoxxx'
               ])

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
    writelines(['baz foo bar',
                'foo bar baz'])
    assert_equal [
      'foo bar baz',
      'baz foo bar'
    ], `#{FZF} -fbar --tiebreak=begin --algo=v2 < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_end
    writelines(['xoxxxxxxxx',
                'xxoxxxxxxx',
                'xxxoxxxxxx',
                'xxxxoxxxx',
                'xxxxxoxxx',
                '  xxxxoxxx'])

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

    writelines(['/bar/baz', '/foo/bar/baz'])
    assert_equal [
      '/foo/bar/baz',
      '/bar/baz'
    ], `#{FZF} -fbaz --tiebreak=end < #{tempname}`.lines(chomp: true)
  end

  def test_tiebreak_length_with_nth
    input = %w[
      1:hell
      123:hello
      12345:he
      1234567:h
    ]
    writelines(input)

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

  def test_tiebreak_chunk
    writelines(['1 foobarbaz ba',
                '2 foobar baz',
                '3 foo barbaz'])

    assert_equal [
      '3 foo barbaz',
      '2 foobar baz',
      '1 foobarbaz ba'
    ], `#{FZF} -fo --tiebreak=chunk < #{tempname}`.lines(chomp: true)

    assert_equal [
      '1 foobarbaz ba',
      '2 foobar baz',
      '3 foo barbaz'
    ], `#{FZF} -fba --tiebreak=chunk < #{tempname}`.lines(chomp: true)

    assert_equal [
      '3 foo barbaz'
    ], `#{FZF} -f'!foobar' --tiebreak=chunk < #{tempname}`.lines(chomp: true)
  end

  def test_boundary_match
    # Underscore boundaries should be ranked lower
    {
      default: [' xyz '] + %w[/xyz/ [xyz] -xyz- -xyz_ _xyz- _xyz_],
      path: ['/xyz/', ' xyz '] + %w[[xyz] -xyz- -xyz_ _xyz- _xyz_],
      history: ['[xyz]', '-xyz-', ' xyz '] + %w[/xyz/ -xyz_ _xyz- _xyz_]
    }.each do |scheme, expected|
      result = `printf -- 'xxyzx\n-xxyz\nxyzx-\n_xyz_\n_xyz-\n-xyz_\n[xyz]\n-xyz-\n xyz \n/xyz/\n' | #{FZF} -f"'xyz'" --scheme=#{scheme}`.lines(chomp: true)
      assert_equal expected, result
    end
  end

  def test_accept_nth
    # Single field selection
    assert_equal 'three', `echo 'one two three' | #{FZF} -d' ' --with-nth 1 --accept-nth -1 -f one`.chomp

    # Multiple field selection
    writelines(['ID001:John:Developer', 'ID002:Jane:Manager', 'ID003:Bob:Designer'])
    assert_equal 'ID001', `#{FZF} -d: --with-nth 2 --accept-nth 1 -f John < #{tempname}`.chomp
    assert_equal "ID002:Manager", `#{FZF} -d: --with-nth 2 --accept-nth 1,3 -f Jane < #{tempname}`.chomp

    # Test with different delimiters
    writelines(['emp001 Alice Engineering', 'emp002 Bob Marketing'])
    assert_equal 'emp001', `#{FZF} -d' ' --with-nth 2 --accept-nth 1 -f Alice < #{tempname}`.chomp
  end
end
