package fzf

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/tui"

	"github.com/junegunn/go-shellwords"
	"github.com/rivo/uniseg"
)

const Usage = `fzf is an interactive filter program for any kind of list.

It implements a "fuzzy" matching algorithm, so you can quickly type in patterns
with omitted characters and still get the results you want.

Project URL: https://github.com/junegunn/fzf
Author: Junegunn Choi <junegunn.c@gmail.com>

Usage: fzf [options]

  Search
    -x, --extended          Extended-search mode
                            (enabled by default; +x or --no-extended to disable)
    -e, --exact             Enable Exact-match
    -i, --ignore-case       Case-insensitive match (default: smart-case match)
    +i, --no-ignore-case    Case-sensitive match
    --scheme=SCHEME         Scoring scheme [default|path|history]
    --literal               Do not normalize latin script letters before matching
    -n, --nth=N[,..]        Comma-separated list of field index expressions
                            for limiting search scope. Each can be a non-zero
                            integer or a range expression ([BEGIN]..[END]).
    --with-nth=N[,..]       Transform the presentation of each line using
                            field index expressions
    -d, --delimiter=STR     Field delimiter regex (default: AWK-style)
    +s, --no-sort           Do not sort the result
    --tail=NUM              Maximum number of items to keep in memory
    --track                 Track the current selection when the result is updated
    --tac                   Reverse the order of the input
    --disabled              Do not perform search
    --tiebreak=CRI[,..]     Comma-separated list of sort criteria to apply
                            when the scores are tied [length|chunk|begin|end|index]
                            (default: length)
  Interface
    -m, --multi[=MAX]       Enable multi-select with tab/shift-tab
    --no-mouse              Disable mouse
    --bind=KEYBINDS         Custom key bindings. Refer to the man page.
    --cycle                 Enable cyclic scroll
    --wrap                  Enable line wrap
    --wrap-sign=STR         Indicator for wrapped lines
    --no-multi-line         Disable multi-line display of items when using --read0
    --keep-right            Keep the right end of the line visible on overflow
    --scroll-off=LINES      Number of screen lines to keep above or below when
                            scrolling to the top or to the bottom (default: 0)
    --no-hscroll            Disable horizontal scroll
    --hscroll-off=COLS      Number of screen columns to keep to the right of the
                            highlighted substring (default: 10)
    --filepath-word         Make word-wise movements respect path separators
    --jump-labels=CHARS     Label characters for jump mode

  Layout
    --height=[~]HEIGHT[%]   Display fzf window below the cursor with the given
                            height instead of using fullscreen.
                            A negative value is calculated as the terminal height
                            minus the given value.
                            If prefixed with '~', fzf will determine the height
                            according to the input size.
    --min-height=HEIGHT     Minimum height when --height is given in percent
                            (default: 10)
    --tmux[=OPTS]           Start fzf in a tmux popup (requires tmux 3.3+)
                            [center|top|bottom|left|right][,SIZE[%]][,SIZE[%]]
                            (default: center,50%)
    --layout=LAYOUT         Choose layout: [default|reverse|reverse-list]
    --border[=STYLE]        Draw border around the finder
                            [rounded|sharp|bold|block|thinblock|double|horizontal|vertical|
                             top|bottom|left|right|none] (default: rounded)
    --border-label=LABEL    Label to print on the border
    --border-label-pos=COL  Position of the border label
                            [POSITIVE_INTEGER: columns from left|
                             NEGATIVE_INTEGER: columns from right][:bottom]
                            (default: 0 or center)
    --margin=MARGIN         Screen margin (TRBL | TB,RL | T,RL,B | T,R,B,L)
    --padding=PADDING       Padding inside border (TRBL | TB,RL | T,RL,B | T,R,B,L)
    --info=STYLE            Finder info style
                            [default|right|hidden|inline[-right][:PREFIX]]
    --info-command=COMMAND  Command to generate info line
    --separator=STR         String to form horizontal separator on info line
    --no-separator          Hide info line separator
    --scrollbar[=C1[C2]]    Scrollbar character(s) (each for main and preview window)
    --no-scrollbar          Hide scrollbar
    --prompt=STR            Input prompt (default: '> ')
    --pointer=STR           Pointer to the current line (default: '▌' or '>')
    --marker=STR            Multi-select marker (default: '┃' or '>')
    --marker-multi-line=STR Multi-select marker for multi-line entries;
                            3 elements for top, middle, and bottom (default: '╻┃╹')
    --header=STR            String to print as header
    --header-lines=N        The first N lines of the input are treated as header
    --header-first          Print header before the prompt line
    --ellipsis=STR          Ellipsis to show when line is truncated (default: '··')

  Display
    --ansi                  Enable processing of ANSI color codes
    --tabstop=SPACES        Number of spaces for a tab character (default: 8)
    --color=COLSPEC         Base scheme (dark|light|16|bw) and/or custom colors
    --highlight-line        Highlight the whole current line
    --no-bold               Do not use bold text

  History
    --history=FILE          History file
    --history-size=N        Maximum number of history entries (default: 1000)

  Preview
    --preview=COMMAND       Command to preview highlighted line ({})
    --preview-window=OPT    Preview window layout (default: right:50%)
                            [up|down|left|right][,SIZE[%]]
                            [,[no]wrap][,[no]cycle][,[no]follow][,[no]hidden]
                            [,border-BORDER_OPT]
                            [,+SCROLL[OFFSETS][/DENOM]][,~HEADER_LINES]
                            [,default][,<SIZE_THRESHOLD(ALTERNATIVE_LAYOUT)]
    --preview-label=LABEL
    --preview-label-pos=N   Same as --border-label and --border-label-pos,
                            but for preview window

  Scripting
    -q, --query=STR         Start the finder with the given query
    -1, --select-1          Automatically select the only match
    -0, --exit-0            Exit immediately when there's no match
    -f, --filter=STR        Filter mode. Do not start interactive finder.
    --print-query           Print query as the first line
    --expect=KEYS           Comma-separated list of keys to complete fzf
    --read0                 Read input delimited by ASCII NUL characters
    --print0                Print output delimited by ASCII NUL characters
    --sync                  Synchronous search for multi-staged filtering
    --with-shell=STR        Shell command and flags to start child processes with
    --listen[=[ADDR:]PORT]  Start HTTP server to receive actions (POST /)
                            (To allow remote process execution, use --listen-unsafe)

  Directory traversal       (Only used when $FZF_DEFAULT_COMMAND is not set)
    --walker=OPTS           [file][,dir][,follow][,hidden] (default: file,follow,hidden)
    --walker-root=DIR       Root directory from which to start walker (default: .)
    --walker-skip=DIRS      Comma-separated list of directory names to skip
                            (default: .git,node_modules)

  Shell integration
    --bash                  Print script to set up Bash shell integration
    --zsh                   Print script to set up Zsh shell integration
    --fish                  Print script to set up Fish shell integration

  Help
    --version               Display version information and exit
    --help                  Show this message
    --man                   Show man page

  Environment variables
    FZF_DEFAULT_COMMAND     Default command to use when input is tty
    FZF_DEFAULT_OPTS        Default options (e.g. '--layout=reverse --info=inline')
    FZF_DEFAULT_OPTS_FILE   Location of the file to read default options from
    FZF_API_KEY             X-API-Key header for HTTP server (--listen)

`

const defaultInfoPrefix = " < "

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
	byChunk
	byLength
	byBegin
	byEnd
)

type heightSpec struct {
	size    float64
	percent bool
	auto    bool
	inverse bool
	index   int
}

type sizeSpec struct {
	size    float64
	percent bool
}

func (s sizeSpec) String() string {
	if s.percent {
		return fmt.Sprintf("%d%%", int(s.size))
	}
	return fmt.Sprintf("%d", int(s.size))
}

func defaultMargin() [4]sizeSpec {
	return [4]sizeSpec{}
}

type trackOption int

const (
	trackDisabled trackOption = iota
	trackEnabled
	trackCurrent
)

type windowPosition int

const (
	posUp windowPosition = iota
	posDown
	posLeft
	posRight
	posCenter
)

type tmuxOptions struct {
	width    sizeSpec
	height   sizeSpec
	position windowPosition
	index    int
}

type layoutType int

const (
	layoutDefault layoutType = iota
	layoutReverse
	layoutReverseList
)

type infoStyle int

const (
	infoDefault infoStyle = iota
	infoRight
	infoInline
	infoInlineRight
	infoHidden
)

type labelOpts struct {
	label  string
	column int
	bottom bool
}

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
	threshold   int
	alternative *previewOpts
}

func (o *previewOpts) Visible() bool {
	return o.size.size > 0 || o.alternative != nil && o.alternative.size.size > 0
}

func (o *previewOpts) Toggle() {
	o.hidden = !o.hidden
}

func defaultTmuxOptions(index int) *tmuxOptions {
	return &tmuxOptions{
		position: posCenter,
		width:    sizeSpec{50, true},
		height:   sizeSpec{50, true},
		index:    index}
}

