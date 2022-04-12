package fzf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/tui"

	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-shellwords"
)

const usage = `usage: fzf [options]

  Search
    -x, --extended        Extended-search mode
                          (enabled by default; +x or --no-extended to disable)
    -e, --exact           Enable Exact-match
    --algo=TYPE           Fuzzy matching algorithm: [v1|v2] (default: v2)
    -i                    Case-insensitive match (default: smart-case match)
    +i                    Case-sensitive match
    --literal             Do not normalize latin script letters before matching
    -n, --nth=N[,..]      Comma-separated list of field index expressions
                          for limiting search scope. Each can be a non-zero
                          integer or a range expression ([BEGIN]..[END]).
    --with-nth=N[,..]     Transform the presentation of each line using
                          field index expressions
    -d, --delimiter=STR   Field delimiter regex (default: AWK-style)
    +s, --no-sort         Do not sort the result
    --tac                 Reverse the order of the input
    --disabled            Do not perform search
    --tiebreak=CRI[,..]   Comma-separated list of sort criteria to apply
                          when the scores are tied [length|begin|end|index]
                          (default: length)

  Interface
    -m, --multi[=MAX]     Enable multi-select with tab/shift-tab
    --no-mouse            Disable mouse
    --bind=KEYBINDS       Custom key bindings. Refer to the man page.
    --cycle               Enable cyclic scroll
    --keep-right          Keep the right end of the line visible on overflow
    --scroll-off=LINES    Number of screen lines to keep above or below when
                          scrolling to the top or to the bottom (default: 0)
    --no-hscroll          Disable horizontal scroll
    --hscroll-off=COLS    Number of screen columns to keep to the right of the
                          highlighted substring (default: 10)
    --filepath-word       Make word-wise movements respect path separators
    --jump-labels=CHARS   Label characters for jump and jump-accept

  Layout
    --height=HEIGHT[%]    Display fzf window below the cursor with the given
                          height instead of using fullscreen
    --min-height=HEIGHT   Minimum height when --height is given in percent
                          (default: 10)
    --layout=LAYOUT       Choose layout: [default|reverse|reverse-list]
    --border[=STYLE]      Draw border around the finder
                          [rounded|sharp|horizontal|vertical|
                           top|bottom|left|right|none] (default: rounded)
    --margin=MARGIN       Screen margin (TRBL | TB,RL | T,RL,B | T,R,B,L)
    --padding=PADDING     Padding inside border (TRBL | TB,RL | T,RL,B | T,R,B,L)
    --info=STYLE          Finder info style [default|inline|hidden]
    --prompt=STR          Input prompt (default: '> ')
    --pointer=STR         Pointer to the current line (default: '>')
    --marker=STR          Multi-select marker (default: '>')
    --header=STR          String to print as header
    --header-lines=N      The first N lines of the input are treated as header
    --header-first        Print header before the prompt line
    --ellipsis=STR        Ellipsis to show when line is truncated (default: '..')

  Display
    --ansi                Enable processing of ANSI color codes
    --tabstop=SPACES      Number of spaces for a tab character (default: 8)
    --color=COLSPEC       Base scheme (dark|light|16|bw) and/or custom colors
    --no-bold             Do not use bold text

  History
    --history=FILE        History file
    --history-size=N      Maximum number of history entries (default: 1000)

  Preview
    --preview=COMMAND     Command to preview highlighted line ({})
    --preview-window=OPT  Preview window layout (default: right:50%)
                          [up|down|left|right][,SIZE[%]]
                          [,[no]wrap][,[no]cycle][,[no]follow][,[no]hidden]
                          [,border-BORDER_OPT]
                          [,+SCROLL[OFFSETS][/DENOM]][,~HEADER_LINES]
                          [,default]

  Scripting
    -q, --query=STR       Start the finder with the given query
    -1, --select-1        Automatically select the only match
    -0, --exit-0          Exit immediately when there's no match
    -f, --filter=STR      Filter mode. Do not start interactive finder.
    --print-query         Print query as the first line
    --expect=KEYS         Comma-separated list of keys to complete fzf
    --read0               Read input delimited by ASCII NUL characters
    --print0              Print output delimited by ASCII NUL characters
    --sync                Synchronous search for multi-staged filtering
    --version             Display version information and exit

  Environment variables
    FZF_DEFAULT_COMMAND   Default command to use when input is tty
    FZF_DEFAULT_OPTS      Default options
                          (e.g. '--layout=reverse --inline-info')

`

// Case denotes case-sensitivity of search
type Case int

// Case-sensitivities
const (
	CaseSmart Case = iota
	CaseIgnore
	CaseRespect
)

// Sort criteria
type criterion int

const (
	byScore criterion = iota
	byLength
	byBegin
	byEnd
)

type sizeSpec struct {
	size    float64
	percent bool
}

func defaultMargin() [4]sizeSpec {
	return [4]sizeSpec{}
}

type windowPosition int

const (
	posUp windowPosition = iota
	posDown
	posLeft
	posRight
)

type layoutType int

const (
	layoutDefault layoutType = iota
	layoutReverse
	layoutReverseList
)

type infoStyle int

const (
	infoDefault infoStyle = iota
	infoInline
	infoHidden
)

type previewOpts struct {
	command     string
	position    windowPosition
	size        sizeSpec
	scroll      string
	hidden      bool
	wrap        bool
	cycle       bool
	follow      bool
	border      tui.BorderShape
	headerLines int
}

func (a previewOpts) sameLayout(b previewOpts) bool {
	return a.size == b.size && a.position == b.position && a.border == b.border && a.hidden == b.hidden
}

func (a previewOpts) sameContentLayout(b previewOpts) bool {
	return a.wrap == b.wrap && a.headerLines == b.headerLines
}

// Options stores the values of command-line options
type Options struct {
	Fuzzy       bool
	FuzzyAlgo   algo.Algo
	Extended    bool
	Phony       bool
	Case        Case
	Normalize   bool
	Nth         []Range
	WithNth     []Range
	Delimiter   Delimiter
	Sort        int
	Tac         bool
	Criteria    []criterion
	Multi       int
	Ansi        bool
	Mouse       bool
	Theme       *tui.ColorTheme
	Black       bool
	Bold        bool
	Height      sizeSpec
	MinHeight   int
	Layout      layoutType
	Cycle       bool
	KeepRight   bool
	Hscroll     bool
	HscrollOff  int
	ScrollOff   int
	FileWord    bool
	InfoStyle   infoStyle
	JumpLabels  string
	Prompt      string
	Pointer     string
	Marker      string
	Query       string
	Select1     bool
	Exit0       bool
	Filter      *string
	ToggleSort  bool
	Expect      map[tui.Event]string
	Keymap      map[tui.Event][]*action
	Preview     previewOpts
	PrintQuery  bool
	ReadZero    bool
	Printer     func(string)
	PrintSep    string
	Sync        bool
	History     *History
	Header      []string
	HeaderLines int
	HeaderFirst bool
	Ellipsis    string
	Margin      [4]sizeSpec
	Padding     [4]sizeSpec
	BorderShape tui.BorderShape
	Unicode     bool
	Tabstop     int
	ClearOnExit bool
	Version     bool
}

