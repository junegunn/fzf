package fzf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/curses"

	"github.com/junegunn/go-shellwords"
)

const usage = `usage: fzf [options]

  Search
    -x, --extended        Extended-search mode
    -e, --extended-exact  Extended-search mode (exact match)
    -i                    Case-insensitive match (default: smart-case match)
    +i                    Case-sensitive match
    -n, --nth=N[,..]      Comma-separated list of field index expressions
                          for limiting search scope. Each can be a non-zero
                          integer or a range expression ([BEGIN]..[END])
        --with-nth=N[,..] Transform item using index expressions within finder
    -d, --delimiter=STR   Field delimiter regex for --nth (default: AWK-style)
    +s, --no-sort         Do not sort the result
        --tac             Reverse the order of the input
        --tiebreak=CRI    Sort criterion when the scores are tied;
                          [length|begin|end|index] (default: length)

  Interface
    -m, --multi           Enable multi-select with tab/shift-tab
        --ansi            Enable processing of ANSI color codes
        --no-mouse        Disable mouse
        --color=COLSPEC   Base scheme (dark|light|16|bw) and/or custom colors
        --black           Use black background
        --reverse         Reverse orientation
        --cycle           Enable cyclic scroll
        --no-hscroll      Disable horizontal scroll
        --inline-info     Display finder info inline with the query
        --prompt=STR      Input prompt (default: '> ')
        --bind=KEYBINDS   Custom key bindings. Refer to the man page.
        --history=FILE    History file
        --history-size=N  Maximum number of history entries (default: 1000)

  Scripting
    -q, --query=STR       Start the finder with the given query
    -1, --select-1        Automatically select the only match
    -0, --exit-0          Exit immediately when there's no match
    -f, --filter=STR      Filter mode. Do not start interactive finder.
        --print-query     Print query as the first line
        --expect=KEYS     Comma-separated list of keys to complete fzf
        --sync            Synchronous search for multi-staged filtering

  Environment variables
    FZF_DEFAULT_COMMAND   Default command to use when input is tty
    FZF_DEFAULT_OPTS      Defaults options. (e.g. '-x -m')

`

// Mode denotes the current search mode
type Mode int

// Search modes
const (
	ModeFuzzy Mode = iota
	ModeExtended
	ModeExtendedExact
)

// Case denotes case-sensitivity of search
type Case int

// Case-sensitivities
const (
	CaseSmart Case = iota
	CaseIgnore
	CaseRespect
)

// Sort criteria
type tiebreak int

const (
	byLength tiebreak = iota
	byBegin
	byEnd
	byIndex
)

// Options stores the values of command-line options
type Options struct {
	Mode       Mode
	Case       Case
	Nth        []Range
	WithNth    []Range
	Delimiter  *regexp.Regexp
	Sort       int
	Tac        bool
	Tiebreak   tiebreak
	Multi      bool
	Ansi       bool
	Mouse      bool
	Theme      *curses.ColorTheme
	Black      bool
	Reverse    bool
	Cycle      bool
	Hscroll    bool
	InlineInfo bool
	Prompt     string
	Query      string
	Select1    bool
	Exit0      bool
	Filter     *string
	ToggleSort bool
	Expect     map[int]string
	Keymap     map[int]actionType
	Execmap    map[int]string
	PrintQuery bool
	ReadZero   bool
	Sync       bool
	History    *History
	Version    bool
}

func defaultTheme() *curses.ColorTheme {
	if strings.Contains(os.Getenv("TERM"), "256") {
		return curses.Dark256
	}
	return curses.Default16
}