func parseTmuxOptions(arg string, index int) (*tmuxOptions, error) {
	var err error
	opts := defaultTmuxOptions(index)
	tokens := splitRegexp.Split(arg, -1)
	errorToReturn := errors.New("invalid tmux option: " + arg + " (expected: [center|top|bottom|left|right][,SIZE[%]][,SIZE[%]])")
	if len(tokens) == 0 || len(tokens) > 3 {
		return nil, errorToReturn
	}

	// Defaults to 'center'
	switch tokens[0] {
	case "top", "up":
		opts.position = posUp
		opts.width = sizeSpec{100, true}
	case "bottom", "down":
		opts.position = posDown
		opts.width = sizeSpec{100, true}
	case "left":
		opts.position = posLeft
		opts.height = sizeSpec{100, true}
	case "right":
		opts.position = posRight
		opts.height = sizeSpec{100, true}
	case "center":
	default:
		tokens = append([]string{"center"}, tokens...)
	}

	// One size given
	var size1 sizeSpec
	if len(tokens) > 1 {
		if size1, err = parseSize(tokens[1], 100, "size"); err != nil {
			return nil, errorToReturn
		}
	}

	// Two sizes given
	var size2 sizeSpec
	if len(tokens) == 3 {
		if size2, err = parseSize(tokens[2], 100, "size"); err != nil {
			return nil, errorToReturn
		}
		opts.width = size1
		opts.height = size2
	} else if len(tokens) == 2 {
		switch tokens[0] {
		case "top", "up":
			opts.height = size1
		case "bottom", "down":
			opts.height = size1
		case "left":
			opts.width = size1
		case "right":
			opts.width = size1
		case "center":
			opts.width = size1
			opts.height = size1
		}
	}

	return opts, nil
}

func parseLabelPosition(opts *labelOpts, arg string) error {
	opts.column = 0
	opts.bottom = false
	var err error
	for _, token := range splitRegexp.Split(strings.ToLower(arg), -1) {
		switch token {
		case "center":
			opts.column = 0
		case "bottom":
			opts.bottom = true
		case "top":
			opts.bottom = false
		default:
			opts.column, err = atoi(token)
		}
	}
	return err
}

func (a previewOpts) aboveOrBelow() bool {
	return a.size.size > 0 && (a.position == posUp || a.position == posDown)
}

func (a previewOpts) sameLayout(b previewOpts) bool {
	return a.size == b.size && a.position == b.position && a.border == b.border && a.hidden == b.hidden && a.threshold == b.threshold &&
		(a.alternative != nil && b.alternative != nil && a.alternative.sameLayout(*b.alternative) ||
			a.alternative == nil && b.alternative == nil)
}

func (a previewOpts) sameContentLayout(b previewOpts) bool {
	return a.wrap == b.wrap && a.headerLines == b.headerLines
}

func firstLine(s string) string {
	return strings.SplitN(s, "\n", 2)[0]
}

type walkerOpts struct {
	file   bool
	dir    bool
	hidden bool
	follow bool
}

// Options stores the values of command-line options
type Options struct {
	Input        chan string
	Output       chan string
	NoWinpty     bool
	Tmux         *tmuxOptions
	ForceTtyIn   bool
	ProxyScript  string
	Bash         bool
	Zsh          bool
	Fish         bool
	Man          bool
	Fuzzy        bool
	FuzzyAlgo    algo.Algo
	Scheme       string
	Extended     bool
	Phony        bool
	Case         Case
	Normalize    bool
	Nth          []Range
	WithNth      []Range
	Delimiter    Delimiter
	Sort         int
	Track        trackOption
	Tac          bool
	Tail         int
	Criteria     []criterion
	Multi        int
	Ansi         bool
	Mouse        bool
	Theme        *tui.ColorTheme
	Black        bool
	Bold         bool
	Height       heightSpec
	MinHeight    int
	Layout       layoutType
	Cycle        bool
	Wrap         bool
	WrapSign     *string
	MultiLine    bool
	CursorLine   bool
	KeepRight    bool
	Hscroll      bool
	HscrollOff   int
	ScrollOff    int
	FileWord     bool
	InfoStyle    infoStyle
	InfoPrefix   string
	InfoCommand  string
	Separator    *string
	JumpLabels   string
	Prompt       string
	Pointer      *string
	Marker       *string
	MarkerMulti  *[3]string
	Query        string
	Select1      bool
	Exit0        bool
	Filter       *string
	ToggleSort   bool
	Expect       map[tui.Event]string
	Keymap       map[tui.Event][]*action
	Preview      previewOpts
	PrintQuery   bool
	ReadZero     bool
	Printer      func(string)
	PrintSep     string
	Sync         bool
	History      *History
	Header       []string
	HeaderLines  int
	HeaderFirst  bool
	Ellipsis     *string
	Scrollbar    *string
	Margin       [4]sizeSpec
	Padding      [4]sizeSpec
	BorderShape  tui.BorderShape
	BorderLabel  labelOpts
	PreviewLabel labelOpts
	Unicode      bool
	Ambidouble   bool
	Tabstop      int
	WithShell    string
	ListenAddr   *listenAddress
	Unsafe       bool
	ClearOnExit  bool
	WalkerOpts   walkerOpts
	WalkerRoot   string
	WalkerSkip   []string
	Version      bool
	Help         bool
	CPUProfile   string
	MEMProfile   string
	BlockProfile string
	MutexProfile string
}

func filterNonEmpty(input []string) []string {
	output := make([]string, 0, len(input))
	for _, str := range input {
		if len(str) > 0 {
			output = append(output, str)
		}
	}
	return output
}

func defaultPreviewOpts(command string) previewOpts {
	return previewOpts{command, posRight, sizeSpec{50, true}, "", false, false, false, false, tui.DefaultBorderShape, 0, 0, nil}
}

func defaultOptions() *Options {
	var theme *tui.ColorTheme
	if os.Getenv("NO_COLOR") != "" {
		theme = tui.NoColorTheme()
	} else {
		theme = tui.EmptyTheme()
	}

	return &Options{
		Bash:         false,
		Zsh:          false,
		Fish:         false,
		Man:          false,
		Fuzzy:        true,
		FuzzyAlgo:    algo.FuzzyMatchV2,
		Scheme:       "default",
		Extended:     true,
		Phony:        false,
		Case:         CaseSmart,
		Normalize:    true,
		Nth:          make([]Range, 0),
		WithNth:      make([]Range, 0),
		Delimiter:    Delimiter{},
		Sort:         1000,
		Track:        trackDisabled,
		Tac:          false,
		Criteria:     []criterion{byScore, byLength},
		Multi:        0,
		Ansi:         false,
		Mouse:        true,
		Theme:        theme,
		Black:        false,
		Bold:         true,
		MinHeight:    10,
		Layout:       layoutDefault,
		Cycle:        false,
		Wrap:         false,
		MultiLine:    true,
		KeepRight:    false,
		Hscroll:      true,
		HscrollOff:   10,
		ScrollOff:    3,
		FileWord:     false,
		InfoStyle:    infoDefault,
		Separator:    nil,
		JumpLabels:   defaultJumpLabels,
		Prompt:       "> ",
		Pointer:      nil,
		Marker:       nil,
		MarkerMulti:  nil,
		Query:        "",
		Select1:      false,
		Exit0:        false,
		Filter:       nil,
		ToggleSort:   false,
		Expect:       make(map[tui.Event]string),
		Keymap:       make(map[tui.Event][]*action),
		Preview:      defaultPreviewOpts(""),
		PrintQuery:   false,
		ReadZero:     false,
		Printer:      func(str string) { fmt.Println(str) },
		PrintSep:     "\n",
		Sync:         false,
		History:      nil,
		Header:       make([]string, 0),
		HeaderLines:  0,
		HeaderFirst:  false,
		Ellipsis:     nil,
		Scrollbar:    nil,
		Margin:       defaultMargin(),
		Padding:      defaultMargin(),
		Unicode:      true,
		Ambidouble:   os.Getenv("RUNEWIDTH_EASTASIAN") == "1",
		Tabstop:      8,
		BorderLabel:  labelOpts{},
		PreviewLabel: labelOpts{},
		Unsafe:       false,
		ClearOnExit:  true,
		WalkerOpts:   walkerOpts{file: true, hidden: true, follow: true},
		WalkerRoot:   ".",
		WalkerSkip:   []string{".git", "node_modules"},
		Help:         false,
		Version:      false}
}

func optString(arg string, prefixes ...string) (bool, string) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(arg, prefix) {
			return true, arg[len(prefix):]
		}
	}
	return false, ""
}

func nextString(args []string, i *int, message string) (string, error) {
	if len(args) > *i+1 {
		*i++
	} else {
		return "", errors.New(message)
	}
	return args[*i], nil
}

func optionalNextString(args []string, i *int) (bool, string) {
	if len(args) > *i+1 && !strings.HasPrefix(args[*i+1], "-") && !strings.HasPrefix(args[*i+1], "+") {
		*i++
		return true, args[*i]
	}
	return false, ""
}

func atoi(str string) (int, error) {
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0, errors.New("not a valid integer: " + str)
	}
	return num, nil
}

func atof(str string) (float64, error) {
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, errors.New("not a valid number: " + str)
	}
	return num, nil
}

func nextInt(args []string, i *int, message string) (int, error) {
	if len(args) > *i+1 {
		*i++
	} else {
		return 0, errors.New(message)
	}
	n, err := atoi(args[*i])
	if err != nil {
		return 0, errors.New(message)
	}
	return n, nil
}

