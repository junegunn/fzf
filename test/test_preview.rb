# frozen_string_literal: true

require_relative 'lib/common'

# Test cases for preview
class TestPreview < TestInteractive
  def test_preview
    tmux.send_keys %(seq 1000 | sed s/^2$// | #{FZF} -m --preview 'sleep 0.2; echo {{}-{+}}' --bind ?:toggle-preview), :Enter
    tmux.until { |lines| assert_includes lines[1], ' {1-1} ' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[1], ' {-} ' }
    tmux.send_keys '555'
    tmux.until { |lines| assert_includes lines[1], ' {555-555} ' }
    tmux.send_keys '?'
    tmux.until { |lines| refute_includes lines[1], ' {555-555} ' }
    tmux.send_keys '?'
    tmux.until { |lines| assert_includes lines[1], ' {555-555} ' }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert lines[-2]&.start_with?('  28/1000 ') }
    tmux.send_keys 'foobar'
    tmux.until { |lines| refute_includes lines[1], ' {55-55} ' }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' {1-1} ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {-1} ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {3-1 } ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {4-1  3} ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {5-1  3 4} ' }
  end

  def test_toggle_preview_without_default_preview_command
    tmux.send_keys %(seq 100 | #{FZF} --bind 'space:preview(echo [{}]),enter:toggle-preview' --preview-window up,border-double), :Enter
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      refute_includes lines[1], '║ [1]'
    end

    # toggle-preview should do nothing
    tmux.send_keys :Enter
    tmux.until { |lines| refute_includes lines[1], '║ [1]' }
    tmux.send_keys :Up
    tmux.until do |lines|
      refute_includes lines[1], '║ [1]'
      refute_includes lines[1], '║ [2]'
    end

    tmux.send_keys :Up
    tmux.until do |lines|
      assert_includes lines, '> 3'
      refute_includes lines[1], '║ [3]'
    end

    # One-off preview action
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines[1], '║ [3]' }

    # toggle-preview to hide it
    tmux.send_keys :Enter
    tmux.until { |lines| refute_includes lines[1], '║ [3]' }

    # toggle-preview again does nothing
    tmux.send_keys :Enter, :Up
    tmux.until do |lines|
      assert_includes lines, '> 4'
      refute_includes lines[1], '║ [4]'
    end
  end

  def test_show_and_hide_preview
    tmux.send_keys %(seq 100 | #{FZF} --preview-window hidden,border-bold --preview 'echo [{}]' --bind 'a:show-preview,b:hide-preview'), :Enter

    # Hidden by default
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      refute_includes lines[1], '┃ [1]'
    end

    # Show
    tmux.send_keys :a
    tmux.until { |lines| assert_includes lines[1], '┃ [1]' }

    # Already shown
    tmux.send_keys :a
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[1], '┃ [2]' }

    # Hide
    tmux.send_keys :b
    tmux.send_keys :Up
    tmux.until do |lines|
      assert_includes lines, '> 3'
      refute_includes lines[1], '┃ [3]'
    end

    # Already hidden
    tmux.send_keys :b
    tmux.send_keys :Up
    tmux.until do |lines|
      assert_includes lines, '> 4'
      refute_includes lines[1], '┃ [4]'
    end

    # Show it again
    tmux.send_keys :a
    tmux.until { |lines| assert_includes lines[1], '┃ [4]' }
  end

  def test_preview_hidden
    tmux.send_keys %(seq 1000 | #{FZF} --preview 'echo {{}-{}-$FZF_PREVIEW_LINES-$FZF_PREVIEW_COLUMNS}' --preview-window down:1:hidden --bind ?:toggle-preview), :Enter
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys '?'
    tmux.until { |lines| assert_match(/ {1-1-1-[0-9]+}/, lines[-2]) }
    tmux.send_keys '555'
    tmux.until { |lines| assert_match(/ {555-555-1-[0-9]+}/, lines[-2]) }
    tmux.send_keys '?'
    tmux.until { |lines| assert_equal '> 555', lines[-1] }
  end

  def test_preview_size_0
    tmux.send_keys %(seq 100 | #{FZF} --reverse --preview 'echo {} >> #{tempname}; echo ' --preview-window 0 --bind space:toggle-preview), :Enter
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      assert_equal '  100/100', lines[1]
      assert_equal '> 1', lines[2]
    end
    wait do
      assert_path_exists tempname
      assert_equal %w[1], File.readlines(tempname, chomp: true)
    end
    tmux.send_keys :Space, :Down, :Down
    tmux.until { |lines| assert_equal '> 3', lines[4] }
    wait do
      assert_path_exists tempname
      assert_equal %w[1], File.readlines(tempname, chomp: true)
    end
    tmux.send_keys :Space, :Down
    tmux.until { |lines| assert_equal '> 4', lines[5] }
    wait do
      assert_path_exists tempname
      assert_equal %w[1 3 4], File.readlines(tempname, chomp: true)
    end
  end

  def test_preview_size_0_hidden
    tmux.send_keys %(seq 100 | #{FZF} --reverse --preview 'echo {} >> #{tempname}; echo ' --preview-window 0,hidden --bind space:toggle-preview), :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys :Down, :Down
    tmux.until { |lines| assert_includes lines, '> 3' }
    wait { refute_path_exists tempname }
    tmux.send_keys :Space
    wait do
      assert_path_exists tempname
      assert_equal %w[3], File.readlines(tempname, chomp: true)
    end
    tmux.send_keys :Down
    wait do
      assert_equal %w[3 4], File.readlines(tempname, chomp: true)
    end
    tmux.send_keys :Space, :Down
    tmux.until { |lines| assert_includes lines, '> 5' }
    tmux.send_keys :Down
    tmux.until { |lines| assert_includes lines, '> 6' }
    tmux.send_keys :Space
    wait do
      assert_equal %w[3 4 6], File.readlines(tempname, chomp: true)
    end
  end

  def test_preview_flags
    tmux.send_keys %(seq 10 | sed 's/^/:: /; s/$/  /' |
        #{FZF} --multi --preview 'echo {{2}/{s2}/{+2}/{+s2}/{q}/{n}/{+n}}'), :Enter
    tmux.until { |lines| assert_includes lines[1], ' {1/1  /1/1  //0/0} ' }
    tmux.send_keys '123'
    tmux.until { |lines| assert_includes lines[1], ' {////123//} ' }
    tmux.send_keys 'C-u', '1'
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' {1/1  /1/1  /1/0/0} ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {10/10  /1/1  /1/9/0} ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' {10/10  /1 10/1   10  /1/9/0 9} ' }
    tmux.send_keys '2'
    tmux.until { |lines| assert_includes lines[1], ' {//1 10/1   10  /12//0 9} ' }
    tmux.send_keys '3'
    tmux.until { |lines| assert_includes lines[1], ' {//1 10/1   10  /123//0 9} ' }
  end

  def test_preview_asterisk
    tmux.send_keys %(seq 5 | #{FZF} --multi --preview 'echo [{}/{+}/{*}/{*n}]' --preview-window '+{1}'), :Enter
    tmux.until { |lines| assert_equal 5, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' [1/1/1 2 3 4 5/0 1 2 3 4] ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' [2/1/1 2 3 4 5/0 1 2 3 4] ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' [3/1 2/1 2 3 4 5/0 1 2 3 4] ' }
    tmux.send_keys '5'
    tmux.until { |lines| assert_includes lines[1], ' [5/1 2/5/4] ' }
    tmux.send_keys '5'
    tmux.until { |lines| assert_includes lines[1], ' [/1 2//] ' }
  end

  def test_preview_file
    tmux.send_keys %[(echo foo bar; echo bar foo) | #{FZF} --multi --preview 'cat {+f} {+f2} {+nf} {+fn}' --print0], :Enter
    tmux.until { |lines| assert_includes lines[1], ' foo barbar00 ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' foo barbar00 ' }
    tmux.send_keys :BTab
    tmux.until { |lines| assert_includes lines[1], ' foo barbar foobarfoo0101 ' }
  end

  def test_preview_q_no_match
    tmux.send_keys %(: | #{FZF} --preview 'echo foo {q} foo'), :Enter
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' foo  foo' }
    tmux.send_keys 'bar'
    tmux.until { |lines| assert_includes lines[1], ' foo bar foo' }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_includes lines[1], ' foo  foo' }
  end

  def test_preview_q_no_match_with_initial_query
    tmux.send_keys %(: | #{FZF} --preview 'echo 1. /{q}/{q:1}/; echo 2. /{q:..}/{q:2}/{q:-1}/; echo 3. /{q:s-2}/{q:-2}/{q:x}/' --query 'foo bar'), :Enter
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], '1. /foo bar/foo/' }
    tmux.until { |lines| assert_includes lines[2], '2. /foo bar/bar/bar/' }
    tmux.until { |lines| assert_includes lines[3], '3. /foo /foo/{q:x}/' }
  end

  def test_preview_update_on_select
    tmux.send_keys %(seq 10 | fzf -m --preview 'echo {+}' --bind a:toggle-all),
                   :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.send_keys 'a'
    tmux.until { |lines| assert(lines.any? { |line| line.include?(' 1 2 3 4 5 ') }) }
    tmux.send_keys 'a'
    tmux.until { |lines| lines.each { |line| refute_includes line, ' 1 2 3 4 5 ' } }
  end

  def test_preview_correct_tab_width_after_ansi_reset_code
    writelines(["\x1b[31m+\x1b[m\t\x1b[32mgreen"])
    tmux.send_keys "#{FZF} --preview 'cat #{tempname}'", :Enter
    tmux.until { |lines| assert_includes lines[1], ' +       green ' }
  end

  def test_preview_bindings_with_default_preview
    tmux.send_keys "seq 10 | #{FZF} --preview 'echo [{}]' --bind 'a:preview(echo [{}{}]),b:preview(echo [{}{}{}]),c:refresh-preview'", :Enter
    tmux.until { |lines| lines.match_count == 10 }
    tmux.until { |lines| assert_includes lines[1], '[1]' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[1], '[11]' }
    tmux.send_keys 'c'
    tmux.until { |lines| assert_includes lines[1], '[1]' }
    tmux.send_keys 'b'
    tmux.until { |lines| assert_includes lines[1], '[111]' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[1], '[2]' }
  end

  def test_preview_bindings_without_default_preview
    tmux.send_keys "seq 10 | #{FZF} --bind 'a:preview(echo [{}{}]),b:preview(echo [{}{}{}]),c:refresh-preview'", :Enter
    tmux.until { |lines| lines.match_count == 10 }
    tmux.until { |lines| refute_includes lines[1], '1' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[1], '[11]' }
    tmux.send_keys 'c' # does nothing
    tmux.until { |lines| assert_includes lines[1], '[11]' }
    tmux.send_keys 'b'
    tmux.until { |lines| assert_includes lines[1], '[111]' }
    tmux.send_keys 9
    tmux.until { |lines| lines.match_count == 1 }
    tmux.until { |lines| refute_includes lines[1], '2' }
    tmux.until { |lines| assert_includes lines[1], '[111]' }
  end

  def test_preview_scroll_begin_constant
    tmux.send_keys "echo foo 123 321 | #{FZF} --preview 'seq 1000' --preview-window left:+123", :Enter
    tmux.until { |lines| assert_match %r{1/1}, lines[-2] }
    tmux.until { |lines| assert_match %r{123.*123/1000}, lines[1] }
  end

  def test_preview_scroll_begin_expr
    tmux.send_keys "echo foo 123 321 | #{FZF} --preview 'seq 1000' --preview-window left:+{3}", :Enter
    tmux.until { |lines| assert_match %r{1/1}, lines[-2] }
    tmux.until { |lines| assert_match %r{321.*321/1000}, lines[1] }
  end

  def test_preview_scroll_begin_and_offset
    ['echo foo 123 321', 'echo foo :123: 321'].each do |input|
      tmux.send_keys "#{input} | #{FZF} --preview 'seq 1000' --preview-window left:+{2}-2", :Enter
      tmux.until { |lines| assert_match %r{1/1}, lines[-2] }
      tmux.until { |lines| assert_match %r{121.*121/1000}, lines[1] }
      tmux.send_keys 'C-c'
    end
  end

  def test_preview_clear_screen
    tmux.send_keys %{seq 100 | #{FZF} --preview 'for i in $(seq 300); do (( i % 200 == 0 )) && printf "\\033[2J"; echo "[$i]"; sleep 0.001; done'}, :Enter
    tmux.until { |lines| lines.match_count == 100 }
    tmux.until { |lines| lines[1]&.include?('[200]') }
  end

  def test_preview_window_follow
    file = Tempfile.new('fzf-follow')
    file.sync = true

    tmux.send_keys %(seq 100 | #{FZF} --preview 'echo start; tail -f "#{file.path}"' --preview-window follow --bind 'up:preview-up,down:preview-down,space:change-preview-window:follow|nofollow' --preview-window '~4'), :Enter
    tmux.until { |lines| lines.match_count == 100 }

    # Write to the temporary file, and check if the preview window is showing
    # the last line of the file
    tmux.until { |lines| assert_includes lines[1], 'start' }
    3.times { file.puts _1 } # header lines
    1000.times { file.puts _1 }
    tmux.until { |lines| assert_includes lines[1], '/1004' }
    tmux.until { |lines| assert_includes lines[-2], '999' }

    # Scroll the preview window and fzf should stop following the file content
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[-2], '998' }
    file.puts 'foo', 'bar'
    tmux.until do |lines|
      assert_includes lines[1], '/1006'
      assert_includes lines[-2], '998'
    end

    # Scroll back to the bottom and fzf should start following the file again
    %w[999 foo bar].each do |item|
      wait do
        tmux.send_keys :Down
        tmux.until { |lines| assert_includes lines[-2], item }
      end
    end
    file.puts 'baz'
    tmux.until do |lines|
      assert_includes lines[1], '/1007'
      assert_includes lines[-2], 'baz'
    end

    # Scroll upwards to stop following
    tmux.send_keys :Up
    wait { assert_includes lines[-2], 'bar' }
    file.puts 'aaa'
    tmux.until do |lines|
      assert_includes lines[1], '/1008'
      assert_includes lines[-2], 'bar'
    end

    # Manually enable following
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines[-2], 'aaa' }
    file.puts 'bbb'
    tmux.until do |lines|
      assert_includes lines[1], '/1009'
      assert_includes lines[-2], 'bbb'
    end

    # Disable following
    tmux.send_keys :Space
    file.puts 'ccc', 'ddd'
    tmux.until do |lines|
      assert_includes lines[1], '/1011'
      assert_includes lines[-2], 'bbb'
    end
  rescue StandardError
    file.close
    file.unlink
  end

  def test_toggle_preview_wrap
    tmux.send_keys "#{FZF} --preview 'for i in $(seq $FZF_PREVIEW_COLUMNS); do echo -n .; done; echo wrapped; echo 2nd line' --bind ctrl-w:toggle-preview-wrap", :Enter
    2.times do
      tmux.until { |lines| assert_includes lines[2], '2nd line' }
      tmux.send_keys 'C-w'
      tmux.until do |lines|
        assert_includes lines[2], 'wrapped'
        assert_includes lines[3], '2nd line'
      end
      tmux.send_keys 'C-w'
    end
  end

  def test_close
    tmux.send_keys "seq 100 | #{FZF} --preview 'echo foo' --bind ctrl-c:close", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], 'foo' }
    tmux.send_keys 'C-c'
    tmux.until { |lines| refute_includes lines[1], 'foo' }
    tmux.send_keys '10'
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.send_keys 'C-c'
    tmux.send_keys 'C-l', 'closed'
    tmux.until { |lines| assert_includes lines[0], 'closed' }
  end

  def test_preview_header
    tmux.send_keys "seq 100 | #{FZF} --bind ctrl-k:preview-up+preview-up,ctrl-j:preview-down+preview-down+preview-down --preview 'seq 1000' --preview-window 'top:+{1}:~3'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    top5 = ->(lines) { lines.drop(1).take(5).map { |s| s[/[0-9]+/] } }
    tmux.until do |lines|
      assert_includes lines[1], '4/1000'
      assert_equal(%w[1 2 3 4 5], top5[lines])
    end
    tmux.send_keys '55'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_equal(%w[1 2 3 55 56], top5[lines])
    end
    tmux.send_keys 'C-J'
    tmux.until do |lines|
      assert_equal(%w[1 2 3 58 59], top5[lines])
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 19, lines.match_count
      assert_equal(%w[1 2 3 5 6], top5[lines])
    end
    tmux.send_keys 'C-K'
    tmux.until { |lines| assert_equal(%w[1 2 3 4 5], top5[lines]) }
  end

  def test_change_preview_window
    tmux.send_keys "seq 1000 | #{FZF} --preview 'echo [[{}]]' --no-preview-border --bind '" \
                   'a:change-preview(echo __{}__),' \
                   'b:change-preview-window(down)+change-preview(echo =={}==)+change-preview-window(up),' \
                   'c:change-preview(),d:change-preview-window(hidden),' \
                   "e:preview(printf ::%${FZF_PREVIEW_COLUMNS}s{})+change-preview-window(up),f:change-preview-window(up,wrap)'", :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.until { |lines| assert_includes lines[0], '[[1]]' }

    # change-preview action permanently changes the preview command set by --preview
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[0], '__1__' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines[0], '__2__' }

    # When multiple change-preview-window actions are bound to a single key,
    # the last one wins and the updated options are immediately applied to the new preview
    tmux.send_keys 'b'
    tmux.until { |lines| assert_equal '==2==', lines[0] }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal '==3==', lines[0] }

    # change-preview with an empty preview command closes the preview window
    tmux.send_keys 'c'
    tmux.until { |lines| refute_includes lines[0], '==' }

    # change-preview again to re-open the preview window
    tmux.send_keys 'a'
    tmux.until { |lines| assert_equal '__3__', lines[0] }

    # Hide the preview window with hidden flag
    tmux.send_keys 'd'
    tmux.until { |lines| refute_includes lines[0], '__3__' }

    # One-off preview
    tmux.send_keys 'e'
    tmux.until do |lines|
      assert_equal '::', lines[0]
      refute_includes lines[1], '3'
    end

    # Wrapped
    tmux.send_keys 'f'
    tmux.until do |lines|
      assert_equal '::', lines[0]
      assert_equal '↳   3', lines[1]
    end
  end

  def test_change_preview_window_should_not_reset_change_preview
    tmux.send_keys "#{FZF} --preview-window up,border-none --bind 'start:change-preview(echo hello)' --bind 'enter:change-preview-window(border-left)'", :Enter
    tmux.until { |lines| assert_includes lines, 'hello' }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_includes lines, '│ hello' }
  end

  def test_change_preview_window_rotate
    tmux.send_keys "seq 100 | #{FZF} --preview-window left,border-none --preview 'echo hello' --bind '" \
                   "a:change-preview-window(right|down|up|hidden|)'", :Enter
    tmux.until { |lines| assert(lines.any? { _1.include?('100/100') }) }
    3.times do
      tmux.until { |lines| lines[0].start_with?('hello') }
      tmux.send_keys 'a'
      tmux.until { |lines| lines[0].end_with?('hello') }
      tmux.send_keys 'a'
      tmux.until { |lines| lines[-1].start_with?('hello') }
      tmux.send_keys 'a'
      tmux.until { |lines| assert_equal 'hello', lines[0] }
      tmux.send_keys 'a'
      tmux.until { |lines| refute_includes lines[0], 'hello' }
      tmux.send_keys 'a'
    end
  end

  def test_change_preview_window_rotate_hidden
    tmux.send_keys "seq 100 | #{FZF} --preview-window hidden --preview 'echo =={}==' --bind '" \
                   "a:change-preview-window(nohidden||down,1|)'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.until { |lines| refute_includes lines[1], '==1==' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[1], '==1==' }
    tmux.send_keys 'a'
    tmux.until { |lines| refute_includes lines[1], '==1==' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[-2], '==1==' }
    tmux.send_keys 'a'
    tmux.until { |lines| refute_includes lines[-2], '==1==' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines[1], '==1==' }
  end

  def test_change_preview_window_rotate_hidden_down
    tmux.send_keys "seq 100 | #{FZF} --bind '?:change-preview-window:up||down|' --preview 'echo =={}==' --preview-window hidden,down,1", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.until { |lines| refute_includes lines[1], '==1==' }
    tmux.send_keys '?'
    tmux.until { |lines| assert_includes lines[1], '==1==' }
    tmux.send_keys '?'
    tmux.until { |lines| refute_includes lines[1], '==1==' }
    tmux.send_keys '?'
    tmux.until { |lines| assert_includes lines[-2], '==1==' }
    tmux.send_keys '?'
    tmux.until { |lines| refute_includes lines[-2], '==1==' }
    tmux.send_keys '?'
    tmux.until { |lines| assert_includes lines[1], '==1==' }
  end

  def test_toggle_alternative_preview_window
    tmux.send_keys "seq 10 | #{FZF} --bind space:toggle-preview --preview-window '<100000(hidden,up,border-none)' --preview 'echo /{}/{}/'", :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.until { |lines| refute_includes lines, '/1/1/' }
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines, '/1/1/' }
  end

  def test_alternative_preview_window_opts
    tmux.send_keys "seq 10 | #{FZF} --preview-border rounded --preview-window '~5,2,+0,<100000(~0,+100,wrap,noinfo)' --preview 'seq 1000'", :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.until do |lines|
      assert_equal ['╭────╮', '│ 10 │', '│ ↳ 0│', '│ 10 │', '│ ↳ 1│'], lines.take(5).map(&:strip)
    end
  end

  def test_preview_window_width_exception
    tmux.send_keys "seq 10 | #{FZF} --scrollbar --preview-window border-left --border --preview 'seq 1000'", :Enter
    tmux.until do |lines|
      assert lines[1]&.end_with?(' 1/1000││')
    end
  end

  def test_preview_window_hidden_on_focus
    tmux.send_keys "seq 3 | #{FZF} --preview 'echo {}' --bind focus:hide-preview", :Enter
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines, '> 2' }
  end

  def test_preview_query_should_not_be_affected_by_search
    tmux.send_keys "seq 1 | #{FZF} --bind 'change:transform-search(echo {q:1})' --preview 'echo [{q}/{}]'", :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys '1'
    tmux.until { |lines| assert lines.any_include?('[1/1]') }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('[1 /1]') }
    tmux.send_keys '2'
    tmux.until do |lines|
      assert lines.any_include?('[1 2/1]')
      assert_equal 1, lines.match_count
    end
  end
end