func defaultOptions() *Options {
	return &Options{
		Mode:       ModeFuzzy,
		Case:       CaseSmart,
		Nth:        make([]Range, 0),
		WithNth:    make([]Range, 0),
		Delimiter:  nil,
		Sort:       1000,
		Tac:        false,
		Tiebreak:   byLength,
		Multi:      false,
		Ansi:       false,
		Mouse:      true,
		Theme:      defaultTheme(),
		Black:      false,
		Reverse:    false,
		Cycle:      false,
		Hscroll:    true,
		InlineInfo: false,
		Prompt:     "> ",
		Query:      "",
		Select1:    false,
		Exit0:      false,
		Filter:     nil,
		ToggleSort: false,
		Expect:     make(map[int]string),
		Keymap:     defaultKeymap(),
		Execmap:    make(map[int]string),
		PrintQuery: false,
		ReadZero:   false,
		Sync:       false,
		History:    nil,
		Version:    false}
}

func help(ok int) {
	os.Stderr.WriteString(usage)
	os.Exit(ok)
}

func errorExit(msg string) {
	os.Stderr.WriteString(msg + "\n")
	help(1)
}

func optString(arg string, prefixes ...string) (bool, string) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(arg, prefix) {
			return true, arg[len(prefix):]
		}
	}
	return false, ""
}

func nextString(args []string, i *int, message string) string {
	if len(args) > *i+1 {
		*i++
	} else {
		errorExit(message)
	}
	return args[*i]
}

func optionalNextString(args []string, i *int) string {
	if len(args) > *i+1 {
		*i++
		return args[*i]
	}
	return ""
}

func atoi(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		errorExit("not a valid integer: " + str)
	}
	return num
}

func nextInt(args []string, i *int, message string) int {
	if len(args) > *i+1 {
		*i++
	} else {
		errorExit(message)
	}
	return atoi(args[*i])
}

func optionalNumeric(args []string, i *int) int {
	if len(args) > *i+1 {
		if strings.IndexAny(args[*i+1], "0123456789") == 0 {
			*i++
		}
	}
	return 1 // Don't care
}

func splitNth(str string) []Range {
	if match, _ := regexp.MatchString("^[0-9,-.]+$", str); !match {
		errorExit("invalid format: " + str)
	}

	tokens := strings.Split(str, ",")
	ranges := make([]Range, len(tokens))
	for idx, s := range tokens {
		r, ok := ParseRange(&s)
		if !ok {
			errorExit("invalid format: " + str)
		}
		ranges[idx] = r
	}
	return ranges
}

func delimiterRegexp(str string) *regexp.Regexp {
	rx, e := regexp.Compile(str)
	if e != nil {
		str = regexp.QuoteMeta(str)
	}

	rx, e = regexp.Compile(fmt.Sprintf("(?:.*?%s)|(?:.+?$)", str))
	if e != nil {
		errorExit("invalid regular expression: " + e.Error())
	}
	return rx
}

func isAlphabet(char uint8) bool {
	return char >= 'a' && char <= 'z'
}

func parseKeyChords(str string, message string) map[int]string {
	if len(str) == 0 {
		errorExit(message)
	}

	tokens := strings.Split(str, ",")
	if str == "," || strings.HasPrefix(str, ",,") || strings.HasSuffix(str, ",,") || strings.Index(str, ",,,") >= 0 {
		tokens = append(tokens, ",")
	}

	chords := make(map[int]string)
	for _, key := range tokens {
		if len(key) == 0 {
			continue // ignore
		}
		lkey := strings.ToLower(key)
		chord := 0
		switch lkey {
		case "up":
			chord = curses.Up
		case "down":
			chord = curses.Down
		case "left":
			chord = curses.Left
		case "right":
			chord = curses.Right
		case "enter", "return":
			chord = curses.CtrlM
		case "space":
			chord = curses.AltZ + int(' ')
		case "bspace", "bs":
			chord = curses.BSpace
		case "alt-bs", "alt-bspace":
			chord = curses.AltBS
		case "tab":
			chord = curses.Tab
		case "btab", "shift-tab":
			chord = curses.BTab
		case "esc":
			chord = curses.ESC
		case "del":
			chord = curses.Del
		case "home":
			chord = curses.Home
		case "end":
			chord = curses.End
		case "pgup", "page-up":
			chord = curses.PgUp
		case "pgdn", "page-down":
			chord = curses.PgDn
		case "shift-left":
			chord = curses.SLeft
		case "shift-right":
			chord = curses.SRight
		default:
			if len(key) == 6 && strings.HasPrefix(lkey, "ctrl-") && isAlphabet(lkey[5]) {
				chord = curses.CtrlA + int(lkey[5]) - 'a'
			} else if len(key) == 5 && strings.HasPrefix(lkey, "alt-") && isAlphabet(lkey[4]) {
				chord = curses.AltA + int(lkey[4]) - 'a'
			} else if len(key) == 2 && strings.HasPrefix(lkey, "f") && key[1] >= '1' && key[1] <= '4' {
				chord = curses.F1 + int(key[1]) - '1'
			} else if utf8.RuneCountInString(key) == 1 {
				chord = curses.AltZ + int([]rune(key)[0])
			} else {
				errorExit("unsupported key: " + key)
			}
		}
		if chord > 0 {
			chords[chord] = key
		}
	}
	return chords
}

