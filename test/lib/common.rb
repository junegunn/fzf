# frozen_string_literal: true

require 'bundler/setup'
require 'minitest/autorun'
require 'fileutils'
require 'English'
require 'shellwords'
require 'erb'
require 'tempfile'
require 'net/http'
require 'json'

TEMPLATE = File.read(File.expand_path('common.sh', __dir__))
FISH_TEMPLATE = File.read(File.expand_path('common.fish', __dir__))
UNSETS = %w[
  FZF_DEFAULT_COMMAND FZF_DEFAULT_OPTS
  FZF_TMUX FZF_TMUX_OPTS
  FZF_CTRL_T_COMMAND FZF_CTRL_T_OPTS
  FZF_ALT_C_COMMAND
  FZF_ALT_C_OPTS FZF_CTRL_R_OPTS
  FZF_API_KEY
].freeze
DEFAULT_TIMEOUT = 10

FILE = File.expand_path(__FILE__)
BASE = File.expand_path('../..', __dir__)
Dir.chdir(BASE)
FZF = %(FZF_DEFAULT_OPTS="--no-scrollbar --gutter ' ' --pointer '>' --marker '>'" FZF_DEFAULT_COMMAND= #{BASE}/bin/fzf).freeze

def wait(timeout = DEFAULT_TIMEOUT)
  since = Time.now
  begin
    yield or raise Minitest::Assertion, 'Assertion failure'
  rescue Minitest::Assertion
    raise if Time.now - since > timeout

    sleep(0.05)
    retry
  end
end

class Shell
  class << self
    def bash
      @bash ||=
        begin
          bashrc = '/tmp/fzf.bash'
          File.open(bashrc, 'w') do |f|
            f.puts ERB.new(TEMPLATE).result(binding)
          end

          "bash --rcfile #{bashrc}"
        end
    end

    def zsh
      @zsh ||=
        begin
          zdotdir = '/tmp/fzf-zsh'
          FileUtils.rm_rf(zdotdir)
          FileUtils.mkdir_p(zdotdir)
          File.open("#{zdotdir}/.zshrc", 'w') do |f|
            f.puts ERB.new(TEMPLATE).result(binding)
          end
          "ZDOTDIR=#{zdotdir} zsh"
        end
    end

    def fish
      @fish ||=
        begin
          confdir = '/tmp/fzf-fish'
          FileUtils.rm_rf(confdir)
          FileUtils.mkdir_p("#{confdir}/fish/conf.d")
          File.open("#{confdir}/fish/conf.d/fzf.fish", 'w') do |f|
            f.puts ERB.new(FISH_TEMPLATE).result(binding)
          end
          "rm -f ~/.local/share/fish/fzf_test_history; XDG_CONFIG_HOME=#{confdir} fish"
        end
    end
  end
end

class Tmux
  attr_reader :win

  def initialize(shell = :bash)
    @win = go(%W[new-window -d -P -F #I #{Shell.send(shell)}]).first
    go(%W[set-window-option -t #{@win} pane-base-index 0])
    return unless shell == :fish

    send_keys 'function fish_prompt; end; clear', :Enter
    self.until(&:empty?)
  end

  def kill
    go(%W[kill-window -t #{win}])
  end

  def focus
    go(%W[select-window -t #{win}])
  end

  def send_keys(*args)
    go(%W[send-keys -t #{win}] + args.map(&:to_s))
  end

  # Simulate a mouse click at the given 1-based column and row using the SGR mouse protocol
  # (xterm mouse mode 1006, which fzf enables). The escape sequence is injected as literal
  # keystrokes via tmux, and fzf parses it like a real terminal mouse event.
  #
  # tmux's own mouse handling intercepts these sequences when `set -g mouse on`, so we toggle
  # mouse off for the duration of the click and restore the previous state afterwards.
  def click(col, row, button: 0)
    prev = go(%w[show-options -gv mouse]).first
    go(%w[set-option -g mouse off])
    begin
      seq = "\e[<#{button};#{col};#{row}M\e[<#{button};#{col};#{row}m"
      go(%W[send-keys -t #{win} -l #{seq}])
    ensure
      go(%W[set-option -g mouse #{prev}]) if prev && !prev.empty?
    end
  end

  def paste(str)
    system('tmux', 'setb', str, ';', 'pasteb', '-t', win, ';', 'send-keys', '-t', win, 'Enter')
  end

  def capture
    go(%W[capture-pane -p -J -t #{win}]).map(&:rstrip).reverse.drop_while(&:empty?).reverse
  end

  # Raw pane capture with ANSI escape sequences preserved.
  def capture_ansi
    go(%W[capture-pane -p -J -e -t #{win}])
  end

  # 3-bit ANSI bg code (40..47) -> color name used in --color options.
  BG_NAMES = %w[black red green yellow blue magenta cyan white].freeze

  # Parse `tmux capture-pane -e` output into per-row bg ranges. Each row is an
  # array of [col_start, col_end, bg] tuples where bg is one of:
  #   'default'
  #   'red' / 'green' / 'blue' / ... (3-bit names)
  #   'bright-red' / ...             (bright variants)
  #   '256:<n>'                      (256-color fallback)
  # ANSI state persists across rows, matching real terminal behavior.
  def bg_ranges
    raw = go(%W[capture-pane -p -J -e -t #{win}])
    bg = 'default'
    raw.map do |row|
      cells = []
      i = 0
      len = row.length
      while i < len
        c = row[i]
        if c == "\e" && row[i + 1] == '['
          j = i + 2
          j += 1 while j < len && row[j] != 'm'
          parts = row[i + 2...j].split(';')
          k = 0
          while k < parts.length
            p = parts[k].to_i
            case p
            when 0, 49 then bg = 'default'
            when 40..47 then bg = BG_NAMES[p - 40]
            when 100..107 then bg = "bright-#{BG_NAMES[p - 100]}"
            when 48
              if parts[k + 1] == '5'
                bg = "256:#{parts[k + 2]}"
                k += 2
              elsif parts[k + 1] == '2'
                bg = "rgb:#{parts[k + 2]}:#{parts[k + 3]}:#{parts[k + 4]}"
                k += 4
              end
            end
            k += 1
          end
          i = j + 1
        else
          cells << bg
          i += 1
        end
      end
      ranges = []
      start = 0
      cells.each_with_index do |b, idx|
        if idx.positive? && b != cells[idx - 1]
          ranges << [start, idx - 1, cells[idx - 1]]
          start = idx
        end
      end
      ranges << [start, cells.length - 1, cells.last] unless cells.empty?
      ranges
    end
  end

  def until(refresh = false, timeout: DEFAULT_TIMEOUT)
    lines = nil
    begin
      wait(timeout) do
        lines = capture
        class << lines
          def counts
            lazy
              .map { |l| l.scan(%r{^. ([0-9]+)/([0-9]+)( \(([0-9]+)\))?}) }
              .reject(&:empty?)
              .first&.first&.map(&:to_i)&.values_at(0, 1, 3) || [0, 0, 0]
          end

          def match_count
            counts[0]
          end

          def item_count
            counts[1]
          end

          def select_count
            counts[2]
          end

          def any_include?(val)
            method = val.is_a?(Regexp) ? :match : :include?
            find { |line| line.send(method, val) }
          end
        end
        yield(lines).tap do |ok|
          send_keys 'C-l' if refresh && !ok
        end
      end
    rescue Minitest::Assertion
      puts $ERROR_INFO.backtrace
      puts '>' * 80
      puts lines
      puts '<' * 80
      raise
    end
    lines
  end

  def prepare
    tries = 0
    begin
      self.until(true) do |lines|
        message = "Prepare[#{tries}]"
        send_keys ' ', 'C-u', :Enter, message, :Left, :Right
        sleep(0.15)
        lines[-1] == message
      end
    rescue Minitest::Assertion
      (tries += 1) < 5 ? retry : raise
    end
    send_keys 'C-u', 'C-l'
  end

  private

  def go(args)
    IO.popen(%w[tmux] + args) { |io| io.readlines(chomp: true) }
  end
end

class TestBase < Minitest::Test
  TEMPNAME = Dir::Tmpname.create(%w[fzf]) {}
  FIFONAME = Dir::Tmpname.create(%w[fzf-fifo]) {}

  def writelines(lines)
    File.write(TEMPNAME, lines.join("\n"))
  end

  def tempname
    TEMPNAME
  end

  def fzf_output
    @thread.join.value.chomp.tap { @thread = nil }
  end

  def fzf_output_lines
    fzf_output.lines(chomp: true)
  end

  def setup
    File.mkfifo(FIFONAME)
  end

  def teardown
    FileUtils.rm_f([TEMPNAME, FIFONAME])
  end

  alias assert_equal_org assert_equal
  def assert_equal(expected, actual)
    # Ignore info separator
    actual = actual&.sub(/\s*─+$/, '') if actual.is_a?(String) && actual&.match?(%r{\d+/\d+})
    assert_equal_org(expected, actual)
  end

  # Run fzf with its output piped to a fifo
  def fzf(*opts)
    raise 'fzf_output not taken' if @thread

    @thread = Thread.new { File.read(FIFONAME) }
    fzf!(*opts) + " > #{FIFONAME.shellescape}"
  end

  def fzf!(*opts)
    opts = opts.filter_map do |o|
      case o
      when Symbol
        o = o.to_s
        o.length > 1 ? "--#{o.tr('_', '-')}" : "-#{o}"
      when String, Numeric
        o.to_s
      end
    end
    "#{FZF} #{opts.join(' ')}"
  end
end

class TestInteractive < TestBase
  attr_reader :tmux

  def setup
    super
    @tmux = Tmux.new
  end

  def teardown
    super
    @tmux.kill
  end
end
