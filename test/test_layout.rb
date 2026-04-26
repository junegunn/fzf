# frozen_string_literal: true

require_relative 'lib/common'

# Test cases that mainly use assert_block to verify the layout of fzf
class TestLayout < TestInteractive
  def assert_block(expected, lines)
    cols = expected.lines.map { it.chomp.length }.max
    top = lines.take(expected.lines.length).map { it[0, cols].rstrip + "\n" }.join.chomp
    bottom = lines.reverse.take(expected.lines.length).reverse.map { it[0, cols].rstrip + "\n" }.join.chomp
    assert_includes [top, bottom], expected.chomp
  end

  def test_vanilla
    tmux.send_keys "seq 1 100000 | #{fzf}", :Enter
    block = <<~BLOCK
        2
      > 1
        100000/100000
      >
    BLOCK
    tmux.until { assert_block(block, it) }

    # Testing basic key bindings
    tmux.send_keys '99', 'C-a', '1', 'C-f', '3', 'C-b', 'C-h', 'C-u', 'C-e', 'C-y', 'C-k', 'Tab', 'BTab'
    block = <<~BLOCK
      > 3910
        391
        856/100000
      > 391
    BLOCK
    tmux.until { assert_block(block, it) }

    tmux.send_keys :Enter
    assert_equal '3910', fzf_output
  end

  def test_header_first
    tmux.send_keys "seq 1000 | #{FZF} --header foobar --header-lines 3 --header-first", :Enter
    block = <<~OUTPUT
      > 4
        3
        2
        1
        997/997
      >
        foobar
    OUTPUT
    tmux.until { assert_block(block, it) }
  end

  def test_header_first_reverse
    tmux.send_keys "seq 1000 | #{FZF} --header foobar --header-lines 3 --header-first --reverse --inline-info", :Enter
    block = <<~OUTPUT
        foobar
      >   < 997/997
        1
        2
        3
      > 4
    OUTPUT
    tmux.until { assert_block(block, it) }
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
      tmux.until { assert_block(expected, it) }
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
      tmux.until { assert_block(expected, it) }
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
    tmux.until { assert_block(expected, it) }
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
    tmux.until { assert_block(expected, it) }
  end

  def test_reload_and_change_cache
    tmux.send_keys "echo bar | #{FZF} --bind 'zero:change-header(foo)+reload(echo foo)+clear-query'", :Enter
    expected = <<~OUTPUT
      > bar
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, it) }
    tmux.send_keys :z
    expected = <<~OUTPUT
      > foo
        foo
        1/1
      >
    OUTPUT
    tmux.until { assert_block(expected, it) }
  end

  def test_toggle_header
    tmux.send_keys "seq 4 | #{FZF} --header-lines 2 --header foo --bind space:toggle-header --header-first --height 10 --border rounded", :Enter
    before = <<~OUTPUT
      ╭───────
      │
      │   4
      │ > 3
      │   2
      │   1
      │   2/2
      │ >
      │   foo
      ╰───────
    OUTPUT
    tmux.until { assert_block(before, it) }
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
    tmux.until { assert_block(after, it) }
    tmux.send_keys :Space
    tmux.until { assert_block(before, it) }
  end

  def test_height_range_fit
    tmux.send_keys 'seq 3 | fzf --height ~100% --info=inline --border rounded', :Enter
    expected = <<~OUTPUT
      ╭──────────
      │ ▌ 3
      │ ▌ 2
      │ > 1
      │ >   < 3/3
      ╰──────────
    OUTPUT
    tmux.until { assert_block(expected, it) }
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
      │ ▌ 3
      │ ▌ 2
      │ > 1
      │ >   < 3/3
      ╰──────────
    OUTPUT
    tmux.until { assert_block(expected, it) }
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
    tmux.until { assert_block(expected, it) }
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
    tmux.until { assert_block(expected, it) }
  end

  def test_height_range_overflow
    tmux.send_keys 'seq 100 | fzf --height ~5 --info=inline --border rounded', :Enter
    expected = <<~OUTPUT
      ╭──────────────
      │ ▌ 2
      │ > 1
      │ >   < 100/100
      ╰──────────────
    OUTPUT
    tmux.until { assert_block(expected, it) }
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
    tmux.until { assert_block(expected, it) }
  end

  def test_fzf_multi_line
    tmux.send_keys %[(echo -en '0\\0'; echo -en '1\\n2\\0'; seq 1000) | fzf --read0 --multi --bind load:select-all --border rounded], :Enter
    block = <<~BLOCK
      │ ▌┃998
      │ ▌┃999
      │ ▌┃1000
      │ ▌╹
      │ ▌╻1
      │ ▌╹2
      │ >>0
      │   3/3 (3)
      │ >
      ╰───────────
    BLOCK
    tmux.until { assert_block(block, it) }
    tmux.send_keys :Up, :Up
    block = <<~BLOCK
      ╭───────
      │ >╻1
      │ >┃2
      │ >┃3
    BLOCK
    tmux.until { assert_block(block, it) }

    block = <<~BLOCK
      │ >┃
      │
      │ >
      ╰───
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_fzf_multi_line_reverse
    tmux.send_keys %[(echo -en '0\\0'; echo -en '1\\n2\\0'; seq 1000) | fzf --read0 --multi --bind load:select-all --border rounded --reverse], :Enter
    block = <<~BLOCK
      ╭───────────
      │ >
      │   3/3 (3)
      │ >>0
      │ ▌╻1
      │ ▌╹2
      │ ▌╻1
      │ ▌┃2
      │ ▌┃3
    BLOCK
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
  end

  def test_gap
    tmux.send_keys %(seq 100 | #{FZF} --gap --border rounded --reverse), :Enter
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
    tmux.until { assert_block(block, it) }
  end

  def test_gap_2
    tmux.send_keys %(seq 100 | #{FZF} --gap=2 --gap-line xyz --border rounded --reverse), :Enter
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block1, it) }

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
    tmux.until { assert_block(block2, it) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, it) }
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
    tmux.until { assert_block(block1, it) }

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
    tmux.until { assert_block(block2, it) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, it) }
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
    tmux.until { assert_block(block1, it) }

    tmux.send_keys :Space
    block2 = <<~BLOCK
      │   4
      │ > 3
      ╰──────────
          2
          1
          98/98 ─
        >
      ╭──────────
      │   hello
      ╰──────────
    BLOCK
    tmux.until { assert_block(block2, it) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, it) }
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
    tmux.until { assert_block(block1, it) }

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
    tmux.until { assert_block(block2, it) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, it) }
  end

  def test_header_border_toggle_with_header_lines_header_first_header_lines_border
    tmux.send_keys %(seq 100 | #{FZF} --list-border rounded --header-border rounded --bind 'space:change-header(hello),enter:change-header()' --header-lines 2 --header-first --header-lines-border double), :Enter
    block1 = <<~BLOCK
      │   5
      │   4
      │ > 3
      ╰──────────
          98/98 ─
        >
      ╔══════════
      ║   2
      ║   1
      ╚══════════
    BLOCK
    tmux.until { assert_block(block1, it) }

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
    tmux.until { assert_block(block2, it) }

    tmux.send_keys :Enter
    tmux.until { assert_block(block1, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
  end

  def test_style_full_adaptive_height
    tmux.send_keys %(seq 1| #{FZF} --style=full:rounded --height=~100% --header-lines=1 --info=default), :Enter
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
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
    tmux.until { assert_block(block, it) }
  end

  def test_min_height_auto_no_input
    tmux.send_keys %(seq 100 | #{FZF} --style full:sharp --no-input --height 1% --min-height 5+), :Enter

    block = <<~BLOCK
      ┌─────────
      │   5
      │   4
      │   3
      │   2
      │ > 1
      └─────────
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_min_height_auto_no_input_reverse_list
    tmux.send_keys %(seq 100 | #{FZF} --style full:sharp --layout reverse-list --no-input --height 1% --min-height 5+ --bind a:show-input,b:hide-input,c:toggle-input), :Enter

    block = <<~BLOCK
      ┌─────────
      │ > 1
      │   2
      │   3
      │   4
      │   5
      └─────────
    BLOCK
    tmux.until { assert_block(block, it) }
    tmux.send_keys :a
    block2 = <<~BLOCK
      ┌─────
      │ > 1
      │   2
      └─────
      ┌─────
      │ >
      └─────
    BLOCK
    tmux.until { assert_block(block2, it) }
    tmux.send_keys :b
    tmux.until { assert_block(block, it) }
    tmux.send_keys :c
    tmux.until { assert_block(block2, it) }
    tmux.send_keys :c
    tmux.until { assert_block(block, it) }
  end

  def test_layout_reverse_list
    prefix = "seq 5 | #{FZF} --layout reverse-list --no-list-border --height ~100% --border sharp "
    suffixes = [
      %(),
      %[--header "$(seq 101 103)"],
      %[--header "$(seq 101 103)" --header-first],
      %[--header "$(seq 101 103)" --header-lines 3],
      %[--header "$(seq 101 103)" --header-lines 3 --header-first],
      %[--header "$(seq 101 103)" --header-border sharp],
      %[--header "$(seq 101 103)" --header-border sharp --header-first],
      %[--header "$(seq 101 103)" --header-border sharp --header-lines 3],
      %[--header "$(seq 101 103)" --header-border sharp --header-lines 3 --header-lines-border sharp],
      %[--header "$(seq 101 103)" --header-border sharp --header-lines 3 --header-lines-border sharp --header-first],
      %[--header "$(seq 101 103)" --header-border sharp --header-lines 3 --header-lines-border sharp --header-first --input-border sharp],
      %[--header "$(seq 101 103)" --header-border sharp --header-lines 3 --header-lines-border sharp --header-first --no-input],
      %[--header "$(seq 101 103)" --input-border sharp],
      %[--header "$(seq 101 103)" --style full:sharp],
      %[--header "$(seq 101 103)" --style full:sharp --header-first]
    ]
    output = <<~BLOCK
      ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌───────── ┌─────── ┌───────── ┌───────── ┌─────────
      │ > 1     │ > 1     │ > 1     │   1     │   1     │ > 1     │ > 1     │   1     │ ┌────── │ ┌────── │ ┌─────── │ ┌───── │ > 1      │ ┌─────── │ ┌───────
      │   2     │   2     │   2     │   2     │   2     │   2     │   2     │   2     │ │ 1     │ │ 1     │ │ 1      │ │ 1    │   2      │ │ > 1    │ │ > 1
      │   3     │   3     │   3     │   3     │   3     │   3     │   3     │   3     │ │ 2     │ │ 2     │ │ 2      │ │ 2    │   3      │ │   2    │ │   2
      │   4     │   4     │   4     │ > 4     │ > 4     │   4     │   4     │ > 4     │ │ 3     │ │ 3     │ │ 3      │ │ 3    │   4      │ │   3    │ │   3
      │   5     │   5     │   5     │   5     │   5     │   5     │   5     │   5     │ └────── │ └────── │ └─────── │ └───── │   5      │ │   4    │ │   4
      │   5/5 ─ │   101   │   5/5 ─ │   101   │   2/2 ─ │ ┌────── │   5/5 ─ │ ┌────── │ > 4     │ > 4     │ > 4      │ > 4    │   101    │ │   5    │ │   5
      │ >       │   102   │ >       │   102   │ >       │ │ 101   │ >       │ │ 101   │   5     │   5     │   5      │   5    │   102    │ └─────── │ └───────
      └──────── │   103   │   101   │   103   │   101   │ │ 102   │ ┌────── │ │ 102   │ ┌────── │   2/2 ─ │ ┌─────── │ ┌───── │   103    │ ┌─────── │ ┌───────
                │   5/5 ─ │   102   │   2/2 ─ │   102   │ │ 103   │ │ 101   │ │ 103   │ │ 101   │ >       │ │   2/2  │ │ 101  │ ┌─────── │ │   101  │ │ >
                │ >       │   103   │ >       │   103   │ └────── │ │ 102   │ └────── │ │ 102   │ ┌────── │ │ >      │ │ 102  │ │   5/5  │ │   102  │ └───────
                └──────── └──────── └──────── └──────── │   5/5 ─ │ │ 103   │   2/2 ─ │ │ 103   │ │ 101   │ └─────── │ │ 103  │ │ >      │ │   103  │ ┌───────
                                                        │ >       │ └────── │ >       │ └────── │ │ 102   │ ┌─────── │ └───── │ └─────── │ └─────── │ │   101
                                                        └──────── └──────── └──────── │   2/2 ─ │ │ 103   │ │ 101    └─────── └───────── │ ┌─────── │ │   102
                                                                                      │ >       │ └────── │ │ 102                        │ │ >      │ │   103
                                                                                      └──────── └──────── │ │ 103                        │ └─────── │ └───────
                                                                                                          │ └───────                     └───────── └─────────
                                                                                                          └─────────
    BLOCK

    expects = []
    output.each_line.first.scan(/\S+/) do
      offset = Regexp.last_match.offset(0)
      expects << output.lines.filter_map { it[offset[0]...offset[1]]&.strip }.take_while { !it.empty? }.join("\n")
    end

    suffixes.zip(expects).each do |suffix, block|
      tmux.send_keys(prefix + suffix, :Enter)
      tmux.until { assert_block(block, it) }

      teardown
      setup
    end
  end

  def test_layout_default_with_footer
    prefix = %[
      seq 3 | #{FZF} --no-list-border --height ~100% \
        --border sharp --footer "$(seq 201 202)" --footer-label FOOT --footer-label-pos 3 \
        --header-label HEAD --header-label-pos 3:bottom \
        --bind 'space:transform-footer-label(echo foot)+change-header-label(head)'
    ].strip + ' '
    suffixes = [
      %(),
      %[--header "$(seq 101 102)"],
      %[--header "$(seq 101 102)" --header-first],
      %[--header "$(seq 101 102)" --header-lines 2],
      %[--header "$(seq 101 102)" --header-lines 2 --header-first],
      %[--header "$(seq 101 102)" --header-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-first],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --no-header-lines-border],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border none],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp --header-first --input-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp --header-first --no-input],
      %[--header "$(seq 101 102)" --footer-border sharp --input-border line],
      %[--header "$(seq 101 102)" --style full:sharp --header-first]
    ]
    output = <<~BLOCK
      ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌───────── ┌──────── ┌──────── ┌─────────
      │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201    │   201   │ ┌─FOOT─ │ ┌─FOOT──
      │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202    │   202   │ │ 201   │ │   201
      │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT── │ ──FOOT─ │ │ 202   │ │   202
      │   3     │   3     │   3     │ > 3     │ > 3     │   3     │   3     │ > 3     │ > 3     │ > 3     │ > 3     │ > 3      │ > 3     │ └────── │ └───────
      │   2     │   2     │   2     │   2     │   2     │   2     │   2     │ ┌────── │ ┌────── │   2     │ ┌────── │ ┌─────── │ ┌────── │   3     │ ┌───────
      │ > 1     │ > 1     │ > 1     │   1     │   1     │ > 1     │ > 1     │ │ 2     │ │ 2     │   1     │ │ 2     │ │ 2      │ │ 2     │   2     │ │   3
      │   3/3 ─ │   101   │   3/3 ─ │   101   │   1/1 ─ │ ┌────── │   3/3 ─ │ │ 1     │ │ 1     │ ┌────── │ │ 1     │ │ 1      │ │ 1     │ > 1     │ │   2
      │ >       │   102   │ >       │   102   │ >       │ │ 101   │ >       │ │ 101   │ │ 101   │ │ 101   │ └────── │ └─────── │ └────── │   101   │ │ > 1
      └──────── │   3/3 ─ │   101   │   1/1 ─ │   101   │ │ 102   │ ┌────── │ │ 102   │ │ 102   │ │ 102   │ ┌────── │ ┌─────── │ ┌────── │   102   │ └───────
                │ >       │   102   │ >       │   102   │ └─HEAD─ │ │ 101   │ └─HEAD─ │ └─HEAD─ │ └─HEAD─ │ │ 101   │ │   1/1  │ │ 101   │ ─────── │ ┌───────
                └──────── └──────── └──────── └──────── │   3/3 ─ │ │ 102   │   1/1 ─ │   1/1 ─ │   1/1 ─ │ │ 102   │ │ >      │ │ 102   │   3/3   │ │ >
                                                        │ >       │ └─HEAD─ │ >       │ >       │ >       │ └─HEAD─ │ └─────── │ └─HEAD─ │ >       │ └───────
                                                        └──────── └──────── └──────── └──────── └──────── │   1/1 ─ │ ┌─────── └──────── └──────── │ ┌───────
                                                                                                          │ >       │ │ 101                        │ │   101
                                                                                                          └──────── │ │ 102                        │ │   102
                                                                                                                    │ └─HEAD──                     │ └─HEAD──
                                                                                                                    └─────────                     └─────────
    BLOCK

    expects = []
    output.each_line.first.scan(/\S+/) do
      offset = Regexp.last_match.offset(0)
      expects << output.lines.filter_map { it[offset[0]...offset[1]]&.strip }.take_while { !it.empty? }.join("\n")
    end

    suffixes.zip(expects).each do |suffix, block|
      tmux.send_keys(prefix + suffix, :Enter)
      tmux.until { assert_block(block, it) }
      tmux.send_keys :Space
      tmux.until { assert_block(block.downcase, it) }

      teardown
      setup
    end
  end

  def test_layout_reverse_list_with_footer
    prefix = %[
      seq 3 | #{FZF} --layout reverse-list --no-list-border --height ~100% \
        --border sharp --footer "$(seq 201 202)" --footer-label FOOT --footer-label-pos 3 \
        --header-label HEAD --header-label-pos 3:bottom \
        --bind 'space:transform-footer-label(echo foot)+change-header-label(head)'
    ].strip + ' '
    suffixes = [
      %(),
      %[--header "$(seq 101 102)"],
      %[--header "$(seq 101 102)" --header-first],
      %[--header "$(seq 101 102)" --header-lines 2],
      %[--header "$(seq 101 102)" --header-lines 2 --header-first],
      %[--header "$(seq 101 102)" --header-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-first],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp --header-first --input-border sharp],
      %[--header "$(seq 101 102)" --header-border sharp --header-lines 2 --header-lines-border sharp --header-first --no-input],
      %[--header "$(seq 101 102)" --footer-border sharp --input-border line],
      %[--header "$(seq 101 102)" --style full:sharp --header-first]
    ]
    output = <<~BLOCK
      ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌──────── ┌───────── ┌──────── ┌──────── ┌─────────
      │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201   │   201    │   201   │ ┌─FOOT─ │ ┌─FOOT──
      │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202   │   202    │   202   │ │ 201   │ │   201
      │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT─ │ ──FOOT── │ ──FOOT─ │ │ 202   │ │   202
      │ > 1     │ > 1     │ > 1     │   1     │   1     │ > 1     │ > 1     │   1     │ ┌────── │ ┌─────── │ ┌────── │ └────── │ └───────
      │   2     │   2     │   2     │   2     │   2     │   2     │   2     │   2     │ │ 1     │ │ 1      │ │ 1     │ > 1     │ ┌───────
      │   3     │   3     │   3     │ > 3     │ > 3     │   3     │   3     │ > 3     │ │ 2     │ │ 2      │ │ 2     │   2     │ │ > 1
      │   3/3 ─ │   101   │   3/3 ─ │   101   │   1/1 ─ │ ┌────── │   3/3 ─ │ ┌────── │ └────── │ └─────── │ └────── │   3     │ │   2
      │ >       │   102   │ >       │   102   │ >       │ │ 101   │ >       │ │ 101   │ > 3     │ > 3      │ > 3     │   101   │ │   3
      └──────── │   3/3 ─ │   101   │   1/1 ─ │   101   │ │ 102   │ ┌────── │ │ 102   │ ┌────── │ ┌─────── │ ┌────── │   102   │ └───────
                │ >       │   102   │ >       │   102   │ └─HEAD─ │ │ 101   │ └─HEAD─ │ │ 101   │ │   1/1  │ │ 101   │ ─────── │ ┌───────
                └──────── └──────── └──────── └──────── │   3/3 ─ │ │ 102   │   1/1 ─ │ │ 102   │ │ >      │ │ 102   │   3/3   │ │ >
                                                        │ >       │ └─HEAD─ │ >       │ └─HEAD─ │ └─────── │ └─HEAD─ │ >       │ └───────
                                                        └──────── └──────── └──────── │   1/1 ─ │ ┌─────── └──────── └──────── │ ┌───────
                                                                                      │ >       │ │ 101                        │ │   101
                                                                                      └──────── │ │ 102                        │ │   102
                                                                                                │ └─HEAD──                     │ └─HEAD──
                                                                                                └─────────                     └─────────
    BLOCK

    expects = []
    output.each_line.first.scan(/\S+/) do
      offset = Regexp.last_match.offset(0)
      expects << output.lines.filter_map { it[offset[0]...offset[1]]&.strip }.take_while { !it.empty? }.join("\n")
    end

    suffixes.zip(expects).each do |suffix, block|
      tmux.send_keys(prefix + suffix, :Enter)
      tmux.until { assert_block(block, it) }
      tmux.send_keys :Space
      tmux.until { assert_block(block.downcase, it) }

      teardown
      setup
    end
  end

  def test_change_header_and_label_at_once
    tmux.send_keys %(seq 10 | #{FZF} --border sharp --header-border sharp --header-label-pos 3 --bind 'focus:change-header(header)+change-header-label(label)'), :Enter
    block = <<~BLOCK
      │ ┌─label──
      │ │ header
      │ └────────
      │   10/10 ─
      │ >
      └──────────
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_label_truncation
    command = <<~CMD
      seq 10 | #{FZF} --style full --border --header-lines=1 --preview ':' \\
        --border-label "#{'b' * 1000}" \\
        --preview-label "#{'p' * 1000}" \\
        --header-label "#{'h' * 1000}" \\
        --header-label "#{'h' * 1000}" \\
        --input-label "#{'i' * 1000}" \\
        --list-label "#{'l' * 1000}"
    CMD
    writelines(command.lines.map(&:chomp))
    tmux.send_keys("sh #{tempname}", :Enter)
    tmux.until do |lines|
      text = lines.join
      assert_includes text, 'b··'
      assert_includes text, 'l··p'
      assert_includes text, 'p··'
      assert_includes text, 'h··'
      assert_includes text, 'i··'
    end
  end

  def test_separator_no_ellipsis
    tmux.send_keys %(seq 10 | #{FZF} --separator "$(seq 1000 | tr '\\n' ' ')"), :Enter
    tmux.until do |lines|
      assert_equal 10, lines.match_count
      refute_includes lines.join, '··'
    end
  end

  def test_header_border_no_pointer_and_marker
    tmux.send_keys %(seq 10 | #{FZF} --header-lines 1 --header-border sharp --no-list-border --pointer '' --marker ''), :Enter
    block = <<~BLOCK
      ┌──────
      │ 1
      └──────
        9/9 ─
      >
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_gutter_default
    tmux.send_keys %(seq 10 | fzf), :Enter
    block = <<~BLOCK
      ▌ 3
      ▌ 2
      > 1
        10/10
      >
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_gutter_default_no_unicode
    tmux.send_keys %(seq 10 | fzf --no-unicode), :Enter
    block = <<~BLOCK
        3
        2
      > 1
        10/10
      >
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  def test_gutter_custom
    tmux.send_keys %(seq 10 | fzf --gutter x), :Enter
    block = <<~BLOCK
      x 3
      x 2
      > 1
        10/10
      >
    BLOCK
    tmux.until { assert_block(block, it) }
  end

  # https://github.com/junegunn/fzf/issues/4537
  def test_no_scrollbar_preview_toggle
    x = 'x' * 300
    y = 'y' * 300
    tmux.send_keys %(yes #{x} | head -1000 | fzf --bind 'tab:toggle-preview' --border --no-scrollbar --preview 'echo #{y}' --preview-window 'border-left'), :Enter

    # │ ▌ xxxxxxxx·· │ yyyyyyyy│
    tmux.until do |lines|
      lines.any? { it.match?(/x·· │ y+│$/) }
    end
    tmux.send_keys :Tab

    # │ ▌ xxxxxxxx·· │
    tmux.until do |lines|
      lines.none? { it.match?(/x··y│$/) }
    end

    tmux.send_keys :Tab
    tmux.until do |lines|
      lines.any? { it.match?(/x·· │ y+│$/) }
    end
  end

  def test_header_and_footer_should_not_be_wider_than_list
    tmux.send_keys %(WIDE=$(printf 'x%.0s' {1..1000}); (echo $WIDE; echo $WIDE) | fzf --header-lines 1 --style full --header-border bottom --header-lines-border top --ellipsis XX --header "$WIDE" --footer "$WIDE" --no-footer-border), :Enter
    tmux.until do |lines|
      matches = lines.filter_map { |line| line[/x+XX/] }
      assert_equal 4, matches.length
      assert_equal 1, matches.uniq.length
    end
  end

  def test_combinations
    skip unless ENV['LONGTEST']

    base = [
      '--pointer=@',
      '--exact',
      '--query=123',
      '--header="$(seq 101 103)"',
      '--header-lines=3',
      '--footer "$(seq 201 203)"',
      '--preview "echo foobar"'
    ]
    options = [
      ['--separator==', '--no-separator'],
      ['--info=default', '--info=inline', '--info=inline-right'],
      ['--no-input-border', '--input-border'],
      ['--no-header-border', '--header-border=none', '--header-border'],
      ['--no-header-lines-border', '--header-lines-border'],
      ['--no-footer-border', '--footer-border'],
      ['--no-list-border', '--list-border'],
      ['--preview-window=right', '--preview-window=up', '--preview-window=down', '--preview-window=left'],
      ['--header-first', '--no-header-first'],
      ['--layout=default', '--layout=reverse', '--layout=reverse-list']
    ]
    # Combination of all options
    combinations = options[0].product(*options.drop(1))
    combinations.each_with_index do |combination, index|
      opts = base + combination
      command = %(seq 1001 2000 | #{FZF} #{opts.join(' ')})
      puts "# #{index + 1}/#{combinations.length}\n#{command}"
      tmux.send_keys command, :Enter
      tmux.until do |lines|
        layout = combination.find { it.start_with?('--layout=') }.split('=').last
        header_first = combination.include?('--header-first')

        # Input
        input = lines.index { it.include?('> 123') }
        assert(input)

        # Info
        info = lines.index { it.include?('11/997') }
        assert(info)

        assert(layout == 'reverse' ? input <= info : input >= info)

        # List
        item1 = lines.index { it.include?('1230') }
        item2 = lines.index { it.include?('1231') }
        assert_equal(item1, layout == 'default' ? item2 + 1 : item2 - 1)

        # Preview
        assert(lines.any? { it.include?('foobar') })

        # Header
        header1 = lines.index { it.include?('101') }
        header2 = lines.index { it.include?('102') }
        assert_equal(header2, header1 + 1)
        assert((layout == 'reverse') == header_first ? input > header1 : input < header1)

        # Footer
        footer1 = lines.index { it.include?('201') }
        footer2 = lines.index { it.include?('202') }
        assert_equal(footer2, footer1 + 1)
        assert(layout == 'reverse' ? footer1 > item2 : footer1 < item2)

        # Header lines
        hline1 = lines.index { it.include?('1001') }
        hline2 = lines.index { it.include?('1002') }
        assert_equal(hline1, layout == 'default' ? hline2 + 1 : hline2 - 1)
        assert(layout == 'reverse' ? hline1 > header1 : hline1 < header1)
      end
      tmux.send_keys :Enter
    end
  end

  # Locate a word in the currently captured screen and click its first character.
  # tmux rows/columns are 1-based; capture indices are 0-based.
  def click_word(word)
    tmux.capture.each_with_index do |line, idx|
      col = line.index(word)
      return tmux.click(col + 1, idx + 1) if col
    end
    flunk("word #{word.inspect} not found on screen")
  end

  # Launch fzf with a click-{header,footer} binding that echoes FZF_CLICK_* into the prompt,
  # then click each word in `clicks` and assert the resulting L/W values.
  # `clicks` is an array of [word_to_click, expected_line].
  def verify_clicks(kind:, opts:, input:, clicks:)
    var = kind.to_s.upcase # HEADER or FOOTER
    binding = "click-#{kind}:transform-prompt:" \
              "echo \"L=$FZF_CLICK_#{var}_LINE W=$FZF_CLICK_#{var}_WORD> \""
    # --multi makes the info line end in " (0)" so the wait regex is unambiguous.
    tmux.send_keys %(#{input} | #{FZF} #{opts} --multi --bind '#{binding}'), :Enter
    # Wait for fzf to fully render before inspecting the screen, otherwise the echoed
    # command line can shadow click targets.
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+ \(0\)}) }
    clicks.each do |word, line|
      click_word(word)
      tmux.until { |lines| assert lines.any_include?("L=#{line} W=#{word}>") }
    end
    tmux.send_keys 'Escape'
  end

  # Header lines (--header-lines) are rendered in reverse display order only under
  # layout=default; in layout=reverse and layout=reverse-list they keep the input order.
  # FZF_CLICK_HEADER_LINE reflects the visual row, so the expected value flips.
  HEADER_CLICKS = [%w[Aaa 1], %w[Bbb 2], %w[Ccc 3]].freeze

  %w[default reverse reverse-list].each do |layout|
    slug = layout.tr('-', '_')

    # Plain --header with no border around the header section.
    define_method(:"test_click_header_plain_#{slug}") do
      verify_clicks(kind: :header,
                    opts: %(--layout=#{layout} --header $'Aaa\\nBbb\\nCcc'),
                    input: 'seq 5',
                    clicks: HEADER_CLICKS)
    end

    # --header with a framing border (--style full gives --header-border=rounded by default).
    define_method(:"test_click_header_border_rounded_#{slug}") do
      verify_clicks(kind: :header,
                    opts: %(--layout=#{layout} --style full --header $'Aaa\\nBbb\\nCcc'),
                    input: 'seq 5',
                    clicks: HEADER_CLICKS)
    end

    # --header-lines consumed from stdin, with its own framing border.
    define_method(:"test_click_header_lines_border_rounded_#{slug}") do
      clicks_hl = if layout == 'default'
                    [%w[Xaa 3], %w[Ybb 2], %w[Zcc 1]]
                  else
                    [%w[Xaa 1], %w[Ybb 2], %w[Zcc 3]]
                  end
      verify_clicks(kind: :header,
                    opts: %(--layout=#{layout} --style full --header-lines 3),
                    input: "(printf 'Xaa\\nYbb\\nZcc\\n'; seq 5)",
                    clicks: clicks_hl)
    end

    # --footer with a framing border.
    define_method(:"test_click_footer_border_rounded_#{slug}") do
      verify_clicks(kind: :footer,
                    opts: %(--layout=#{layout} --style full --footer $'Foo\\nBar\\nBaz'),
                    input: 'seq 5',
                    clicks: [%w[Foo 1], %w[Bar 2], %w[Baz 3]])
    end

    # --header and --header-lines combined. Click-header numbering concatenates the two
    # sections, but the order depends on the layout:
    #   layoutReverse:     custom header (1..N), then header-lines (N+1..N+M)
    #   layoutDefault:     header-lines (1..M, reversed visually), then custom header (M+1..M+N)
    #   layoutReverseList: header-lines (1..M), then custom header (M+1..M+N)
    define_method(:"test_click_header_combined_#{slug}") do
      clicks = case layout
               when 'reverse'
                 [%w[Aaa 1], %w[Bbb 2], %w[Ccc 3], %w[Xaa 4], %w[Ybb 5], %w[Zcc 6]]
               when 'default'
                 [%w[Aaa 4], %w[Bbb 5], %w[Ccc 6], %w[Xaa 3], %w[Ybb 2], %w[Zcc 1]]
               else # reverse-list
                 [%w[Aaa 4], %w[Bbb 5], %w[Ccc 6], %w[Xaa 1], %w[Ybb 2], %w[Zcc 3]]
               end
      verify_clicks(kind: :header,
                    opts: %(--layout=#{layout} --header $'Aaa\\nBbb\\nCcc' --header-lines 3),
                    input: "(printf 'Xaa\\nYbb\\nZcc\\n'; seq 5)",
                    clicks: clicks)
    end

    # Inline header inside a rounded list border.
    define_method(:"test_click_header_border_inline_#{slug}") do
      opts = %(--layout=#{layout} --style full --header $'Aaa\\nBbb\\nCcc' --header-border=inline)
      verify_clicks(kind: :header, opts: opts, input: 'seq 5', clicks: HEADER_CLICKS)
    end

    # Inline header inside a horizontal list border (top+bottom only, no T-junctions).
    define_method(:"test_click_header_border_inline_horizontal_list_#{slug}") do
      opts = %(--layout=#{layout} --style full --list-border=horizontal --header $'Aaa\\nBbb\\nCcc' --header-border=inline)
      verify_clicks(kind: :header, opts: opts, input: 'seq 5', clicks: HEADER_CLICKS)
    end

    # Inline header-lines inside a rounded list border.
    define_method(:"test_click_header_lines_border_inline_#{slug}") do
      clicks_hl = if layout == 'default'
                    [%w[Xaa 3], %w[Ybb 2], %w[Zcc 1]]
                  else
                    [%w[Xaa 1], %w[Ybb 2], %w[Zcc 3]]
                  end
      opts = %(--layout=#{layout} --style full --header-lines 3 --header-lines-border=inline)
      verify_clicks(kind: :header, opts: opts,
                    input: "(printf 'Xaa\\nYbb\\nZcc\\n'; seq 5)",
                    clicks: clicks_hl)
    end

    # Inline footer inside a rounded list border.
    define_method(:"test_click_footer_border_inline_#{slug}") do
      opts = %(--layout=#{layout} --style full --footer $'Foo\\nBar\\nBaz' --footer-border=inline)
      verify_clicks(kind: :footer, opts: opts, input: 'seq 5',
                    clicks: [%w[Foo 1], %w[Bar 2], %w[Baz 3]])
    end
  end

  # An inline section requesting far more rows than the terminal can fit must not
  # break the layout. The list frame must still render inside the pane with both
  # corners visible and the prompt line present.
  def test_inline_header_lines_oversized
    tmux.send_keys %(seq 10000 | #{FZF} --style full --header-border inline --header-lines 9999), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    lines = tmux.capture
    # Rounded (light) and sharp (tcell) default border glyphs.
    top_corners = /[╭┌]/
    bottom_corners = /[╰└]/
    assert(lines.any? { |l| l.match?(top_corners) }, "list frame top missing: #{lines.inspect}")
    assert(lines.any? { |l| l.match?(bottom_corners) }, "list frame bottom missing: #{lines.inspect}")
    assert(lines.any? { |l| l.include?('>') }, "prompt missing: #{lines.inspect}")
    tmux.send_keys 'Escape'
  end

  # A non-inline section that consumes all available rows must still render without
  # crashing when another section is inline but has no budget. The inline section's
  # content is clipped to 0 but the layout proceeds.
  def test_inline_footer_starved_by_non_inline_header
    tmux.send_keys %(seq 10000 | #{FZF} --style full --footer-border inline --footer "$(seq 1000)" --header "$(seq 1000)"), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    lines = tmux.capture
    assert(lines.any? { |l| l.include?('>') }, "prompt missing: #{lines.inspect}")
    tmux.send_keys 'Escape'
  end

  # Without a line-drawing --list-border, --header-border=inline must silently
  # fall back to the `line` style (documented behavior).
  def test_inline_falls_back_without_list_border
    tmux.send_keys %(seq 5 | #{FZF} --list-border=none --header HEADER --header-border=inline), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    lines = tmux.capture
    assert(lines.any? { |l| l.include?('HEADER') }, "header missing: #{lines.inspect}")
    # Neither list frame corners (rounded/sharp) nor T-junction runes appear,
    # since we've fallen back to a plain line separator.
    assert(lines.none? { |l| l.match?(/[╭╮╰╯┌┐└┘├┤]/) }, "unexpected frame glyphs: #{lines.inspect}")
    tmux.send_keys 'Escape'
  end

  # Regression: when --header-border=inline falls back to `line` because the
  # list border can't host an inline separator, the header-border color must
  # inherit from `border`, not `list-border`. The effective shape is `line`,
  # so color inheritance must match what `line` rendering would use.
  def test_inline_fallback_does_not_inherit_list_border_color
    # Marker attribute (bold) on list-border. If HeaderBorder wrongly inherits
    # from ListBorder, the header separator characters will carry the bold
    # attribute. --info=hidden and --no-separator strip other separator lines
    # so the only row of `─` chars is the header separator.
    tmux.send_keys %(seq 5 | #{FZF} --list-border=none --header HEADER --header-border=inline --info=hidden --no-separator --color=bg:-1,list-border:red:bold), :Enter
    sep_row = nil
    tmux.until do |_|
      sep_row = tmux.capture_ansi.find do |row|
        stripped = row.gsub(/\e\[[\d;]*m/, '').rstrip
        stripped.match?(/\A─+\z/)
      end
      !sep_row.nil?
    end
    # Bold (1) or red fg (31) on the header separator means it inherited from
    # list-border even though the effective shape is `line` (non-inline).
    refute_match(/\e\[(?:[\d;]*;)?(?:1|31)(?:;[\d;]*)?m─/, sep_row,
                 "header separator inherited list-border attr: #{sep_row.inspect}")
    tmux.send_keys 'Escape'
  end

  # Inline takes precedence over --header-first: the main header stays
  # inside the list frame instead of moving below the input.
  def test_inline_header_border_overrides_header_first
    tmux.send_keys %(seq 5 | #{FZF} --style full --header foo --header-first --header-border inline), :Enter
    tmux.until do |lines|
      foo_idx = lines.index { |l| l.match?(/\A│\s+foo\s+│\z/) }
      input_idx = lines.index { |l| l.match?(%r{\A│\s+>\s+\d+/\d+\s+│\z}) }
      foo_idx && input_idx && foo_idx < input_idx
    end
  end

  # With both sections present, --header-first still moves the main --header
  # below the input while --header-lines-border=inline keeps header-lines
  # inside the list frame.
  def test_inline_header_lines_with_header_first_and_main_header
    tmux.send_keys %(seq 5 | #{FZF} --style full --header foo --header-lines 1 --header-first --header-lines-border inline), :Enter
    tmux.until do |lines|
      one_idx = lines.index { |l| l.match?(/\A│\s+1\s+│\z/) }
      foo_idx = lines.index { |l| l.match?(/\A│\s+foo\s+│\z/) }
      input_idx = lines.index { |l| l.match?(%r{\A│\s+>\s+\d+/\d+\s+│\z}) }
      one_idx && foo_idx && input_idx && one_idx < input_idx && input_idx < foo_idx
    end
  end

  # With no main --header, --header-first previously repositioned
  # header-lines. Inline now takes precedence: header-lines stays inside
  # the list frame.
  def test_inline_header_lines_with_header_first_no_main_header
    tmux.send_keys %(seq 5 | #{FZF} --style full --header-lines 1 --header-first --header-lines-border inline), :Enter
    tmux.until do |lines|
      one_idx = lines.index { |l| l.match?(/\A│\s+1\s+│\z/) }
      input_idx = lines.index { |l| l.match?(%r{\A│\s+>\s+\d+/\d+\s+│\z}) }
      one_idx && input_idx && one_idx < input_idx
    end
  end

  # Regression: with --header-border=inline and --header-lines but no
  # --header, the inline slot was sized for header-lines only. After
  # change-header added a main header line, resizeIfNeeded tolerated the
  # too-small slot, so the header-lines line got displaced and disappeared.
  def test_inline_change_header_grows_slot
    tmux.send_keys %(seq 5 | #{FZF} --style full --header-lines 1 --header-border inline --bind space:change-header:tada), :Enter
    tmux.until { |lines| lines.any_include?(/\A│\s+1\s+│\z/) }
    tmux.send_keys :Space
    tmux.until do |lines|
      lines.any_include?(/\A│\s+1\s+│\z/) && lines.any_include?(/\A│\s+tada\s+│\z/)
    end
  end

  # Regression: with --footer-border=inline, change-footer that grows the
  # footer line count left the inline slot sized for the old length, so
  # extra lines were clipped.
  def test_inline_change_footer_grows_slot
    tmux.send_keys %(seq 5 | #{FZF} --style full --footer-border inline --footer one --bind $'space:change-footer:one\\ntwo'), :Enter
    tmux.until { |lines| lines.any_include?(/\A│\s+one\s+│\z/) }
    tmux.send_keys :Space
    tmux.until do |lines|
      lines.any_include?(/\A│\s+one\s+│\z/) && lines.any_include?(/\A│\s+two\s+│\z/)
    end
  end

  # Invalid inline combinations must be rejected at startup.
  def test_inline_rejected_on_unsupported_options
    [
      ['--border=inline', 'inline border is only supported'],
      ['--list-border=inline', 'inline border is only supported'],
      ['--input-border=inline', 'inline border is only supported'],
      ['--preview-window=border-inline --preview :', 'invalid preview window option: border-inline'],
      ['--header-border=inline --header-lines-border=sharp --header-lines=1',
       '--header-border=inline requires --header-lines-border to be inline or unset']
    ].each do |args, expected|
      output = `#{FZF} #{args} < /dev/null 2>&1`
      refute_equal 0, $CHILD_STATUS.exitstatus, "expected non-zero exit for: #{args}"
      assert_includes output, expected, "wrong error for: #{args}"
    end
  end

  private

  # Count rows whose entire width is a single `color` range.
  def count_full_rows(ranges_by_row, color)
    ranges_by_row.count { |r| r.length == 1 && r[0][2] == color }
  end

  # Wait until `tmux.bg_ranges` has at least `count` fully-`color` rows; return them.
  def wait_for_full_rows(color, count)
    ranges = nil
    tmux.until do |_|
      ranges = tmux.bg_ranges
      count_full_rows(ranges, color) >= count
    end
    ranges
  end

  public

  # Inline header's entire section (outer edge + content-row verticals + separator)
  # carries the header-bg color; list rows below carry list-bg.
  def test_inline_header_bg_color
    tmux.send_keys %(seq 5 | #{FZF} --list-border --reverse --header HEADER --header-border=inline --color=bg:-1,header-border:white,list-border:white,header-bg:red,list-bg:green), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    # 3 fully-red rows: top edge, header content, separator.
    ranges = wait_for_full_rows('red', 3)
    assert_equal_org(3, count_full_rows(ranges, 'red'))
    # List rows below (>=5) are fully green.
    assert_operator count_full_rows(ranges, 'green'), :>=, 5
    tmux.send_keys 'Escape'
  end

  # Regression: when --header-lines-border=inline is the only inline section
  # (no --header-border), the section must still use header-bg, not list-bg.
  def test_inline_header_lines_bg_without_main_header
    tmux.send_keys %(seq 5 | #{FZF} --list-border --reverse --header-lines 2 --header-lines-border=inline --color=bg:-1,header-border:white,list-border:white,header-bg:red,list-bg:green), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    # Top edge + 2 content rows + separator = 4 fully-red rows.
    ranges = wait_for_full_rows('red', 4)
    assert_equal_org(4, count_full_rows(ranges, 'red'))
    tmux.send_keys 'Escape'
  end

  # Inline footer's entire section carries footer-bg; list rows above carry list-bg.
  def test_inline_footer_bg_color
    tmux.send_keys %(seq 5 | #{FZF} --list-border --footer FOOTER --footer-border=inline --color=bg:-1,footer-border:white,list-border:white,footer-bg:blue,list-bg:green), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    ranges = wait_for_full_rows('blue', 3)
    assert_equal_org(3, count_full_rows(ranges, 'blue'))
    tmux.send_keys 'Escape'
  end

  # The list-label's bg is swapped to match the adjacent inline section so it reads as
  # part of the section frame rather than a list-colored island on a section-colored edge.
  def test_list_label_bg_on_inline_section_edge
    tmux.send_keys %(seq 5 | #{FZF} --list-border --reverse --header HEADER --header-border=inline --list-label=LL --color=bg:-1,header-border:white,list-border:white,header-bg:red,list-bg:green,list-label:yellow:bold), :Enter
    tmux.until { |lines| lines.any_include?(%r{ [0-9]+/[0-9]+}) }
    # The label sits on the header-owned top edge, so the entire row must be a
    # single red run (no green breaks where the label cells are).
    ranges = wait_for_full_rows('red', 3)
    assert_operator count_full_rows(ranges, 'red'), :>=, 3
    tmux.send_keys 'Escape'
  end
end