func parseTiebreak(str string) tiebreak {
	switch strings.ToLower(str) {
	case "length":
		return byLength
	case "index":
		return byIndex
	case "begin":
		return byBegin
	case "end":
		return byEnd
	default:
		errorExit("invalid sort criterion: " + str)
	}
	return byLength
}

func dupeTheme(theme *curses.ColorTheme) *curses.ColorTheme {
	dupe := *theme
	return &dupe
}

func parseTheme(defaultTheme *curses.ColorTheme, str string) *curses.ColorTheme {
	theme := dupeTheme(defaultTheme)
	for _, str := range strings.Split(strings.ToLower(str), ",") {
		switch str {
		case "dark":
			theme = dupeTheme(curses.Dark256)
		case "light":
			theme = dupeTheme(curses.Light256)
		case "16":
			theme = dupeTheme(curses.Default16)
		case "bw", "no":
			theme = nil
		default:
			fail := func() {
				errorExit("invalid color specification: " + str)
			}
			// Color is disabled
			if theme == nil {
				errorExit("colors disabled; cannot customize colors")
			}

			pair := strings.Split(str, ":")
			if len(pair) != 2 {
				fail()
			}
			ansi32, err := strconv.Atoi(pair[1])
			if err != nil || ansi32 < -1 || ansi32 > 255 {
				fail()
			}
			ansi := int16(ansi32)
			switch pair[0] {
			case "fg":
				theme.Fg = ansi
				theme.UseDefault = theme.UseDefault && ansi < 0
			case "bg":
				theme.Bg = ansi
				theme.UseDefault = theme.UseDefault && ansi < 0
			case "fg+":
				theme.Current = ansi
			case "bg+":
				theme.DarkBg = ansi
			case "hl":
				theme.Match = ansi
			case "hl+":
				theme.CurrentMatch = ansi
			case "prompt":
				theme.Prompt = ansi
			case "spinner":
				theme.Spinner = ansi
			case "info":
				theme.Info = ansi
			case "pointer":
				theme.Cursor = ansi
			case "marker":
				theme.Selected = ansi
			default:
				fail()
			}
		}
	}
	return theme
}

var executeRegexp *regexp.Regexp

func firstKey(keymap map[int]string) int {
	for k := range keymap {
		return k
	}
	return 0
}

const (
	escapedColon = 0
	escapedComma = 1
)

