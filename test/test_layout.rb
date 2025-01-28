# frozen_string_literal: true

require_relative 'lib/common'

# Test cases that mainly use assert_block to verify the layout of fzf
class TestLayout < TestInteractive
  def assert_block(expected, lines)
    cols = expected.lines.map { it.chomp.length }.max
    top = lines.take(expected.lines.length).map { _1[0, cols].rstrip + "\n" }.join
    bottom = lines.reverse.take(expected.lines.length).reverse.map { _1[0, cols].rstrip + "\n" }.join
    assert_includes [top, bottom], expected
  end

  def test_vanilla
    tmux.send_keys "seq 1 100000 | #{fzf}", :Enter
    block = <<~BLOCK
        2
      > 1
        100000/100000
      >
    BLOCK
    tmux.until { assert_block(block, _1) }

    # Testing basic key bindings
    tmux.send_keys '99', 'C-a', '1', 'C-f', '3', 'C-b', 'C-h', 'C-u', 'C-e', 'C-y', 'C-k', 'Tab', 'BTab'
    block = <<~BLOCK
      > 3910
        391
        856/100000
      > 391
    BLOCK
    tmux.until { assert_block(block, _1) }

    tmux.send_keys :Enter
    assert_equal '3910', fzf_output
  end

  def test_header_first
    tmux.send_keys "seq 1000 | #{FZF} --header foobar --header-lines 3 --header-first", :Enter
    block = <<~OUTPUT
      > 4
        997/997
      >
        3
        2
        1
        foobar
    OUTPUT
    tmux.until { assert_block(block, _1) }
  end

  def test_header_first_reverse
    tmux.send_keys "seq 1000 | #{FZF} --header foobar --header-lines 3 --header-first --reverse --inline-info", :Enter
    block = <<~OUTPUT
        foobar
        1
        2
        3
      >   < 997/997
      > 4
    OUTPUT
    tmux.until { assert_block(block, _1) }
  end

  def test_change_and_transform_header
    [
      'space:change-header:$(seq 4)',
      'space:transform-header:seq 4'
    ].each_with_index do |binding, i|
      tmux.send_keys %(seq 3 | #{FZF} --header-lines 2 --header bar --bind "#{binding}"), :Enter
      expected = <<~OUTPUT
        > 3
          2
          1
          bar
          1/1
        >
      OUTPUT
      tmux.until { assert_block(expected, _1) }
      tmux.send_keys :Space
      expected = <<~OUTPUT
        > 3
          2
          1
          1
          2
          3
          4
          1/1
        >
      OUTPUT
      tmux.until { assert_block(expected, _1) }
      next unless i.zero?

      teardown
      setup
    end
  end

  def test_change_header
    tmux.send_keys %(seq 3 | #{FZF} --header-lines 2 --header bar --bind "space:change-header:$(seq 4)"), :Enter
    expected = <<~OUTPUT
      > 3
        2
        1
        bar
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, _1) }
    tmux.send_keys :Space
    expected = <<~OUTPUT
      > 3
        2
        1
        1
        2
        3
        4
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_reload_and_change_cache
    tmux.send_keys "echo bar | #{FZF} --bind 'zero:change-header(foo)+reload(echo foo)+clear-query'", :Enter
    expected = <<~OUTPUT
      > bar
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, _1) }
    tmux.send_keys :z
    expected = <<~OUTPUT
      > foo
        foo
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_toggle_header
    tmux.send_keys "seq 4 | #{FZF} --header-lines 2 --header foo --bind space:toggle-header --header-first --height 10 --border rounded", :Enter
    before = <<~OUTPUT
      ╭───────
      │
      │   4
      │ > 3
      │   2/2
      │ >
      │   2
      │   1
      │   foo
      ╰───────
    OUTPUT
    tmux.until { assert_block(before, _1) }
    tmux.send_keys :Space
    after = <<~OUTPUT
      ╭───────
      │
      │
      │
      │
      │   4
      │ > 3
      │   2/2
      │ >
      ╰───────
    OUTPUT
    tmux.until { assert_block(after, _1) }
    tmux.send_keys :Space
    tmux.until { assert_block(before, _1) }
  end

  def test_height_range_fit
    tmux.send_keys 'seq 3 | fzf --height ~100% --info=inline --border rounded', :Enter
    expected = <<~OUTPUT
      ╭──────────
      │   3
      │   2
      │ > 1
      │ >   < 3/3
      ╰──────────
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_height_range_fit_preview_above
    tmux.send_keys 'seq 3 | fzf --height ~100% --info=inline --border rounded --preview-window border-rounded --preview "seq {}" --preview-window up,60%', :Enter
    expected = <<~OUTPUT
      ╭──────────
      │ ╭────────
      │ │ 1
      │ │
      │ │
      │ │
      │ ╰────────
      │   3
      │   2
      │ > 1
      │ >   < 3/3
      ╰──────────
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_height_range_fit_preview_above_alternative
    tmux.send_keys 'seq 3 | fzf --height ~100% --border=sharp --preview "seq {}" --preview-window up,40%,border-bottom --padding 1 --exit-0 --header hello --header-lines=2', :Enter
    expected = <<~OUTPUT
      ┌─────────
      │
      │  1
      │  2
      │  3
      │  ───────
      │  > 3
      │    2
      │    1
      │    hello
      │    1/1 ─
      │  >
      │
      └─────────
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_height_range_fit_preview_left
    tmux.send_keys "seq 3 | fzf --height ~100% --border=vertical --preview 'seq {}' --preview-window left,5,border-right --padding 1 --exit-0 --header $'hello\\nworld' --header-lines=2", :Enter
    expected = <<~OUTPUT
      │
      │  1     │ > 3
      │  2     │   2
      │  3     │   1
      │        │   hello
      │        │   world
      │        │   1/1 ─
      │        │ >
      │
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_height_range_overflow
    tmux.send_keys 'seq 100 | fzf --height ~5 --info=inline --border rounded', :Enter
    expected = <<~OUTPUT
      ╭──────────────
      │   2
      │ > 1
      │ >   < 100/100
      ╰──────────────
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_no_extra_newline_issue_3209
    tmux.send_keys(%(seq 100 | #{FZF} --height 10 --preview-window up,wrap,border-rounded --preview 'printf "─%.0s" $(seq 1 "$((FZF_PREVIEW_COLUMNS - 5))"); printf $"\\e[7m%s\\e[0m" title; echo; echo something'), :Enter)
    expected = <<~OUTPUT
      ╭──────────
      │ ─────────
      │ something
      │
      ╰──────────
        3
        2
      > 1
        100/100 ─
      >
    OUTPUT
    tmux.until { assert_block(expected, _1) }
  end

  def test_fzf_multi_line
    tmux.send_keys %[(echo -en '0\\0'; echo -en '1\\n2\\0'; seq 1000) | fzf --read0 --multi --bind load:select-all --border rounded], :Enter
    block = <<~BLOCK
      │  ┃998
      │  ┃999
      │  ┃1000
      │  ╹
      │  ╻1
      │  ╹2
      │ >>0
      │   3/3 (3)
      │ >
      ╰───────────
    BLOCK
    tmux.until { assert_block(block, _1) }
    tmux.send_keys :Up, :Up
    block = <<~BLOCK
      ╭───────
      │ >╻1
      │ >┃2
      │ >┃3
    BLOCK
    tmux.until { assert_block(block, _1) }

    block = <<~BLOCK
      │ >┃
      │
      │ >
      ╰───
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_fzf_multi_line_reverse
    tmux.send_keys %[(echo -en '0\\0'; echo -en '1\\n2\\0'; seq 1000) | fzf --read0 --multi --bind load:select-all --border rounded --reverse], :Enter
    block = <<~BLOCK
      ╭───────────
      │ >
      │   3/3 (3)
      │ >>0
      │  ╻1
      │  ╹2
      │  ╻1
      │  ┃2
      │  ┃3
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_fzf_multi_line_no_pointer_and_marker
    tmux.send_keys %[(echo -en '0\\0'; echo -en '1\\n2\\0'; seq 1000) | fzf --read0 --multi --bind load:select-all --border rounded --reverse --pointer '' --marker '' --marker-multi-line ''], :Enter
    block = <<~BLOCK
      ╭───────────
      │ >
      │   3/3 (3)
      │ 0
      │ 1
      │ 2
      │ 1
      │ 2
      │ 3
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_gap
    tmux.send_keys %(seq 100 | #{FZF} --gap --border --reverse), :Enter
    block = <<~BLOCK
      ╭─────────────────
      │ >
      │   100/100 ──────
      │ > 1
      │   ┈┈┈┈┈┈┈┈┈┈┈┈┈┈
      │   2
      │   ┈┈┈┈┈┈┈┈┈┈┈┈┈┈
      │   3
      │   ┈┈┈┈┈┈┈┈┈┈┈┈┈┈
      │   4
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_gap_2
    tmux.send_keys %(seq 100 | #{FZF} --gap=2 --gap-line xyz --border --reverse), :Enter
    block = <<~BLOCK
      ╭─────────────────
      │ >
      │   100/100 ──────
      │ > 1
      │
      │   xyzxyzxyzxyzxy
      │   2
      │
      │   xyzxyzxyzxyzxy
      │   3
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_list_border_and_label
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --list-border double --list-label list --list-label-pos 2:bottom --header-lines 3 --query 1 --padding 1,2), :Enter
    block = <<~BLOCK
      │   ║   11
      │   ║ > 10
      │   ║   3
      │   ║   2
      │   ║   1
      │   ║   19/97 ─
      │   ║ > 1
      │   ╚list══════
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_input_border_and_label
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --input-border bold --input-label input --input-label-pos 2 --header-lines 3 --query 1 --padding 1,2), :Enter
    block = <<~BLOCK
      │     11
      │   > 10
      │     3
      │     2
      │     1
      │   ┏input━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_input_border_and_label_header_first
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --input-border bold --input-label input --input-label-pos 2 --header-lines 3 --query 1 --padding 1,2 --header-first), :Enter
    block = <<~BLOCK
      │     11
      │   > 10
      │   ┏input━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │     3
      │     2
      │     1
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_list_input_border_and_label
    tmux.send_keys %(
      seq 100 | #{FZF} --border rounded --list-border double --input-border bold --list-label-pos 2:bottom --input-label-pos 2 --header-lines 3 --query 1 --padding 1,2 \
      --bind 'start:transform-input-label(echo INPUT)+transform-list-label(echo LIST)' \
      --bind 'space:change-input-label( input )+change-list-label( list )'
    ).strip, :Enter
    block = <<~BLOCK
      │   ║   11
      │   ║ > 10
      │   ╚LIST══════
      │       3
      │       2
      │       1
      │   ┏INPUT━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
    tmux.send_keys :Space
    block = <<~BLOCK
      │   ║   11
      │   ║ > 10
      │   ╚ list ════
      │       3
      │       2
      │       1
      │   ┏ input ━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_list_input_border_and_label_header_first
    tmux.send_keys %(
      seq 100 | #{FZF} --border rounded --list-border double --input-border bold --list-label-pos 2:bottom --input-label-pos 2 --header-lines 3 --query 1 --padding 1,2 \
      --bind 'start:transform-input-label(echo INPUT)+transform-list-label(echo LIST)' \
      --bind 'space:change-input-label( input )+change-list-label( list )' --header-first
    ).strip, :Enter
    block = <<~BLOCK
      │   ║   11
      │   ║ > 10
      │   ╚LIST══════
      │   ┏INPUT━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │       3
      │       2
      │       1
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
    tmux.send_keys :Space
    block = <<~BLOCK
      │   ║   11
      │   ║ > 10
      │   ╚ list ════
      │   ┏ input ━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗━━━━━━━━━━
      │       3
      │       2
      │       1
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_header_border_and_label
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2), :Enter
    block = <<~BLOCK
      │     12
      │     11
      │   > 10
      │   ┌────────
      │   │ 3
      │   │ 2
      │   │ 1
      │   └header──
      │     19/97 ─
      │   > 1
      │
      ╰────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_header_border_toggle
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()'), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │   3
      │   2
      │ > 1
      │   100/100 ─
      │ >
      ╰────────────
    BLOCK
    tmux.until { assert_block(block1, _1) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │   3
      │   2
      │ > 1
      ╰────────────
      ╭────────────
      │   hello
      ╰────────────
          100/100 ─
        >
    BLOCK
    tmux.until { assert_block(block2, _1) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, _1) }
  end

  def test_header_border_toggle_with_header_lines
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()' --header-lines 2), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │ > 3
      ╰──────────
      ╭──────────
      │   2
      │   1
      ╰──────────
          98/98 ─
        >
    BLOCK
    tmux.until { assert_block(block1, _1) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │   4
      │ > 3
      ╰──────────
      ╭──────────
      │   2
      │   1
      │   hello
      ╰──────────
          98/98 ─
        >
    BLOCK
    tmux.until { assert_block(block2, _1) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, _1) }
  end

  def test_header_border_toggle_with_header_lines_header_first
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()' --header-lines 2 --header-first), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │ > 3
      ╰──────────
          98/98 ─
        >
      ╭──────────
      │   2
      │   1
      ╰──────────
    BLOCK
    tmux.until { assert_block(block1, _1) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │   4
      │ > 3
      ╰──────────
          98/98 ─
        >
      ╭──────────
      │   2
      │   1
      │   hello
      ╰──────────
    BLOCK
    tmux.until { assert_block(block2, _1) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, _1) }
  end

  def test_header_border_toggle_with_header_lines_header_lines_border
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()' --header-lines 2 --header-lines-border double), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │ > 3
      ╰──────────
      ╔══════════
      ║   2
      ║   1
      ╚══════════
          98/98 ─
        >
    BLOCK
    tmux.until { assert_block(block1, _1) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │ > 3
      ╰──────────
      ╔══════════
      ║   2
      ║   1
      ╚══════════
      ╭──────────
      │   hello
      ╰──────────
          98/98 ─
        >
    BLOCK
    tmux.until { assert_block(block2, _1) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, _1) }
  end

  def test_header_border_toggle_with_header_lines_header_first_header_lines_border
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()' --header-lines 2 --header-first --header-lines-border double), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │ > 3
      ╰──────────
      ╔══════════
      ║   2
      ║   1
      ╚══════════
          98/98 ─
        >
    BLOCK
    tmux.until { assert_block(block1, _1) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │ > 3
      ╰──────────
      ╔══════════
      ║   2
      ║   1
      ╚══════════
          98/98 ─
        >
      ╭──────────
      │   hello
      ╰──────────
    BLOCK
    tmux.until { assert_block(block2, _1) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, _1) }
  end

  def test_header_border_and_label_header_first
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2 --header-first), :Enter
    block = <<~BLOCK
      │     12
      │     11
      │   > 10
      │     19/97 ─
      │   > 1
      │   ┌────────
      │   │ 3
      │   │ 2
      │   │ 1
      │   └header──
      │
      ╰────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_header_border_and_label_with_list_border
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --list-border double --list-label list --list-label-pos 2:bottom --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2), :Enter
    block = <<~BLOCK
      │   ║   12
      │   ║   11
      │   ║ > 10
      │   ╚list══════
      │   ┌──────────
      │   │   3
      │   │   2
      │   │   1
      │   └header────
      │       19/97 ─
      │     > 1
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_header_border_and_label_with_list_border_header_first
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --list-border double --list-label list --list-label-pos 2:bottom --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2 --header-first), :Enter
    block = <<~BLOCK
      │   ║   12
      │   ║   11
      │   ║ > 10
      │   ╚list══════
      │       19/97 ─
      │     > 1
      │   ┌──────────
      │   │   3
      │   │   2
      │   │   1
      │   └header────
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_all_borders
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --list-border double --list-label list --list-label-pos 2:bottom --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2 --input-border bold --input-label input --input-label-pos 2:bottom), :Enter
    block = <<~BLOCK
      │   ║   12
      │   ║   11
      │   ║ > 10
      │   ╚list══════
      │   ┌──────────
      │   │   3
      │   │   2
      │   │   1
      │   └header────
      │   ┏━━━━━━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗input━━━━━
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_all_borders_header_first
    tmux.send_keys %(seq 100 | #{FZF} --border rounded --list-border double --list-label list --list-label-pos 2:bottom --header-lines 3 --header-border sharp --header-label header --header-label-pos 2:bottom --query 1 --padding 1,2 --input-border bold --input-label input --input-label-pos 2:bottom --header-first), :Enter
    block = <<~BLOCK
      │   ║   12
      │   ║   11
      │   ║ > 10
      │   ╚list══════
      │   ┏━━━━━━━━━━
      │   ┃   19/97
      │   ┃ > 1
      │   ┗input━━━━━
      │   ┌──────────
      │   │   3
      │   │   2
      │   │   1
      │   └header────
      │
      ╰──────────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_style_full_adaptive_height
    tmux.send_keys %(seq 1| #{FZF} --style=full --height=~100% --header-lines=1 --info=default), :Enter
    block = <<~BLOCK
      ╭────────
      ╰────────
      ╭────────
      │   1
      ╰────────
      ╭────────
      │   0/0
      │ >
      ╰────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_style_full_adaptive_height_double
    tmux.send_keys %(seq 1| #{FZF} --style=full:double --border --height=~100% --header-lines=1 --info=default), :Enter
    block = <<~BLOCK
      ╔══════════
      ║ ╔════════
      ║ ╚════════
      ║ ╔════════
      ║ ║   1
      ║ ╚════════
      ║ ╔════════
      ║ ║   0/0
      ║ ║ >
      ║ ╚════════
      ╚══════════
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_preview_window_noinfo
    # │ 1        ││
    tmux.send_keys %(#{FZF} --preview 'seq 1000' --preview-window top,noinfo --scrollbar --bind space:change-preview-window:info), :Enter
    tmux.until do |lines|
      assert lines[1]&.start_with?('│ 1')
      assert lines[1]&.end_with?('  ││')
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines[1]&.start_with?('│ 1')
      assert lines[1]&.end_with?('1000││')
    end
  end

  def test_min_height_no_auto
    tmux.send_keys %(seq 100 | #{FZF} --border sharp --style full:sharp --height 1% --min-height 5), :Enter

    block = <<~BLOCK
      ┌───────
      │ ┌─────
      │ │ >
      │ └─────
      └───────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end

  def test_min_height_auto
    tmux.send_keys %(seq 100 | #{FZF} --style full:sharp --height 1% --min-height 5+), :Enter

    block = <<~BLOCK
      ┌─────────
      │   5
      │   4
      │   3
      │   2
      │ > 1
      └─────────
      ┌─────────
      │ >
      └─────────
    BLOCK
    tmux.until { assert_block(block, _1) }
  end
end
