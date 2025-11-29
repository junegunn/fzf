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

  def trigger
    '**'
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
    tmux.send_keys "cat /tmp/fzf-test/10#{trigger}", :Tab
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
    tmux.send_keys "cat ~#{user}#{trigger}", :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys "/#{user}"
    tmux.until { |lines| assert(lines.any? { |l| l.end_with?("/#{user}") }) }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      assert_match %r{cat .*/#{user}}, lines[-1]
    end

    # ~INVALID_USERNAME**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys "cat ~such#{trigger}", :Tab
    tmux.until(true) { |lines| assert lines.any_include?('no~such~user') }
    tmux.send_keys :Enter
    tmux.until(true) do |lines|
      if shell == :fish
        # Fish's string escape quotes filenames with ~ to prevent tilde expansion
        assert_equal 'cat no\\~such\\~user', lines[-1]
      else
        assert_equal 'cat no~such~user', lines[-1]
      end
    end

    # /tmp/fzf\ test**<TAB>
    tmux.send_keys 'C-u'
    tmux.send_keys "cat /tmp/fzf\\ test/#{trigger}", :Tab
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
    tmux.send_keys "cat /tmp/fzf-test/hidden#{trigger}", :Tab
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
    tmux.send_keys "ls /#{trigger}", :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys :Enter
  end

  def test_dir_completion
    (1..100).each do |idx|
      FileUtils.mkdir_p("/tmp/fzf-test/d#{idx}")
    end
    FileUtils.touch('/tmp/fzf-test/d55/xxx')
    tmux.prepare
    tmux.send_keys "cd /tmp/fzf-test/#{trigger}", :Tab
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
    skip('fish background job format differs') if shell == :fish
    tmux.send_keys 'sleep 12345 &', :Enter
    lines = tmux.until { |lines| assert lines[-1]&.start_with?('[1] ') }
    pid = lines[-1]&.split&.last
    tmux.prepare
    tmux.send_keys 'C-L'
    tmux.send_keys "kill #{trigger}", :Tab
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
    if shell == :fish
      tmux.send_keys 'function _fzf_compgen_path; echo $argv[1]; seq 10; end', :Enter
    else
      tmux.send_keys '_fzf_compgen_path() { echo "$1"; seq 10; }', :Enter
    end
    tmux.prepare
    tmux.send_keys "ls /tmp/#{trigger}", :Tab
    tmux.until { |lines| assert_equal 11, lines.match_count }
    tmux.send_keys :Tab, :Tab, :Tab
    tmux.until { |lines| assert_equal 3, lines.select_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls /tmp 1 2', lines[-1] }
  end

  def test_unset_completion
    skip('fish has native completion for set and unset variables') if shell == :fish
    tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    tmux.prepare

    # Using tmux
    tmux.send_keys "unset FZFFOOBR#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'unset FZFFOOBAR', lines[-1] }
    tmux.send_keys 'C-c'

    # FZF_TMUX=1
    new_shell
    tmux.focus
    tmux.send_keys "unset FZFFOOBR#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'unset FZFFOOBAR', lines[-1] }
  end

  def test_completion_in_command_sequence
    if shell == :fish
      FileUtils.mkdir_p('/tmp/fzf-test-seq')
      FileUtils.touch('/tmp/fzf-test-seq/fzffoobar')
    else
      tmux.send_keys 'export FZFFOOBAR=BAZ', :Enter
    end
    tmux.prepare

    triggers = ['**', '~~', '++', 'ff', '/']
    triggers.push('&', '[', ';', '`') if instance_of?(TestZsh)

    triggers.each do |trigger|
      set_var('FZF_COMPLETION_TRIGGER', trigger)
      if shell == :fish
        command = "echo foo; QUX=THUD ls /tmp/fzf-test-seq/fzffoobr#{trigger}"
        expected = 'echo foo; QUX=THUD ls /tmp/fzf-test-seq/fzffoobar'
      else
        command = "echo foo; QUX=THUD unset FZFFOOBR#{trigger}"
        expected = 'echo foo; QUX=THUD unset FZFFOOBAR'
      end
      tmux.send_keys command.sub(/(;|`)$/, '\\\\\1'), :Tab
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal expected, lines[-1] }
    end
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-seq') if shell == :fish
  end

  def test_file_completion_unicode
    FileUtils.mkdir_p('/tmp/fzf-test')
    # Shell-agnostic file creation
    File.write('/tmp/fzf-test/fzf-unicode 테스트1', "test3\n")
    File.write('/tmp/fzf-test/fzf-unicode 테스트2', "test4\n")
    tmux.send_keys 'cd /tmp/fzf-test', :Enter
    tmux.prepare
    tmux.send_keys "cat fzf-unicode#{trigger}", :Tab
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
    skip('bash-specific _comprun/declare syntax') if shell == :fish
    tmux.send_keys 'eval "_fzf$(declare -f _comprun)"', :Enter
    %w[f g].each do |command|
      tmux.prepare
      tmux.send_keys "#{command} b#{trigger}", :Tab
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
    skip('fish uses native ssh completion') if shell == :fish
    (1..5).each { |i| FileUtils.touch("/tmp/fzf-test-ssh-#{i}") }

    tmux.send_keys "ssh jg@localhost#{trigger}", :Tab
    tmux.until do |lines|
      assert_operator lines.match_count, :>=, 1
    end

    tmux.send_keys :Enter
    tmux.until { |lines| assert lines.any_include?('ssh jg@localhost') }
    tmux.send_keys " -i /tmp/fzf-test-ssh#{trigger}", :Tab
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

    tmux.send_keys "localhost#{trigger}", :Tab
    tmux.until do |lines|
      assert_operator lines.match_count, :>=, 1
    end
  end

  def test_option_equals_completion
    FileUtils.mkdir_p('/tmp/fzf-test-opt-eq')
    FileUtils.touch('/tmp/fzf-test-opt-eq/file1.txt')
    FileUtils.touch('/tmp/fzf-test-opt-eq/file2.txt')
    tmux.prepare

    # Test --opt=**<TAB>
    if shell != :zsh
      tmux.send_keys "some-command --output=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '1'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('--output=/tmp/fzf-test-opt-eq/file1.txt') }
    end

    # Test -o=**<TAB>
    if shell != :zsh
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -o=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '2'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-o=/tmp/fzf-test-opt-eq/file2.txt') }
    end

    # Test --opt=/**<TAB> (long option with equals and slash prefix)
    if shell != :zsh
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command --output=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '1'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('--output=/tmp/fzf-test-opt-eq/file1.txt') }
    end

    # Test -o=/**<TAB> (short option with equals and slash prefix)
    if shell != :zsh
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -o=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '2'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-o=/tmp/fzf-test-opt-eq/file2.txt') }
    end

    # Test -o/**<TAB> (short option without equals)
    if shell == :fish
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -o/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '2'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-o/tmp/fzf-test-opt-eq/file2.txt') }
    end

    # Test -- --opt=**<TAB>
    if shell == :bash
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -- --output=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '1'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-- --output=/tmp/fzf-test-opt-eq/file1.txt') }
    end

    # Test -- -o=**<TAB>
    if shell == :bash
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -- -o=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '2'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-- -o=/tmp/fzf-test-opt-eq/file2.txt') }
    end

    # Test -- --opt=/**<TAB> (long option with equals and slash prefix after --)
    if shell == :bash
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -- --output=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '1'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-- --output=/tmp/fzf-test-opt-eq/file1.txt') }
    end

    # Test -- -o=/**<TAB> (short option with equals and slash prefix after --)
    if shell == :bash
      tmux.send_keys 'C-u'
      tmux.send_keys "some-command -- -o=/tmp/fzf-test-opt-eq/file#{trigger}", :Tab
      tmux.until do |lines|
        assert_equal 2, lines.match_count
        assert_includes lines, '> file'
      end
      tmux.send_keys '2'
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until(true) { |lines| assert lines[-1]&.include?('-- -o=/tmp/fzf-test-opt-eq/file2.txt') }
    end
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-opt-eq')
  end

  def test_filename_with_newline
    skip('this test fails on bash/zsh, they replace the newline with a space') if shell != :fish
    FileUtils.mkdir_p('/tmp/fzf-test-newline')
    FileUtils.touch("/tmp/fzf-test-newline/xyz\nwith\nnewlines")
    tmux.prepare
    tmux.send_keys "cat /tmp/fzf-test-newline/xyz#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'cat /tmp/fzf-test-newline/xyz\\nwith\\nnewlines', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-newline')
  end

  def test_path_with_special_chars
    FileUtils.mkdir_p('/tmp/fzf-test-[special]')
    FileUtils.touch('/tmp/fzf-test-[special]/xyz123')
    tmux.prepare
    tmux.send_keys "ls /tmp/fzf-test-\\[special\\]/xyz#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls /tmp/fzf-test-\\[special\\]/xyz123', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-[special]')
  end

  def test_dollar_sign_in_path
    FileUtils.mkdir_p('/tmp/fzf-test-$dollar')
    FileUtils.touch('/tmp/fzf-test-$dollar/xyz123')
    tmux.prepare
    if shell == :fish
      tmux.send_keys "ls /tmp/fzf-test-\\$dollar/xyz#{trigger}", :Tab
    else
      tmux.send_keys "ls '/tmp/fzf-test-$dollar/'xyz#{trigger}", :Tab
    end
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls /tmp/fzf-test-\\$dollar/xyz123', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-$dollar')
  end

  def test_completion_after_double_dash
    FileUtils.mkdir_p('/tmp/fzf-test-ddash')
    FileUtils.touch('/tmp/fzf-test-ddash/--xyz123')
    tmux.prepare
    tmux.send_keys "ls -- /tmp/fzf-test-ddash/--xyz#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls -- /tmp/fzf-test-ddash/--xyz123', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-ddash')
  end

  def test_double_dash_with_equals
    FileUtils.mkdir_p('/tmp/fzf-test-ddash-eq')
    FileUtils.touch('/tmp/fzf-test-ddash-eq/--foo=bar')
    tmux.prepare
    tmux.send_keys "ls -- /tmp/fzf-test-ddash-eq/--foo#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls -- /tmp/fzf-test-ddash-eq/--foo=bar', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-ddash-eq')
  end

  def test_query_with_dollar_sign
    FileUtils.mkdir_p('/tmp/fzf-test-dollar-query')
    FileUtils.touch('/tmp/fzf-test-dollar-query/file.fish')
    tmux.prepare
    tmux.send_keys "ls /tmp/fzf-test-dollar-query/.fish$#{trigger}", :Tab
    tmux.until { |lines| assert_equal 1, lines.match_count }
    tmux.send_keys :Enter
    tmux.until(true) { |lines| assert_equal 'ls /tmp/fzf-test-dollar-query/file.fish', lines[-1] }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-dollar-query')
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
    tmux.send_keys "fake /tmp/foo#{trigger}", :Tab
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }
    tmux.send_keys 'C-c'

    tmux.prepare
    tmux.send_keys 'fake /tmp/foo'
    tmux.send_keys :Tab, 'C-u'

    tmux.prepare
    tmux.send_keys "fake /tmp/foo#{trigger}", :Tab
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
      tmux.send_keys "#{command} FZFFOOBR#{trigger}", :Tab
      tmux.until { |lines| assert_equal 1, lines.match_count }
      tmux.send_keys :Enter
      tmux.until { |lines| assert_equal "#{command} FZFFOOBAR", lines[-1] }
      tmux.send_keys 'C-c'
    end
  end
end

class TestFish < TestBase
  include TestShell
  include CompletionTest

  def shell
    :fish
  end

  def trigger
    '++'
  end

  def new_shell
    tmux.send_keys 'env FZF_TMUX=1 XDG_CONFIG_HOME=/tmp/fzf-fish fish', :Enter
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

  def test_single_flag_completion
    tmux.prepare
    tmux.send_keys "ls -#{trigger}", :Tab

    # Should launch fzf with flag options
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }

    # Should include common flags
    tmux.until { |lines| assert lines.any_include?('-a') }

    # Select one and verify insertion
    tmux.send_keys :Enter
    tmux.until { |lines| assert_match(/ls -\w+/, lines[-1]) }
  end

  def test_double_flag_completion
    tmux.prepare
    tmux.send_keys "ls --#{trigger}", :Tab

    # Should launch fzf with flag options
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }

    # Should include common flags
    tmux.until { |lines| assert lines.any_include?('--all') }

    # Select one and verify insertion
    tmux.send_keys :Enter
    tmux.until { |lines| assert_match(/ls --\w+/, lines[-1]) }
  end

  def test_command_completion
    tmux.prepare
    tmux.send_keys "ma#{trigger}", :Tab

    # Should launch fzf with matching commands
    tmux.until { |lines| assert_operator lines.match_count, :>, 0 }

    # Filter to specific command
    tmux.send_keys 'keconv'
    tmux.until do |lines|
      assert_equal 1, lines.match_count
      assert lines.any_include?('makeconv')
    end

    tmux.send_keys :Enter
    tmux.until { |lines| assert_equal 'makeconv', lines[-1] }
  end

  def test_argument_completion_after_command
    FileUtils.mkdir_p('/tmp/fzf-test-args')
    FileUtils.touch('/tmp/fzf-test-args/match.txt')

    tmux.prepare
    tmux.send_keys "ls /tmp/fzf-test-args/ma#{trigger}", :Tab

    # Should show file completion (NOT command completion)
    tmux.until do |lines|
      assert_operator lines.match_count, :>, 0
      assert lines.any_include?('match.txt')
    end

    tmux.send_keys :Enter
    tmux.until { |lines| assert_match(%r{ls /tmp/fzf-test-args/match\.txt}, lines[-1]) }
  ensure
    FileUtils.rm_rf('/tmp/fzf-test-args')
  end
end