func defaultPreviewOpts(command string) previewOpts {
	return previewOpts{command, posRight, sizeSpec{50, true}, "", false, false, false, false, tui.BorderRounded, 0}
}

func defaultOptions() *Options {
	return &Options{
		Fuzzy:       true,
		FuzzyAlgo:   algo.FuzzyMatchV2,
		Extended:    true,
		Phony:       false,
		Case:        CaseSmart,
		Normalize:   true,
		Nth:         make([]Range, 0),
		WithNth:     make([]Range, 0),
		Delimiter:   Delimiter{},
		Sort:        1000,
		Tac:         false,
		Criteria:    []criterion{byScore, byLength},
		Multi:       0,
		Ansi:        false,
		Mouse:       true,
		Theme:       tui.EmptyTheme(),
		Black:       false,
		Bold:        true,
		MinHeight:   10,
		Layout:      layoutDefault,
		Cycle:       false,
		KeepRight:   false,
		Hscroll:     true,
		HscrollOff:  10,
		ScrollOff:   0,
		FileWord:    false,
		InfoStyle:   infoDefault,
		JumpLabels:  defaultJumpLabels,
		Prompt:      "> ",
		Pointer:     ">",
		Marker:      ">",
		Query:       "",
		Select1:     false,
		Exit0:       false,
		Filter:      nil,
		ToggleSort:  false,
		Expect:      make(map[tui.Event]string),
		Keymap:      make(map[tui.Event][]*action),
		Preview:     defaultPreviewOpts(""),
		PrintQuery:  false,
		ReadZero:    false,
		Printer:     func(str string) { fmt.Println(str) },
		PrintSep:    "\n",
		Sync:        false,
		History:     nil,
		Header:      make([]string, 0),
		HeaderLines: 0,
		HeaderFirst: false,
		Ellipsis:    "..",
		Margin:      defaultMargin(),
		Padding:     defaultMargin(),
		Unicode:     true,
		Tabstop:     8,
		ClearOnExit: true,
		Version:     false}
}

func help(code int) {
	os.Stdout.WriteString(usage)
	os.Exit(code)
}

func errorExit(msg string) {
	os.Stderr.WriteString(msg + "\n")
	os.Exit(exitError)
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

func optionalNextString(args []string, i *int) (bool, string) {
	if len(args) > *i+1 && !strings.HasPrefix(args[*i+1], "-") && !strings.HasPrefix(args[*i+1], "+") {
		*i++
		return true, args[*i]
	}
	return false, ""
}

func atoi(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		errorExit("not a valid integer: " + str)
	}
	return num
}