func optionalNumeric(args []string, i *int, defaultValue int) (int, error) {
	if len(args) > *i+1 {
		if strings.IndexAny(args[*i+1], "0123456789") == 0 {
			*i++
			n, err := atoi(args[*i])
			if err != nil {
				return 0, err
			}
			return n, nil
		}
	}
	return defaultValue, nil
}

func splitNth(str string) ([]Range, error) {
	if match, _ := regexp.MatchString("^[0-9,-.]+$", str); !match {
		return nil, errors.New("invalid format: " + str)
	}

	tokens := strings.Split(str, ",")
	ranges := make([]Range, len(tokens))
	for idx, s := range tokens {
		r, ok := ParseRange(&s)
		if !ok {
			return nil, errors.New("invalid format: " + str)
		}
		ranges[idx] = r
	}
	return ranges, nil
}

func delimiterRegexp(str string) Delimiter {
	// Special handling of \t
	str = strings.ReplaceAll(str, "\\t", "\t")

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

func parseAlgo(str string) (algo.Algo, error) {
	switch str {
	case "v1":
		return algo.FuzzyMatchV1, nil
	case "v2":
		return algo.FuzzyMatchV2, nil
	}
	return nil, errors.New("invalid algorithm (expected: v1 or v2)")
}

func processScheme(opts *Options) error {
	if !algo.Init(opts.Scheme) {
		return errors.New("invalid scoring scheme (expected: default|path|history)")
	}
	if opts.Scheme == "history" {
		opts.Criteria = []criterion{byScore}
	}
	return nil
}

func parseBorder(str string, optional bool) (tui.BorderShape, error) {
	switch str {
	case "rounded":
		return tui.BorderRounded, nil
	case "sharp":
		return tui.BorderSharp, nil
	case "bold":
		return tui.BorderBold, nil
	case "block":
		return tui.BorderBlock, nil
	case "thinblock":
		return tui.BorderThinBlock, nil
	case "double":
		return tui.BorderDouble, nil
	case "horizontal":
		return tui.BorderHorizontal, nil
	case "vertical":
		return tui.BorderVertical, nil
	case "top":
		return tui.BorderTop, nil
	case "bottom":
		return tui.BorderBottom, nil
	case "left":
		return tui.BorderLeft, nil
	case "right":
		return tui.BorderRight, nil
	case "none":
		return tui.BorderNone, nil
	}
	if optional && str == "" {
		return tui.DefaultBorderShape, nil
	}
	return tui.BorderNone, errors.New("invalid border style (expected: rounded|sharp|bold|block|thinblock|double|horizontal|vertical|top|bottom|left|right|none)")
}

func parseKeyChords(str string, message string) (map[tui.Event]string, error) {
	return parseKeyChordsImpl(str, message)
}

func parseKeyChordsImpl(str string, message string) (map[tui.Event]string, error) {
	if len(str) == 0 {
		return nil, errors.New(message)
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
		case "backspace", "bspace", "bs":
			add(tui.Backspace)
		case "ctrl-space":
			add(tui.CtrlSpace)
		case "ctrl-delete":
			add(tui.CtrlDelete)
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
		case "start":
			add(tui.Start)
		case "load":
			add(tui.Load)
		case "focus":
			add(tui.Focus)
		case "result":
			add(tui.Result)
		case "resize":
			add(tui.Resize)
		case "one":
			add(tui.One)
		case "zero":
			add(tui.Zero)
		case "jump":
			add(tui.Jump)
		case "jump-cancel":
			add(tui.JumpCancel)
		case "click-header":
			add(tui.ClickHeader)
		case "alt-enter", "alt-return":
			chords[tui.CtrlAltKey('m')] = key
		case "alt-space":
			chords[tui.AltKey(' ')] = key
		case "alt-bs", "alt-bspace", "alt-backspace":
			add(tui.AltBackspace)
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
			add(tui.ShiftTab)
		case "esc":
			add(tui.Esc)
		case "delete", "del":
			add(tui.Delete)
		case "home":
			add(tui.Home)
		case "end":
			add(tui.End)
		case "insert":
			add(tui.Insert)
		case "pgup", "page-up":
			add(tui.PageUp)
		case "pgdn", "page-down":
			add(tui.PageDown)
		case "alt-shift-up", "shift-alt-up":
			add(tui.AltShiftUp)
		case "alt-shift-down", "shift-alt-down":
			add(tui.AltShiftDown)
		case "alt-shift-left", "shift-alt-left":
			add(tui.AltShiftLeft)
		case "alt-shift-right", "shift-alt-right":
			add(tui.AltShiftRight)
		case "shift-up":
			add(tui.ShiftUp)
		case "shift-down":
			add(tui.ShiftDown)
		case "shift-left":
			add(tui.ShiftLeft)
		case "shift-right":
			add(tui.ShiftRight)
		case "shift-delete":
			add(tui.ShiftDelete)
		case "left-click":
			add(tui.LeftClick)
		case "right-click":
			add(tui.RightClick)
		case "shift-left-click":
			add(tui.SLeftClick)
		case "shift-right-click":
			add(tui.SRightClick)
		case "double-click":
			add(tui.DoubleClick)
		case "scroll-up":
			add(tui.ScrollUp)
		case "scroll-down":
			add(tui.ScrollDown)
		case "shift-scroll-up":
			add(tui.SScrollUp)
		case "shift-scroll-down":
			add(tui.SScrollDown)
		case "preview-scroll-up":
			add(tui.PreviewScrollUp)
		case "preview-scroll-down":
			add(tui.PreviewScrollDown)
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
				return nil, errors.New("unsupported key: " + key)
			}
		}
	}
	return chords, nil
}

func parseTiebreak(str string) ([]criterion, error) {
	criteria := []criterion{byScore}
	hasIndex := false
	hasChunk := false
	hasLength := false
	hasBegin := false
	hasEnd := false
	check := func(notExpected *bool, name string) error {
		if *notExpected {
			return errors.New("duplicate sort criteria: " + name)
		}
		if hasIndex {
			return errors.New("index should be the last criterion")
		}
		*notExpected = true
		return nil
	}
	for _, str := range strings.Split(strings.ToLower(str), ",") {
		switch str {
		case "index":
			if err := check(&hasIndex, "index"); err != nil {
				return nil, err
			}
		case "chunk":
			if err := check(&hasChunk, "chunk"); err != nil {
				return nil, err
			}
			criteria = append(criteria, byChunk)
		case "length":
			if err := check(&hasLength, "length"); err != nil {
				return nil, err
			}
			criteria = append(criteria, byLength)
		case "begin":
			if err := check(&hasBegin, "begin"); err != nil {
				return nil, err
			}
			criteria = append(criteria, byBegin)
		case "end":
			if err := check(&hasEnd, "end"); err != nil {
				return nil, err
			}
			criteria = append(criteria, byEnd)
		default:
			return nil, errors.New("invalid sort criterion: " + str)
		}
	}
	if len(criteria) > 4 {
		return nil, errors.New("at most 3 tiebreaks are allowed: " + str)
	}
	return criteria, nil
}

func dupeTheme(theme *tui.ColorTheme) *tui.ColorTheme {
	dupe := *theme
	return &dupe
}

func parseTheme(defaultTheme *tui.ColorTheme, str string) (*tui.ColorTheme, error) {
	var err error
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
				// Let the code proceed to simplify the error handling
				err = errors.New("invalid color specification: " + str)
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
					case "strikethrough":
						cattr.Attr |= tui.StrikeThrough
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
			case "current-fg", "fg+":
				mergeAttr(&theme.Current)
			case "current-bg", "bg+":
				mergeAttr(&theme.DarkBg)
			case "selected-fg":
				mergeAttr(&theme.SelectedFg)
			case "selected-bg":
				mergeAttr(&theme.SelectedBg)
			case "gutter":
				mergeAttr(&theme.Gutter)
			case "hl":
				mergeAttr(&theme.Match)
			case "current-hl", "hl+":
				mergeAttr(&theme.CurrentMatch)
			case "selected-hl":
				mergeAttr(&theme.SelectedMatch)
			case "border":
				mergeAttr(&theme.Border)
			case "preview-border":
				mergeAttr(&theme.PreviewBorder)
			case "separator":
				mergeAttr(&theme.Separator)
			case "scrollbar":
				mergeAttr(&theme.Scrollbar)
			case "preview-scrollbar":
				mergeAttr(&theme.PreviewScrollbar)
			case "label":
				mergeAttr(&theme.BorderLabel)
			case "preview-label":
				mergeAttr(&theme.PreviewLabel)
			case "prompt":
				mergeAttr(&theme.Prompt)
			case "spinner":
				mergeAttr(&theme.Spinner)
			case "info":
				mergeAttr(&theme.Info)
			case "pointer":
				mergeAttr(&theme.Cursor)
			case "marker":
				mergeAttr(&theme.Marker)
			case "header":
				mergeAttr(&theme.Header)
			default:
				fail()
			}
		}
	}
	return theme, err
}

func parseWalkerOpts(str string) (walkerOpts, error) {
	opts := walkerOpts{}
	for _, str := range strings.Split(strings.ToLower(str), ",") {
		switch str {
		case "file":
			opts.file = true
		case "dir":
			opts.dir = true
		case "hidden":
			opts.hidden = true
		case "follow":
			opts.follow = true
		case "":
			// Ignored
		default:
			return opts, errors.New("invalid walker option: " + str)
		}
	}
	if !opts.file && !opts.dir {
		return opts, errors.New("at least one of 'file' or 'dir' should be specified")
	}
	return opts, nil
}