func parseKeymap(keymap map[int]actionType, execmap map[int]string, toggleSort bool, str string) (map[int]actionType, map[int]string, bool) {
	if executeRegexp == nil {
		// Backreferences are not supported.
		// "~!@#$%^&*;/|".each_char.map { |c| Regexp.escape(c) }.map { |c| "#{c}[^#{c}]*#{c}" }.join('|')
		executeRegexp = regexp.MustCompile(
			"(?s):execute:.*|:execute(\\([^)]*\\)|\\[[^\\]]*\\]|~[^~]*~|![^!]*!|@[^@]*@|\\#[^\\#]*\\#|\\$[^\\$]*\\$|%[^%]*%|\\^[^\\^]*\\^|&[^&]*&|\\*[^\\*]*\\*|;[^;]*;|/[^/]*/|\\|[^\\|]*\\|)")
	}
	masked := executeRegexp.ReplaceAllStringFunc(str, func(src string) string {
		return ":execute(" + strings.Repeat(" ", len(src)-10) + ")"
	})
	masked = strings.Replace(masked, "::", string([]rune{escapedColon, ':'}), -1)
	masked = strings.Replace(masked, ",:", string([]rune{escapedComma, ':'}), -1)

	idx := 0
	for _, pairStr := range strings.Split(masked, ",") {
		origPairStr := str[idx : idx+len(pairStr)]
		idx += len(pairStr) + 1

		pair := strings.SplitN(pairStr, ":", 2)
		if len(pair) < 2 {
			errorExit("bind action not specified: " + origPairStr)
		}
		var key int
		if len(pair[0]) == 1 && pair[0][0] == escapedColon {
			key = ':' + curses.AltZ
		} else if len(pair[0]) == 1 && pair[0][0] == escapedComma {
			key = ',' + curses.AltZ
		} else {
			keys := parseKeyChords(pair[0], "key name required")
			key = firstKey(keys)
		}

		act := origPairStr[len(pair[0])+1 : len(origPairStr)]
		actLower := strings.ToLower(act)
		switch actLower {
		case "ignore":
			keymap[key] = actIgnore
		case "beginning-of-line":
			keymap[key] = actBeginningOfLine
		case "abort":
			keymap[key] = actAbort
		case "accept":
			keymap[key] = actAccept
		case "backward-char":
			keymap[key] = actBackwardChar
		case "backward-delete-char":
			keymap[key] = actBackwardDeleteChar
		case "backward-word":
			keymap[key] = actBackwardWord
		case "clear-screen":
			keymap[key] = actClearScreen
		case "delete-char":
			keymap[key] = actDeleteChar
		case "end-of-line":
			keymap[key] = actEndOfLine
		case "forward-char":
			keymap[key] = actForwardChar
		case "forward-word":
			keymap[key] = actForwardWord
		case "kill-line":
			keymap[key] = actKillLine
		case "kill-word":
			keymap[key] = actKillWord
		case "unix-line-discard", "line-discard":
			keymap[key] = actUnixLineDiscard
		case "unix-word-rubout", "word-rubout":
			keymap[key] = actUnixWordRubout
		case "yank":
			keymap[key] = actYank
		case "backward-kill-word":
			keymap[key] = actBackwardKillWord
		case "toggle-down":
			keymap[key] = actToggleDown
		case "toggle-up":
			keymap[key] = actToggleUp
		case "toggle-all":
			keymap[key] = actToggleAll
		case "select-all":
			keymap[key] = actSelectAll
		case "deselect-all":
			keymap[key] = actDeselectAll
		case "toggle":
			keymap[key] = actToggle
		case "down":
			keymap[key] = actDown
		case "up":
			keymap[key] = actUp
		case "page-up":
			keymap[key] = actPageUp
		case "page-down":
			keymap[key] = actPageDown
		case "previous-history":
			keymap[key] = actPreviousHistory
		case "next-history":
			keymap[key] = actNextHistory
		case "toggle-sort":
			keymap[key] = actToggleSort
			toggleSort = true
		default:
			if isExecuteAction(actLower) {
				keymap[key] = actExecute
				if act[7] == ':' {
					execmap[key] = act[8:]
				} else {
					execmap[key] = act[8 : len(act)-1]
				}
			} else {
				errorExit("unknown action: " + act)
			}
		}
	}
	return keymap, execmap, toggleSort
}

func isExecuteAction(str string) bool {
	if !strings.HasPrefix(str, "execute") || len(str) < 9 {
		return false
	}
	b := str[7]
	e := str[len(str)-1]
	if b == ':' || b == '(' && e == ')' || b == '[' && e == ']' ||
		b == e && strings.ContainsAny(string(b), "~!@#$%^&*;/|") {
		return true
	}
	return false
}