func atof(str string) float64 {
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		errorExit("not a valid number: " + str)
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

func optionalNumeric(args []string, i *int, defaultValue int) int {
	if len(args) > *i+1 {
		if strings.IndexAny(args[*i+1], "0123456789") == 0 {
			*i++
			return atoi(args[*i])
		}
	}
	return defaultValue
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

func delimiterRegexp(str string) Delimiter {
	// Special handling of \t
	str = strings.Replace(str, "\\t", "\t", -1)

	// 1. Pattern does not contain any special character
	if regexp.QuoteMeta(str) == str {
		return Delimiter{str: &str}
	}

	rx, e := regexp.Compile(str)
	// 2. Pattern is not a valid regular expression
	if e != nil {
		return Delimiter{str: &str}
	}

	// 3. Pattern as regular expression. Slow.
	return Delimiter{regex: rx}
}

func isAlphabet(char uint8) bool {
	return char >= 'a' && char <= 'z'
}

func isNumeric(char uint8) bool {
	return char >= '0' && char <= '9'
}

func parseAlgo(str string) algo.Algo {
	switch str {
	case "v1":
		return algo.FuzzyMatchV1
	case "v2":
		return algo.FuzzyMatchV2
	default:
		errorExit("invalid algorithm (expected: v1 or v2)")
	}
	return algo.FuzzyMatchV2
}

func parseBorder(str string, optional bool) tui.BorderShape {
	switch str {
	case "rounded":
		return tui.BorderRounded
	case "sharp":
		return tui.BorderSharp
	case "horizontal":
		return tui.BorderHorizontal
	case "vertical":
		return tui.BorderVertical
	case "top":
		return tui.BorderTop
	case "bottom":
		return tui.BorderBottom
	case "left":
		return tui.BorderLeft
	case "right":
		return tui.BorderRight
	case "none":
		return tui.BorderNone
	default:
		if optional && str == "" {
			return tui.BorderRounded
		}
		errorExit("invalid border style (expected: rounded|sharp|horizontal|vertical|top|bottom|left|right|none)")
	}
	return tui.BorderNone
}

func parseKeyChords(str string, message string) map[tui.Event]string {
	if len(str) == 0 {
		errorExit(message)
	}

	str = regexp.MustCompile("(?i)(alt-),").ReplaceAllString(str, "$1"+string([]rune{escapedComma}))
	tokens := strings.Split(str, ",")
	if str == "," || strings.HasPrefix(str, ",,") || strings.HasSuffix(str, ",,") || strings.Contains(str, ",,,") {
		tokens = append(tokens, ",")
	}

	chords := make(map[tui.Event]string)
	for _, key := range tokens {
		if len(key) == 0 {
			continue // ignore
		}
		key = strings.ReplaceAll(key, string([]rune{escapedComma}), ",")
		lkey := strings.ToLower(key)
		add := func(e tui.EventType) {
			chords[e.AsEvent()] = key
		}
		switch lkey {
		case "up":
			add(tui.Up)
		case "down":
			add(tui.Down)
		case "left":
			add(tui.Left)
		case "right":
			add(tui.Right)
		case "enter", "return":
			add(tui.CtrlM)
		case "space":
			chords[tui.Key(' ')] = key
		case "bspace", "bs":
			add(tui.BSpace)
		case "ctrl-space":
			add(tui.CtrlSpace)
		case "ctrl-^", "ctrl-6":
			add(tui.CtrlCaret)
		case "ctrl-/", "ctrl-_":
			add(tui.CtrlSlash)
		case "ctrl-\\":
			add(tui.CtrlBackSlash)
		case "ctrl-]":
			add(tui.CtrlRightBracket)
		case "change":
			add(tui.Change)
		case "backward-eof":
			add(tui.BackwardEOF)
		case "alt-enter", "alt-return":
			chords[tui.CtrlAltKey('m')] = key
		case "alt-space":
			chords[tui.AltKey(' ')] = key
		case "alt-bs", "alt-bspace":
			add(tui.AltBS)
		case "alt-up":
			add(tui.AltUp)
		case "alt-down":
			add(tui.AltDown)
		case "alt-left":
			add(tui.AltLeft)
		case "alt-right":
			add(tui.AltRight)
		case "tab":
			add(tui.Tab)
		case "btab", "shift-tab":
			add(tui.BTab)
		case "esc":
			add(tui.ESC)
		case "del":
			add(tui.Del)
		case "home":
			add(tui.Home)
		case "end":
			add(tui.End)
		case "insert":
			add(tui.Insert)
		case "pgup", "page-up":
			add(tui.PgUp)
		case "pgdn", "page-down":
			add(tui.PgDn)
		case "alt-shift-up", "shift-alt-up":
			add(tui.AltSUp)
		case "alt-shift-down", "shift-alt-down":
			add(tui.AltSDown)
		case "alt-shift-left", "shift-alt-left":
			add(tui.AltSLeft)
		case "alt-shift-right", "shift-alt-right":
			add(tui.AltSRight)
		case "shift-up":
			add(tui.SUp)
		case "shift-down":
			add(tui.SDown)
		case "shift-left":
			add(tui.SLeft)
		case "shift-right":
			add(tui.SRight)
		case "left-click":
			add(tui.LeftClick)
		case "right-click":
			add(tui.RightClick)
		case "double-click":
			add(tui.DoubleClick)
		case "f10":
			add(tui.F10)
		case "f11":
			add(tui.F11)
		case "f12":
			add(tui.F12)
		default:
			runes := []rune(key)
			if len(key) == 10 && strings.HasPrefix(lkey, "ctrl-alt-") && isAlphabet(lkey[9]) {
				chords[tui.CtrlAltKey(rune(key[9]))] = key
			} else if len(key) == 6 && strings.HasPrefix(lkey, "ctrl-") && isAlphabet(lkey[5]) {
				add(tui.EventType(tui.CtrlA.Int() + int(lkey[5]) - 'a'))
			} else if len(runes) == 5 && strings.HasPrefix(lkey, "alt-") {
				r := runes[4]
				switch r {
				case escapedColon:
					r = ':'
				case escapedComma:
					r = ','
				case escapedPlus:
					r = '+'
				}
				chords[tui.AltKey(r)] = key
			} else if len(key) == 2 && strings.HasPrefix(lkey, "f") && key[1] >= '1' && key[1] <= '9' {
				add(tui.EventType(tui.F1.Int() + int(key[1]) - '1'))
			} else if len(runes) == 1 {
				chords[tui.Key(runes[0])] = key
			} else {
				errorExit("unsupported key: " + key)
			}
		}
	}
	return chords
}

func parseTiebreak(str string) []criterion {
	criteria := []criterion{byScore}
	hasIndex := false
	hasLength := false
	hasBegin := false
	hasEnd := false
	check := func(notExpected *bool, name string) {
		if *notExpected {
			errorExit("duplicate sort criteria: " + name)
		}
		if hasIndex {
			errorExit("index should be the last criterion")
		}
		*notExpected = true
	}
	for _, str := range strings.Split(strings.ToLower(str), ",") {
		switch str {
		case "index":
			check(&hasIndex, "index")
		case "length":
			check(&hasLength, "length")
			criteria = append(criteria, byLength)
		case "begin":
			check(&hasBegin, "begin")
			criteria = append(criteria, byBegin)
		case "end":
			check(&hasEnd, "end")
			criteria = append(criteria, byEnd)
		default:
			errorExit("invalid sort criterion: " + str)
		}
	}
	return criteria
}

func dupeTheme(theme *tui.ColorTheme) *tui.ColorTheme {
	dupe := *theme
	return &dupe
}

func parseTheme(defaultTheme *tui.ColorTheme, str string) *tui.ColorTheme {
	theme := dupeTheme(defaultTheme)
	rrggbb := regexp.MustCompile("^#[0-9a-fA-F]{6}$")
	for _, str := range strings.Split(strings.ToLower(str), ",") {
		switch str {
		case "dark":
			theme = dupeTheme(tui.Dark256)
		case "light":
			theme = dupeTheme(tui.Light256)
		case "16":
			theme = dupeTheme(tui.Default16)
		case "bw", "no":
			theme = tui.NoColorTheme()
		default:
			fail := func() {
				errorExit("invalid color specification: " + str)
			}
			// Color is disabled
			if theme == nil {
				continue
			}

			components := strings.Split(str, ":")
			if len(components) < 2 {
				fail()
			}

			mergeAttr := func(cattr *tui.ColorAttr) {
				for _, component := range components[1:] {
					switch component {
					case "regular":
						cattr.Attr = tui.AttrRegular
					case "bold", "strong":
						cattr.Attr |= tui.Bold
					case "dim":
						cattr.Attr |= tui.Dim
					case "italic":
						cattr.Attr |= tui.Italic
					case "underline":
						cattr.Attr |= tui.Underline
					case "blink":
						cattr.Attr |= tui.Blink
					case "reverse":
						cattr.Attr |= tui.Reverse
					case "black":
						cattr.Color = tui.Color(0)
					case "red":
						cattr.Color = tui.Color(1)
					case "green":
						cattr.Color = tui.Color(2)
					case "yellow":
						cattr.Color = tui.Color(3)
					case "blue":
						cattr.Color = tui.Color(4)
					case "magenta":
						cattr.Color = tui.Color(5)
					case "cyan":
						cattr.Color = tui.Color(6)
					case "white":
						cattr.Color = tui.Color(7)
					case "bright-black", "gray", "grey":
						cattr.Color = tui.Color(8)
					case "bright-red":
						cattr.Color = tui.Color(9)
					case "bright-green":
						cattr.Color = tui.Color(10)
					case "bright-yellow":
						cattr.Color = tui.Color(11)
					case "bright-blue":
						cattr.Color = tui.Color(12)
					case "bright-magenta":
						cattr.Color = tui.Color(13)
					case "bright-cyan":
						cattr.Color = tui.Color(14)
					case "bright-white":
						cattr.Color = tui.Color(15)
					case "":
					default:
						if rrggbb.MatchString(component) {
							cattr.Color = tui.HexToColor(component)
						} else {
							ansi32, err := strconv.Atoi(component)
							if err != nil || ansi32 < -1 || ansi32 > 255 {
								fail()
							}
							cattr.Color = tui.Color(ansi32)
						}
					}
				}
			}
			switch components[0] {
			case "query", "input":
				mergeAttr(&theme.Input)
			case "disabled":
				mergeAttr(&theme.Disabled)
			case "fg":
				mergeAttr(&theme.Fg)
			case "bg":
				mergeAttr(&theme.Bg)
			case "preview-fg":
				mergeAttr(&theme.PreviewFg)
			case "preview-bg":
				mergeAttr(&theme.PreviewBg)
			case "fg+":
				mergeAttr(&theme.Current)
			case "bg+":
				mergeAttr(&theme.DarkBg)
			case "gutter":
				mergeAttr(&theme.Gutter)
			case "hl":
				mergeAttr(&theme.Match)
			case "hl+":
				mergeAttr(&theme.CurrentMatch)
			case "border":
				mergeAttr(&theme.Border)
			case "prompt":
				mergeAttr(&theme.Prompt)
			case "spinner":
				mergeAttr(&theme.Spinner)
			case "info":
				mergeAttr(&theme.Info)
			case "pointer":
				mergeAttr(&theme.Cursor)
			case "marker":
				mergeAttr(&theme.Selected)
			case "header":
				mergeAttr(&theme.Header)
			default:
				fail()
			}
		}
	}
	return theme
}

var executeRegexp *regexp.Regexp

func firstKey(keymap map[tui.Event]string) tui.Event {
	for k := range keymap {
		return k
	}
	return tui.EventType(0).AsEvent()
}

const (
	escapedColon = 0
	escapedComma = 1
	escapedPlus  = 2
)

func init() {
	// Backreferences are not supported.
	// "~!@#$%^&*;/|".each_char.map { |c| Regexp.escape(c) }.map { |c| "#{c}[^#{c}]*#{c}" }.join('|')
	executeRegexp = regexp.MustCompile(
		`(?si)[:+](execute(?:-multi|-silent)?|reload|preview|change-prompt|change-preview-window|change-preview|(?:re|un)bind):.+|[:+](execute(?:-multi|-silent)?|reload|preview|change-prompt|change-preview-window|change-preview|(?:re|un)bind)(\([^)]*\)|\[[^\]]*\]|~[^~]*~|![^!]*!|@[^@]*@|\#[^\#]*\#|\$[^\$]*\$|%[^%]*%|\^[^\^]*\^|&[^&]*&|\*[^\*]*\*|;[^;]*;|/[^/]*/|\|[^\|]*\|)`)
}

func parseKeymap(keymap map[tui.Event][]*action, str string) {
	masked := executeRegexp.ReplaceAllStringFunc(str, func(src string) string {
		symbol := ":"
		if strings.HasPrefix(src, "+") {
			symbol = "+"
		}
		prefix := symbol + "execute"
		if strings.HasPrefix(src[1:], "reload") {
			prefix = symbol + "reload"
		} else if strings.HasPrefix(src[1:], "change-preview-window") {
			prefix = symbol + "change-preview-window"
		} else if strings.HasPrefix(src[1:], "change-preview") {
			prefix = symbol + "change-preview"
		} else if strings.HasPrefix(src[1:], "preview") {
			prefix = symbol + "preview"
		} else if strings.HasPrefix(src[1:], "unbind") {
			prefix = symbol + "unbind"
		} else if strings.HasPrefix(src[1:], "rebind") {
			prefix = symbol + "rebind"
		} else if strings.HasPrefix(src[1:], "change-prompt") {
			prefix = symbol + "change-prompt"
		} else if src[len(prefix)] == '-' {
			c := src[len(prefix)+1]
			if c == 's' || c == 'S' {
				prefix += "-silent"
			} else {
				prefix += "-multi"
			}
		}
		return prefix + "(" + strings.Repeat(" ", len(src)-len(prefix)-2) + ")"
	})
	masked = strings.Replace(masked, "::", string([]rune{escapedColon, ':'}), -1)
	masked = strings.Replace(masked, ",:", string([]rune{escapedComma, ':'}), -1)
	masked = strings.Replace(masked, "+:", string([]rune{escapedPlus, ':'}), -1)

	idx := 0
	for _, pairStr := range strings.Split(masked, ",") {
		origPairStr := str[idx : idx+len(pairStr)]
		idx += len(pairStr) + 1

		pair := strings.SplitN(pairStr, ":", 2)
		if len(pair) < 2 {
			errorExit("bind action not specified: " + origPairStr)
		}
		var key tui.Event
		if len(pair[0]) == 1 && pair[0][0] == escapedColon {
			key = tui.Key(':')
		} else if len(pair[0]) == 1 && pair[0][0] == escapedComma {
			key = tui.Key(',')
		} else if len(pair[0]) == 1 && pair[0][0] == escapedPlus {
			key = tui.Key('+')
		} else {
			keys := parseKeyChords(pair[0], "key name required")
			key = firstKey(keys)
		}

		idx2 := len(pair[0]) + 1
		specs := strings.Split(pair[1], "+")
		actions := make([]*action, 0, len(specs))
		appendAction := func(types ...actionType) {
			actions = append(actions, toActions(types...)...)
		}
		prevSpec := ""
		for specIndex, maskedSpec := range specs {
			spec := origPairStr[idx2 : idx2+len(maskedSpec)]
			idx2 += len(maskedSpec) + 1
			spec = prevSpec + spec
			specLower := strings.ToLower(spec)
			switch specLower {
			case "ignore":
				appendAction(actIgnore)
			case "beginning-of-line":
				appendAction(actBeginningOfLine)
			case "abort":
				appendAction(actAbort)
			case "accept":
				appendAction(actAccept)
			case "accept-non-empty":
				appendAction(actAcceptNonEmpty)
			case "print-query":
				appendAction(actPrintQuery)
			case "refresh-preview":
				appendAction(actRefreshPreview)
			case "replace-query":
				appendAction(actReplaceQuery)
			case "backward-char":
				appendAction(actBackwardChar)
			case "backward-delete-char":
				appendAction(actBackwardDeleteChar)
			case "backward-delete-char/eof":
				appendAction(actBackwardDeleteCharEOF)
			case "backward-word":
				appendAction(actBackwardWord)
			case "clear-screen":
				appendAction(actClearScreen)
			case "delete-char":
				appendAction(actDeleteChar)
			case "delete-char/eof":
				appendAction(actDeleteCharEOF)
			case "deselect":
				appendAction(actDeselect)
			case "end-of-line":
				appendAction(actEndOfLine)
			case "cancel":
				appendAction(actCancel)
			case "clear-query":
				appendAction(actClearQuery)
			case "clear-selection":
				appendAction(actClearSelection)
			case "forward-char":
				appendAction(actForwardChar)
			case "forward-word":
				appendAction(actForwardWord)
			case "jump":
				appendAction(actJump)
			case "jump-accept":
				appendAction(actJumpAccept)
			case "kill-line":
				appendAction(actKillLine)
			case "kill-word":
				appendAction(actKillWord)
			case "unix-line-discard", "line-discard":
				appendAction(actUnixLineDiscard)
			case "unix-word-rubout", "word-rubout":
				appendAction(actUnixWordRubout)
			case "yank":
				appendAction(actYank)
			case "backward-kill-word":
				appendAction(actBackwardKillWord)
			case "toggle-down":
				appendAction(actToggle, actDown)
			case "toggle-up":
				appendAction(actToggle, actUp)
			case "toggle-in":
				appendAction(actToggleIn)
			case "toggle-out":
				appendAction(actToggleOut)
			case "toggle-all":
				appendAction(actToggleAll)
			case "toggle-search":
				appendAction(actToggleSearch)
			case "select":
				appendAction(actSelect)
			case "select-all":
				appendAction(actSelectAll)
			case "deselect-all":
				appendAction(actDeselectAll)
			case "close":
				appendAction(actClose)
			case "toggle":
				appendAction(actToggle)
			case "down":
				appendAction(actDown)
			case "up":
				appendAction(actUp)
			case "first", "top":
				appendAction(actFirst)
			case "last":
				appendAction(actLast)
			case "page-up":
				appendAction(actPageUp)
			case "page-down":
				appendAction(actPageDown)
			case "half-page-up":
				appendAction(actHalfPageUp)
			case "half-page-down":
				appendAction(actHalfPageDown)
			case "previous-history":
				appendAction(actPreviousHistory)
			case "next-history":
				appendAction(actNextHistory)
			case "toggle-preview":
				appendAction(actTogglePreview)
			case "toggle-preview-wrap":
				appendAction(actTogglePreviewWrap)
			case "toggle-sort":
				appendAction(actToggleSort)
			case "preview-top":
				appendAction(actPreviewTop)
			case "preview-bottom":
				appendAction(actPreviewBottom)
			case "preview-up":
				appendAction(actPreviewUp)
			case "preview-down":
				appendAction(actPreviewDown)
			case "preview-page-up":
				appendAction(actPreviewPageUp)
			case "preview-page-down":
				appendAction(actPreviewPageDown)
			case "preview-half-page-up":
				appendAction(actPreviewHalfPageUp)
			case "preview-half-page-down":
				appendAction(actPreviewHalfPageDown)
			case "enable-search":
				appendAction(actEnableSearch)
			case "disable-search":
				appendAction(actDisableSearch)
			case "put":
				if key.Type == tui.Rune && unicode.IsGraphic(key.Char) {
					appendAction(actRune)
				} else {
					errorExit("unable to put non-printable character: " + pair[0])
				}
			default:
				t := isExecuteAction(specLower)
				if t == actIgnore {
					if specIndex == 0 && specLower == "" {
						actions = append(keymap[key], actions...)
					} else {
						errorExit("unknown action: " + spec)
					}
				} else {
					var offset int
					switch t {
					case actReload:
						offset = len("reload")
					case actPreview:
						offset = len("preview")
					case actChangePreviewWindow:
						offset = len("change-preview-window")
					case actChangePreview:
						offset = len("change-preview")
					case actChangePrompt:
						offset = len("change-prompt")
					case actUnbind:
						offset = len("unbind")
					case actRebind:
						offset = len("rebind")
					case actExecuteSilent:
						offset = len("execute-silent")
					case actExecuteMulti:
						offset = len("execute-multi")
					default:
						offset = len("execute")
					}
					var actionArg string
					if spec[offset] == ':' {
						if specIndex == len(specs)-1 {
							actionArg = spec[offset+1:]
							actions = append(actions, &action{t: t, a: actionArg})
						} else {
							prevSpec = spec + "+"
							continue
						}
					} else {
						actionArg = spec[offset+1 : len(spec)-1]
						actions = append(actions, &action{t: t, a: actionArg})
					}
					if t == actUnbind || t == actRebind {
						parseKeyChords(actionArg, spec[0:offset]+" target required")
					} else if t == actChangePreviewWindow {
						opts := previewOpts{}
						for _, arg := range strings.Split(actionArg, "|") {
							parsePreviewWindow(&opts, arg)
						}
					}
				}
			}
			prevSpec = ""
		}
		keymap[key] = actions
	}
}

func isExecuteAction(str string) actionType {
	matches := executeRegexp.FindAllStringSubmatch(":"+str, -1)
	if matches == nil || len(matches) != 1 {
		return actIgnore
	}
	prefix := matches[0][1]
	if len(prefix) == 0 {
		prefix = matches[0][2]
	}
	switch prefix {
	case "reload":
		return actReload
	case "unbind":
		return actUnbind
	case "rebind":
		return actRebind
	case "preview":
		return actPreview
	case "change-preview-window":
		return actChangePreviewWindow
	case "change-preview":
		return actChangePreview
	case "change-prompt":
		return actChangePrompt
	case "execute":
		return actExecute
	case "execute-silent":
		return actExecuteSilent
	case "execute-multi":
		return actExecuteMulti
	}
	return actIgnore
}

func parseToggleSort(keymap map[tui.Event][]*action, str string) {
	keys := parseKeyChords(str, "key name required")
	if len(keys) != 1 {
		errorExit("multiple keys specified")
	}
	keymap[firstKey(keys)] = toActions(actToggleSort)
}

func strLines(str string) []string {
	return strings.Split(strings.TrimSuffix(str, "\n"), "\n")
}

func parseSize(str string, maxPercent float64, label string) sizeSpec {
	var val float64
	percent := strings.HasSuffix(str, "%")
	if percent {
		val = atof(str[:len(str)-1])
		if val < 0 {
			errorExit(label + " must be non-negative")
		}
		if val > maxPercent {
			errorExit(fmt.Sprintf("%s too large (max: %d%%)", label, int(maxPercent)))
		}
	} else {
		if strings.Contains(str, ".") {
			errorExit(label + " (without %) must be a non-negative integer")
		}

		val = float64(atoi(str))
		if val < 0 {
			errorExit(label + " must be non-negative")
		}
	}
	return sizeSpec{val, percent}
}

func parseHeight(str string) sizeSpec {
	size := parseSize(str, 100, "height")
	return size
}

func parseLayout(str string) layoutType {
	switch str {
	case "default":
		return layoutDefault
	case "reverse":
		return layoutReverse
	case "reverse-list":
		return layoutReverseList
	default:
		errorExit("invalid layout (expected: default / reverse / reverse-list)")
	}
	return layoutDefault
}

func parseInfoStyle(str string) infoStyle {
	switch str {
	case "default":
		return infoDefault
	case "inline":
		return infoInline
	case "hidden":
		return infoHidden
	default:
		errorExit("invalid info style (expected: default / inline / hidden)")
	}
	return infoDefault
}

func parsePreviewWindow(opts *previewOpts, input string) {
	delimRegex := regexp.MustCompile("[:,]") // : for backward compatibility
	sizeRegex := regexp.MustCompile("^[0-9]+%?$")
	offsetRegex := regexp.MustCompile(`^(\+{-?[0-9]+})?([+-][0-9]+)*(-?/[1-9][0-9]*)?$`)
	headerRegex := regexp.MustCompile("^~(0|[1-9][0-9]*)$")
	tokens := delimRegex.Split(input, -1)
	for _, token := range tokens {
		switch token {
		case "":
		case "default":
			*opts = defaultPreviewOpts(opts.command)
		case "hidden":
			opts.hidden = true
		case "nohidden":
			opts.hidden = false
		case "wrap":
			opts.wrap = true
		case "nowrap":
			opts.wrap = false
		case "cycle":
			opts.cycle = true
		case "nocycle":
			opts.cycle = false
		case "up", "top":
			opts.position = posUp
		case "down", "bottom":
			opts.position = posDown
		case "left":
			opts.position = posLeft
		case "right":
			opts.position = posRight
		case "rounded", "border", "border-rounded":
			opts.border = tui.BorderRounded
		case "sharp", "border-sharp":
			opts.border = tui.BorderSharp
		case "noborder", "border-none":
			opts.border = tui.BorderNone
		case "border-horizontal":
			opts.border = tui.BorderHorizontal
		case "border-vertical":
			opts.border = tui.BorderVertical
		case "border-top":
			opts.border = tui.BorderTop
		case "border-bottom":
			opts.border = tui.BorderBottom
		case "border-left":
			opts.border = tui.BorderLeft
		case "border-right":
			opts.border = tui.BorderRight
		case "follow":
			opts.follow = true
		case "nofollow":
			opts.follow = false
		default:
			if headerRegex.MatchString(token) {
				opts.headerLines = atoi(token[1:])
			} else if sizeRegex.MatchString(token) {
				opts.size = parseSize(token, 99, "window size")
			} else if offsetRegex.MatchString(token) {
				opts.scroll = token
			} else {
				errorExit("invalid preview window option: " + token)
			}
		}
	}
}

func parseMargin(opt string, margin string) [4]sizeSpec {
	margins := strings.Split(margin, ",")
	checked := func(str string) sizeSpec {
		return parseSize(str, 49, opt)
	}
	switch len(margins) {
	case 1:
		m := checked(margins[0])
		return [4]sizeSpec{m, m, m, m}
	case 2:
		tb := checked(margins[0])
		rl := checked(margins[1])
		return [4]sizeSpec{tb, rl, tb, rl}
	case 3:
		t := checked(margins[0])
		rl := checked(margins[1])
		b := checked(margins[2])
		return [4]sizeSpec{t, rl, b, rl}
	case 4:
		return [4]sizeSpec{
			checked(margins[0]), checked(margins[1]),
			checked(margins[2]), checked(margins[3])}
	default:
		errorExit("invalid " + opt + ": " + margin)
	}
	return defaultMargin()
}

func parseOptions(opts *Options, allArgs []string) {
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
	validateJumpLabels := false
	validatePointer := false
	validateMarker := false
	validateEllipsis := false
	for i := 0; i < len(allArgs); i++ {
		arg := allArgs[i]
		switch arg {
		case "-h", "--help":
			help(exitOk)
		case "-x", "--extended":
			opts.Extended = true
		case "-e", "--exact":
			opts.Fuzzy = false
		case "--extended-exact":
			// Note that we now don't have --no-extended-exact
			opts.Fuzzy = false
			opts.Extended = true
		case "+x", "--no-extended":
			opts.Extended = false
		case "+e", "--no-exact":
			opts.Fuzzy = true
		case "-q", "--query":
			opts.Query = nextString(allArgs, &i, "query string required")
		case "-f", "--filter":
			filter := nextString(allArgs, &i, "query string required")
			opts.Filter = &filter
		case "--literal":
			opts.Normalize = false
		case "--no-literal":
			opts.Normalize = true
		case "--algo":
			opts.FuzzyAlgo = parseAlgo(nextString(allArgs, &i, "algorithm required (v1|v2)"))
		case "--expect":
			for k, v := range parseKeyChords(nextString(allArgs, &i, "key names required"), "key names required") {
				opts.Expect[k] = v
			}
		case "--no-expect":
			opts.Expect = make(map[tui.Event]string)
		case "--enabled", "--no-phony":
			opts.Phony = false
		case "--disabled", "--phony":
			opts.Phony = true
		case "--tiebreak":
			opts.Criteria = parseTiebreak(nextString(allArgs, &i, "sort criterion required"))
		case "--bind":
			parseKeymap(opts.Keymap, nextString(allArgs, &i, "bind expression required"))
		case "--color":
			_, spec := optionalNextString(allArgs, &i)
			if len(spec) == 0 {
				opts.Theme = tui.EmptyTheme()
			} else {
				opts.Theme = parseTheme(opts.Theme, spec)
			}
		case "--toggle-sort":
			parseToggleSort(opts.Keymap, nextString(allArgs, &i, "key name required"))
		case "-d", "--delimiter":
			opts.Delimiter = delimiterRegexp(nextString(allArgs, &i, "delimiter required"))
		case "-n", "--nth":
			opts.Nth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "--with-nth":
			opts.WithNth = splitNth(nextString(allArgs, &i, "nth expression required"))
		case "-s", "--sort":
			opts.Sort = optionalNumeric(allArgs, &i, 1)
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
			opts.Multi = optionalNumeric(allArgs, &i, maxMulti)
		case "+m", "--no-multi":
			opts.Multi = 0
		case "--ansi":
			opts.Ansi = true
		case "--no-ansi":
			opts.Ansi = false
		case "--no-mouse":
			opts.Mouse = false
		case "+c", "--no-color":
			opts.Theme = tui.NoColorTheme()
		case "+2", "--no-256":
			opts.Theme = tui.Default16
		case "--black":
			opts.Black = true
		case "--no-black":
			opts.Black = false
		case "--bold":
			opts.Bold = true
		case "--no-bold":
			opts.Bold = false
		case "--layout":
			opts.Layout = parseLayout(
				nextString(allArgs, &i, "layout required (default / reverse / reverse-list)"))
		case "--reverse":
			opts.Layout = layoutReverse
		case "--no-reverse":
			opts.Layout = layoutDefault
		case "--cycle":
			opts.Cycle = true
		case "--no-cycle":
			opts.Cycle = false
		case "--keep-right":
			opts.KeepRight = true
		case "--no-keep-right":
			opts.KeepRight = false
		case "--hscroll":
			opts.Hscroll = true
		case "--no-hscroll":
			opts.Hscroll = false
		case "--hscroll-off":
			opts.HscrollOff = nextInt(allArgs, &i, "hscroll offset required")
		case "--scroll-off":
			opts.ScrollOff = nextInt(allArgs, &i, "scroll offset required")
		case "--filepath-word":
			opts.FileWord = true
		case "--no-filepath-word":
			opts.FileWord = false
		case "--info":
			opts.InfoStyle = parseInfoStyle(
				nextString(allArgs, &i, "info style required"))
		case "--no-info":
			opts.InfoStyle = infoHidden
		case "--inline-info":
			opts.InfoStyle = infoInline
		case "--no-inline-info":
			opts.InfoStyle = infoDefault
		case "--jump-labels":
			opts.JumpLabels = nextString(allArgs, &i, "label characters required")
			validateJumpLabels = true
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
		case "--print0":
			opts.Printer = func(str string) { fmt.Print(str, "\x00") }
			opts.PrintSep = "\x00"
		case "--no-print0":
			opts.Printer = func(str string) { fmt.Println(str) }
			opts.PrintSep = "\n"
		case "--print-query":
			opts.PrintQuery = true
		case "--no-print-query":
			opts.PrintQuery = false
		case "--prompt":
			opts.Prompt = nextString(allArgs, &i, "prompt string required")
		case "--pointer":
			opts.Pointer = nextString(allArgs, &i, "pointer sign string required")
			validatePointer = true
		case "--marker":
			opts.Marker = nextString(allArgs, &i, "selected sign string required")
			validateMarker = true
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
		case "--no-header":
			opts.Header = []string{}
		case "--no-header-lines":
			opts.HeaderLines = 0
		case "--header":
			opts.Header = strLines(nextString(allArgs, &i, "header string required"))
		case "--header-lines":
			opts.HeaderLines = atoi(
				nextString(allArgs, &i, "number of header lines required"))
		case "--header-first":
			opts.HeaderFirst = true
		case "--no-header-first":
			opts.HeaderFirst = false
		case "--ellipsis":
			opts.Ellipsis = nextString(allArgs, &i, "ellipsis string required")
			validateEllipsis = true
		case "--preview":
			opts.Preview.command = nextString(allArgs, &i, "preview command required")
		case "--no-preview":
			opts.Preview.command = ""
		case "--preview-window":
			parsePreviewWindow(&opts.Preview,
				nextString(allArgs, &i, "preview window layout required: [up|down|left|right][,SIZE[%]][,border-BORDER_OPT][,wrap][,cycle][,hidden][,+SCROLL[OFFSETS][/DENOM]][,~HEADER_LINES][,default]"))
		case "--height":
			opts.Height = parseHeight(nextString(allArgs, &i, "height required: HEIGHT[%]"))
		case "--min-height":
			opts.MinHeight = nextInt(allArgs, &i, "height required: HEIGHT")
		case "--no-height":
			opts.Height = sizeSpec{}
		case "--no-margin":
			opts.Margin = defaultMargin()
		case "--no-padding":
			opts.Padding = defaultMargin()
		case "--no-border":
			opts.BorderShape = tui.BorderNone
		case "--border":
			hasArg, arg := optionalNextString(allArgs, &i)
			opts.BorderShape = parseBorder(arg, !hasArg)
		case "--no-unicode":
			opts.Unicode = false
		case "--unicode":
			opts.Unicode = true
		case "--margin":
			opts.Margin = parseMargin(
				"margin",
				nextString(allArgs, &i, "margin required (TRBL / TB,RL / T,RL,B / T,R,B,L)"))
		case "--padding":
			opts.Padding = parseMargin(
				"padding",
				nextString(allArgs, &i, "padding required (TRBL / TB,RL / T,RL,B / T,R,B,L)"))
		case "--tabstop":
			opts.Tabstop = nextInt(allArgs, &i, "tab stop required")
		case "--clear":
			opts.ClearOnExit = true
		case "--no-clear":
			opts.ClearOnExit = false
		case "--version":
			opts.Version = true
		default:
			if match, value := optString(arg, "--algo="); match {
				opts.FuzzyAlgo = parseAlgo(value)
			} else if match, value := optString(arg, "-q", "--query="); match {
				opts.Query = value
			} else if match, value := optString(arg, "-f", "--filter="); match {
				opts.Filter = &value
			} else if match, value := optString(arg, "-d", "--delimiter="); match {
				opts.Delimiter = delimiterRegexp(value)
			} else if match, value := optString(arg, "--border="); match {
				opts.BorderShape = parseBorder(value, false)
			} else if match, value := optString(arg, "--prompt="); match {
				opts.Prompt = value
			} else if match, value := optString(arg, "--pointer="); match {
				opts.Pointer = value
				validatePointer = true
			} else if match, value := optString(arg, "--marker="); match {
				opts.Marker = value
				validateMarker = true
			} else if match, value := optString(arg, "-n", "--nth="); match {
				opts.Nth = splitNth(value)
			} else if match, value := optString(arg, "--with-nth="); match {
				opts.WithNth = splitNth(value)
			} else if match, _ := optString(arg, "-s", "--sort="); match {
				opts.Sort = 1 // Don't care
			} else if match, value := optString(arg, "-m", "--multi="); match {
				opts.Multi = atoi(value)
			} else if match, value := optString(arg, "--height="); match {
				opts.Height = parseHeight(value)
			} else if match, value := optString(arg, "--min-height="); match {
				opts.MinHeight = atoi(value)
			} else if match, value := optString(arg, "--layout="); match {
				opts.Layout = parseLayout(value)
			} else if match, value := optString(arg, "--info="); match {
				opts.InfoStyle = parseInfoStyle(value)
			} else if match, value := optString(arg, "--toggle-sort="); match {
				parseToggleSort(opts.Keymap, value)
			} else if match, value := optString(arg, "--expect="); match {
				for k, v := range parseKeyChords(value, "key names required") {
					opts.Expect[k] = v
				}
			} else if match, value := optString(arg, "--tiebreak="); match {
				opts.Criteria = parseTiebreak(value)
			} else if match, value := optString(arg, "--color="); match {
				opts.Theme = parseTheme(opts.Theme, value)
			} else if match, value := optString(arg, "--bind="); match {
				parseKeymap(opts.Keymap, value)
			} else if match, value := optString(arg, "--history="); match {
				setHistory(value)
			} else if match, value := optString(arg, "--history-size="); match {
				setHistoryMax(atoi(value))
			} else if match, value := optString(arg, "--header="); match {
				opts.Header = strLines(value)
			} else if match, value := optString(arg, "--header-lines="); match {
				opts.HeaderLines = atoi(value)
			} else if match, value := optString(arg, "--ellipsis="); match {
				opts.Ellipsis = value
				validateEllipsis = true
			} else if match, value := optString(arg, "--preview="); match {
				opts.Preview.command = value
			} else if match, value := optString(arg, "--preview-window="); match {
				parsePreviewWindow(&opts.Preview, value)
			} else if match, value := optString(arg, "--margin="); match {
				opts.Margin = parseMargin("margin", value)
			} else if match, value := optString(arg, "--padding="); match {
				opts.Padding = parseMargin("padding", value)
			} else if match, value := optString(arg, "--tabstop="); match {
				opts.Tabstop = atoi(value)
			} else if match, value := optString(arg, "--hscroll-off="); match {
				opts.HscrollOff = atoi(value)
			} else if match, value := optString(arg, "--scroll-off="); match {
				opts.ScrollOff = atoi(value)
			} else if match, value := optString(arg, "--jump-labels="); match {
				opts.JumpLabels = value
				validateJumpLabels = true
			} else {
				errorExit("unknown option: " + arg)
			}
		}
	}

	if opts.HeaderLines < 0 {
		errorExit("header lines must be a non-negative integer")
	}

	if opts.HscrollOff < 0 {
		errorExit("hscroll offset must be a non-negative integer")
	}

	if opts.ScrollOff < 0 {
		errorExit("scroll offset must be a non-negative integer")
	}

	if opts.Tabstop < 1 {
		errorExit("tab stop must be a positive integer")
	}

	if len(opts.JumpLabels) == 0 {
		errorExit("empty jump labels")
	}

	if validateJumpLabels {
		for _, r := range opts.JumpLabels {
			if r < 32 || r > 126 {
				errorExit("non-ascii jump labels are not allowed")
			}
		}
	}

	if validatePointer {
		if err := validateSign(opts.Pointer, "pointer"); err != nil {
			errorExit(err.Error())
		}
	}

	if validateMarker {
		if err := validateSign(opts.Marker, "marker"); err != nil {
			errorExit(err.Error())
		}
	}

	if validateEllipsis {
		for _, r := range opts.Ellipsis {
			if !unicode.IsGraphic(r) {
				errorExit("invalid character in ellipsis")
			}
		}
	}
}

func validateSign(sign string, signOptName string) error {
	if sign == "" {
		return fmt.Errorf("%v cannot be empty", signOptName)
	}
	for _, r := range sign {
		if !unicode.IsGraphic(r) {
			return fmt.Errorf("invalid character in %v", signOptName)
		}
	}
	if runewidth.StringWidth(sign) > 2 {
		return fmt.Errorf("%v display width should be up to 2", signOptName)
	}
	return nil
}

func postProcessOptions(opts *Options) {
	if !opts.Version && !tui.IsLightRendererSupported() && opts.Height.size > 0 {
		errorExit("--height option is currently not supported on this platform")
	}
	// Default actions for CTRL-N / CTRL-P when --history is set
	if opts.History != nil {
		if _, prs := opts.Keymap[tui.CtrlP.AsEvent()]; !prs {
			opts.Keymap[tui.CtrlP.AsEvent()] = toActions(actPreviousHistory)
		}
		if _, prs := opts.Keymap[tui.CtrlN.AsEvent()]; !prs {
			opts.Keymap[tui.CtrlN.AsEvent()] = toActions(actNextHistory)
		}
	}

	// Extend the default key map
	keymap := defaultKeymap()
	for key, actions := range opts.Keymap {
		var lastChangePreviewWindow *action
		for _, act := range actions {
			switch act.t {
			case actToggleSort:
				// To display "+S"/"-S" on info line
				opts.ToggleSort = true
			case actChangePreviewWindow:
				lastChangePreviewWindow = act
			}
		}
		// Re-organize actions so that we only keep the last change-preview-window
		// and it comes first in the list.
		//  *  change-preview-window(up,+10)+preview(sleep 3; cat {})+change-preview-window(up,+20)
		//  -> change-preview-window(up,+20)+preview(sleep 3; cat {})
		if lastChangePreviewWindow != nil {
			reordered := []*action{lastChangePreviewWindow}
			for _, act := range actions {
				if act.t != actChangePreviewWindow {
					reordered = append(reordered, act)
				}
			}
			actions = reordered
		}
		keymap[key] = actions
	}
	opts.Keymap = keymap

	// If we're not using extended search mode, --nth option becomes irrelevant
	// if it contains the whole range
	if !opts.Extended || len(opts.Nth) == 1 {
		for _, r := range opts.Nth {
			if r.begin == rangeEllipsis && r.end == rangeEllipsis {
				opts.Nth = make([]Range, 0)
				return
			}
		}
	}

	if opts.Bold {
		theme := opts.Theme
		boldify := func(c tui.ColorAttr) tui.ColorAttr {
			dup := c
			if !theme.Colored {
				dup.Attr |= tui.Bold
			} else if (c.Attr & tui.AttrRegular) == 0 {
				dup.Attr |= tui.Bold
			}
			return dup
		}
		theme.Current = boldify(theme.Current)
		theme.CurrentMatch = boldify(theme.CurrentMatch)
		theme.Prompt = boldify(theme.Prompt)
		theme.Input = boldify(theme.Input)
		theme.Cursor = boldify(theme.Cursor)
		theme.Spinner = boldify(theme.Spinner)
	}
}

// ParseOptions parses command-line options
func ParseOptions() *Options {
	opts := defaultOptions()

	// Options from Env var
	words, _ := shellwords.Parse(os.Getenv("FZF_DEFAULT_OPTS"))
	if len(words) > 0 {
		parseOptions(opts, words)
	}

	// Options from command-line arguments
	parseOptions(opts, os.Args[1:])

	postProcessOptions(opts)
	return opts
}