var (
	executeRegexp    *regexp.Regexp
	splitRegexp      *regexp.Regexp
	actionNameRegexp *regexp.Regexp
)

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
	executeRegexp = regexp.MustCompile(
		`(?si)[:+](become|execute(?:-multi|-silent)?|reload(?:-sync)?|preview|(?:change|transform)-(?:header|query|prompt|border-label|preview-label)|transform|change-(?:preview-window|preview|multi)|(?:re|un)bind|pos|put|print)`)
	splitRegexp = regexp.MustCompile("[,:]+")
	actionNameRegexp = regexp.MustCompile("(?i)^[a-z-]+")
}

func maskActionContents(action string) string {
	masked := ""
Loop:
	for len(action) > 0 {
		loc := executeRegexp.FindStringIndex(action)
		if loc == nil {
			masked += action
			break
		}
		masked += action[:loc[1]]
		action = action[loc[1]:]
		if len(action) == 0 {
			break
		}
		cs := string(action[0])
		var ce string
		switch action[0] {
		case ':':
			masked += strings.Repeat(" ", len(action))
			break Loop
		case '(':
			ce = ")"
		case '{':
			ce = "}"
		case '[':
			ce = "]"
		case '<':
			ce = ">"
		case '~', '!', '@', '#', '$', '%', '^', '&', '*', ';', '/', '|':
			ce = string(cs)
		default:
			continue
		}
		cs = regexp.QuoteMeta(cs)
		ce = regexp.QuoteMeta(ce)

		// @$ or @+
		loc = regexp.MustCompile(fmt.Sprintf(`(?s)^%s.*?(%s[+,]|%s$)`, cs, ce, ce)).FindStringIndex(action)
		if loc == nil {
			masked += action
			break
		}
		// Keep + or , at the end
		lastChar := action[loc[1]-1]
		if lastChar == '+' || lastChar == ',' {
			loc[1]--
		}
		masked += strings.Repeat(" ", loc[1])
		action = action[loc[1]:]
	}
	masked = strings.ReplaceAll(masked, "::", string([]rune{escapedColon, ':'}))
	masked = strings.ReplaceAll(masked, ",:", string([]rune{escapedComma, ':'}))
	masked = strings.ReplaceAll(masked, "+:", string([]rune{escapedPlus, ':'}))
	return masked
}

func parseSingleActionList(str string) ([]*action, error) {
	// We prepend a colon to satisfy executeRegexp and remove it later
	masked := maskActionContents(":" + str)[1:]
	return parseActionList(masked, str, []*action{}, false)
}

func parseActionList(masked string, original string, prevActions []*action, putAllowed bool) ([]*action, error) {
	maskedStrings := strings.Split(masked, "+")
	originalStrings := make([]string, len(maskedStrings))
	idx := 0
	for i, maskedString := range maskedStrings {
		originalStrings[i] = original[idx : idx+len(maskedString)]
		idx += len(maskedString) + 1
	}
	actions := make([]*action, 0, len(maskedStrings))
	appendAction := func(types ...actionType) {
		actions = append(actions, toActions(types...)...)
	}
	prevSpec := ""
	for specIndex, spec := range originalStrings {
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
		case "accept-or-print-query":
			appendAction(actAcceptOrPrintQuery)
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
			appendAction(actBackwardDeleteCharEof)
		case "backward-word":
			appendAction(actBackwardWord)
		case "clear-screen":
			appendAction(actClearScreen)
		case "delete-char":
			appendAction(actDeleteChar)
		case "delete-char/eof":
			appendAction(actDeleteCharEof)
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
		case "toggle-track":
			appendAction(actToggleTrack)
		case "toggle-track-current":
			appendAction(actToggleTrackCurrent)
		case "toggle-header":
			appendAction(actToggleHeader)
		case "toggle-wrap":
			appendAction(actToggleWrap)
		case "show-header":
			appendAction(actShowHeader)
		case "hide-header":
			appendAction(actHideHeader)
		case "track", "track-current":
			appendAction(actTrackCurrent)
		case "untrack-current":
			appendAction(actUntrackCurrent)
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
		case "prev-history", "previous-history":
			appendAction(actPrevHistory)
		case "next-history":
			appendAction(actNextHistory)
		case "prev-selected":
			appendAction(actPrevSelected)
		case "next-selected":
			appendAction(actNextSelected)
		case "show-preview":
			appendAction(actShowPreview)
		case "hide-preview":
			appendAction(actHidePreview)
		case "toggle-preview":
			appendAction(actTogglePreview)
		case "toggle-preview-wrap":
			appendAction(actTogglePreviewWrap)
		case "toggle-sort":
			appendAction(actToggleSort)
		case "offset-up":
			appendAction(actOffsetUp)
		case "offset-down":
			appendAction(actOffsetDown)
		case "offset-middle":
			appendAction(actOffsetMiddle)
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
			if putAllowed {
				appendAction(actChar)
			} else {
				return nil, errors.New("unable to put non-printable character")
			}
		default:
			t := isExecuteAction(specLower)
			if t == actIgnore {
				if specIndex == 0 && specLower == "" {
					actions = append(prevActions, actions...)
				} else if specLower == "change-multi" {
					appendAction(actChangeMulti)
				} else {
					return nil, errors.New("unknown action: " + spec)
				}
			} else {
				offset := len(actionNameRegexp.FindString(spec))
				var actionArg string
				if spec[offset] == ':' {
					if specIndex == len(originalStrings)-1 {
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
				switch t {
				case actUnbind, actRebind:
					if _, err := parseKeyChordsImpl(actionArg, spec[0:offset]+" target required"); err != nil {
						return nil, err
					}
				case actChangePreviewWindow:
					opts := previewOpts{}
					for _, arg := range strings.Split(actionArg, "|") {
						// Make sure that each expression is valid
						if err := parsePreviewWindowImpl(&opts, arg); err != nil {
							return nil, err
						}
					}
				}
			}
		}
		prevSpec = ""
	}
	return actions, nil
}

func parseKeymap(keymap map[tui.Event][]*action, str string) error {
	var err error
	masked := maskActionContents(str)
	idx := 0
	for _, pairStr := range strings.Split(masked, ",") {
		origPairStr := str[idx : idx+len(pairStr)]
		idx += len(pairStr) + 1

		pair := strings.SplitN(pairStr, ":", 2)
		if len(pair) < 2 {
			return errors.New("bind action not specified: " + origPairStr)
		}
		var key tui.Event
		if len(pair[0]) == 1 && pair[0][0] == escapedColon {
			key = tui.Key(':')
		} else if len(pair[0]) == 1 && pair[0][0] == escapedComma {
			key = tui.Key(',')
		} else if len(pair[0]) == 1 && pair[0][0] == escapedPlus {
			key = tui.Key('+')
		} else {
			keys, err := parseKeyChordsImpl(pair[0], "key name required")
			if err != nil {
				return err
			}
			key = firstKey(keys)
		}
		putAllowed := key.Type == tui.Rune && unicode.IsGraphic(key.Char)
		keymap[key], err = parseActionList(pair[1], origPairStr[len(pair[0])+1:], keymap[key], putAllowed)
		if err != nil {
			return err
		}
	}
	return nil
}

func isExecuteAction(str string) actionType {
	masked := maskActionContents(":" + str)[1:]
	if masked == str {
		// Not masked
		return actIgnore
	}

	prefix := actionNameRegexp.FindString(str)
	switch prefix {
	case "become":
		return actBecome
	case "reload":
		return actReload
	case "reload-sync":
		return actReloadSync
	case "unbind":
		return actUnbind
	case "rebind":
		return actRebind
	case "preview":
		return actPreview
	case "change-border-label":
		return actChangeBorderLabel
	case "change-header":
		return actChangeHeader
	case "change-preview-label":
		return actChangePreviewLabel
	case "change-preview-window":
		return actChangePreviewWindow
	case "change-preview":
		return actChangePreview
	case "change-prompt":
		return actChangePrompt
	case "change-query":
		return actChangeQuery
	case "change-multi":
		return actChangeMulti
	case "pos":
		return actPosition
	case "execute":
		return actExecute
	case "execute-silent":
		return actExecuteSilent
	case "execute-multi":
		return actExecuteMulti
	case "print":
		return actPrint
	case "put":
		return actPut
	case "transform":
		return actTransform
	case "transform-border-label":
		return actTransformBorderLabel
	case "transform-preview-label":
		return actTransformPreviewLabel
	case "transform-header":
		return actTransformHeader
	case "transform-prompt":
		return actTransformPrompt
	case "transform-query":
		return actTransformQuery
	}
	return actIgnore
}

func parseToggleSort(keymap map[tui.Event][]*action, str string) error {
	keys, err := parseKeyChords(str, "key name required")
	if err != nil {
		return err
	}
	if len(keys) != 1 {
		return errors.New("multiple keys specified")
	}
	keymap[firstKey(keys)] = toActions(actToggleSort)
	return nil
}

func strLines(str string) []string {
	return strings.Split(strings.TrimSuffix(str, "\n"), "\n")
}

func parseSize(str string, maxPercent float64, label string) (sizeSpec, error) {
	var spec = sizeSpec{}
	var val float64
	var err error
	percent := strings.HasSuffix(str, "%")
	if percent {
		if val, err = atof(str[:len(str)-1]); err != nil {
			return spec, err
		}

		if val < 0 {
			return spec, errors.New(label + " must be non-negative")
		}
		if val > maxPercent {
			return spec, fmt.Errorf("%s too large (max: %d%%)", label, int(maxPercent))
		}
	} else {
		if strings.Contains(str, ".") {
			return spec, errors.New(label + " (without %) must be a non-negative integer")
		}

		i, err := atoi(str)
		if err != nil {
			return spec, err
		}
		val = float64(i)
		if val < 0 {
			return spec, errors.New(label + " must be non-negative")
		}
	}
	return sizeSpec{val, percent}, nil
}

func parseHeight(str string, index int) (heightSpec, error) {
	heightSpec := heightSpec{index: index}
	if strings.HasPrefix(str, "~") {
		heightSpec.auto = true
		str = str[1:]
	}
	if strings.HasPrefix(str, "-") {
		if heightSpec.auto {
			return heightSpec, errors.New("negative(-) height is not compatible with adaptive(~) height")
		}
		heightSpec.inverse = true
		str = str[1:]
	}

	size, err := parseSize(str, 100, "height")
	if err != nil {
		return heightSpec, err
	}
	heightSpec.size = size.size
	heightSpec.percent = size.percent
	return heightSpec, nil
}

func parseLayout(str string) (layoutType, error) {
	switch str {
	case "default":
		return layoutDefault, nil
	case "reverse":
		return layoutReverse, nil
	case "reverse-list":
		return layoutReverseList, nil
	}
	return layoutDefault, errors.New("invalid layout (expected: default / reverse / reverse-list)")
}

func parseInfoStyle(str string) (infoStyle, string, error) {
	switch str {
	case "default":
		return infoDefault, "", nil
	case "right":
		return infoRight, "", nil
	case "inline":
		return infoInline, defaultInfoPrefix, nil
	case "inline-right":
		return infoInlineRight, "", nil
	case "hidden":
		return infoHidden, "", nil
	}
	type infoSpec struct {
		name  string
		style infoStyle
	}
	for _, spec := range []infoSpec{
		{"inline", infoInline},
		{"inline-right", infoInlineRight}} {
		if strings.HasPrefix(str, spec.name+":") {
			return spec.style, strings.ReplaceAll(str[len(spec.name)+1:], "\n", " "), nil
		}
	}
	return infoDefault, "", errors.New("invalid info style (expected: default|right|hidden|inline[-right][:PREFIX])")
}

func parsePreviewWindow(opts *previewOpts, input string) error {
	return parsePreviewWindowImpl(opts, input)
}

func parsePreviewWindowImpl(opts *previewOpts, input string) error {
	var err error
	tokenRegex := regexp.MustCompile(`[:,]*(<([1-9][0-9]*)\(([^)<]+)\)|[^,:]+)`)
	sizeRegex := regexp.MustCompile("^[0-9]+%?$")
	offsetRegex := regexp.MustCompile(`^(\+{-?[0-9]+})?([+-][0-9]+)*(-?/[1-9][0-9]*)?$`)
	headerRegex := regexp.MustCompile("^~(0|[1-9][0-9]*)$")
	tokens := tokenRegex.FindAllStringSubmatch(input, -1)
	var alternative string
	for _, match := range tokens {
		if len(match[2]) > 0 {
			if opts.threshold, err = atoi(match[2]); err != nil {
				return err
			}
			alternative = match[3]
			continue
		}
		token := match[1]
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
		case "border-bold":
			opts.border = tui.BorderBold
		case "border-block":
			opts.border = tui.BorderBlock
		case "border-thinblock":
			opts.border = tui.BorderThinBlock
		case "border-double":
			opts.border = tui.BorderDouble
		case "noborder", "border-none":
			opts.border = tui.BorderNone
		case "border-horizontal":
			opts.border = tui.BorderHorizontal
		case "border-vertical":
			opts.border = tui.BorderVertical
		case "border-up", "border-top":
			opts.border = tui.BorderTop
		case "border-down", "border-bottom":
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
				if opts.headerLines, err = atoi(token[1:]); err != nil {
					return err
				}
			} else if sizeRegex.MatchString(token) {
				if opts.size, err = parseSize(token, 99, "window size"); err != nil {
					return err
				}
			} else if offsetRegex.MatchString(token) {
				opts.scroll = token
			} else {
				return errors.New("invalid preview window option: " + token)
			}
		}
	}
	if len(alternative) > 0 {
		alternativeOpts := *opts
		opts.alternative = &alternativeOpts
		opts.alternative.hidden = false
		opts.alternative.alternative = nil
		err = parsePreviewWindowImpl(opts.alternative, alternative)
	}
	return err
}

