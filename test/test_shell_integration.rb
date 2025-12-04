# frozen_string_literal: true

require_relative 'lib/common'

# Testing shell integration
module TestShell
  attr_reader :tmux

  def setup
    @tmux = Tmux.new(shell)
    tmux.prepare
  end

  def teardown
    @tmux.kill
  end

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "export #{name}='#{val}'", :Enter
    tmux.prepare
  end

  def unset_var(name)
    tmux.prepare
    tmux.send_keys "unset #{name}", :Enter
    tmux.prepare
  end

  def test_ctrl_t
    set_var('FZF_CTRL_T_COMMAND', 'seq 100')

    tmux.prepare
    tmux.send_keys 'C-t'
    tmux.until { |lines| assert_equal 100, lines.match_count }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| assert lines.any_include?(' (3)') }
    tmux.send_keys :Enter
    tmux.until { |lines| assert lines.any_include?('1 2 3') }
    tmux.send_keys 'C-c'
  end

  def test_ctrl_t_unicode
    writelines(['fzf-unicode 테스트1', 'fzf-unicode 테스트2'])
    set_var('FZF_CTRL_T_COMMAND', "cat #{tempname}")

    tmux.prepare
    tmux.send_keys 'echo ', 'C-t'
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.send_keys 'fzf-unicode'
    tmux.until { |lines| assert_equal 2, lines.match_count }

    tmux.send_keys '1'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 1, lines.select_count }

    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 2, lines.match_count }

    tmux.send_keys '2'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 2, lines.select_count }

    tmux.send_keys :Enter
    tmux.until { |lines| assert_match(/echo .*fzf-unicode.*1.* .*fzf-unicode.*2/, lines.join) }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'fzf-unicode 테스트1 fzf-unicode 테스트2', lines[-1] }
  end

  def test_alt_c
    tmux.prepare
    tmux.send_keys :Escape, :c
    lines = tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    expected = lines.reverse.find { |l| l.start_with?('> ') }[2..].chomp('/')
    tmux.send_keys :Enter
    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| assert lines[-1]&.end_with?(expected) }
  end

  def test_alt_c_command
    set_var('FZF_ALT_C_COMMAND', 'echo /tmp')

    tmux.prepare
    tmux.send_keys 'cd /', :Enter

    tmux.prepare
    tmux.send_keys :Escape, :c
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter

    tmux.prepare
    tmux.send_keys :pwd, :Enter
    tmux.until { |lines| assert_equal '/tmp', lines[-1] }
  end

  def test_ctrl_r
    tmux.prepare
    tmux.send_keys 'echo 1st', :Enter
    tmux.prepare
    tmux.send_keys 'echo 2nd', :Enter
    tmux.prepare
    tmux.send_keys 'echo 3d', :Enter
    tmux.prepare
    3.times do
      tmux.send_keys 'echo 3rd', :Enter
      tmux.prepare
    end
    tmux.send_keys 'echo 4th', :Enter
    tmux.prepare
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys 'e3d'
    # Duplicates removed: 3d (1) + 3rd (1) => 2 matches
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.until { |lines| assert lines[-3]&.end_with?(' echo 3d') }
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert lines[-3]&.end_with?(' echo 3rd') }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'echo 3rd', lines[-1] }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal '3rd', lines[-1] }
  end

  def test_ctrl_r_multiline
    # NOTE: Current bash implementation shows an extra new line if there's
    # only entry in the history
    tmux.send_keys ':', :Enter
    tmux.send_keys 'echo "foo', :Enter, 'bar"', :Enter
    tmux.until { |lines| assert_equal %w[foo bar], lines[-2..] }
    tmux.prepare
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_equal '>', lines[-1] }
    tmux.send_keys 'foo bar'
    tmux.until { |lines| assert_includes lines[-4], '"foo' } unless shell == :zsh
    tmux.until { |lines| assert lines[-3]&.match?(/bar"␊?/) }
    tmux.send_keys :Enter
    tmux.until { |lines| assert lines[-1]&.match?(/bar"␊?/) }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal %w[foo bar], lines[-2..] }
  end

  def test_ctrl_r_abort
    skip("doesn't restore the original line when search is aborted pre Bash 4") if shell == :bash && `#{Shell.bash} --version`[/(?<= version )\d+/].to_i < 4
    %w[foo ' "].each do |query|
      tmux.prepare
      tmux.send_keys :Enter, query
      tmux.until { |lines| assert lines[-1]&.start_with?(query) }
      tmux.send_keys 'C-r'
      tmux.until { |lines| assert_equal "> #{query}", lines[-1] }
      tmux.send_keys 'C-g'
      tmux.until { |lines| assert lines[-1]&.start_with?(query) }
    end
  end
end

module CompletionTest
  def test_file_completion
    FileUtils.mkdir_p('/tmp/fzf-test')
    FileUtils.mkdir_p('/tmp/fzf test')
    (1..100).each { |i| FileUtils.touch("/tmp/fzf-test/#{i}") }
    ['no~such~user', '/tmp/fzf test/foobar'].each do |f|
      FileUtils.touch(File.expand_path(f))
    end
    tmux.prepare
    tmux.send_keys 'cat /tmp/fzf-test/10**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys ' !d'
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.send_keys :Tab, :Tab
    tmux.until { |lines| assert_equal 2, lines.select_count }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      assert_equal 'cat /tmp/fzf-test/10 /tmp/fzf-test/100', lines[-1]
    end

    # ~USERNAME**<TAB>
    user = `whoami`.chomp
    tmux.send_keys 'C-u'
    tmux.send_keys "cat ~#{user}**", :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys "/#{user}"
    tmux.until { |lines| assert(lines.any? { |l| l.end_with?("/#{user}") }) }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      assert_match %r{cat .*/#{user}}, lines[-1]
    end

    # ~INVALID_USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat ~such**', :Tab
    tmux.until(true) { |lines| assert lines.any_include?('no~such~user') }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'cat no~such~user', lines[-1] }

    # /tmp/fzf\ test**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat /tmp/fzf\ test/**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys 'foobar$'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert lines.any_include?('> /tmp/fzf test/foobar')
    end
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'cat /tmp/fzf\ test/foobar', lines[-1] }

    # Should include hidden files
    (1..100).each { |i| FileUtils.touch("/tmp/fzf-test/.hidden-#{i}") }
    tmux.send_keys 'C-u'
    tmux.send_keys 'cat /tmp/fzf-test/hidden**', :Tab
    tmux.until(true) do |lines|
      assert_equal 100, lines.match_count
      assert lines.any_include?('/tmp/fzf-test/.hidden-')
    end
    tmux.send_keys :Enter
  ensure
    ['/tmp/fzf-test', '/tmp/fzf test', '~/.fzf-home', 'no~such~user'].each do |f|
      FileUtils.rm_rf(File.expand_path(f))
    end
  end

  def test_file_completion_root
    tmux.send_keys 'ls /**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys :Enter
  end

  def test_dir_completion
    (1..100).each do |idx|
      FileUtils.mkdir_p("/tmp/fzf-test/d#{idx}")
    end
    FileUtils.touch('/tmp/fzf-test/d55/xxx')
    tmux.prepare
    tmux.send_keys 'cd /tmp/fzf-test/**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys :Tab, :Tab # Tab does not work here
    tmux.send_keys 55
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert_includes lines, '> 55'
      assert_includes lines, '> /tmp/fzf-test/d55/'
    end
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'cd /tmp/fzf-test/d55/', lines[-1] }
    tmux.send_keys :xx
    tmux.until { |lines| assert_equal 'cd /tmp/fzf-test/d55/xx', lines[-1] }

    # Should not match regular files (bash-only)
    if instance_of?(TestBash)
      tmux.send_keys :Tab
      tmux.until { |lines| assert_equal 'cd /tmp/fzf-test/d55/xx', lines[-1] }
    end

    # Fail back to plusdirs
    tmux.send_keys :BSpace, :BSpace, :BSpace
    tmux.until { |lines| assert_equal 'cd /tmp/fzf-test/d55', lines[-1] }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 'cd /tmp/fzf-test/d55/', lines[-1] }
  end

  def test_process_completion
    tmux.send_keys 'sleep 12345 &', :Enter
    lines = tmux.until { |lines| assert lines[-1]&.start_with?('[1] ') }
    pid = lines[-1]&.split&.last
    tmux.prepare
    tmux.send_keys 'C-L'
    tmux.send_keys 'kill **', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys 'sleep12345'
    tmux.until { |lines| assert lines.any_include?('sleep 12345') }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal "kill #{pid}", lines[-1] }
  ensure
    if pid
      begin
        Process.kill('KILL', pid.to_i)
      rescue StandardError
        nil
      end
    end
  end

  def test_custom_completion
    tmux.send_keys '_fzf_compgen_path() { echo "$1"; seq 10; }', :Enter
    tmux.prepare
    tmux.send_keys 'ls /tmp/**', :Tab
    tmux.until { |lines| assert_equal 11, lines.match_count }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| assert_equal 3, lines.select_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls /tmp 1 2', lines[-1] }
  end

  def test_unset_completion
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    tmux.prepare

    # Using tmux
    tmux.send_keys 'unset FZFFOOBR**', :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'unset FZFFOOBAR', lines[-1] }
    tmux.send_keys 'C-c'

    # FZF_TMUX=1
    new_shell
    tmux.focus
    tmux.send_keys 'unset FZFFOOBR**', :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'unset FZFFOOBAR', lines[-1] }
  end

  def test_completion_in_command_sequence
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    tmux.prepare

    triggers = ['**', '~~', '++', 'ff', '/']
    triggers.push('&', '[', ';', '`') if instance_of?(TestZsh)

    triggers.each do |trigger|
      set_var('FZF_COMPLETION_TRIGGER', trigger)
      command = "echo foo; QUX=THUD unset FZFFOOBR#{trigger}"
      tmux.send_keys command.sub(/(;|`)$/, '\\\\\1'), :Tab
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal 'echo foo; QUX=THUD unset FZFFOOBAR', lines[-1] }
    end
  end

  def test_file_completion_unicode
    FileUtils.mkdir_p('/tmp/fzf-test')
    tmux.paste "cd /tmp/fzf-test; echo test3 > $'fzf-unicode \\355\\205\\214\\354\\212\\244\\355\\212\\2701'; echo test4 > $'fzf-unicode \\355\\205\\214\\354\\212\\244\\355\\212\\2702'"
    tmux.prepare
    tmux.send_keys 'cat fzf-unicode**', :Tab
    tmux.until { |lines| assert_equal 2, lines.match_count }

    tmux.send_keys '1'
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 1, lines.select_count }

    tmux.send_keys :BSpace
    tmux.until { |lines| assert_equal 2, lines.match_count }

    tmux.send_keys '2'
    tmux.until { |lines| assert_equal 1, lines.select_count }
    tmux.send_keys :Tab
    tmux.until { |lines| assert_equal 2, lines.select_count }

    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_match(/cat .*fzf-unicode.*1.* .*fzf-unicode.*2/, lines[-1]) }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal %w[test3 test4], lines[-2..] }
  end

  def test_custom_completion_api
    tmux.send_keys 'eval "_fzf$(declare -f _comprun)"', :Enter
    %w[f g].each do |command|
      tmux.prepare
      tmux.send_keys "#{command} b**", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.item_count
        assert_equal 1, lines.match_count
        assert lines.any_include?("prompt-#{command}")
        assert lines.any_include?("preview-#{command}-bar")
      end
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal "#{command} #{command}barbar", lines[-1] }
      tmux.send_keys 'C-u'
    end
  ensure
    tmux.prepare
    tmux.send_keys 'unset -f _fzf_comprun', :Enter
  end

  def test_ssh_completion
    (1..5).each { |i| FileUtils.touch("/tmp/fzf-test-ssh-#{i}") }

    tmux.send_keys 'ssh jg@localhost**', :Tab
    tmux.until do |lines|
      assert_operator lines.match_count, :>=, 1
    end

    tmux.send_keys :Enter
    tmux.until { |lines| assert lines.any_include?('ssh jg@localhost') }
    tmux.send_keys ' -i /tmp/fzf-test-ssh**', :Tab
    tmux.until do |lines|
      assert_operator lines.match_count, :>=, 5
      assert_equal 0, lines.select_count
    end
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until do |lines|
      assert_equal 3, lines.select_count
    end
    tmux.send_keys :Enter
    tmux.until { |lines| assert lines.any_include?('ssh jg@localhost  -i /tmp/fzf-test-ssh-') }

    tmux.send_keys 'localhost**', :Tab
    tmux.until do |lines|
      assert_operator lines.match_count, :>=, 1
    end
  end
end

class TestBash < TestBase
  include TestShell
  include CompletionTest

  def shell
    :bash
  end

  def new_shell
    tmux.prepare
    tmux.send_keys "FZF_TMUX=1 #{Shell.bash}", :Enter
    tmux.prepare
  end

  def test_dynamic_completion_loader
    tmux.paste 'touch /tmp/foo; _fzf_completion_loader=1'
    tmux.paste '_completion_loader() { complete -o default fake; }'
    tmux.paste 'complete -F _fzf_path_completion -o default -o bashdefault fake'
    tmux.send_keys 'fake /tmp/foo**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys 'C-c'

    tmux.prepare
    tmux.send_keys 'fake /tmp/foo'
    tmux.send_keys :Tab, 'C-u'

    tmux.prepare
    tmux.send_keys 'fake /tmp/foo**', :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
  end
end

class TestZsh < TestBase
  include TestShell
  include CompletionTest

  def shell
    :zsh
  end

  def new_shell
    tmux.send_keys "FZF_TMUX=1 #{Shell.zsh}", :Enter
    tmux.prepare
  end

  def test_complete_quoted_command
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    ['unset', '\unset', "'unset'"].each do |command|
      tmux.prepare
      tmux.send_keys "#{command} FZFFOOBR**", :Tab
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal "#{command} FZFFOOBAR", lines[-1] }
      tmux.send_keys 'C-c'
    end
  end

  # Helper function to run test with Perl and again with Awk
  def self.test_perl_and_awk(name, &block)
    define_method("test_#{name}") do
      instance_eval(&block)
    end

    define_method("test_#{name}_awk") do
      tmux.send_keys "unset 'commands[perl]'", :Enter
      tmux.prepare
      # Verify perl is actually unset (0 = not found)
      tmux.send_keys 'echo ${+commands[perl]}', :Enter
      tmux.until { |lines| assert_equal '0', lines[-1] }
      tmux.prepare
      instance_eval(&block)
    end
  end

  def prepare_ctrl_r_test
    tmux.send_keys ':', :Enter
    tmux.send_keys 'echo match-collision', :Enter
    tmux.prepare
    tmux.send_keys 'echo "line 1', :Enter, '2 line 2"', :Enter
    tmux.prepare
    tmux.send_keys 'echo "foo', :Enter, 'bar"', :Enter
    tmux.prepare
    tmux.send_keys 'echo "bar', :Enter, 'foo"', :Enter
    tmux.prepare
    tmux.send_keys 'echo "trailing_space "', :Enter
    tmux.prepare
    tmux.send_keys 'cat <<EOF | wc -c', :Enter, 'qux thud', :Enter, 'EOF', :Enter
    tmux.prepare
    tmux.send_keys 'C-l', 'C-r'
  end

  test_perl_and_awk 'ctrl_r_accept_or_print_query' do
    set_var('FZF_CTRL_R_OPTS', '--bind enter:accept-or-print-query')
    prepare_ctrl_r_test
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys '1 foobar'
    tmux.until { |lines| assert_equal 0, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal '1 foobar', lines[-1] }
  end

  test_perl_and_awk 'ctrl_r_multiline_index_collision' do
    # Leading number in multi-line history content is not confused with index
    prepare_ctrl_r_test
    tmux.send_keys "'line 1"
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until do |lines|
      assert_equal ['echo "line 1', '2 line 2"'], lines[-2..]
    end
  end

  test_perl_and_awk 'ctrl_r_multi_selection' do
    prepare_ctrl_r_test
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| assert_includes lines[-2], '(3)' }
    tmux.send_keys :Enter
    tmux.until do |lines|
      assert_equal ['cat <<EOF | wc -c', 'qux thud', 'EOF', 'echo "trailing_space "', 'echo "bar', 'foo"'], lines[-6..]
    end
  end

  test_perl_and_awk 'ctrl_r_no_multi_selection' do
    set_var('FZF_CTRL_R_OPTS', '--no-multi')
    prepare_ctrl_r_test
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys :BTab, :BTab, :BTab
    tmux.until { |lines| refute_includes lines[-2], '(3)' }
    tmux.send_keys :Enter
    tmux.until do |lines|
      assert_equal ['cat <<EOF | wc -c', 'qux thud', 'EOF'], lines[-3..]
    end
  end

  # NOTE: 'Perl/$history' won't see foreign cmds immediately, unlike 'awk/fc'.
  # Perl passes only because another cmd runs between mocking and triggering C-r
  # https://github.com/junegunn/fzf/issues/4061
  # https://zsh.org/mla/users/2024/msg00692.html
  test_perl_and_awk 'ctrl_r_foreign_commands' do
    histfile = "#{tempname}-foreign-hist"
    tmux.send_keys "HISTFILE=#{histfile}", :Enter
    tmux.prepare
    # SHARE_HISTORY picks up foreign commands; marked with * in fc
    tmux.send_keys 'setopt SHARE_HISTORY', :Enter
    tmux.prepare
    tmux.send_keys 'fzf_cmd_local', :Enter
    tmux.prepare
    # Mock foreign command (for testing only; don't edit your HISTFILE this way)
    tmux.send_keys "echo ': 0:0;fzf_cmd_foreign' >> $HISTFILE", :Enter
    tmux.prepare
    # Verify fc shows foreign command with asterisk
    tmux.send_keys 'fc -rl -1', :Enter
    tmux.until { |lines| assert lines.any? { |l| l.match?(/^\s*\d+\* fzf_cmd_foreign/) } }
    tmux.prepare
    # Test ctrl-r correctly extracts the foreign command
    tmux.send_keys 'C-r'
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys '^fzf_cmd_'
    tmux.until { |lines| assert_equal 2, lines.match_count }
    tmux.send_keys :BTab, :BTab
    tmux.until { |lines| assert_includes lines[-2], '(2)' }
    tmux.send_keys :Enter
    tmux.until do |lines|
      assert_equal ['fzf_cmd_foreign', 'fzf_cmd_local'], lines[-2..]
    end
  ensure
    FileUtils.rm_f(histfile)
  end
end

class TestFish < TestBase
  include TestShell

  def shell
    :fish
  end

  def new_shell
    tmux.send_keys 'env FZF_TMUX=1 FZF_DEFAULT_OPTS=--no-scrollbar fish', :Enter
    tmux.send_keys 'function fish_prompt; end; clear', :Enter
    tmux.until { |lines| assert_empty lines }
  end

  def set_var(name, val)
    tmux.prepare
    tmux.send_keys "set -g #{name} '#{val}'", :Enter
    tmux.prepare
  end

  def test_ctrl_r_multi
    tmux.send_keys ':', :Enter
    tmux.send_keys 'echo "foo', :Enter, 'bar"', :Enter
    tmux.prepare
    tmux.send_keys 'echo "bar', :Enter, 'foo"', :Enter
    tmux.prepare
    tmux.send_keys 'C-l', 'C-r'
    block = <<~BLOCK
      echo "foo
      bar"
      echo "bar
      foo"
    BLOCK
    tmux.until do |lines|
      block.lines.each_with_index do |line, idx|
        assert_includes lines[-6 + idx], line.chomp
      end
    end
    tmux.send_keys :BTab, :BTab
    tmux.until { |lines| assert_includes lines[-2], '(2)' }
    tmux.send_keys :Enter
    block = <<~BLOCK
      echo "bar
      foo"
      echo "foo
      bar"
    BLOCK
    tmux.until do |lines|
      assert_equal block.lines.map(&:chomp), lines
    end
  end
end
