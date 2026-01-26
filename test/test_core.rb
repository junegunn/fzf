# frozen_string_literal: true

require_relative 'lib/common'

# Testing basic features of fzf
class TestCore < TestInteractive
  def test_fzf_default_command
    tmux.send_keys fzf.sub('FZF_DEFAULT_COMMAND=', "FZF_DEFAULT_COMMAND='echo hello'"), :Enter
    tmux.until { |lines| assert_equal '> hello', lines[-3] }

    tmux.send_keys :Enter
    assert_equal 'hello', fzf_output
  end

  def test_fzf_default_command_failure
    tmux.send_keys fzf.sub('FZF_DEFAULT_COMMAND=', 'FZF_DEFAULT_COMMAND=false'), :Enter
    tmux.until { |lines| assert_includes lines[-2], '  [Command failed: false] ─' }
    tmux.send_keys :Enter
  end

  def test_key_bindings
    tmux.send_keys "#{FZF} -q 'foo bar foo-bar'", :Enter
    tmux.until { |lines| assert_equal '> foo bar foo-bar', lines.last }

    # CTRL-A
    tmux.send_keys 'C-A', '('
    tmux.until { |lines| assert_equal '> (foo bar foo-bar', lines.last }

    # META-F
    tmux.send_keys :Escape, :f, ')'
    tmux.until { |lines| assert_equal '> (foo) bar foo-bar', lines.last }

    # CTRL-B
    tmux.send_keys 'C-B', 'var'
    tmux.until { |lines| assert_equal '> (foovar) bar foo-bar', lines.last }

    # Left, CTRL-D
    tmux.send_keys :Left, :Left, 'C-D'
    tmux.until { |lines| assert_equal '> (foovr) bar foo-bar', lines.last }

    # META-BS
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| assert_equal '> (r) bar foo-bar', lines.last }

    # CTRL-Y
    tmux.send_keys 'C-Y', 'C-Y'
    tmux.until { |lines| assert_equal '> (foovfoovr) bar foo-bar', lines.last }

    # META-B
    tmux.send_keys :Escape, :b, :Space, :Space
    tmux.until { |lines| assert_equal '> (  foovfoovr) bar foo-bar', lines.last }

    # CTRL-F / Right
    tmux.send_keys 'C-F', :Right, '/'
    tmux.until { |lines| assert_equal '> (  fo/ovfoovr) bar foo-bar', lines.last }

    # CTRL-H / BS
    tmux.send_keys 'C-H', :BSpace
    tmux.until { |lines| assert_equal '> (  fovfoovr) bar foo-bar', lines.last }

    # CTRL-E
    tmux.send_keys 'C-E', 'baz'
    tmux.until { |lines| assert_equal '> (  fovfoovr) bar foo-barbaz', lines.last }

    # CTRL-U
    tmux.send_keys 'C-U'
    tmux.until { |lines| assert_equal '>', lines.last }

    # CTRL-Y
    tmux.send_keys 'C-Y'
    tmux.until { |lines| assert_equal '> (  fovfoovr) bar foo-barbaz', lines.last }

    # CTRL-W
    tmux.send_keys 'C-W', 'bar-foo'
    tmux.until { |lines| assert_equal '> (  fovfoovr) bar bar-foo', lines.last }

    # META-D
    tmux.send_keys :Escape, :b, :Escape, :b, :Escape, :d, 'C-A', 'C-Y'
    tmux.until { |lines| assert_equal '> bar(  fovfoovr) bar -foo', lines.last }

    # CTRL-M
    tmux.send_keys 'C-M'
    tmux.until { |lines| refute_equal '>', lines.last }
  end

  def test_file_word
    tmux.send_keys "#{FZF} -q '--/foo bar/foo-bar/baz' --filepath-word", :Enter
    tmux.until { |lines| assert_equal '> --/foo bar/foo-bar/baz', lines.last }

    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :b
    tmux.send_keys :Escape, :d
    tmux.send_keys :Escape, :f
    tmux.send_keys :Escape, :BSpace
    tmux.until { |lines| assert_equal '> --///baz', lines.last }
  end

  def test_multi_order
    tmux.send_keys "seq 1 10 | #{fzf(:multi)}", :Enter
    tmux.until { |lines| assert_equal '>', lines.last }

    tmux.send_keys :Tab, :Up, :Up, :Tab, :Tab, :Tab, # 3, 2
                   'C-K', 'C-K', 'C-K', 'C-K', :BTab, :BTab, # 5, 6
                   :PgUp, 'C-J', :Down, :Tab, :Tab # 8, 7
    tmux.until { |lines| assert_equal '  10/10 (6)', lines[-2] }
    tmux.send_keys 'C-M'
    assert_equal %w[3 2 5 6 8 7], fzf_output_lines
  end

  def test_subword_forward
    tmux.send_keys "#{FZF} --bind K:kill-subword,F:forward-subword -q 'foo bar foo-bar fooFooBar'", :Enter, :Home
    tmux.until { |lines| assert_equal '> foo bar foo-bar fooFooBar', lines.last }

    tmux.send_keys 'F', :Delete
    tmux.until { |lines| assert_equal '> foobar foo-bar fooFooBar', lines.last }

    tmux.send_keys 'K'
    tmux.until { |lines| assert_equal '> foo foo-bar fooFooBar', lines.last }

    tmux.send_keys 'F', 'K'
    tmux.until { |lines| assert_equal '> foo foo fooFooBar', lines.last }

    tmux.send_keys 'F', 'F', 'K'
    tmux.until { |lines| assert_equal '> foo foo fooFoo', lines.last }
  end

  def test_subword_backward
    tmux.send_keys "#{FZF} --bind K:backward-kill-subword,B:backward-subword -q 'foo bar foo-bar fooBar'", :Enter
    tmux.until { |lines| assert_equal '> foo bar foo-bar fooBar', lines.last }

    tmux.send_keys 'B', :BSpace
    tmux.until { |lines| assert_equal '> foo bar foo-bar foBar', lines.last }

    tmux.send_keys 'K'
    tmux.until { |lines| assert_equal '> foo bar foo-bar Bar', lines.last }

    tmux.send_keys 'B', :BSpace
    tmux.until { |lines| assert_equal '> foo bar foobar Bar', lines.last }

    tmux.send_keys 'B', 'B', :BSpace
    tmux.until { |lines| assert_equal '> foobar foobar Bar', lines.last }
  end

  def test_multi_max
    tmux.send_keys "seq 1 10 | #{FZF} -m 3 --bind A:select-all,T:toggle-all --preview 'echo [{+}]/{}'", :Enter

    tmux.until { |lines| assert_equal 10, lines.match_count }

    tmux.send_keys '1'
    tmux.until do |lines|
      assert_includes lines[1], ' [1]/1 '
      assert lines[-2]&.start_with?('  2/10 ')
    end

    tmux.send_keys 'A'
    tmux.until do |lines|
      assert_includes lines[1], ' [1 10]/1 '
      assert lines[-2]&.start_with?('  2/10 (2/3)')
    end

    tmux.send_keys :BSpace
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (2/3)') }

    tmux.send_keys 'T'
    tmux.until do |lines|
      assert_includes lines[1], ' [2 3 4]/1 '
      assert lines[-2]&.start_with?('  10/10 (3/3)')
    end

    %w[T A].each do |key|
      tmux.send_keys key
      tmux.until do |lines|
        assert_includes lines[1], ' [1 5 6]/1 '
        assert lines[-2]&.start_with?('  10/10 (3/3)')
      end
    end

    tmux.send_keys :BTab
    tmux.until do |lines|
      assert_includes lines[1], ' [5 6]/2 '
      assert lines[-2]&.start_with?('  10/10 (2/3)')
    end

    [:BTab, :BTab, 'A'].each do |key|
      tmux.send_keys key
      tmux.until do |lines|
        assert_includes lines[1], ' [5 6 2]/3 '
        assert lines[-2]&.start_with?('  10/10 (3/3)')
      end
    end

    tmux.send_keys '2'
    tmux.until { |lines| assert lines[-2]&.start_with?('  1/10 (3/3)') }

    tmux.send_keys 'T'
    tmux.until do |lines|
      assert_includes lines[1], ' [5 6]/2 '
      assert lines[-2]&.start_with?('  1/10 (2/3)')
    end

    tmux.send_keys :BSpace
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (2/3)') }

    tmux.send_keys 'A'
    tmux.until do |lines|
      assert_includes lines[1], ' [5 6 1]/1 '
      assert lines[-2]&.start_with?('  10/10 (3/3)')
    end
  end

  def test_multi_action
    tmux.send_keys "seq 10 | #{FZF} --bind 'a:change-multi,b:change-multi(3),c:change-multi(xxx),d:change-multi(0)'", :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 ') }
    tmux.send_keys 'a'
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (0)') }
    tmux.send_keys 'b'
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (0/3)') }
    tmux.send_keys :BTab
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (1/3)') }
    tmux.send_keys 'c'
    tmux.send_keys :BTab
    tmux.until { |lines| assert lines[-2]&.start_with?('  10/10 (2/3)') }
    tmux.send_keys 'd'
    tmux.until do |lines|
      assert lines[-2]&.start_with?('  10/10 ') && !lines[-2]&.include?('(')
    end
  end

  def test_with_nth
    [true, false].each do |multi|
      tmux.send_keys "(echo '  1st 2nd 3rd/';
                       echo '  first second third/') |
                       #{fzf(multi && :multi, :x, :nth, 2, :with_nth, '2,-1,1')}",
                     :Enter
      tmux.until { |lines| assert_equal multi ? '  2/2 (0)' : '  2/2', lines[-2] }

      # Transformed list
      lines = tmux.capture
      assert_equal '  second third/first', lines[-4]
      assert_equal '> 2nd 3rd/1st',        lines[-3]

      # However, the output must not be transformed
      if multi
        tmux.send_keys :BTab, :BTab
        tmux.until { |lines| assert_equal '  2/2 (2)', lines[-2] }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/', '  first second third/'], fzf_output_lines
      else
        tmux.send_keys '^', '3'
        tmux.until { |lines| assert_equal '  1/2', lines[-2] }
        tmux.send_keys :Enter
        assert_equal ['  1st 2nd 3rd/'], fzf_output_lines
      end
    end
  end

  def test_scroll
    [true, false].each do |rev|
      tmux.send_keys "seq 1 100 | #{fzf(rev && :reverse)}", :Enter
      tmux.until { |lines| assert_equal '  100/100', lines[rev ? 1 : -2] }
      tmux.send_keys(*Array.new(110) { rev ? :Down : :Up })
      tmux.until { |lines| assert_includes lines, '> 100' }
      tmux.send_keys :Enter
      assert_equal '100', fzf_output
    end
  end

  def test_select_1
    tmux.send_keys "seq 1 100 | #{fzf(:with_nth, '..,..', :print_query, :q, 5555, :'1')}", :Enter
    assert_equal %w[5555 55], fzf_output_lines
  end

  def test_select_1_accept_nth
    tmux.send_keys "seq 1 100 | #{fzf(:with_nth, '..,..', :print_query, :q, 5555, :'1', :accept_nth, '"{1} // {1}"')}", :Enter
    assert_equal ['5555', '55 // 55'], fzf_output_lines
  end

  def test_exit_0
    tmux.send_keys "seq 1 100 | #{fzf(:with_nth, '..,..', :print_query, :q, 555_555, :'0')}", :Enter
    assert_equal %w[555555], fzf_output_lines
  end

  def test_select_1_exit_0_fail
    [:'0', :'1', %i[1 0]].each do |opt|
      tmux.send_keys "seq 1 100 | #{fzf(:print_query, :multi, :q, 5, *opt)}", :Enter
      tmux.until { |lines| assert_equal '> 5', lines.last }
      tmux.send_keys :BTab, :BTab, :BTab
      tmux.until { |lines| assert_equal '  19/100 (3)', lines[-2] }
      tmux.send_keys :Enter
      assert_equal %w[5 5 50 51], fzf_output_lines
    end
  end

  def test_query_unicode
    tmux.paste "(echo abc; echo $'\\352\\260\\200\\353\\202\\230\\353\\213\\244') | #{fzf(:query, "$'\\352\\260\\200\\353\\213\\244'")}"
    tmux.until { |lines| assert_equal '  1/2', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[가나다], fzf_output_lines
  end

  def test_sync
    tmux.send_keys "seq 1 100 | #{FZF} --multi | awk '{print $1 $1}' | #{fzf(:sync)}", :Enter
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys 9
    tmux.until { |lines| assert_equal '  19/100 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  19/100 (3)', lines[-2] }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys 'C-K', :Enter
    assert_equal %w[9090], fzf_output_lines
  end

  def test_tac
    tmux.send_keys "seq 1 1000 | #{fzf(:tac, :multi)}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  1000/1000 (3)', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[1000 999 998], fzf_output_lines
  end

  def test_tac_sort
    tmux.send_keys "seq 1 1000 | #{fzf(:tac, :multi)}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys '99'
    tmux.until { |lines| assert_equal '  28/1000 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  28/1000 (3)', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[99 999 998], fzf_output_lines
  end

  def test_tac_nosort
    tmux.send_keys "seq 1 1000 | #{fzf(:tac, :no_sort, :multi)}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys '00'
    tmux.until { |lines| assert_equal '  10/1000 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  10/1000 (3)', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[1000 900 800], fzf_output_lines
  end

  def test_expect
    test = lambda do |key, feed, expected = key|
      tmux.send_keys "seq 1 100 | #{fzf(:expect, key, :prompt, "[#{key}]")}", :Enter
      tmux.until { |lines| assert_equal '  100/100', lines[-2] }
      tmux.send_keys '55'
      tmux.until { |lines| assert_equal '  1/100', lines[-2] }
      tmux.send_keys(*feed)
      tmux.prepare
      assert_equal [expected, '55'], fzf_output_lines
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

  def test_expect_with_bound_actions
    tmux.send_keys "seq 1 100 | #{fzf('--query 1 --print-query --expect z --bind z:up+up')}", :Enter
    tmux.until { |lines| assert_equal 20, lines.match_count }
    tmux.send_keys('z')
    assert_equal %w[1 z 1], fzf_output_lines
  end

  def test_expect_print_query
    tmux.send_keys "seq 1 100 | #{fzf('--expect=alt-z', :print_query)}", :Enter
    tmux.until { |lines| assert_equal '  100/100', lines[-2] }
    tmux.send_keys '55'
    tmux.until { |lines| assert_equal '  1/100', lines[-2] }
    tmux.send_keys :Escape, :z
    assert_equal %w[55 alt-z 55], fzf_output_lines
  end

  def test_expect_printable_character_print_query
    tmux.send_keys "seq 1 100 | #{fzf('--expect=z --print-query')}", :Enter
    tmux.until { |lines| assert_equal '  100/100', lines[-2] }
    tmux.send_keys '55'
    tmux.until { |lines| assert_equal '  1/100', lines[-2] }
    tmux.send_keys 'z'
    assert_equal %w[55 z 55], fzf_output_lines
  end

  def test_expect_print_query_select_1
    tmux.send_keys "seq 1 100 | #{fzf('-q55 -1 --expect=alt-z --print-query')}", :Enter
    assert_equal ['55', '', '55'], fzf_output_lines
  end

  def test_toggle_sort
    ['--toggle-sort=ctrl-r', '--bind=ctrl-r:toggle-sort'].each do |opt|
      tmux.send_keys "seq 1 111 | #{fzf("-m +s --tac #{opt} -q11")}", :Enter
      tmux.until { |lines| assert_equal '> 111', lines[-3] }
      tmux.send_keys :Tab
      tmux.until { |lines| assert_equal '  4/111 -S (1)', lines[-2] }
      tmux.send_keys 'C-R'
      tmux.until { |lines| assert_equal '> 11', lines[-3] }
      tmux.send_keys :Tab
      tmux.until { |lines| assert_equal '  4/111 +S (2)', lines[-2] }
      tmux.send_keys :Enter
      assert_equal %w[111 11], fzf_output_lines
    end
  end

  def test_invalid_cache
    tmux.send_keys "(echo d; echo D; echo x) | #{fzf('-q d')}", :Enter
    tmux.until { |lines| assert_equal '  2/3', lines[-2] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '  3/3', lines[-2] }
    tmux.send_keys :D
    tmux.until { |lines| assert_equal '  1/3', lines[-2] }
    tmux.send_keys :Enter
  end

  def test_invalid_cache_query_type
    command = %[(echo 'foo$bar'; echo 'barfoo'; echo 'foo^bar'; echo "foo'1-2"; seq 100) | #{FZF}]

    # Suffix match
    tmux.send_keys command, :Enter
    tmux.until { |lines| assert_equal 104, lines.match_count }
    tmux.send_keys 'foo$'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'bar'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter

    # Prefix match
    tmux.prepare
    tmux.send_keys command, :Enter
    tmux.until { |lines| assert_equal 104, lines.match_count }
    tmux.send_keys '^bar'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'C-a', 'foo'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter

    # Exact match
    tmux.prepare
    tmux.send_keys command, :Enter
    tmux.until { |lines| assert_equal 104, lines.match_count }
    tmux.send_keys "'12"
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'C-a', 'foo'
    tmux.until { |lines| assert_equal 1, lines.match_count }
  end

  def test_bind
    tmux.send_keys "seq 1 1000 | #{fzf('-m --bind=ctrl-j:accept,u,:,U:up,X,,,Z:toggle-up,t:toggle')}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys 'uU:', 'X,Z', 'tt', 'uu', 'ttt', 'C-j'
    assert_equal %w[4 5 6 9], fzf_output_lines
  end

  def test_bind_print_query
    tmux.send_keys "seq 1 1000 | #{fzf('-m --bind=ctrl-j:print-query')}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys 'print-my-query', 'C-j'
    assert_equal %w[print-my-query], fzf_output_lines
  end

  def test_bind_replace_query
    tmux.send_keys "seq 1 1000 | #{fzf('--print-query --bind=ctrl-j:replace-query')}", :Enter
    tmux.send_keys '1'
    tmux.until { |lines| assert_equal '  272/1000', lines[-2] }
    tmux.send_keys 'C-k', 'C-j'
    tmux.until { |lines| assert_equal '  29/1000', lines[-2] }
    tmux.until { |lines| assert_equal '> 10', lines[-1] }
  end

  def test_select_all_deselect_all_toggle_all
    tmux.send_keys "seq 100 | #{fzf('--bind ctrl-a:select-all,ctrl-d:deselect-all,ctrl-t:toggle-all --multi')}", :Enter
    tmux.until { |lines| assert_equal '  100/100 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal '  100/100 (3)', lines[-2] }
    tmux.send_keys 'C-t'
    tmux.until { |lines| assert_equal '  100/100 (97)', lines[-2] }
    tmux.send_keys 'C-a'
    tmux.until { |lines| assert_equal '  100/100 (100)', lines[-2] }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| assert_equal '  100/100 (98)', lines[-2] }
    tmux.send_keys '100'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys 'C-d'
    tmux.until { |lines| assert_equal '  1/100 (97)', lines[-2] }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'C-d'
    tmux.until { |lines| assert_equal '  100/100 (0)', lines[-2] }
    tmux.send_keys :BTab, :BTab
    tmux.until { |lines| assert_equal '  100/100 (2)', lines[-2] }
    tmux.send_keys 0
    tmux.until { |lines| assert_equal '  10/100 (2)', lines[-2] }
    tmux.send_keys 'C-a'
    tmux.until { |lines| assert_equal '  10/100 (12)', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[1 2 10 20 30 40 50 60 70 80 90 100],
                 fzf_output_lines
  end

  def test_history
    history_file = '/tmp/fzf-test-history'

    # History with limited number of entries
    FileUtils.rm_f(history_file)
    opts = "--history=#{history_file} --history-size=4"
    input = %w[00 11 22 33 44]
    input.each do |keys|
      tmux.prepare
      tmux.send_keys "seq 100 | #{FZF} #{opts}", :Enter
      tmux.until { |lines| assert_equal '  100/100', lines[-2] }
      tmux.send_keys keys
      tmux.until { |lines| assert_equal '  1/100', lines[-2] }
      tmux.send_keys :Enter
    end
    wait do
      assert_path_exists history_file
      assert_equal input[1..], File.readlines(history_file, chomp: true)
    end

    # Update history entries (not changed on disk)
    tmux.send_keys "seq 100 | #{FZF} #{opts}", :Enter
    tmux.until { |lines| assert_equal '  100/100', lines[-2] }
    tmux.send_keys 'C-p'
    tmux.until { |lines| assert_equal '> 44', lines[-1] }
    tmux.send_keys 'C-p'
    tmux.until { |lines| assert_equal '> 33', lines[-1] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '> 3', lines[-1] }
    tmux.send_keys 1
    tmux.until { |lines| assert_equal '> 31', lines[-1] }
    tmux.send_keys 'C-p'
    tmux.until { |lines| assert_equal '> 22', lines[-1] }
    tmux.send_keys 'C-n'
    tmux.until { |lines| assert_equal '> 31', lines[-1] }
    tmux.send_keys 0
    tmux.until { |lines| assert_equal '> 310', lines[-1] }
    tmux.send_keys :Enter
    wait do
      assert_path_exists history_file
      assert_equal %w[22 33 44 310], File.readlines(history_file, chomp: true)
    end

    # Respect --bind option
    tmux.send_keys "seq 100 | #{FZF} #{opts} --bind ctrl-p:next-history,ctrl-n:previous-history", :Enter
    tmux.until { |lines| assert_equal '  100/100', lines[-2] }
    tmux.send_keys 'C-n', 'C-n', 'C-n', 'C-n', 'C-p'
    tmux.until { |lines| assert_equal '> 33', lines[-1] }
    tmux.send_keys :Enter
  ensure
    FileUtils.rm_f(history_file)
  end

  def test_cycle
    tmux.send_keys "seq 8 | #{FZF} --cycle", :Enter
    tmux.until { |lines| assert_equal '  8/8', lines[-2] }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal '> 8', lines[-10] }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal '> 7', lines[-9] }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal '> 8', lines[-10] }
    tmux.send_keys :PgUp
    tmux.until { |lines| assert_equal '> 8', lines[-10] }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal '> 1', lines[-3] }
    tmux.send_keys :PgDn
    tmux.until { |lines| assert_equal '> 1', lines[-3] }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal '> 8', lines[-10] }
  end

  def test_header_lines
    tmux.send_keys "seq 100 | #{fzf('--header-lines=10 -q 5')}", :Enter
    2.times do
      tmux.until do |lines|
        assert_equal '  18/90', lines[-2]
        assert_equal '  1', lines[-3]
        assert_equal '  2', lines[-4]
        assert_equal '> 50', lines[-13]
      end
      tmux.send_keys :Down
    end
    tmux.send_keys :Enter
    assert_equal '50', fzf_output
  end

  def test_header_lines_reverse
    tmux.send_keys "seq 100 | #{fzf('--header-lines=10 -q 5 --reverse')}", :Enter
    2.times do
      tmux.until do |lines|
        assert_equal '  18/90', lines[1]
        assert_equal '  1', lines[2]
        assert_equal '  2', lines[3]
        assert_equal '> 50', lines[12]
      end
      tmux.send_keys :Up
    end
    tmux.send_keys :Enter
    assert_equal '50', fzf_output
  end

  def test_header_lines_reverse_list
    tmux.send_keys "seq 100 | #{fzf('--header-lines=10 -q 5 --layout=reverse-list')}", :Enter
    2.times do
      tmux.until do |lines|
        assert_equal '  9', lines[8]
        assert_equal '  10', lines[9]
        assert_equal '> 50', lines[10]
        assert_equal '  18/90', lines[-2]
      end
      tmux.send_keys :Up
    end
    tmux.send_keys :Enter
    assert_equal '50', fzf_output
  end

  def test_header_lines_overflow
    tmux.send_keys "seq 100 | #{fzf('--header-lines=200')}", :Enter
    tmux.until do |lines|
      assert_equal '  0/0', lines[-2]
      assert_equal '  1', lines[-3]
    end
    tmux.send_keys :Enter
    assert_equal '', fzf_output
  end

  def test_header_lines_with_nth
    tmux.send_keys "seq 100 | #{fzf('--header-lines 5 --with-nth 1,1,1,1,1')}", :Enter
    tmux.until do |lines|
      assert_equal '  95/95', lines[-2]
      assert_equal '  11111', lines[-3]
      assert_equal '  55555', lines[-7]
      assert_equal '> 66666', lines[-8]
    end
    tmux.send_keys :Enter
    assert_equal '6', fzf_output
  end

  def test_header
    tmux.send_keys %[seq 100 | #{FZF} --header "$(head -5 #{FILE})"], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  100/100', lines[-2]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[-7..-3]
      assert_equal '> 1', lines[-8]
    end
  end

  def test_header_reverse
    tmux.send_keys %[seq 100 | #{FZF} --header "$(head -5 #{FILE})" --reverse], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  100/100', lines[1]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[2..6]
      assert_equal '> 1', lines[7]
    end
  end

  def test_header_reverse_list
    tmux.send_keys %[seq 100 | #{FZF} --header "$(head -5 #{FILE})" --layout=reverse-list], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  100/100', lines[-2]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[-7..-3]
      assert_equal '> 1', lines[0]
    end
  end

  def test_header_and_header_lines
    tmux.send_keys %[seq 100 | #{FZF} --header-lines 10 --header "$(head -5 #{FILE})"], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  90/90', lines[-2]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[-7...-2]
      assert_equal ('  1'..'  10').to_a.reverse, lines[-17...-7]
    end
  end

  def test_header_and_header_lines_reverse
    tmux.send_keys %[seq 100 | #{FZF} --reverse --header-lines 10 --header "$(head -5 #{FILE})"], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  90/90', lines[1]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[2...7]
      assert_equal ('  1'..'  10').to_a, lines[7...17]
    end
  end

  def test_header_and_header_lines_reverse_list
    tmux.send_keys %[seq 100 | #{FZF} --layout=reverse-list --header-lines 10 --header "$(head -5 #{FILE})"], :Enter
    header = File.readlines(FILE, chomp: true).take(5)
    tmux.until do |lines|
      assert_equal '  90/90', lines[-2]
      assert_equal header.map { |line| "  #{line}".rstrip }, lines[-7...-2]
      assert_equal ('  1'..'  10').to_a, lines.take(10)
    end
  end

  def test_cancel
    tmux.send_keys "seq 10 | #{FZF} --bind 2:cancel", :Enter
    tmux.until { |lines| assert_equal '  10/10', lines[-2] }
    tmux.send_keys '123'
    tmux.until do |lines|
      assert_equal '> 3', lines[-1]
      assert_equal '  1/10', lines[-2]
    end
    tmux.send_keys 'C-y', 'C-y'
    tmux.until { |lines| assert_equal '> 311', lines[-1] }
    tmux.send_keys 2
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys 2
    tmux.prepare
  end

  def test_margin
    tmux.send_keys "yes | head -1000 | #{FZF} --margin 5,3", :Enter
    tmux.until do |lines|
      assert_equal '', lines[4]
      assert_equal '     y', lines[5]
    end
    tmux.send_keys :Enter
  end

  def test_margin_reverse
    tmux.send_keys "seq 1000 | #{FZF} --margin 7,5 --reverse", :Enter
    tmux.until { |lines| assert_equal '       1000/1000', lines[1 + 7] }
    tmux.send_keys :Enter
  end

  def test_margin_reverse_list
    tmux.send_keys "yes | head -1000 | #{FZF} --margin 5,3 --layout=reverse-list", :Enter
    tmux.until do |lines|
      assert_equal '', lines[4]
      assert_equal '   > y', lines[5]
    end
    tmux.send_keys :Enter
  end

  def test_tabstop
    writelines(%W[f\too\tba\tr\tbaz\tbarfooq\tux])
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
        assert_equal exp, lines[-3]
      end
      tmux.send_keys :Enter
    end
  end

  def test_exit_0_exit_code
    `echo foo | #{FZF} -q bar -0`
    assert_equal 1, $CHILD_STATUS.exitstatus
  end

  def test_invalid_option
    lines = `#{FZF} --foobar 2>&1`
    assert_equal 2, $CHILD_STATUS.exitstatus
    assert_includes lines, 'unknown option: --foobar'
  end

  def test_exitstatus_empty
    { '99' => '0', '999' => '1' }.each do |query, status|
      tmux.send_keys "seq 100 | #{FZF} -q #{query}; echo --$?--", :Enter
      tmux.until { |lines| assert_match %r{ [10]/100}, lines[-2] }
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal "--#{status}--", lines.last }
    end
  end

  def test_hscroll_off
    writelines([('=' * 10_000) + '0123456789'])
    [0, 3, 6].each do |off|
      tmux.prepare
      tmux.send_keys "#{FZF} --hscroll-off=#{off} -q 0 --bind space:toggle-hscroll < #{tempname}", :Enter
      tmux.until { |lines| assert lines[-3]&.end_with?((0..off).to_a.join + '··') }
      tmux.send_keys '9'
      tmux.until { |lines| assert lines[-3]&.end_with?('789') }
      tmux.send_keys :Space
      tmux.until { |lines| assert lines[-3]&.end_with?('=··') }
      tmux.send_keys :Space
      tmux.until { |lines| assert lines[-3]&.end_with?('789') }
      tmux.send_keys :Enter
    end
  end

  def test_partial_caching
    tmux.send_keys 'seq 1000 | fzf -e', :Enter
    tmux.until { |lines| assert_equal '  1000/1000', lines[-2] }
    tmux.send_keys 11
    tmux.until { |lines| assert_equal '  19/1000', lines[-2] }
    tmux.send_keys 'C-a', "'"
    tmux.until { |lines| assert_equal '  28/1000', lines[-2] }
    tmux.send_keys :Enter
  end

  def test_jump
    tmux.send_keys "seq 1000 | #{fzf("--multi --jump-labels 12345 --bind 'ctrl-j:jump'")}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '5 5', lines[-7] }
    tmux.until { |lines| assert_equal '  6', lines[-8] }
    tmux.send_keys '5'
    tmux.until { |lines| assert_equal '> 5', lines[-7] }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal ' >5', lines[-7] }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '5>5', lines[-7] }
    tmux.send_keys '2'
    tmux.until { |lines| assert_equal '> 2', lines[-4] }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal ' >2', lines[-4] }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '5>5', lines[-7] }

    # Press any key other than jump labels to cancel jump
    tmux.send_keys '6'
    tmux.until { |lines| assert_equal '> 1', lines[-3] }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal '>>1', lines[-3] }
    tmux.send_keys :Enter
    assert_equal %w[5 2 1], fzf_output_lines
  end

  def test_jump_accept
    tmux.send_keys "seq 1000 | #{fzf("--multi --jump-labels 12345 --bind 'ctrl-j:jump-accept'")}", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '5 5', lines[-7] }
    tmux.send_keys '3'
    assert_equal '3', fzf_output
  end

  def test_jump_events
    tmux.send_keys "seq 1000 | #{FZF} --multi --jump-labels 12345 --bind 'ctrl-j:jump,jump:preview(echo jumped to {}),jump-cancel:preview(echo jump cancelled at {})'", :Enter
    tmux.until { |lines| assert_equal '  1000/1000 (0)', lines[-2] }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_includes lines[-7], '5 5' }
    tmux.send_keys '3'
    tmux.until { |lines| assert(lines.any? { it.include?('jumped to 3') }) }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_includes lines[-7], '5 5' }
    tmux.send_keys 'C-c'
    tmux.until { |lines| assert(lines.any? { it.include?('jump cancelled at 3') }) }
  end

  def test_jump_no_pointer
    tmux.send_keys "seq 100 | #{FZF} --pointer= --jump-labels 12345 --bind ctrl-j:jump", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '5 5', lines[-7] }
    tmux.send_keys 'C-c'
    tmux.until { |lines| assert_equal ' 5', lines[-7] }
  end

  def test_jump_no_pointer_no_marker
    tmux.send_keys "seq 100 | #{FZF} --pointer= --marker= --jump-labels 12345 --bind ctrl-j:jump", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'C-j'
    tmux.until { |lines| assert_equal '55', lines[-7] }
    tmux.send_keys 'C-c'
    tmux.until { |lines| assert_equal '5', lines[-7] }
  end

  def test_pointer
    tmux.send_keys "seq 10 | #{fzf("--pointer '>>'")}", :Enter
    # Assert that specified pointer is displayed
    tmux.until { |lines| assert_equal '>> 1', lines[-3] }
  end

  def test_pointer_with_jump
    tmux.send_keys "seq 10 | #{FZF} --multi --jump-labels 12345 --bind 'ctrl-j:jump' --pointer '>>'", :Enter
    tmux.until { |lines| assert_equal '  10/10 (0)', lines[-2] }
    tmux.send_keys 'C-j'
    # Correctly padded jump label should appear
    tmux.until { |lines| assert_equal '5  5', lines[-7] }
    tmux.until { |lines| assert_equal '   6', lines[-8] }
    tmux.send_keys '5'
    # Assert that specified pointer is displayed
    tmux.until { |lines| assert_equal '>> 5', lines[-7] }
  end

  def test_marker
    tmux.send_keys "seq 10 | #{FZF} --multi --marker '>>'", :Enter
    tmux.until { |lines| assert_equal '  10/10 (0)', lines[-2] }
    tmux.send_keys :BTab
    # Assert that specified marker is displayed
    tmux.until { |lines| assert_equal ' >>1', lines[-3] }
  end

  def test_no_clear
    tmux.send_keys "seq 10 | #{fzf('--no-clear --inline-info --height 5')}", :Enter
    prompt = '>   < 10/10'
    tmux.until { |lines| assert_equal prompt, lines[-1] }
    tmux.send_keys :Enter
    assert_equal %w[1], fzf_output_lines
    tmux.until { |lines| assert_equal prompt, lines[-1] }
  end

  def test_info_hidden
    tmux.send_keys 'seq 10 | fzf --info=hidden --no-separator', :Enter
    tmux.until { |lines| assert_equal '> 1', lines[-2] }
  end

  def test_info_inline_separator
    tmux.send_keys 'seq 10 | fzf --info=inline:___ --no-separator', :Enter
    tmux.until { |lines| assert_equal '>  ___10/10', lines[-1] }
  end

  def test_change_first_last
    tmux.send_keys %(seq 1000 | #{FZF} --bind change:first,alt-Z:last), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal '> 2', lines[-4] }
    tmux.send_keys 1
    tmux.until { |lines| assert_equal '> 1', lines[-3] }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal '> 10', lines[-4] }
    tmux.send_keys 1
    tmux.until { |lines| assert_equal '> 11', lines[-3] }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal '> 1', lines[-3] }
    tmux.send_keys :Escape, 'Z'
    tmux.until { |lines| assert_equal '> 1000', lines[0] }
    tmux.send_keys :Enter
  end

  def test_pos
    tmux.send_keys %(seq 1000 | #{FZF} --bind 'a:pos(3),b:pos(-3),c:pos(1),d:pos(-1),e:pos(0)' --preview 'echo {}/{}'), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys :a
    tmux.until { |lines| assert_includes lines[1], ' 3/3' }
    tmux.send_keys :b
    tmux.until { |lines| assert_includes lines[1], ' 998/998' }
    tmux.send_keys :c
    tmux.until { |lines| assert_includes lines[1], ' 1/1' }
    tmux.send_keys :d
    tmux.until { |lines| assert_includes lines[1], ' 1000/1000' }
    tmux.send_keys :e
    tmux.until { |lines| assert_includes lines[1], ' 1/1' }
  end

  def test_put
    tmux.send_keys %(seq 1000 | #{FZF} --bind 'a:put+put,b:put+put(ravo)' --preview 'echo {q}/{q}'), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys :a
    tmux.until { |lines| assert_includes lines[1], ' aa/aa' }
    tmux.send_keys :b
    tmux.until { |lines| assert_includes lines[1], ' aabravo/aabravo' }
  end

  def test_accept_non_empty
    tmux.send_keys %(seq 1000 | #{fzf('--print-query --bind enter:accept-non-empty')}), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys 'foo'
    tmux.until { |lines| assert_equal '  0/1000', lines[-2] }
    # fzf doesn't exit since there's no selection
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal '  0/1000', lines[-2] }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal '  1000/1000', lines[-2] }
    tmux.send_keys '999'
    tmux.until { |lines| assert_equal '  1/1000', lines[-2] }
    tmux.send_keys :Enter
    assert_equal %w[999 999], fzf_output_lines
  end

  def test_accept_non_empty_with_multi_selection
    tmux.send_keys %(seq 1000 | #{fzf('-m --print-query --bind enter:accept-non-empty')}), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal '  1000/1000 (1)', lines[-2] }
    tmux.send_keys 'foo'
    tmux.until { |lines| assert_equal '  0/1000 (1)', lines[-2] }
    # fzf will exit in this case even though there's no match for the current query
    tmux.send_keys :Enter
    assert_equal %w[foo 1], fzf_output_lines
  end

  def test_accept_non_empty_with_empty_list
    tmux.send_keys %(: | #{fzf('-q foo --print-query --bind enter:accept-non-empty')}), :Enter
    tmux.until { |lines| assert_equal '  0/0', lines[-2] }
    tmux.send_keys :Enter
    # fzf will exit anyway since input list is empty
    assert_equal %w[foo], fzf_output_lines
  end

  def test_accept_or_print_query_without_match
    tmux.send_keys %(seq 1000 | #{fzf('--bind enter:accept-or-print-query')}), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys 99_999
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :Enter
    assert_equal %w[99999], fzf_output_lines
  end

  def test_accept_or_print_query_with_match
    tmux.send_keys %(seq 1000 | #{fzf('--bind enter:accept-or-print-query')}), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys '^99$'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    assert_equal %w[99], fzf_output_lines
  end

  def test_accept_or_print_query_with_multi_selection
    tmux.send_keys %(seq 1000 | #{fzf('--bind enter:accept-or-print-query --multi')}), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_equal 3, lines.select_count }
    tmux.send_keys 99_999
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :Enter
    assert_equal %w[1 2 3], fzf_output_lines
  end

  def test_inverse_only_search_should_not_sort_the_result
    # Filter
    assert_equal %w[aaaaa b ccc],
                 `printf '%s\n' aaaaa b ccc BAD | #{FZF} -f '!bad'`.lines(chomp: true)

    # Interactive
    tmux.send_keys %(printf '%s\n' aaaaa b ccc BAD | #{FZF} -q '!bad'), :Enter
    tmux.until do |lines|
      assert_equal 4, lines.item_count
      assert_equal 3, lines.match_count
    end
    tmux.until { |lines| assert_equal '> aaaaa', lines[-3] }
    tmux.until { |lines| assert_equal '  b', lines[-4] }
    tmux.until { |lines| assert_equal '  ccc', lines[-5] }
  end

  def test_disabled
    tmux.send_keys %(seq 1000 | #{FZF} --query 333 --disabled --bind a:enable-search,b:disable-search,c:toggle-search --preview 'echo {} {q}'), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' 1 333 ' }
    tmux.send_keys 'foo'
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' 1 333foo ' }

    # Already disabled, no change
    tmux.send_keys 'b'
    tmux.until { |lines| assert_equal 1000, lines.match_count }

    # Enable search
    tmux.send_keys 'a'
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :BSpace, :BSpace, :BSpace
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' 333 333 ' }

    # Toggle search -> disabled again, but retains the previous result
    tmux.send_keys 'c'
    tmux.send_keys 'foo'
    tmux.until { |lines| assert_includes lines[1], ' 333 333foo ' }
    tmux.until { |lines| assert_equal 1, lines.match_count }

    # Enabled, no match
    tmux.send_keys 'c'
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.until { |lines| assert_includes lines[1], ' 333foo ' }
  end

  def test_clear_query
    tmux.send_keys %(: | #{FZF} --query foo --bind space:clear-query), :Enter
    tmux.until { |lines| assert_equal 0, lines.item_count }
    tmux.until { |lines| assert_equal '> foo', lines.last }
    tmux.send_keys 'C-a', 'bar'
    tmux.until { |lines| assert_equal '> barfoo', lines.last }
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal '>', lines.last }
  end

  def test_change_query
    tmux.send_keys %(: | #{FZF} --query foo --bind space:change-query:foobar), :Enter
    tmux.until { |lines| assert_equal 0, lines.item_count }
    tmux.until { |lines| assert_equal '> foo', lines.last }
    tmux.send_keys :Space, 'baz'
    tmux.until { |lines| assert_equal '> foobarbaz', lines.last }
  end

  def test_transform_query
    tmux.send_keys %{#{FZF} --bind 'ctrl-r:transform-query(rev <<< {q}),ctrl-u:transform-query: tr "[:lower:]" "[:upper:]" <<< {q}' --query bar}, :Enter
    tmux.until { |lines| assert_equal '> bar', lines[-1] }
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_equal '> rab', lines[-1] }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal '> RAB', lines[-1] }
  end

  def test_transform_prompt
    tmux.send_keys %{#{FZF} --bind 'ctrl-r:transform-query(rev <<< {q}),ctrl-u:transform-query: tr "[:lower:]" "[:upper:]" <<< {q}' --query bar}, :Enter
    tmux.until { |lines| assert_equal '> bar', lines[-1] }
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_equal '> rab', lines[-1] }
    tmux.send_keys 'C-u'
    tmux.until { |lines| assert_equal '> RAB', lines[-1] }
  end

  def test_transform
    tmux.send_keys %{#{FZF} --bind 'focus:transform:echo "change-prompt({fzf:action})"'}, :Enter
    tmux.until { |lines| assert_equal 'start', lines[-1] }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal 'up', lines[-1] }
  end

  def test_search
    tmux.send_keys %(seq 100 | #{FZF} --query 0 --bind space:search:1), :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal 20, lines.match_count }
    tmux.send_keys '0'
    tmux.until { |lines| assert_equal 1, lines.match_count }
  end

  def test_transform_search
    tmux.send_keys %(seq 1000 | #{FZF} --bind 'change:transform-search:echo {q}{q}'), :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    tmux.send_keys '1'
    tmux.until { |lines| assert_equal 28, lines.match_count }
    tmux.send_keys :BSpace, '0'
    tmux.until { |lines| assert_equal 10, lines.match_count }
  end

  def test_clear_selection
    tmux.send_keys %(seq 100 | #{FZF} --multi --bind space:clear-selection), :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal '  100/100 (1)', lines[-2] }
    tmux.send_keys 'foo'
    tmux.until { |lines| assert_equal '  0/100 (1)', lines[-2] }
    tmux.send_keys :Space
    tmux.until { |lines| assert_equal '  0/100 (0)', lines[-2] }
  end

  def test_backward_delete_char_eof
    tmux.send_keys "seq 1000 | #{FZF} --bind 'bs:backward-delete-char/eof'", :Enter
    tmux.until { |lines| assert_equal '  1000/1000', lines[-2] }
    tmux.send_keys '11'
    tmux.until { |lines| assert_equal '> 11', lines[-1] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '> 1', lines[-1] }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys :BSpace
    tmux.prepare
  end

  def test_strip_xterm_osc_sequence
    %W[\x07 \x1b\\].each do |esc|
      writelines([%(printf $1"\e]4;3;rgb:aa/bb/cc#{esc} "$2)])
      File.chmod(0o755, tempname)
      tmux.prepare
      tmux.send_keys \
        %(echo foo bar | #{FZF} --preview '#{tempname} {2} {1}'), :Enter

      tmux.until { |lines| assert lines.any_include?('bar foo') }
      tmux.send_keys :Enter
    end
  end

  def test_keep_right
    tmux.send_keys "seq 10000 | #{FZF} --read0 --keep-right --no-multi-line --bind space:toggle-multi-line", :Enter
    tmux.until { |lines| assert lines.any_include?('9999␊10000') }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('> 1') }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('9999␊10000') }
  end

  def test_freeze_left_keep_right
    tmux.send_keys %[seq 10000 | #{FZF} --read0 --delimiter "\n" --freeze-left 3 --keep-right --ellipsis XX --no-multi-line --bind space:toggle-multi-line], :Enter
    tmux.until { |lines| assert_match(/^> 1␊2␊3XX.*10000␊$/, lines[-3]) }
    tmux.send_keys '5'
    tmux.until { |lines| assert_match(/^> 1␊2␊3␊4␊5␊.*XX$/, lines[-3]) }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('> 1') }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('1␊2␊3␊4␊5␊') }
  end

  def test_freeze_left_and_right
    tmux.send_keys %[seq 10000 | tr "\n" ' ' | #{FZF} --freeze-left 3 --freeze-right 3 --ellipsis XX], :Enter
    tmux.until { |lines| assert_match(/XX9998 9999 10000$/, lines[-3]) }
    tmux.send_keys "'1000"
    tmux.until { |lines| assert_match(/^> 1 2 3XX.*XX9998 9999 10000$/,lines[-3]) }
  end

  def test_freeze_left_and_right_delimiter
    tmux.send_keys %[seq 10000 | tr "\n" ' ' | sed 's/ / , /g' | #{FZF} --freeze-left 3 --freeze-right 3 --ellipsis XX --delimiter ' , '], :Enter
    tmux.until { |lines| assert_match(/XX, 9999 , 10000 ,$/, lines[-3]) }
    tmux.send_keys "'1000"
    tmux.until { |lines| assert_match(/^> 1 , 2 , 3 ,XX.*XX, 9999 , 10000 ,$/,lines[-3]) }
  end

  def test_freeze_right_exceed_range
    tmux.send_keys %[seq 10000 | tr "\n" ' ' | #{FZF} --freeze-right 100000 --ellipsis XX], :Enter
    ['', "'1000"].each do |query|
      tmux.send_keys query
      tmux.until { |lines| assert lines.any_include?("> #{query}".strip) }
      tmux.until do |lines|
        assert_match(/ 9998 9999 10000$/, lines[-3])
        assert_equal(1, lines[-3].scan('XX').size)
      end
    end
  end

  def test_freeze_right_exceed_range_with_freeze_left
    tmux.send_keys %[seq 10000 | tr "\n" ' ' | #{FZF} --freeze-left 3  --freeze-right 100000 --ellipsis XX], :Enter
    tmux.until do |lines|
      assert_match(/^> 1 2 3XX.*9998 9999 10000$/, lines[-3])
      assert_equal(1, lines[-3].scan('XX').size)
    end
  end

  def test_freeze_right_with_ellipsis_and_scrolling
    tmux.send_keys "{ seq 6; ruby -e 'print \"g\"*1000, \"\\n\"'; seq 8 100; } | #{FZF} --ellipsis='777' --freeze-right 1 --scroll-off 0 --bind a:offset-up", :Enter
    tmux.until { |lines| assert_equal '  100/100', lines[-2] }
    tmux.send_keys(*Array.new(6) { :a })
    tmux.until do |lines|
      assert_match(/> 777g+$/, lines[-3])
      assert_equal 1, lines.count { |l| l.end_with?('g') }
    end
  end

  def test_backward_eof
    tmux.send_keys "echo foo | #{FZF} --bind 'backward-eof:reload(seq 100)'", :Enter
    tmux.until { |lines| lines.item_count == 1 && lines.match_count == 1 }
    tmux.send_keys 'x'
    tmux.until { |lines| lines.item_count == 1 && lines.match_count == 0 }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines.item_count == 1 && lines.match_count == 1 }
    tmux.send_keys :BSpace
    tmux.until { |lines| lines.item_count == 100 && lines.match_count == 100 }
  end

  def test_change_prompt
    tmux.send_keys "#{FZF} --bind 'a:change-prompt(a> ),b:change-prompt:b> ' --query foo", :Enter
    tmux.until { |lines| assert_equal '> foo', lines[-1] }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_equal 'a> foo', lines[-1] }
    tmux.send_keys 'b'
    tmux.until { |lines| assert_equal 'b> foo', lines[-1] }
  end

  def test_select_deselect
    tmux.send_keys "seq 3 | #{FZF} --multi --bind up:deselect+up,down:select+down", :Enter
    tmux.until { |lines| assert_equal 3, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 1, lines.select_count }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal 0, lines.select_count }
    tmux.send_keys :Down, :Down
    tmux.until { |lines| assert_equal 2, lines.select_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 1, lines.select_count }
    tmux.send_keys :Down, :Down
    tmux.until { |lines| assert_equal 2, lines.select_count }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal 1, lines.select_count }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal 1, lines.select_count }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal 2, lines.select_count }
  end

  def test_unbind_rebind_toggle_bind
    tmux.send_keys "seq 100 | #{FZF} --bind 'c:clear-query,d:unbind(c,d),e:rebind(c,d),f:toggle-bind(c)'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'ab'
    tmux.until { |lines| assert_equal '> ab', lines[-1] }
    tmux.send_keys 'c'
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys 'dabcd'
    tmux.until { |lines| assert_equal '> abcd', lines[-1] }
    tmux.send_keys 'ecabddc'
    tmux.until { |lines| assert_equal '> abdc', lines[-1] }
    tmux.send_keys 'fcabfc'
    tmux.until { |lines| assert_equal '> abc', lines[-1] }
    tmux.send_keys 'fc'
    tmux.until { |lines| assert_equal '>', lines[-1] }
  end

  def test_scroll_off
    tmux.send_keys "seq 1000 | #{FZF} --scroll-off=3 --bind l:last", :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    height = tmux.until { |lines| lines }.first.to_i
    tmux.send_keys :PgUp
    tmux.until do |lines|
      assert_equal height + 3, lines.first.to_i
      assert_equal "> #{height}", lines[3].strip
    end
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal "> #{height + 1}", lines[3].strip }
    tmux.send_keys 'l'
    tmux.until { |lines| assert_equal '> 1000', lines.first.strip }
    tmux.send_keys :PgDn
    tmux.until { |lines| assert_equal "> #{1000 - height + 1}", lines.reverse[5].strip }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal "> #{1000 - height}", lines.reverse[5].strip }
  end

  def test_scroll_off_large
    tmux.send_keys "seq 1000 | #{FZF} --scroll-off=9999", :Enter
    tmux.until { |lines| assert_equal 1000, lines.match_count }
    height = tmux.until { |lines| lines }.first.to_i
    tmux.send_keys :PgUp
    tmux.until { |lines| assert_equal "> #{height}", lines[height / 2].strip }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal "> #{height + 1}", lines[height / 2].strip }
    tmux.send_keys :Up
    tmux.until { |lines| assert_equal "> #{height + 2}", lines[height / 2].strip }
    tmux.send_keys :Down
    tmux.until { |lines| assert_equal "> #{height + 1}", lines[height / 2].strip }
  end

  def test_ellipsis
    tmux.send_keys 'seq 1000 | tr "\n" , | fzf --ellipsis=SNIPSNIP -e -q500', :Enter
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.until { |lines| assert_match(/^> SNIPSNIP.*SNIPSNIP$/, lines[-3]) }
  end

  def test_start_event
    tmux.send_keys 'seq 100 | fzf --multi --sync --preview-window hidden:border-none --bind "start:select-all+last+preview(echo welcome)"', :Enter
    tmux.until do |lines|
      assert_match(/>100.*welcome/, lines[0])
      assert_includes(lines[-2], '100/100 (100)')
    end
  end

  def test_focus_event
    tmux.send_keys 'seq 100 | fzf --bind "focus:transform-prompt(echo [[{}]]),?:unbind(focus)"', :Enter
    tmux.until { |lines| assert_includes(lines[-1], '[[1]]') }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes(lines[-1], '[[2]]') }
    tmux.send_keys :X
    tmux.until { |lines| assert_includes(lines[-1], '[[]]') }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_includes(lines[-1], '[[1]]') }
    tmux.send_keys :X
    tmux.until { |lines| assert_includes(lines[-1], '[[]]') }
    tmux.send_keys '?'
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.until { |lines| refute_includes(lines[-1], '[[1]]') }
  end

  def test_result_event
    tmux.send_keys '(echo 0; seq 10) | fzf --bind "result:pos(2)"', :Enter
    tmux.until { |lines| assert_equal 11, lines.match_count }
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys '9'
    tmux.until { |lines| assert_includes lines, '> 9' }
    tmux.send_keys :BSpace
    tmux.until { |lines| assert_includes lines, '> 1' }
  end

  def test_labels_center
    tmux.send_keys 'echo x | fzf --border --border-label foobar --preview : --preview-label barfoo --bind "space:change-border-label(foobarfoo)+change-preview-label(barfoobar),enter:transform-border-label(echo foo{}foo)+transform-preview-label(echo bar{}bar)"', :Enter
    tmux.until do
      assert_includes(it[0], '─foobar─')
      assert_includes(it[1], '─barfoo─')
    end
    tmux.send_keys :space
    tmux.until do
      assert_includes(it[0], '─foobarfoo─')
      assert_includes(it[1], '─barfoobar─')
    end
    tmux.send_keys :Enter
    tmux.until do
      assert_includes(it[0], '─fooxfoo─')
      assert_includes(it[1], '─barxbar─')
    end
  end

  def test_labels_left
    tmux.send_keys ': | fzf --border rounded --preview-window border-rounded --border-label foobar --border-label-pos 2 --preview : --preview-label barfoo --preview-label-pos 2', :Enter
    tmux.until do
      assert_includes(it[0], '╭foobar─')
      assert_includes(it[1], '╭barfoo─')
    end
  end

  def test_labels_right
    tmux.send_keys ': | fzf --border rounded --preview-window border-rounded --border-label foobar --border-label-pos -2 --preview : --preview-label barfoo --preview-label-pos -2', :Enter
    tmux.until do
      assert_includes(it[0], '─foobar╮')
      assert_includes(it[1], '─barfoo╮')
    end
  end

  def test_labels_bottom
    tmux.send_keys ': | fzf --border rounded --preview-window border-rounded --border-label foobar --border-label-pos 2:bottom --preview : --preview-label barfoo --preview-label-pos -2:bottom', :Enter
    tmux.until do
      assert_includes(it[-1], '╰foobar─')
      assert_includes(it[-2], '─barfoo╯')
    end
  end

  def test_labels_variables
    tmux.send_keys ': | fzf --border --border-label foobar --preview "echo \$FZF_BORDER_LABEL // \$FZF_PREVIEW_LABEL" --preview-label barfoo --bind "space:change-border-label(barbaz)+change-preview-label(bazbar)+refresh-preview,enter:transform-border-label(echo 123)+transform-preview-label(echo 456)+refresh-preview"', :Enter
    tmux.until do
      assert_includes(it[0], '─foobar─')
      assert_includes(it[1], '─barfoo─')
      assert_includes(it[2], ' foobar // barfoo ')
    end
    tmux.send_keys :Space
    tmux.until do
      assert_includes(it[0], '─barbaz─')
      assert_includes(it[1], '─bazbar─')
      assert_includes(it[2], ' barbaz // bazbar ')
    end
    tmux.send_keys :Enter
    tmux.until do
      assert_includes(it[0], '─123─')
      assert_includes(it[1], '─456─')
      assert_includes(it[2], ' 123 // 456 ')
    end
  end

  def test_info_separator_unicode
    tmux.send_keys 'seq 100 | fzf -q55', :Enter
    tmux.until { assert_includes(it[-2], '  1/100 ─') }
  end

  def test_info_separator_no_unicode
    tmux.send_keys 'seq 100 | fzf -q55 --no-unicode', :Enter
    tmux.until { assert_includes(it[-2], '  1/100 -') }
  end

  def test_info_separator_repeat
    tmux.send_keys 'seq 100 | fzf -q55 --separator _-', :Enter
    tmux.until { assert_includes(it[-2], '  1/100 _-_-') }
  end

  def test_info_separator_ansi_colors_and_tabs
    tmux.send_keys "seq 100 | fzf -q55 --tabstop 4 --separator $'\\x1b[33ma\\tb'", :Enter
    tmux.until { assert_includes(it[-2], '  1/100 a   ba   ba') }
  end

  def test_info_no_separator
    tmux.send_keys 'seq 100 | fzf -q55 --no-separator', :Enter
    tmux.until { assert_operator(it[-2], :==, '  1/100') }
  end

  def test_info_right
    tmux.send_keys "#{FZF} --info=right --separator x --bind 'start:reload:seq 100; sleep 10'", :Enter
    tmux.until { assert_match(%r{xxx [⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏] 100/100}, it[-2]) }
  end

  def test_info_inline_right
    tmux.send_keys "#{FZF} --info=inline-right --bind 'start:reload:seq 100; sleep 10'", :Enter
    tmux.until { assert_match(%r{[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏] 100/100}, it[-1]) }
  end

  def test_info_inline_right_clearance
    tmux.send_keys "seq 100000 | #{FZF} --info inline-right", :Enter
    tmux.until { assert_match(%r{100000/100000}, it[-1]) }
    tmux.send_keys 'x'
    tmux.until { assert_match(%r{     0/100000}, it[-1]) }
  end

  def test_info_command
    tmux.send_keys(%(seq 10000 | #{FZF} --separator x --info-command 'echo -e "--\\x1b[33m$FZF_POS\\x1b[m/$FZF_INFO--"'), :Enter)
    tmux.until { assert_match(%r{^  --1/10000/10000-- xx}, it[-2]) }
    tmux.send_keys :Up
    tmux.until { assert_match(%r{^  --2/10000/10000-- xx}, it[-2]) }
  end

  def test_info_command_inline
    tmux.send_keys(%(seq 10000 | #{FZF} --separator x --info-command 'echo -e "--\\x1b[33m$FZF_POS\\x1b[m/$FZF_INFO--"' --info inline:xx), :Enter)
    tmux.until { assert_match(%r{^>  xx--1/10000/10000-- xx}, it[-1]) }
  end

  def test_info_command_right
    tmux.send_keys(%(seq 10000 | #{FZF} --separator x --info-command 'echo -e "--\\x1b[33m$FZF_POS\\x1b[m/$FZF_INFO--"' --info right), :Enter)
    tmux.until { assert_match(%r{xx --1/10000/10000-- *$}, it[-2]) }
  end

  def test_info_command_inline_right
    tmux.send_keys(%(seq 10000 | #{FZF} --info-command 'echo -e "--\\x1b[33m$FZF_POS\\x1b[m/$FZF_INFO--"' --info inline-right), :Enter)
    tmux.until { assert_match(%r{   --1/10000/10000-- *$}, it[-1]) }
  end

  def test_info_command_inline_right_no_ansi
    tmux.send_keys(%(seq 10000 | #{FZF} --info-command 'echo -e "--$FZF_POS/$FZF_INFO--"' --info inline-right), :Enter)
    tmux.until { assert_match(%r{   --1/10000/10000-- *$}, it[-1]) }
  end

  def test_info_command_and_focus
    tmux.send_keys(%(seq 100 | #{FZF} --separator x --info-command 'echo $FZF_POS' --bind focus:clear-query), :Enter)
    tmux.until { assert_match(/^  1 xx/, it[-2]) }
    tmux.send_keys :Up
    tmux.until { assert_match(/^  2 xx/, it[-2]) }
  end

  def test_prev_next_selected
    tmux.send_keys 'seq 10 | fzf --multi --bind ctrl-n:next-selected,ctrl-p:prev-selected', :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.send_keys :BTab, :BTab, :Up, :BTab
    tmux.until { |lines| assert_equal 3, lines.select_count }
    tmux.send_keys 'C-n'
    tmux.until { |lines| assert_includes lines, '>>4' }
    tmux.send_keys 'C-n'
    tmux.until { |lines| assert_includes lines, '>>2' }
    tmux.send_keys 'C-n'
    tmux.until { |lines| assert_includes lines, '>>1' }
    tmux.send_keys 'C-n'
    tmux.until { |lines| assert_includes lines, '>>4' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| assert_includes lines, '>>1' }
    tmux.send_keys 'C-p'
    tmux.until { |lines| assert_includes lines, '>>2' }
  end

  def test_track
    tmux.send_keys "seq 1000 | #{FZF} --query 555 --track --bind t:toggle-track", :Enter
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, '> 555'
    end
    tmux.send_keys :BSpace
    index = tmux.until do |lines|
      assert_equal 28, lines.match_count
      assert_includes lines, '> 555'
    end.index('> 555')
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 271, lines.match_count
      assert_equal '> 555', lines[index]
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_equal '> 555', lines[index]
    end
    tmux.send_keys '555'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, '> 555'
      assert_includes lines[-2], '+T'
    end
    tmux.send_keys 't'
    tmux.until do |lines|
      refute_includes lines[-2], '+T'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 28, lines.match_count
      assert_includes lines, '> 55'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 271, lines.match_count
      assert_includes lines, '> 5'
    end
    tmux.send_keys 't'
    tmux.until do |lines|
      assert_includes lines[-2], '+T'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 5'
    end
  end

  def test_track_action
    tmux.send_keys "seq 1000 | #{FZF} --pointer x --query 555 --bind t:track,T:up+track", :Enter
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, 'x 555'
      assert_includes lines, '> 555'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 28, lines.match_count
      assert_includes lines, 'x 55'
      assert_includes lines, '> 55'
    end
    tmux.send_keys :t
    tmux.until do |lines|
      assert_includes lines[-2], '+t'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 271, lines.match_count
      assert_includes lines, 'x 55'
      assert_includes lines, '> 5'
    end

    # Automatically disabled when the tracking item is no longer visible
    tmux.send_keys '4'
    tmux.until do |lines|
      assert_equal 28, lines.match_count
      refute_includes lines[-2], '+t'
    end
    tmux.send_keys :BSpace
    tmux.until do |lines|
      assert_equal 271, lines.match_count
      assert_includes lines, 'x 52'
      assert_includes lines, '> 5'
    end
    tmux.send_keys :t
    tmux.until do |lines|
      assert_includes lines[-2], '+t'
    end

    # Automatically disabled when the focus has moved
    tmux.send_keys :Up
    tmux.until do |lines|
      assert_includes lines, 'x 53'
      refute_includes lines[-2], '+t'
    end

    # Should work even when combined with a focus moving actions
    tmux.send_keys 'T'
    tmux.until do |lines|
      assert_includes lines, 'x 54'
      assert_includes lines[-2], '+t'
    end

    tmux.send_keys 'T'
    tmux.until do |lines|
      assert_includes lines, 'x 55'
      assert_includes lines[-2], '+t'
    end
  end

  def test_one_and_zero
    tmux.send_keys "seq 10 | #{FZF} --bind 'zero:preview(echo no match),one:preview(echo {} is the only match)'", :Enter
    tmux.send_keys '1'
    tmux.until do |lines|
      assert_equal 2, lines.match_count
      refute(lines.any? { it.include?('only match') })
      refute(lines.any? { it.include?('no match') })
    end
    tmux.send_keys '0'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert(lines.any? { it.include?('only match') })
    end
    tmux.send_keys '0'
    tmux.until do |lines|
      assert_equal 0, lines.match_count
      assert(lines.any? { it.include?('no match') })
    end
  end

  def test_height_range_with_exit_0
    tmux.send_keys "seq 10 | #{FZF} --height ~10% --exit-0", :Enter
    tmux.until { |lines| assert_equal 10, lines.match_count }
    tmux.send_keys :c
    tmux.until { |lines| assert_equal 0, lines.match_count }
  end

  def test_delete_with_modifiers
    if ENV['GITHUB_ACTION']
      # Expected: "[3]"
      # Actual: "[]3;5~"
      skip('CTRL-DELETE is not properly handled in GitHub Actions environment')
    end
    tmux.send_keys "seq 100 | #{FZF} --bind 'ctrl-delete:up+up,shift-delete:down,focus:transform-prompt:echo [{}]'", :Enter
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys 'C-Delete'
    tmux.until { |lines| assert_equal '[3]', lines[-1] }
    tmux.send_keys 'S-Delete'
    tmux.until { |lines| assert_equal '[2]', lines[-1] }
  end

  def test_fzf_pos
    tmux.send_keys "seq 100 | #{FZF} --preview 'echo $FZF_POS / $FZF_MATCH_COUNT'", :Enter
    tmux.until { |lines| assert(lines.any? { |line| line.include?('1 / 100') }) }
    tmux.send_keys :Up
    tmux.until { |lines| assert(lines.any? { |line| line.include?('2 / 100') }) }
    tmux.send_keys '99'
    tmux.until { |lines| assert(lines.any? { |line| line.include?('1 / 1') }) }
    tmux.send_keys '99'
    tmux.until { |lines| assert(lines.any? { |line| line.include?('0 / 0') }) }
  end

  def test_change_nth
    input = [
      *[''] * 1000,
      'foo bar bar bar bar',
      'foo foo bar bar bar',
      'foo foo foo bar bar',
      'foo foo foo foo bar',
      *[''] * 1000
    ]
    writelines(input)
    nths = '1,2..4,-1,-3..,..2'
    tmux.send_keys %(#{FZF} -qfoo -n#{nths} --bind 'space:change-nth(2|3|4|5|),result:transform-prompt:echo "[$FZF_NTH] "' < #{tempname}), :Enter

    tmux.until do |lines|
      assert lines.any_include?("[#{nths}] foo")
      assert_equal 4, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines.any_include?('[2] foo')
      assert_equal 3, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines.any_include?('[3] foo')
      assert_equal 2, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines.any_include?('[4] foo')
      assert_equal 1, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines.any_include?('[5] foo')
      assert_equal 0, lines.match_count
    end
    tmux.send_keys :Space
    tmux.until do |lines|
      assert lines.any_include?("[#{nths}] foo")
      assert_equal 4, lines.match_count
    end
  end

  def test_env_vars
    def env_vars
      return {} unless File.exist?(tempname)

      File.readlines(tempname).select { it.start_with?('FZF_') }.to_h do
        key, val = it.chomp.split('=', 2)
        [key.to_sym, val]
      end
    end

    tmux.send_keys %(seq 100 | #{FZF} --multi --reverse --preview-window 0 --preview 'env | grep ^FZF_ | sort > #{tempname}' --no-input --bind enter:show-input+refresh-preview,space:disable-search+refresh-preview), :Enter
    expected = {
      FZF_DIRECTION: 'down',
      FZF_TOTAL_COUNT: '100',
      FZF_MATCH_COUNT: '100',
      FZF_SELECT_COUNT: '0',
      FZF_ACTION: 'start',
      FZF_KEY: '',
      FZF_POS: '1',
      FZF_QUERY: '',
      FZF_POINTER: '>',
      FZF_PROMPT: '> ',
      FZF_INPUT_STATE: 'hidden'
    }
    tmux.until do
      assert_equal expected, env_vars.slice(*expected.keys)
    end
    tmux.send_keys :Enter
    tmux.until do
      expected.merge!(FZF_INPUT_STATE: 'enabled', FZF_ACTION: 'show-input', FZF_KEY: 'enter')
      assert_equal expected, env_vars.slice(*expected.keys)
    end
    tmux.send_keys :Tab, :Tab
    tmux.until do
      expected.merge!(FZF_ACTION: 'toggle-down', FZF_KEY: 'tab', FZF_POS: '3', FZF_SELECT_COUNT: '2')
      assert_equal expected, env_vars.slice(*expected.keys)
    end
    tmux.send_keys '99'
    tmux.until do
      expected.merge!(FZF_ACTION: 'char', FZF_KEY: '9', FZF_QUERY: '99', FZF_MATCH_COUNT: '1', FZF_POS: '1')
      assert_equal expected, env_vars.slice(*expected.keys)
    end
    tmux.send_keys :Space
    tmux.until do
      expected.merge!(FZF_INPUT_STATE: 'disabled', FZF_ACTION: 'disable-search', FZF_KEY: 'space')
      assert_equal expected, env_vars.slice(*expected.keys)
    end
  end

  def test_abort_action_chain
    tmux.send_keys %(seq 100 | #{FZF} --bind 'load:accept+up+up' > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      assert_equal '1', File.read(tempname).chomp
    end
    tmux.send_keys %(seq 100 | #{FZF} --bind 'load:abort+become(echo {})' > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      assert_equal '', File.read(tempname).chomp
    end
  end

  def test_exclude_multi
    tmux.send_keys %(seq 1000 | #{FZF} --multi --bind 'a:exclude-multi,b:reload(seq 1000),c:reload-sync(seq 1000)'), :Enter

    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 1'
    end
    tmux.send_keys :a
    tmux.until do |lines|
      assert_includes lines, '> 2'
      assert_equal 999, lines.match_count
    end
    tmux.send_keys :Up, :BTab, :BTab, :BTab, :a
    tmux.until do |lines|
      assert_equal 996, lines.match_count
      assert_includes lines, '> 9'
    end
    tmux.send_keys :b
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 5'
    end
    tmux.send_keys :Tab, :Tab, :Tab, :a
    tmux.until do |lines|
      assert_equal 997, lines.match_count
      assert_includes lines, '> 2'
    end
    tmux.send_keys :c
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 2'
    end

    # TODO: We should also check the behavior of 'exclude' during reloads
  end

  def test_exclude
    tmux.send_keys %(seq 1000 | #{FZF} --multi --bind 'a:exclude,b:reload(seq 1000),c:reload-sync(seq 1000)'), :Enter

    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 1'
    end
    tmux.send_keys :a
    tmux.until do |lines|
      assert_includes lines, '> 2'
      assert_equal 999, lines.match_count
    end
    tmux.send_keys :Up, :BTab, :BTab, :BTab, :a
    tmux.until do |lines|
      assert_equal 998, lines.match_count
      assert_equal 3, lines.select_count
      assert_includes lines, '> 7'
    end
    tmux.send_keys :b
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_equal 0, lines.select_count
      assert_includes lines, '> 5'
    end
    tmux.send_keys :Tab, :Tab, :Tab, :a
    tmux.until do |lines|
      assert_equal 999, lines.match_count
      assert_equal 3, lines.select_count
      assert_includes lines, '>>3'
    end
    tmux.send_keys :a
    tmux.until do |lines|
      assert_equal 998, lines.match_count
      assert_equal 2, lines.select_count
      assert_includes lines, '>>4'
    end
    tmux.send_keys :c
    tmux.until do |lines|
      assert_equal 1000, lines.match_count
      assert_includes lines, '> 2'
    end

    # TODO: We should also check the behavior of 'exclude' during reloads
  end

  def test_accept_nth
    tmux.send_keys %((echo "foo  bar  baz"; echo "bar baz  foo") | #{FZF} --multi --accept-nth 2,2 --sync --bind start:select-all+accept > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      assert_equal ['bar  bar', 'baz  baz'], File.readlines(tempname, chomp: true)
    end
  end

  def test_accept_nth_string_delimiter
    tmux.send_keys %(echo "foo  ,bar,baz" | #{FZF} -d, --accept-nth 2,2,1,3,1 --sync --bind start:accept > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      # Last delimiter and the whitespaces are removed
      assert_equal ['bar,bar,foo  ,bazfoo'], File.readlines(tempname, chomp: true)
    end
  end

  def test_accept_nth_regex_delimiter
    tmux.send_keys %(echo "foo  :,:bar,baz" | #{FZF} --delimiter='[:,]+' --accept-nth 2,2,1,3,1 --sync --bind start:accept > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      # Last delimiter and the whitespaces are removed
      assert_equal ['bar,bar,foo  :,:bazfoo'], File.readlines(tempname, chomp: true)
    end
  end

  def test_accept_nth_regex_delimiter_strip_last
    tmux.send_keys %((echo "foo:,bar:,baz"; echo "foo:,bar:,baz:,qux:,") | #{FZF} --multi --delimiter='[:,]+' --accept-nth 2.. --sync --bind 'load:select-all+accept' > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      # Last delimiter and the whitespaces are removed
      assert_equal ['bar:,baz', 'bar:,baz:,qux'], File.readlines(tempname, chomp: true)
    end
  end

  def test_accept_nth_template
    tmux.send_keys %(echo "foo  ,bar,baz" | #{FZF} -d, --accept-nth '[{n}] 1st: {1}, 3rd: {3}, 2nd: {2}' --sync --bind start:accept > #{tempname}), :Enter
    wait do
      assert_path_exists tempname
      # Last delimiter and the whitespaces are removed
      assert_equal ['[0] 1st: foo, 3rd: baz, 2nd: bar'], File.readlines(tempname, chomp: true)
    end
  end

  def test_ghost
    tmux.send_keys %(seq 100 | #{FZF} --prompt 'X ' --ghost 'Type in query ...' --bind 'space:change-ghost:Y Z' --bind 'enter:transform-ghost:echo Z Y'), :Enter
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      assert_includes lines, 'X Type in query ...'
    end
    tmux.send_keys '100'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, 'X 100'
    end
    tmux.send_keys 'C-u'
    tmux.until do |lines|
      assert_equal 100, lines.match_count
      assert_includes lines, 'X Type in query ...'
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines, 'X Y Z' }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_includes lines, 'X Z Y' }
  end

  def test_ghost_inline
    tmux.send_keys %(seq 100 | #{FZF} --info 'inline: Y' --no-separator --prompt 'X ' --ghost 'Type in query ...'), :Enter
    tmux.until do |lines|
      assert_includes lines, 'X Type in query ... Y100/100'
    end
    tmux.send_keys '100'
    tmux.until do |lines|
      assert_includes lines, 'X 100  Y1/100'
    end
    tmux.send_keys 'C-u'
    tmux.until do |lines|
      assert_includes lines, 'X Type in query ... Y100/100'
    end
  end

  def test_offset_middle
    tmux.send_keys %(seq 1000 | #{FZF} --sync --no-input --reverse --height 5 --scroll-off 0 --bind space:offset-middle), :Enter
    line = nil
    tmux.until { |lines| line = lines.index('> 1') }
    tmux.send_keys :PgDn
    tmux.until { |lines| assert_includes lines[line + 4], '> 5' }
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines[line + 2], '> 5' }
  end

  def test_no_input_query
    tmux.send_keys %(seq 1000 | #{FZF} --no-input --query 555 --bind space:toggle-input), :Enter
    tmux.until { |lines| assert_includes lines, '> 555' }
    tmux.send_keys :Space
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, '> 555'
    end
  end

  def test_no_input_change_query
    tmux.send_keys %(seq 1000 | #{FZF} --multi --query 999 --no-input --bind 'enter:show-input+change-query(555)+hide-input,space:change-query(555)+select'), :Enter
    tmux.until { |lines| assert_includes lines, '> 999' }
    tmux.send_keys :Space
    tmux.until do |lines|
      assert_includes lines, '>>999'
      refute_includes lines, '> 555'
    end
    tmux.send_keys :Enter
    tmux.until do |lines|
      refute_includes lines, '>>999'
      assert_includes lines, '> 555'
    end
  end

  def test_search_override_query_in_no_input_mode
    tmux.send_keys %(seq 1000 | #{FZF} --sync --no-input --bind 'enter:show-input+change-query(555)+hide-input+search(999),space:search(111)+show-input+change-query(777)'), :Enter
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_includes lines, '> 999' }
    tmux.send_keys :Space
    tmux.until { |lines| assert_includes lines, '> 777' }
  end

  def test_change_pointer
    tmux.send_keys %(seq 2 | #{FZF} --bind 'a:change-pointer(a),b:change-pointer(bb),c:change-pointer(),d:change-pointer(ddd)'), :Enter
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines, 'a 1' }
    tmux.send_keys 'b'
    tmux.until { |lines| assert_includes lines, 'bb 1' }
    tmux.send_keys 'c'
    tmux.until { |lines| assert_includes lines, ' 1' }
    tmux.send_keys 'd'
    tmux.until { |lines| refute_includes lines, 'ddd 1' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines, ' 2' }
  end

  def test_transform_pointer
    tmux.send_keys %(seq 2 | #{FZF} --bind 'a:transform-pointer(echo a),b:transform-pointer(echo bb),c:transform-pointer(),d:transform-pointer(echo ddd)'), :Enter
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys 'a'
    tmux.until { |lines| assert_includes lines, 'a 1' }
    tmux.send_keys 'b'
    tmux.until { |lines| assert_includes lines, 'bb 1' }
    tmux.send_keys 'c'
    tmux.until { |lines| assert_includes lines, ' 1' }
    tmux.send_keys 'd'
    tmux.until { |lines| refute_includes lines, 'ddd 1' }
    tmux.send_keys :Up
    tmux.until { |lines| assert_includes lines, ' 2' }
  end

  def test_change_header_on_header_window
    tmux.send_keys %(seq 100 | #{FZF} --list-border --input-border --bind 'start:change-header(foo),space:change-header(bar)'), :Enter
    tmux.until do |lines|
      assert lines.any_include?('100/100')
      assert lines.any_include?('foo')
    end
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('bar') }
  end

  def test_trailing_new_line
    tmux.send_keys %(echo -en "foo\n" | fzf --read0 --no-multi-line), :Enter
    tmux.until { |lines| assert_includes lines, '> foo␊' }
  end

  def test_async_transform
    time = Time.now
    tmux.send_keys %(
      seq 100 | #{FZF} --style full --border --preview : \
          --bind 'focus:bg-transform-header(sleep 0.5; echo th.)' \
          --bind 'focus:+bg-transform-footer(sleep 0.5; echo tf.)' \
          --bind 'focus:+bg-transform-border-label(sleep 0.5; echo tbl.)' \
          --bind "focus:+bg-transform-preview-label(sleep 0.5; echo tpl.)" \
          --bind 'focus:+bg-transform-input-label(sleep 0.5; echo til.)' \
          --bind 'focus:+bg-transform-list-label(sleep 0.5; echo tll.)' \
          --bind 'focus:+bg-transform-header-label(sleep 0.5; echo thl.)' \
          --bind 'focus:+bg-transform-footer-label(sleep 0.5; echo tfl.)' \
          --bind 'focus:+bg-transform-prompt(sleep 0.5; echo tp.)' \
          --bind 'focus:+bg-transform-ghost(sleep 0.5; echo tg.)'
    ).strip, :Enter
    tmux.until do |lines|
      assert lines.any_include?('100/100')
      %w[th tf tbl tpl til tll thl tfl tp tg].each do
        assert lines.any_include?("#{it}.")
      end
    end
    elapsed = Time.now - time
    assert_operator elapsed, :<, 2
  end

  def test_bg_cancel
    tmux.send_keys %(seq 0 1 | #{FZF} --bind 'space:bg-cancel+bg-transform-header(sleep {}; echo [{}])'), :Enter
    tmux.until { assert_equal 2, it.match_count }
    tmux.send_keys '1'
    tmux.until { assert_equal 1, it.match_count }
    tmux.send_keys :Space
    tmux.send_keys :BSpace
    tmux.until { assert_equal 2, it.match_count }
    tmux.send_keys :Space
    tmux.until { |lines| assert lines.any_include?('[0]') }
    sleep(2)
    tmux.until do |lines|
      assert lines.any_include?('[0]')
      refute lines.any_include?('[1]')
    end
  end

  def test_render_order
    tmux.send_keys %(seq 100 | #{FZF} --bind='focus:preview(echo boom)+change-footer(bam)'), :Enter
    tmux.until { assert_equal 100, it.match_count }
    tmux.until { assert it.any_include?('boom') }
    tmux.until { assert it.any_include?('bam') }
  end

  def test_multi_event
    tmux.send_keys %(seq 100 | #{FZF} --multi --bind 'multi:transform-footer:(( FZF_SELECT_COUNT )) && echo "Selected $FZF_SELECT_COUNT item(s)"'), :Enter
    tmux.until { assert_equal 100, it.match_count }
    tmux.send_keys :Tab
    tmux.until { assert_equal 1, it.select_count }
    tmux.until { assert it.any_include?('Selected 1 item(s)') }
    tmux.send_keys :Tab
    tmux.until { assert_equal 0, it.select_count }
    tmux.until { refute it.any_include?('Selected') }
  end

  def test_preserve_selection_on_revision_bump
    tmux.send_keys %(seq 100 | #{FZF} --multi --sync --query "'1" --bind 'a:select-all+change-header(pressed a),b:change-header(pressed b)+change-nth(1),c:exclude'), :Enter
    tmux.until do
      assert_equal 20, it.match_count
      assert_equal 0, it.select_count
    end
    tmux.send_keys :a
    tmux.until do
      assert_equal 20, it.match_count
      assert_equal 20, it.select_count
      assert it.any_include?('pressed a')
    end
    tmux.send_keys :b
    tmux.until do
      assert_equal 20, it.match_count
      assert_equal 20, it.select_count
      refute it.any_include?('pressed a')
      assert it.any_include?('pressed b')
    end
    tmux.send_keys :a
    tmux.until do
      assert_equal 20, it.match_count
      assert_equal 20, it.select_count
      assert it.any_include?('pressed a')
      refute it.any_include?('pressed b')
    end
    tmux.send_keys :c
    tmux.until do
      assert_equal 19, it.match_count
      assert_equal 19, it.select_count
    end
  end

  def test_trigger
    tmux.send_keys %(seq 100 | #{FZF} --bind 'a:up+trigger(a),b:trigger(a,a,b,a)'), :Enter
    tmux.until { assert_equal 100, it.match_count }
    tmux.until { |lines| assert_includes lines, '> 1' }
    tmux.send_keys :a
    tmux.until { |lines| assert_includes lines, '> 3' }
    tmux.send_keys :b
    tmux.until { |lines| assert_includes lines, '> 9' }
  end

  def test_change_nth_unset_default
    tmux.send_keys %(echo foo bar | #{FZF} --nth 2 --query fb --bind space:change-nth:), :Enter
    tmux.until do
      assert_equal 1, it.item_count
      assert_equal 0, it.match_count
    end

    tmux.send_keys :Space

    tmux.until do
      assert_equal 1, it.item_count
      assert_equal 1, it.match_count
    end
  end
end