func parseMargin(opt string, margin string) ([4]sizeSpec, error) {
	margins := strings.Split(margin, ",")
	checked := func(str string) (sizeSpec, error) {
		return parseSize(str, 49, opt)
	}
	switch len(margins) {
	case 1:
		m, e := checked(margins[0])
		return [4]sizeSpec{m, m, m, m}, e
	case 2:
		tb, e := checked(margins[0])
		if e != nil {
			return defaultMargin(), e
		}
		rl, e := checked(margins[1])
		if e != nil {
			return defaultMargin(), e
		}
		return [4]sizeSpec{tb, rl, tb, rl}, nil
	case 3:
		t, e := checked(margins[0])
		if e != nil {
			return defaultMargin(), e
		}
		rl, e := checked(margins[1])
		if e != nil {
			return defaultMargin(), e
		}
		b, e := checked(margins[2])
		if e != nil {
			return defaultMargin(), e
		}
		return [4]sizeSpec{t, rl, b, rl}, nil
	case 4:
		t, e := checked(margins[0])
		if e != nil {
			return defaultMargin(), e
		}
		r, e := checked(margins[1])
		if e != nil {
			return defaultMargin(), e
		}
		b, e := checked(margins[2])
		if e != nil {
			return defaultMargin(), e
		}
		l, e := checked(margins[3])
		if e != nil {
			return defaultMargin(), e
		}
		return [4]sizeSpec{t, r, b, l}, nil
	}
	return [4]sizeSpec{}, errors.New("invalid " + opt + ": " + margin)
}

func parseMarkerMultiLine(str string) (*[3]string, error) {
	if str == "" {
		return &[3]string{}, nil
	}
	gr := uniseg.NewGraphemes(str)
	parts := []string{}
	totalWidth := 0
	for gr.Next() {
		s := string(gr.Runes())
		totalWidth += uniseg.StringWidth(s)
		parts = append(parts, s)
	}

	result := [3]string{}
	if totalWidth != 3 && totalWidth != 6 {
		return &result, fmt.Errorf("invalid total marker width: %d (expected: 0, 3 or 6)", totalWidth)
	}

	expected := totalWidth / 3
	idx := 0
	for _, part := range parts {
		expected -= uniseg.StringWidth(part)
		result[idx] += part
		if expected <= 0 {
			idx++
			expected = totalWidth / 3
		}
		if idx == 3 {
			break
		}
	}
	return &result, nil
}