func checkToggleSort(keymap map[int]actionType, str string) map[int]actionType {
	keys := parseKeyChords(str, "key name required")
	if len(keys) != 1 {
		errorExit("multiple keys specified")
	}
	keymap[firstKey(keys)] = actToggleSort
	return keymap
}

func parseOptions(opts *Options, allArgs []string) {
	keymap := make(map[int]actionType)
	var historyMax int
	if opts.History == nil {
		historyMax = defaultHistoryMax
	} else {
		historyMax = opts.History.maxSize
	}
	setHistory := func(path string) {
		h, e := NewHistory(path, historyMax)
		if e != nil {
			errorExit(e.Error())
		}
		opts.History = h
	}
	setHistoryMax := func(max int) {
		historyMax = max
		if historyMax < 1 {
			errorExit("history max must be a positive integer")
		}
		if opts.History != nil {
			opts.History.maxSize = historyMax
		}
	}
	for i := 0; i < len(allArgs); i++ {
		arg := allArgs[i]
		switch arg {
		case "-h", "--help":
			help(0)
		case "-x", "--extended":
			opts.Mode = ModeExtended
		case "-e", "--extended-exact":
			opts.Mode = ModeExtendedExact
		case "+x", "--no-extended", "+e", "--no-extended-exact":
			opts.Mode = ModeFuzzy
		case "-q", "--query":
			opts.Query = nextString(allArgs, &i, "query string required")
		case "-f", "--filter":
			filter := nextString(allArgs, &i, "query string required")
			opts.Filter = &filter
		case "--expect":
			opts.Expect = parseKeyChords(nextString(allArgs, &i, "key names required"), "key names required")
		case "--tiebreak":
			opts.Tiebreak = parseTiebreak(nextString(allArgs, &i, "sort criterion required"))
		case "--bind":
			keymap, opts.Execmap, opts.ToggleSort =
				parseKeymap(keymap, opts.Execmap, opts.ToggleSort, nextString(allArgs, &i, "bind expression required"))
		case "--color":
			spec := optionalNextString(allArgs, &i)
			if len(spec) == 0 {
				opts.Theme = defaultTheme()
			} else {
				opts.Theme = parseTheme(opts.Theme, spec)
			}
		case "--toggle-sort":
			keymap = checkToggleSort(keymap, nextString(allArgs, &i, "key name required"))
			opts.ToggleSort = true
		case "-d", "--delimiter":
			opts.Delimiter = delimiterRegexp(nextString(allArgs, &i, "delimiter required"))
		case "-n", "--nth":
			opts.Nth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "--with-nth":
			opts.WithNth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "-s", "--sort":
			opts.Sort = optionalNumeric(allArgs, &i)
		case "+s", "--no-sort":
			opts.Sort = 0
		case "--tac":
			opts.Tac = true
		case "--no-tac":
			opts.Tac = false
		case "-i":
			opts.Case = CaseIgnore
		case "+i":
			opts.Case = CaseRespect
		case "-m", "--multi":
			opts.Multi = true
		case "+m", "--no-multi":
			opts.Multi = false
		case "--ansi":
			opts.Ansi = true
		case "--no-ansi":
			opts.Ansi = false
		case "--no-mouse":
			opts.Mouse = false
		case "+c", "--no-color":
			opts.Theme = nil
		case "+2", "--no-256":
			opts.Theme = curses.Default16
		case "--black":
			opts.Black = true
		case "--no-black":
			opts.Black = false
		case "--reverse":
			opts.Reverse = true
		case "--no-reverse":
			opts.Reverse = false
		case "--cycle":
			opts.Cycle = true
		case "--no-cycle":
			opts.Cycle = false
		case "--hscroll":
			opts.Hscroll = true
		case "--no-hscroll":
			opts.Hscroll = false
		case "--inline-info":
			opts.InlineInfo = true
		case "--no-inline-info":
			opts.InlineInfo = false
		case "-1", "--select-1":
			opts.Select1 = true
		case "+1", "--no-select-1":
			opts.Select1 = false
		case "-0", "--exit-0":
			opts.Exit0 = true
		case "+0", "--no-exit-0":
			opts.Exit0 = false
		case "--read0":
			opts.ReadZero = true
		case "--no-read0":
			opts.ReadZero = false
		case "--print-query":
			opts.PrintQuery = true
		case "--no-print-query":
			opts.PrintQuery = false
		case "--prompt":
			opts.Prompt = nextString(allArgs, &i, "prompt string required")
		case "--sync":
			opts.Sync = true
		case "--no-sync":
			opts.Sync = false
		case "--async":
			opts.Sync = false
		case "--no-history":
			opts.History = nil
		case "--history":
			setHistory(nextString(allArgs, &i, "history file path required"))
		case "--history-size":
			setHistoryMax(nextInt(allArgs, &i, "history max size required"))
		case "--version":
			opts.Version = true
		default:
			if match, value := optString(arg, "-q", "--query="); match {
				opts.Query = value
			} else if match, value := optString(arg, "-f", "--filter="); match {
				opts.Filter = &value
			} else if match, value := optString(arg, "-d", "--delimiter="); match {
				opts.Delimiter = delimiterRegexp(value)
			} else if match, value := optString(arg, "--prompt="); match {
				opts.Prompt = value
			} else if match, value := optString(arg, "-n", "--nth="); match {
				opts.Nth = splitNth(value)
			} else if match, value := optString(arg, "--with-nth="); match {
				opts.WithNth = splitNth(value)
			} else if match, _ := optString(arg, "-s", "--sort="); match {
				opts.Sort = 1 // Don't care
			} else if match, value := optString(arg, "--toggle-sort="); match {
				keymap = checkToggleSort(keymap, value)
				opts.ToggleSort = true
			} else if match, value := optString(arg, "--expect="); match {
				opts.Expect = parseKeyChords(value, "key names required")
			} else if match, value := optString(arg, "--tiebreak="); match {
				opts.Tiebreak = parseTiebreak(value)
			} else if match, value := optString(arg, "--color="); match {
				opts.Theme = parseTheme(opts.Theme, value)
			} else if match, value := optString(arg, "--bind="); match {
				keymap, opts.Execmap, opts.ToggleSort =
					parseKeymap(keymap, opts.Execmap, opts.ToggleSort, value)
			} else if match, value := optString(arg, "--history="); match {
				setHistory(value)
			} else if match, value := optString(arg, "--history-size="); match {
				setHistoryMax(atoi(value))
			} else {
				errorExit("unknown option: " + arg)
			}
		}
	}

	// Change default actions for CTRL-N / CTRL-P when --history is used
	if opts.History != nil {
		if _, prs := keymap[curses.CtrlP]; !prs {
			keymap[curses.CtrlP] = actPreviousHistory
		}
		if _, prs := keymap[curses.CtrlN]; !prs {
			keymap[curses.CtrlN] = actNextHistory
		}
	}

	// Override default key bindings
	for key, act := range keymap {
		opts.Keymap[key] = act
	}

	// If we're not using extended search mode, --nth option becomes irrelevant
	// if it contains the whole range
	if opts.Mode == ModeFuzzy || len(opts.Nth) == 1 {
		for _, r := range opts.Nth {
			if r.begin == rangeEllipsis && r.end == rangeEllipsis {
				opts.Nth = make([]Range, 0)
				return
			}
		}
	}
}

// ParseOptions parses command-line options
func ParseOptions() *Options {
	opts := defaultOptions()

	// Options from Env var
	words, _ := shellwords.Parse(os.Getenv("FZF_DEFAULT_OPTS"))
	parseOptions(opts, words)

	// Options from command-line arguments
	parseOptions(opts, os.Args[1:])
	return opts
}