func parseOptions(index *int, opts *Options, allArgs []string) error {
	var err error
	var historyMax int
	if opts.History == nil {
		historyMax = defaultHistoryMax
	} else {
		historyMax = opts.History.maxSize
	}
	setHistory := func(path string) error {
		h, e := NewHistory(path, historyMax)
		if e != nil {
			return e
		}
		opts.History = h
		return nil
	}
	setHistoryMax := func(max int) error {
		historyMax = max
		if historyMax < 1 {
			return errors.New("history max must be a positive integer")
		}
		if opts.History != nil {
			opts.History.maxSize = historyMax
		}
		return nil
	}
	validateJumpLabels := false
	clearExitingOpts := func() {
		// Last-one-wins strategy
		opts.Bash = false
		opts.Zsh = false
		opts.Fish = false
		opts.Help = false
		opts.Version = false
		opts.Man = false
	}
	startIndex := *index
	for i := 0; i < len(allArgs); i++ {
		arg := allArgs[i]
		index := i + startIndex
		switch arg {
		case "--man":
			clearExitingOpts()
			opts.Man = true
		case "--bash":
			clearExitingOpts()
			opts.Bash = true
		case "--zsh":
			clearExitingOpts()
			opts.Zsh = true
		case "--fish":
			clearExitingOpts()
			opts.Fish = true
		case "-h", "--help":
			clearExitingOpts()
			opts.Help = true
		case "--version":
			clearExitingOpts()
			opts.Version = true
		case "--no-winpty":
			opts.NoWinpty = true
		case "--tmux":
			given, str := optionalNextString(allArgs, &i)
			if given {
				if opts.Tmux, err = parseTmuxOptions(str, index); err != nil {
					return err
				}
			} else {
				opts.Tmux = defaultTmuxOptions(index)
			}
		case "--no-tmux":
			opts.Tmux = nil
		case "--force-tty-in":
			// NOTE: We need this because `system('fzf --tmux < /dev/tty')` doesn't
			// work on Neovim. Same as '-' option of fzf-tmux.
			opts.ForceTtyIn = true
		case "--no-force-tty-in":
			opts.ForceTtyIn = false
		case "--proxy-script":
			if opts.ProxyScript, err = nextString(allArgs, &i, ""); err != nil {
				return err
			}
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
			if opts.Query, err = nextString(allArgs, &i, "query string required"); err != nil {
				return err
			}
		case "-f", "--filter":
			filter, err := nextString(allArgs, &i, "query string required")
			if err != nil {
				return err
			}
			opts.Filter = &filter
		case "--literal":
			opts.Normalize = false
		case "--no-literal":
			opts.Normalize = true
		case "--algo":
			str, err := nextString(allArgs, &i, "algorithm required (v1|v2)")
			if err != nil {
				return err
			}
			if opts.FuzzyAlgo, err = parseAlgo(str); err != nil {
				return err
			}
		case "--scheme":
			str, err := nextString(allArgs, &i, "scoring scheme required (default|path|history)")
			if err != nil {
				return err
			}
			opts.Scheme = strings.ToLower(str)
		case "--expect":
			str, err := nextString(allArgs, &i, "key names required")
			if err != nil {
				return err
			}
			chords, err := parseKeyChords(str, "key names required")
			if err != nil {
				return err
			}
			for k, v := range chords {
				opts.Expect[k] = v
			}
		case "--no-expect":
			opts.Expect = make(map[tui.Event]string)
		case "--enabled", "--no-phony":
			opts.Phony = false
		case "--disabled", "--phony":
			opts.Phony = true
		case "--tiebreak":
			str, err := nextString(allArgs, &i, "sort criterion required")
			if err != nil {
				return err
			}
			if opts.Criteria, err = parseTiebreak(str); err != nil {
				return err
			}
		case "--bind":
			str, err := nextString(allArgs, &i, "bind expression required")
			if err != nil {
				return err
			}
			if err := parseKeymap(opts.Keymap, str); err != nil {
				return err
			}
		case "--color":
			_, spec := optionalNextString(allArgs, &i)
			if len(spec) == 0 {
				opts.Theme = tui.EmptyTheme()
			} else {
				if opts.Theme, err = parseTheme(opts.Theme, spec); err != nil {
					return err
				}
			}
		case "--toggle-sort":
			str, err := nextString(allArgs, &i, "key name required")
			if err != nil {
				return err
			}
			if err := parseToggleSort(opts.Keymap, str); err != nil {
				return err
			}
		case "-d", "--delimiter":
			str, err := nextString(allArgs, &i, "delimiter required")
			if err != nil {
				return err
			}
			opts.Delimiter = delimiterRegexp(str)
		case "-n", "--nth":
			str, err := nextString(allArgs, &i, "nth expression required")
			if err != nil {
				return err
			}
			if opts.Nth, err = splitNth(str); err != nil {
				return err
			}
		case "--with-nth":
			str, err := nextString(allArgs, &i, "nth expression required")
			if err != nil {
				return err
			}
			if opts.WithNth, err = splitNth(str); err != nil {
				return err
			}
		case "-s", "--sort":
			if opts.Sort, err = optionalNumeric(allArgs, &i, 1); err != nil {
				return err
			}
		case "+s", "--no-sort":
			opts.Sort = 0
		case "--track":
			opts.Track = trackEnabled
		case "--no-track":
			opts.Track = trackDisabled
		case "--tac":
			opts.Tac = true
		case "--no-tac":
			opts.Tac = false
		case "--tail":
			if opts.Tail, err = nextInt(allArgs, &i, "number of items to keep required"); err != nil {
				return err
			}
			if opts.Tail <= 0 {
				return errors.New("number of items to keep must be a positive integer")
			}
		case "--no-tail":
			opts.Tail = 0
		case "-i", "--ignore-case":
			opts.Case = CaseIgnore
		case "+i", "--no-ignore-case":
			opts.Case = CaseRespect
		case "-m", "--multi":
			if opts.Multi, err = optionalNumeric(allArgs, &i, maxMulti); err != nil {
				return err
			}
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
			str, err := nextString(allArgs, &i, "layout required (default / reverse / reverse-list)")
			if err != nil {
				return err
			}
			if opts.Layout, err = parseLayout(str); err != nil {
				return err
			}
		case "--reverse":
			opts.Layout = layoutReverse
		case "--no-reverse":
			opts.Layout = layoutDefault
		case "--cycle":
			opts.Cycle = true
		case "--highlight-line":
			opts.CursorLine = true
		case "--no-highlight-line":
			opts.CursorLine = false
		case "--no-cycle":
			opts.Cycle = false
		case "--wrap":
			opts.Wrap = true
		case "--no-wrap":
			opts.Wrap = false
		case "--wrap-sign":
			str, err := nextString(allArgs, &i, "wrap sign required")
			if err != nil {
				return err
			}
			opts.WrapSign = &str
		case "--multi-line":
			opts.MultiLine = true
		case "--no-multi-line":
			opts.MultiLine = false
		case "--keep-right":
			opts.KeepRight = true
		case "--no-keep-right":
			opts.KeepRight = false
		case "--hscroll":
			opts.Hscroll = true
		case "--no-hscroll":
			opts.Hscroll = false
		case "--hscroll-off":
			if opts.HscrollOff, err = nextInt(allArgs, &i, "hscroll offset required"); err != nil {
				return err
			}
		case "--scroll-off":
			if opts.ScrollOff, err = nextInt(allArgs, &i, "scroll offset required"); err != nil {
				return err
			}
		case "--filepath-word":
			opts.FileWord = true
		case "--no-filepath-word":
			opts.FileWord = false
		case "--info":
			str, err := nextString(allArgs, &i, "info style required")
			if err != nil {
				return err
			}
			if opts.InfoStyle, opts.InfoPrefix, err = parseInfoStyle(str); err != nil {
				return err
			}
		case "--info-command":
			if opts.InfoCommand, err = nextString(allArgs, &i, "info command required"); err != nil {
				return err
			}
		case "--no-info-command":
			opts.InfoCommand = ""
		case "--no-info":
			opts.InfoStyle = infoHidden
		case "--inline-info":
			opts.InfoStyle = infoInline
			opts.InfoPrefix = defaultInfoPrefix
		case "--no-inline-info":
			opts.InfoStyle = infoDefault
		case "--separator":
			separator, err := nextString(allArgs, &i, "separator character required")
			if err != nil {
				return err
			}
			opts.Separator = &separator
		case "--no-separator":
			nosep := ""
			opts.Separator = &nosep
		case "--scrollbar":
			given, bar := optionalNextString(allArgs, &i)
			if given {
				opts.Scrollbar = &bar
			} else {
				opts.Scrollbar = nil
			}
		case "--no-scrollbar":
			noBar := ""
			opts.Scrollbar = &noBar
		case "--jump-labels":
			if opts.JumpLabels, err = nextString(allArgs, &i, "label characters required"); err != nil {
				return err
			}
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
			opts.Prompt, err = nextString(allArgs, &i, "prompt string required")
			if err != nil {
				return err
			}
		case "--pointer":
			str, err := nextString(allArgs, &i, "pointer sign required")
			if err != nil {
				return err
			}
			str = firstLine(str)
			opts.Pointer = &str
		case "--marker":
			str, err := nextString(allArgs, &i, "marker sign required")
			if err != nil {
				return err
			}
			str = firstLine(str)
			opts.Marker = &str
		case "--marker-multi-line":
			str, err := nextString(allArgs, &i, "marker sign for multi-line entries required")
			if err != nil {
				return err
			}
			if opts.MarkerMulti, err = parseMarkerMultiLine(firstLine(str)); err != nil {
				return err
			}
		case "--sync":
			opts.Sync = true
		case "--no-sync", "--async":
			opts.Sync = false
		case "--no-history":
			opts.History = nil
		case "--history":
			str, err := nextString(allArgs, &i, "history file path required")
			if err != nil {
				return err
			}
			if err := setHistory(str); err != nil {
				return err
			}
		case "--history-size":
			n, err := nextInt(allArgs, &i, "history max size required")
			if err != nil {
				return err
			}
			if err := setHistoryMax(n); err != nil {
				return err
			}
		case "--no-header":
			opts.Header = []string{}
		case "--no-header-lines":
			opts.HeaderLines = 0
		case "--header":
			str, err := nextString(allArgs, &i, "header string required")
			if err != nil {
				return err
			}
			opts.Header = strLines(str)
		case "--header-lines":
			if opts.HeaderLines, err = nextInt(allArgs, &i, "number of header lines required"); err != nil {
				return err
			}
		case "--header-first":
			opts.HeaderFirst = true
		case "--no-header-first":
			opts.HeaderFirst = false
		case "--ellipsis":
			str, err := nextString(allArgs, &i, "ellipsis string required")
			if err != nil {
				return err
			}
			str = firstLine(str)
			opts.Ellipsis = &str
		case "--preview":
			if opts.Preview.command, err = nextString(allArgs, &i, "preview command required"); err != nil {
				return err
			}
		case "--no-preview":
			opts.Preview.command = ""
		case "--preview-window":
			str, err := nextString(allArgs, &i, "preview window layout required: [up|down|left|right][,SIZE[%]][,border-BORDER_OPT][,wrap][,cycle][,hidden][,+SCROLL[OFFSETS][/DENOM]][,~HEADER_LINES][,default]")
			if err != nil {
				return err
			}
			if err := parsePreviewWindow(&opts.Preview, str); err != nil {
				return err
			}
		case "--height":
			str, err := nextString(allArgs, &i, "height required: [~]HEIGHT[%]")
			if err != nil {
				return err
			}
			if opts.Height, err = parseHeight(str, index); err != nil {
				return err
			}
		case "--min-height":
			if opts.MinHeight, err = nextInt(allArgs, &i, "height required: HEIGHT"); err != nil {
				return err
			}
		case "--no-height":
			opts.Height = heightSpec{}
		case "--no-margin":
			opts.Margin = defaultMargin()
		case "--no-padding":
			opts.Padding = defaultMargin()
		case "--no-border":
			opts.BorderShape = tui.BorderNone
		case "--border":
			hasArg, arg := optionalNextString(allArgs, &i)
			if opts.BorderShape, err = parseBorder(arg, !hasArg); err != nil {
				return err
			}
		case "--no-border-label":
			opts.BorderLabel.label = ""
		case "--border-label":
			opts.BorderLabel.label, err = nextString(allArgs, &i, "label required")
			if err != nil {
				return err
			}
		case "--border-label-pos":
			pos, err := nextString(allArgs, &i, "label position required (positive or negative integer or 'center')")
			if err != nil {
				return err
			}
			if err := parseLabelPosition(&opts.BorderLabel, pos); err != nil {
				return err
			}
		case "--no-preview-label":
			opts.PreviewLabel.label = ""
		case "--preview-label":
			if opts.PreviewLabel.label, err = nextString(allArgs, &i, "preview label required"); err != nil {
				return err
			}
		case "--preview-label-pos":
			pos, err := nextString(allArgs, &i, "preview label position required (positive or negative integer or 'center')")
			if err != nil {
				return err
			}
			if err := parseLabelPosition(&opts.PreviewLabel, pos); err != nil {
				return err
			}
		case "--no-unicode":
			opts.Unicode = false
		case "--unicode":
			opts.Unicode = true
		case "--ambidouble":
			opts.Ambidouble = true
		case "--no-ambidouble":
			opts.Ambidouble = false
		case "--margin":
			str, err := nextString(allArgs, &i, "margin required (TRBL / TB,RL / T,RL,B / T,R,B,L)")
			if err != nil {
				return err
			}
			if opts.Margin, err = parseMargin("margin", str); err != nil {
				return err
			}
		case "--padding":
			str, err := nextString(allArgs, &i, "padding required (TRBL / TB,RL / T,RL,B / T,R,B,L)")
			if err != nil {
				return err
			}
			if opts.Padding, err = parseMargin("padding", str); err != nil {
				return err
			}
		case "--tabstop":
			if opts.Tabstop, err = nextInt(allArgs, &i, "tab stop required"); err != nil {
				return err
			}
		case "--with-shell":
			if opts.WithShell, err = nextString(allArgs, &i, "shell command and flags required"); err != nil {
				return err
			}
		case "--listen", "--listen-unsafe":
			given, str := optionalNextString(allArgs, &i)
			addr := defaultListenAddr
			if given {
				var err error
				addr, err = parseListenAddress(str)
				if err != nil {
					return err
				}
			}
			opts.ListenAddr = &addr
			opts.Unsafe = arg == "--listen-unsafe"
		case "--no-listen", "--no-listen-unsafe":
			opts.ListenAddr = nil
			opts.Unsafe = false
		case "--clear":
			opts.ClearOnExit = true
		case "--no-clear":
			opts.ClearOnExit = false
		case "--walker":
			str, err := nextString(allArgs, &i, "walker options required [file][,dir][,follow][,hidden]")
			if err != nil {
				return err
			}
			if opts.WalkerOpts, err = parseWalkerOpts(str); err != nil {
				return err
			}
		case "--walker-root":
			if opts.WalkerRoot, err = nextString(allArgs, &i, "directory required"); err != nil {
				return err
			}
		case "--walker-skip":
			str, err := nextString(allArgs, &i, "directory names to ignore required")
			if err != nil {
				return err
			}
			opts.WalkerSkip = filterNonEmpty(strings.Split(str, ","))
		case "--profile-cpu":
			if opts.CPUProfile, err = nextString(allArgs, &i, "file path required: cpu"); err != nil {
				return err
			}
		case "--profile-mem":
			if opts.MEMProfile, err = nextString(allArgs, &i, "file path required: mem"); err != nil {
				return err
			}
		case "--profile-block":
			if opts.BlockProfile, err = nextString(allArgs, &i, "file path required: block"); err != nil {
				return err
			}
		case "--profile-mutex":
			if opts.MutexProfile, err = nextString(allArgs, &i, "file path required: mutex"); err != nil {
				return err
			}
		case "--":
			// Ignored
		default:
			if match, value := optString(arg, "--algo="); match {
				if opts.FuzzyAlgo, err = parseAlgo(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--tmux="); match {
				if opts.Tmux, err = parseTmuxOptions(value, index); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--scheme="); match {
				opts.Scheme = strings.ToLower(value)
			} else if match, value := optString(arg, "-q", "--query="); match {
				opts.Query = value
			} else if match, value := optString(arg, "-f", "--filter="); match {
				opts.Filter = &value
			} else if match, value := optString(arg, "-d", "--delimiter="); match {
				opts.Delimiter = delimiterRegexp(value)
			} else if match, value := optString(arg, "--border="); match {
				if opts.BorderShape, err = parseBorder(value, false); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--border-label="); match {
				opts.BorderLabel.label = value
			} else if match, value := optString(arg, "--border-label-pos="); match {
				if err := parseLabelPosition(&opts.BorderLabel, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--preview-label="); match {
				opts.PreviewLabel.label = value
			} else if match, value := optString(arg, "--preview-label-pos="); match {
				if err := parseLabelPosition(&opts.PreviewLabel, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--wrap-sign="); match {
				opts.WrapSign = &value
			} else if match, value := optString(arg, "--prompt="); match {
				opts.Prompt = value
			} else if match, value := optString(arg, "--pointer="); match {
				str := firstLine(value)
				opts.Pointer = &str
			} else if match, value := optString(arg, "--marker="); match {
				str := firstLine(value)
				opts.Marker = &str
			} else if match, value := optString(arg, "--marker-multi-line="); match {
				if opts.MarkerMulti, err = parseMarkerMultiLine(firstLine(value)); err != nil {
					return err
				}
			} else if match, value := optString(arg, "-n", "--nth="); match {
				if opts.Nth, err = splitNth(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--with-nth="); match {
				if opts.WithNth, err = splitNth(value); err != nil {
					return err
				}
			} else if match, _ := optString(arg, "-s", "--sort="); match {
				opts.Sort = 1 // Don't care
			} else if match, value := optString(arg, "-m", "--multi="); match {
				if opts.Multi, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--height="); match {
				if opts.Height, err = parseHeight(value, index); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--min-height="); match {
				if opts.MinHeight, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--layout="); match {
				if opts.Layout, err = parseLayout(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--info="); match {
				if opts.InfoStyle, opts.InfoPrefix, err = parseInfoStyle(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--info-command="); match {
				opts.InfoCommand = value
			} else if match, value := optString(arg, "--separator="); match {
				opts.Separator = &value
			} else if match, value := optString(arg, "--scrollbar="); match {
				opts.Scrollbar = &value
			} else if match, value := optString(arg, "--toggle-sort="); match {
				if err := parseToggleSort(opts.Keymap, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--expect="); match {
				chords, err := parseKeyChords(value, "key names required")
				if err != nil {
					return err
				}
				for k, v := range chords {
					opts.Expect[k] = v
				}
			} else if match, value := optString(arg, "--tiebreak="); match {
				if opts.Criteria, err = parseTiebreak(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--color="); match {
				if opts.Theme, err = parseTheme(opts.Theme, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--bind="); match {
				if err := parseKeymap(opts.Keymap, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--history="); match {
				if err := setHistory(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--history-size="); match {
				n, err := atoi(value)
				if err != nil {
					return err
				}
				if err := setHistoryMax(n); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--header="); match {
				opts.Header = strLines(value)
			} else if match, value := optString(arg, "--header-lines="); match {
				if opts.HeaderLines, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--ellipsis="); match {
				str := firstLine(value)
				opts.Ellipsis = &str
			} else if match, value := optString(arg, "--preview="); match {
				opts.Preview.command = value
			} else if match, value := optString(arg, "--preview-window="); match {
				if err := parsePreviewWindow(&opts.Preview, value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--margin="); match {
				if opts.Margin, err = parseMargin("margin", value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--padding="); match {
				if opts.Padding, err = parseMargin("padding", value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--tabstop="); match {
				if opts.Tabstop, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--with-shell="); match {
				opts.WithShell = value
			} else if match, value := optString(arg, "--listen="); match {
				addr, err := parseListenAddress(value)
				if err != nil {
					return err
				}
				opts.ListenAddr = &addr
				opts.Unsafe = false
			} else if match, value := optString(arg, "--listen-unsafe="); match {
				addr, err := parseListenAddress(value)
				if err != nil {
					return err
				}
				opts.ListenAddr = &addr
				opts.Unsafe = true
			} else if match, value := optString(arg, "--walker="); match {
				if opts.WalkerOpts, err = parseWalkerOpts(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--walker-root="); match {
				opts.WalkerRoot = value
			} else if match, value := optString(arg, "--walker-skip="); match {
				opts.WalkerSkip = filterNonEmpty(strings.Split(value, ","))
			} else if match, value := optString(arg, "--hscroll-off="); match {
				if opts.HscrollOff, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--scroll-off="); match {
				if opts.ScrollOff, err = atoi(value); err != nil {
					return err
				}
			} else if match, value := optString(arg, "--jump-labels="); match {
				opts.JumpLabels = value
				validateJumpLabels = true
			} else if match, value := optString(arg, "--tail="); match {
				if opts.Tail, err = atoi(value); err != nil {
					return err
				}
				if opts.Tail <= 0 {
					return errors.New("number of items to keep must be a positive integer")
				}
			} else {
				return errors.New("unknown option: " + arg)
			}
		}
	}
	*index += len(allArgs)

	if opts.HeaderLines < 0 {
		return errors.New("header lines must be a non-negative integer")
	}

	if opts.HscrollOff < 0 {
		return errors.New("hscroll offset must be a non-negative integer")
	}

	if opts.ScrollOff < 0 {
		return errors.New("scroll offset must be a non-negative integer")
	}

	if opts.Tabstop < 1 {
		return errors.New("tab stop must be a positive integer")
	}

	if len(opts.JumpLabels) == 0 {
		return errors.New("empty jump labels")
	}

	if validateJumpLabels {
		for _, r := range opts.JumpLabels {
			if r < 32 || r > 126 {
				return errors.New("non-ascii jump labels are not allowed")
			}
		}
	}
	return err
}

func validateSign(sign string, signOptName string) error {
	if uniseg.StringWidth(sign) > 2 {
		return fmt.Errorf("%v display width should be up to 2", signOptName)
	}
	return nil
}

func validateOptions(opts *Options) error {
	if opts.Pointer != nil {
		if err := validateSign(*opts.Pointer, "pointer"); err != nil {
			return err
		}
	}

	if opts.Marker != nil {
		if err := validateSign(*opts.Marker, "marker"); err != nil {
			return err
		}
	}

	if opts.Scrollbar != nil {
		runes := []rune(*opts.Scrollbar)
		if len(runes) > 2 {
			return errors.New("--scrollbar should be given one or two characters")
		}
		for _, r := range runes {
			if uniseg.StringWidth(string(r)) != 1 {
				return errors.New("scrollbar display width should be 1")
			}
		}
	}

	if opts.Height.auto {
		for _, s := range []sizeSpec{opts.Margin[0], opts.Margin[2]} {
			if s.percent {
				return errors.New("adaptive height is not compatible with top/bottom percent margin")
			}
		}
		for _, s := range []sizeSpec{opts.Padding[0], opts.Padding[2]} {
			if s.percent {
				return errors.New("adaptive height is not compatible with top/bottom percent padding")
			}
		}
	}

	return nil
}

// This function can have side-effects and alter some global states.
// So we run it on fzf.Run and not on ParseOptions.
func postProcessOptions(opts *Options) error {
	if opts.Ambidouble {
		uniseg.EastAsianAmbiguousWidth = 2
	}

	if opts.BorderShape == tui.BorderUndefined {
		opts.BorderShape = tui.BorderNone
	}

	if opts.Pointer == nil {
		defaultPointer := "▌"
		if !opts.Unicode {
			defaultPointer = ">"
		}
		opts.Pointer = &defaultPointer
	}

	markerLen := 1
	if opts.Marker == nil {
		if opts.MarkerMulti != nil && opts.MarkerMulti[0] == "" {
			empty := ""
			opts.Marker = &empty
			markerLen = 0
		} else {
			// "▎" looks better, but not all terminals render it correctly
			defaultMarker := "┃"
			if !opts.Unicode {
				defaultMarker = ">"
			}
			opts.Marker = &defaultMarker
		}
	} else {
		markerLen = uniseg.StringWidth(*opts.Marker)
	}

	markerMultiLen := 1
	if opts.MarkerMulti == nil {
		if *opts.Marker == "" {
			opts.MarkerMulti = &[3]string{}
			markerMultiLen = 0
		} else if opts.Unicode {
			opts.MarkerMulti = &[3]string{"╻", "┃", "╹"}
		} else {
			opts.MarkerMulti = &[3]string{".", "|", "'"}
		}
	} else {
		markerMultiLen = uniseg.StringWidth(opts.MarkerMulti[0])
	}
	diff := markerMultiLen - markerLen
	if diff > 0 {
		padded := *opts.Marker + strings.Repeat(" ", diff)
		opts.Marker = &padded
	} else if diff < 0 {
		for idx := range opts.MarkerMulti {
			opts.MarkerMulti[idx] += strings.Repeat(" ", -diff)
		}
	}

	// Default actions for CTRL-N / CTRL-P when --history is set
	if opts.History != nil {
		if _, prs := opts.Keymap[tui.CtrlP.AsEvent()]; !prs {
			opts.Keymap[tui.CtrlP.AsEvent()] = toActions(actPrevHistory)
		}
		if _, prs := opts.Keymap[tui.CtrlN.AsEvent()]; !prs {
			opts.Keymap[tui.CtrlN.AsEvent()] = toActions(actNextHistory)
		}
	}

	// Extend the default key map
	keymap := defaultKeymap()
	for key, actions := range opts.Keymap {
		reordered := []*action{}
		for _, act := range actions {
			switch act.t {
			case actToggleSort:
				// To display "+S"/"-S" on info line
				opts.ToggleSort = true
			case actTogglePreview, actShowPreview, actHidePreview, actChangePreviewWindow:
				reordered = append(reordered, act)
			}
		}

		// Re-organize actions so that we put actions that change the preview window first in the list.
		//  *  change-preview-window(up,+10)+preview(sleep 3; cat {})+change-preview-window(up,+20)
		//  -> change-preview-window(up,+10)+change-preview-window(up,+20)+preview(sleep 3; cat {})
		if len(reordered) > 0 {
			for _, act := range actions {
				switch act.t {
				case actTogglePreview, actShowPreview, actHidePreview, actChangePreviewWindow:
				default:
					reordered = append(reordered, act)
				}
			}
			actions = reordered
		}
		keymap[key] = actions
	}
	opts.Keymap = keymap

	// If 'double-click' is left unbound, bind it to the action bound to 'enter'
	if _, prs := opts.Keymap[tui.DoubleClick.AsEvent()]; !prs {
		opts.Keymap[tui.DoubleClick.AsEvent()] = opts.Keymap[tui.CtrlM.AsEvent()]
	}

	// If we're not using extended search mode, --nth option becomes irrelevant
	// if it contains the whole range
	if !opts.Extended || len(opts.Nth) == 1 {
		for _, r := range opts.Nth {
			if r.begin == rangeEllipsis && r.end == rangeEllipsis {
				opts.Nth = make([]Range, 0)
				break
			}
		}
	}

	if opts.Bold {
		theme := opts.Theme
		boldify := func(c tui.ColorAttr) tui.ColorAttr {
			dup := c
			if (c.Attr & tui.AttrRegular) == 0 {
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

	// If --height option is not supported on the platform, just ignore it
	if !tui.IsLightRendererSupported() && opts.Height.size > 0 {
		opts.Height = heightSpec{}
	}

	if err := opts.initProfiling(); err != nil {
		return errors.New("failed to start pprof profiles: " + err.Error())
	}

	return processScheme(opts)
}

func parseShellWords(str string) ([]string, error) {
	parser := shellwords.NewParser()
	parser.ParseComment = true
	return parser.Parse(str)
}

// ParseOptions parses command-line options
func ParseOptions(useDefaults bool, args []string) (*Options, error) {
	opts := defaultOptions()
	index := 0

	if useDefaults {
		// 1. Options from $FZF_DEFAULT_OPTS_FILE
		if path := os.Getenv("FZF_DEFAULT_OPTS_FILE"); path != "" {
			bytes, err := os.ReadFile(path)
			if err != nil {
				return nil, errors.New("$FZF_DEFAULT_OPTS_FILE: " + err.Error())
			}

			words, parseErr := parseShellWords(string(bytes))
			if parseErr != nil {
				return nil, errors.New(path + ": " + parseErr.Error())
			}
			if len(words) > 0 {
				if err := parseOptions(&index, opts, words); err != nil {
					return nil, errors.New(path + ": " + err.Error())
				}
			}
		}

		// 2. Options from $FZF_DEFAULT_OPTS string
		words, parseErr := parseShellWords(os.Getenv("FZF_DEFAULT_OPTS"))
		if parseErr != nil {
			return nil, errors.New("$FZF_DEFAULT_OPTS: " + parseErr.Error())
		}
		if len(words) > 0 {
			if err := parseOptions(&index, opts, words); err != nil {
				return nil, errors.New("$FZF_DEFAULT_OPTS: " + err.Error())
			}
		}
	}

	// 3. Options from command-line arguments
	if err := parseOptions(&index, opts, args); err != nil {
		return nil, err
	}

	// 4. Final validation of merged options
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	return opts, nil
}

func (opts *Options) extractReloadOnStart() string {
	cmd := ""
	if actions, prs := opts.Keymap[tui.Start.AsEvent()]; prs {
		filtered := []*action{}
		for _, action := range actions {
			if action.t == actReload || action.t == actReloadSync {
				cmd = action.a
			} else {
				filtered = append(filtered, action)
			}
		}
		opts.Keymap[tui.Start.AsEvent()] = filtered
	}
	return cmd
}
