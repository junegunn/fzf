package fzf

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

// import "github.com/pkg/profile"

/*
Placeholder regex is used to extract placeholders from fzf's template
strings. Acts as input validation for parsePlaceholder function.
Describes the syntax, but it is fairly lenient.

The following pseudo regex has been reverse engineered from the
implementation. It is overly strict, but better describes what's possible.
As such it is not useful for validation, but rather to generate test
cases for example.

	\\?(?:                                      # escaped type
	    {\+?s?f?RANGE(?:,RANGE)*}               # token type
	    |{q}                                    # query type
	    |{\+?n?f?}                              # item type (notice no mandatory element inside brackets)
	)
	RANGE = (?:
	    (?:-?[0-9]+)?\.\.(?:-?[0-9]+)?          # ellipsis syntax for token range (x..y)
	    |-?[0-9]+                               # shorthand syntax (x..x)
	)
*/
var placeholder *regexp.Regexp
var whiteSuffix *regexp.Regexp
var offsetComponentRegex *regexp.Regexp
var offsetTrimCharsRegex *regexp.Regexp
var activeTempFiles []string

const clearCode string = "\x1b[2J"

func init() {
	placeholder = regexp.MustCompile(`\\?(?:{[+sf]*[0-9,-.]*}|{q}|{\+?f?nf?})`)
	whiteSuffix = regexp.MustCompile(`\s*$`)
	offsetComponentRegex = regexp.MustCompile(`([+-][0-9]+)|(-?/[1-9][0-9]*)`)
	offsetTrimCharsRegex = regexp.MustCompile(`[^0-9/+-]`)
	activeTempFiles = []string{}
}

type jumpMode int

const (
	jumpDisabled jumpMode = iota
	jumpEnabled
	jumpAcceptEnabled
)

type resumableState int

const (
	disabledState resumableState = iota
	pausedState
	enabledState
)

func (s resumableState) Enabled() bool {
	return s == enabledState
}

func (s *resumableState) Force(flag bool) {
	if flag {
		*s = enabledState
	} else {
		*s = disabledState
	}
}

func (s *resumableState) Set(flag bool) {
	if *s == disabledState {
		return
	}

	if flag {
		*s = enabledState
	} else {
		*s = pausedState
	}
}

type previewer struct {
	version    int64
	lines      []string
	offset     int
	scrollable bool
	final      bool
	following  resumableState
	spinner    string
	bar        []bool
}

type previewed struct {
	version  int64
	numLines int
	offset   int
	filled   bool
}

type eachLine struct {
	line string
	err  error
}

type itemLine struct {
	offset   int
	current  bool
	selected bool
	label    string
	queryLen int
	width    int
	bar      bool
	result   Result
}

type fitpad struct {
	fit int
	pad int
}

var emptyLine = itemLine{}

type labelPrinter func(tui.Window, int)

// Terminal represents terminal input/output
type Terminal struct {
	initDelay          time.Duration
	infoStyle          infoStyle
	infoSep            string
	separator          labelPrinter
	separatorLen       int
	spinner            []string
	prompt             func()
	promptLen          int
	borderLabel        labelPrinter
	borderLabelLen     int
	borderLabelOpts    labelOpts
	previewLabel       labelPrinter
	previewLabelLen    int
	previewLabelOpts   labelOpts
	pointer            string
	pointerLen         int
	pointerEmpty       string
	marker             string
	markerLen          int
	markerEmpty        string
	queryLen           [2]int
	layout             layoutType
	fullscreen         bool
	keepRight          bool
	hscroll            bool
	hscrollOff         int
	scrollOff          int
	wordRubout         string
	wordNext           string
	cx                 int
	cy                 int
	offset             int
	xoffset            int
	yanked             []rune
	input              []rune
	multi              int
	sort               bool
	toggleSort         bool
	track              trackOption
	delimiter          Delimiter
	expect             map[tui.Event]string
	keymap             map[tui.Event][]*action
	keymapOrg          map[tui.Event][]*action
	pressed            string
	printQuery         bool
	history            *History
	cycle              bool
	headerVisible      bool
	headerFirst        bool
	headerLines        int
	header             []string
	header0            []string
	ellipsis           string
	scrollbar          string
	previewScrollbar   string
	ansi               bool
	tabstop            int
	margin             [4]sizeSpec
	padding            [4]sizeSpec
	unicode            bool
	listenPort         *int
	borderShape        tui.BorderShape
	cleanExit          bool
	paused             bool
	border             tui.Window
	window             tui.Window
	pborder            tui.Window
	pwindow            tui.Window
	borderWidth        int
	count              int
	progress           int
	hasLoadActions     bool
	triggerLoad        bool
	reading            bool
	running            bool
	failed             *string
	jumping            jumpMode
	jumpLabels         string
	printer            func(string)
	printsep           string
	merger             *Merger
	selected           map[int32]selectedItem
	version            int64
	revision           int
	reqBox             *util.EventBox
	initialPreviewOpts previewOpts
	previewOpts        previewOpts
	activePreviewOpts  *previewOpts
	previewer          previewer
	previewed          previewed
	previewBox         *util.EventBox
	eventBox           *util.EventBox
	mutex              sync.Mutex
	initFunc           func()
	prevLines          []itemLine
	suppress           bool
	sigstop            bool
	startChan          chan fitpad
	killChan           chan int
	serverChan         chan []*action
	eventChan          chan tui.Event
	slab               *util.Slab
	theme              *tui.ColorTheme
	tui                tui.Renderer
	executing          *util.AtomicBool
}

type selectedItem struct {
	at   time.Time
	item *Item
}

type byTimeOrder []selectedItem

func (a byTimeOrder) Len() int {
	return len(a)
}

func (a byTimeOrder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byTimeOrder) Less(i, j int) bool {
	return a[i].at.Before(a[j].at)
}

const (
	reqPrompt util.EventType = iota
	reqInfo
	reqHeader
	reqList
	reqJump
	reqRefresh
	reqReinit
	reqFullRedraw
	reqRedrawBorderLabel
	reqRedrawPreviewLabel
	reqClose
	reqPrintQuery
	reqPreviewEnqueue
	reqPreviewDisplay
	reqPreviewRefresh
	reqPreviewDelayed
	reqQuit
)

type action struct {
	t actionType
	a string
}

type actionType int

const (
	actIgnore actionType = iota
	actInvalid
	actRune
	actMouse
	actBeginningOfLine
	actAbort
	actAccept
	actAcceptNonEmpty
	actBackwardChar
	actBackwardDeleteChar
	actBackwardDeleteCharEOF
	actBackwardWord
	actCancel
	actChangeBorderLabel
	actChangeHeader
	actChangePreviewLabel
	actChangePrompt
	actChangeQuery
	actClearScreen
	actClearQuery
	actClearSelection
	actClose
	actDeleteChar
	actDeleteCharEOF
	actEndOfLine
	actForwardChar
	actForwardWord
	actKillLine
	actKillWord
	actUnixLineDiscard
	actUnixWordRubout
	actYank
	actBackwardKillWord
	actSelectAll
	actDeselectAll
	actToggle
	actToggleSearch
	actToggleAll
	actToggleDown
	actToggleUp
	actToggleIn
	actToggleOut
	actToggleTrack
	actToggleHeader
	actTrack
	actDown
	actUp
	actPageUp
	actPageDown
	actPosition
	actHalfPageUp
	actHalfPageDown
	actJump
	actJumpAccept
	actPrintQuery
	actRefreshPreview
	actReplaceQuery
	actToggleSort
	actShowPreview
	actHidePreview
	actTogglePreview
	actTogglePreviewWrap
	actTransformBorderLabel
	actTransformHeader
	actTransformPreviewLabel
	actTransformPrompt
	actTransformQuery
	actPreview
	actChangePreview
	actChangePreviewWindow
	actPreviewTop
	actPreviewBottom
	actPreviewUp
	actPreviewDown
	actPreviewPageUp
	actPreviewPageDown
	actPreviewHalfPageUp
	actPreviewHalfPageDown
	actPrevHistory
	actPrevSelected
	actPut
	actNextHistory
	actNextSelected
	actExecute
	actExecuteSilent
	actExecuteMulti // Deprecated
	actSigStop
	actFirst
	actLast
	actReload
	actReloadSync
	actDisableSearch
	actEnableSearch
	actSelect
	actDeselect
	actUnbind
	actRebind
	actBecome
)

type placeholderFlags struct {
	plus          bool
	preserveSpace bool
	number        bool
	query         bool
	file          bool
}

type searchRequest struct {
	sort    bool
	sync    bool
	command *string
	changed bool
}

type previewRequest struct {
	template     string
	pwindow      tui.Window
	scrollOffset int
	list         []*Item
}

type previewResult struct {
	version int64
	lines   []string
	offset  int
	spinner string
}

func toActions(types ...actionType) []*action {
	actions := make([]*action, len(types))
	for idx, t := range types {
		actions[idx] = &action{t: t, a: ""}
	}
	return actions
}

func defaultKeymap() map[tui.Event][]*action {
	keymap := make(map[tui.Event][]*action)
	add := func(e tui.EventType, a actionType) {
		keymap[e.AsEvent()] = toActions(a)
	}
	addEvent := func(e tui.Event, a actionType) {
		keymap[e] = toActions(a)
	}

	add(tui.Invalid, actInvalid)
	add(tui.Resize, actClearScreen)
	add(tui.CtrlA, actBeginningOfLine)
	add(tui.CtrlB, actBackwardChar)
	add(tui.CtrlC, actAbort)
	add(tui.CtrlG, actAbort)
	add(tui.CtrlQ, actAbort)
	add(tui.ESC, actAbort)
	add(tui.CtrlD, actDeleteCharEOF)
	add(tui.CtrlE, actEndOfLine)
	add(tui.CtrlF, actForwardChar)
	add(tui.CtrlH, actBackwardDeleteChar)
	add(tui.BSpace, actBackwardDeleteChar)
	add(tui.Tab, actToggleDown)
	add(tui.BTab, actToggleUp)
	add(tui.CtrlJ, actDown)
	add(tui.CtrlK, actUp)
	add(tui.CtrlL, actClearScreen)
	add(tui.CtrlM, actAccept)
	add(tui.CtrlN, actDown)
	add(tui.CtrlP, actUp)
	add(tui.CtrlU, actUnixLineDiscard)
	add(tui.CtrlW, actUnixWordRubout)
	add(tui.CtrlY, actYank)
	if !util.IsWindows() {
		add(tui.CtrlZ, actSigStop)
	}

	addEvent(tui.AltKey('b'), actBackwardWord)
	add(tui.SLeft, actBackwardWord)
	addEvent(tui.AltKey('f'), actForwardWord)
	add(tui.SRight, actForwardWord)
	addEvent(tui.AltKey('d'), actKillWord)
	add(tui.AltBS, actBackwardKillWord)

	add(tui.Up, actUp)
	add(tui.Down, actDown)
	add(tui.Left, actBackwardChar)
	add(tui.Right, actForwardChar)

	add(tui.Home, actBeginningOfLine)
	add(tui.End, actEndOfLine)
	add(tui.Del, actDeleteChar)
	add(tui.PgUp, actPageUp)
	add(tui.PgDn, actPageDown)

	add(tui.SUp, actPreviewUp)
	add(tui.SDown, actPreviewDown)

	add(tui.Mouse, actMouse)
	add(tui.LeftClick, actIgnore)
	add(tui.RightClick, actToggle)
	return keymap
}

func trimQuery(query string) []rune {
	return []rune(strings.Replace(query, "\t", " ", -1))
}

func hasPreviewAction(opts *Options) bool {
	for _, actions := range opts.Keymap {
		for _, action := range actions {
			if action.t == actPreview || action.t == actChangePreview {
				return true
			}
		}
	}
	return false
}

func makeSpinner(unicode bool) []string {
	if unicode {
		return []string{`⠋`, `⠙`, `⠹`, `⠸`, `⠼`, `⠴`, `⠦`, `⠧`, `⠇`, `⠏`}
	}
	return []string{`-`, `\`, `|`, `/`, `-`, `\`, `|`, `/`}
}

func evaluateHeight(opts *Options, termHeight int) int {
	if opts.Height.percent {
		return util.Max(int(opts.Height.size*float64(termHeight)/100.0), opts.MinHeight)
	}
	return int(opts.Height.size)
}

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := trimQuery(opts.Query)
	var delay time.Duration
	if opts.Tac {
		delay = initialDelayTac
	} else {
		delay = initialDelay
	}
	var previewBox *util.EventBox
	// We need to start previewer if HTTP server is enabled even when --preview option is not specified
	if len(opts.Preview.command) > 0 || hasPreviewAction(opts) || opts.ListenPort != nil {
		previewBox = util.NewEventBox()
	}
	var renderer tui.Renderer
	fullscreen := !opts.Height.auto && (opts.Height.size == 0 || opts.Height.percent && opts.Height.size == 100)
	if fullscreen {
		if tui.HasFullscreenRenderer() {
			renderer = tui.NewFullscreenRenderer(opts.Theme, opts.Black, opts.Mouse)
		} else {
			renderer = tui.NewLightRenderer(opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit,
				true, func(h int) int { return h })
		}
	} else {
		maxHeightFunc := func(termHeight int) int {
			// Minimum height required to render fzf excluding margin and padding
			effectiveMinHeight := minHeight
			if previewBox != nil && opts.Preview.aboveOrBelow() {
				effectiveMinHeight += 1 + borderLines(opts.Preview.border)
			}
			if opts.InfoStyle.noExtraLine() {
				effectiveMinHeight--
			}
			effectiveMinHeight += borderLines(opts.BorderShape)
			return util.Min(termHeight, util.Max(evaluateHeight(opts, termHeight), effectiveMinHeight))
		}
		renderer = tui.NewLightRenderer(opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit, false, maxHeightFunc)
	}
	wordRubout := "[^\\pL\\pN][\\pL\\pN]"
	wordNext := "[\\pL\\pN][^\\pL\\pN]|(.$)"
	if opts.FileWord {
		sep := regexp.QuoteMeta(string(os.PathSeparator))
		wordRubout = fmt.Sprintf("%s[^%s]", sep, sep)
		wordNext = fmt.Sprintf("[^%s]%s|(.$)", sep, sep)
	}
	keymapCopy := make(map[tui.Event][]*action)
	for key, action := range opts.Keymap {
		keymapCopy[key] = action
	}
	t := Terminal{
		initDelay:          delay,
		infoStyle:          opts.InfoStyle,
		infoSep:            opts.InfoSep,
		separator:          nil,
		spinner:            makeSpinner(opts.Unicode),
		queryLen:           [2]int{0, 0},
		layout:             opts.Layout,
		fullscreen:         fullscreen,
		keepRight:          opts.KeepRight,
		hscroll:            opts.Hscroll,
		hscrollOff:         opts.HscrollOff,
		scrollOff:          opts.ScrollOff,
		wordRubout:         wordRubout,
		wordNext:           wordNext,
		cx:                 len(input),
		cy:                 0,
		offset:             0,
		xoffset:            0,
		yanked:             []rune{},
		input:              input,
		multi:              opts.Multi,
		sort:               opts.Sort > 0,
		toggleSort:         opts.ToggleSort,
		track:              opts.Track,
		delimiter:          opts.Delimiter,
		expect:             opts.Expect,
		keymap:             opts.Keymap,
		keymapOrg:          keymapCopy,
		pressed:            "",
		printQuery:         opts.PrintQuery,
		history:            opts.History,
		margin:             opts.Margin,
		padding:            opts.Padding,
		unicode:            opts.Unicode,
		listenPort:         opts.ListenPort,
		borderShape:        opts.BorderShape,
		borderWidth:        1,
		borderLabel:        nil,
		borderLabelOpts:    opts.BorderLabel,
		previewLabel:       nil,
		previewLabelOpts:   opts.PreviewLabel,
		cleanExit:          opts.ClearOnExit,
		paused:             opts.Phony,
		cycle:              opts.Cycle,
		headerVisible:      true,
		headerFirst:        opts.HeaderFirst,
		headerLines:        opts.HeaderLines,
		header:             []string{},
		header0:            opts.Header,
		ellipsis:           opts.Ellipsis,
		ansi:               opts.Ansi,
		tabstop:            opts.Tabstop,
		hasLoadActions:     false,
		triggerLoad:        false,
		reading:            true,
		running:            true,
		failed:             nil,
		jumping:            jumpDisabled,
		jumpLabels:         opts.JumpLabels,
		printer:            opts.Printer,
		printsep:           opts.PrintSep,
		merger:             EmptyMerger(0),
		selected:           make(map[int32]selectedItem),
		reqBox:             util.NewEventBox(),
		initialPreviewOpts: opts.Preview,
		previewOpts:        opts.Preview,
		previewer:          previewer{0, []string{}, 0, false, true, disabledState, "", []bool{}},
		previewed:          previewed{0, 0, 0, false},
		previewBox:         previewBox,
		eventBox:           eventBox,
		mutex:              sync.Mutex{},
		suppress:           true,
		sigstop:            false,
		slab:               util.MakeSlab(slab16Size, slab32Size),
		theme:              opts.Theme,
		startChan:          make(chan fitpad, 1),
		killChan:           make(chan int),
		serverChan:         make(chan []*action, 10),
		eventChan:          make(chan tui.Event, 1),
		tui:                renderer,
		initFunc:           func() { renderer.Init() },
		executing:          util.NewAtomicBool(false)}
	t.prompt, t.promptLen = t.parsePrompt(opts.Prompt)
	t.pointer, t.pointerLen = t.processTabs([]rune(opts.Pointer), 0)
	t.marker, t.markerLen = t.processTabs([]rune(opts.Marker), 0)
	// Pre-calculated empty pointer and marker signs
	t.pointerEmpty = strings.Repeat(" ", t.pointerLen)
	t.markerEmpty = strings.Repeat(" ", t.markerLen)
	t.borderLabel, t.borderLabelLen = t.ansiLabelPrinter(opts.BorderLabel.label, &tui.ColBorderLabel, false)
	t.previewLabel, t.previewLabelLen = t.ansiLabelPrinter(opts.PreviewLabel.label, &tui.ColPreviewLabel, false)
	if opts.Separator == nil || len(*opts.Separator) > 0 {
		bar := "─"
		if opts.Separator != nil {
			bar = *opts.Separator
		} else if !t.unicode {
			bar = "-"
		}
		t.separator, t.separatorLen = t.ansiLabelPrinter(bar, &tui.ColSeparator, true)
	}
	if t.unicode {
		t.borderWidth = runewidth.RuneWidth('│')
	}
	if opts.Scrollbar == nil {
		if t.unicode && t.borderWidth == 1 {
			t.scrollbar = "│"
		} else {
			t.scrollbar = "|"
		}
		t.previewScrollbar = t.scrollbar
	} else {
		runes := []rune(*opts.Scrollbar)
		if len(runes) > 0 {
			t.scrollbar = string(runes[0])
			t.previewScrollbar = t.scrollbar
			if len(runes) > 1 {
				t.previewScrollbar = string(runes[1])
			}
		}
	}

	_, t.hasLoadActions = t.keymap[tui.Load.AsEvent()]

	if t.listenPort != nil {
		err, port := startHttpServer(*t.listenPort, t.serverChan)
		if err != nil {
			errorExit(err.Error())
		}
		t.listenPort = &port
	}

	return &t
}

func (t *Terminal) environ() []string {
	env := os.Environ()
	if t.listenPort != nil {
		env = append(env, fmt.Sprintf("FZF_PORT=%d", *t.listenPort))
	}
	return env
}

func borderLines(shape tui.BorderShape) int {
	switch shape {
	case tui.BorderHorizontal, tui.BorderRounded, tui.BorderSharp, tui.BorderBold, tui.BorderBlock, tui.BorderThinBlock, tui.BorderDouble:
		return 2
	case tui.BorderTop, tui.BorderBottom:
		return 1
	}
	return 0
}

func (t *Terminal) visibleHeaderLines() int {
	if !t.headerVisible {
		return 0
	}
	return len(t.header0) + t.headerLines
}

// Extra number of lines needed to display fzf
func (t *Terminal) extraLines() int {
	extra := t.visibleHeaderLines() + 1
	if !t.noInfoLine() {
		extra++
	}
	return extra
}

func (t *Terminal) MaxFitAndPad(opts *Options) (int, int) {
	_, screenHeight, marginInt, paddingInt := t.adjustMarginAndPadding()
	padHeight := marginInt[0] + marginInt[2] + paddingInt[0] + paddingInt[2]
	fit := screenHeight - padHeight - t.extraLines()
	return fit, padHeight
}

func (t *Terminal) ansiLabelPrinter(str string, color *tui.ColorPair, fill bool) (labelPrinter, int) {
	// Nothing to do
	if len(str) == 0 {
		return nil, 0
	}

	// Extract ANSI color codes
	str = firstLine(str)
	text, colors, _ := extractColor(str, nil, nil)
	runes := []rune(text)

	// Simpler printer for strings without ANSI colors or tab characters
	if colors == nil && !strings.ContainsRune(str, '\t') {
		length := util.StringWidth(str)
		if length == 0 {
			return nil, 0
		}
		printFn := func(window tui.Window, limit int) {
			if length > limit {
				trimmedRunes, _ := t.trimRight(runes, limit)
				window.CPrint(*color, string(trimmedRunes))
			} else if fill {
				window.CPrint(*color, util.RepeatToFill(str, length, limit))
			} else {
				window.CPrint(*color, str)
			}
		}
		return printFn, len(text)
	}

	// Printer that correctly handles ANSI color codes and tab characters
	item := &Item{text: util.RunesToChars(runes), colors: colors}
	length := t.displayWidth(runes)
	if length == 0 {
		return nil, 0
	}
	result := Result{item: item}
	var offsets []colorOffset
	printFn := func(window tui.Window, limit int) {
		if offsets == nil {
			// tui.Col* are not initialized until renderer.Init()
			offsets = result.colorOffsets(nil, t.theme, *color, *color, false)
		}
		for limit > 0 {
			if length > limit {
				trimmedRunes, _ := t.trimRight(runes, limit)
				t.printColoredString(window, trimmedRunes, offsets, *color)
				break
			} else if fill {
				t.printColoredString(window, runes, offsets, *color)
				limit -= length
			} else {
				t.printColoredString(window, runes, offsets, *color)
				break
			}
		}
	}
	return printFn, length
}

func (t *Terminal) parsePrompt(prompt string) (func(), int) {
	var state *ansiState
	prompt = firstLine(prompt)
	trimmed, colors, _ := extractColor(prompt, state, nil)
	item := &Item{text: util.ToChars([]byte(trimmed)), colors: colors}

	// "Prompt>  "
	//  -------    // Do not apply ANSI attributes to the trailing whitespaces
	//             // unless the part has a non-default ANSI state
	loc := whiteSuffix.FindStringIndex(trimmed)
	if loc != nil {
		blankState := ansiOffset{[2]int32{int32(loc[0]), int32(loc[1])}, ansiState{-1, -1, tui.AttrClear, -1}}
		if item.colors != nil {
			lastColor := (*item.colors)[len(*item.colors)-1]
			if lastColor.offset[1] < int32(loc[1]) {
				blankState.offset[0] = lastColor.offset[1]
				colors := append(*item.colors, blankState)
				item.colors = &colors
			}
		} else {
			colors := []ansiOffset{blankState}
			item.colors = &colors
		}
	}
	output := func() {
		t.printHighlighted(
			Result{item: item}, tui.ColPrompt, tui.ColPrompt, false, false)
	}
	_, promptLen := t.processTabs([]rune(trimmed), 0)

	return output, promptLen
}

func (t *Terminal) noInfoLine() bool {
	return t.infoStyle.noExtraLine()
}

func getScrollbar(total int, height int, offset int) (int, int) {
	if total == 0 || total <= height {
		return 0, 0
	}
	barLength := util.Max(1, height*height/total)
	var barStart int
	if total == height {
		barStart = 0
	} else {
		barStart = (height - barLength) * offset / (total - height)
	}
	return barLength, barStart
}

func (t *Terminal) getScrollbar() (int, int) {
	return getScrollbar(t.merger.Length(), t.maxItems(), t.offset)
}

// Input returns current query string
func (t *Terminal) Input() (bool, []rune) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.paused, copySlice(t.input)
}

// UpdateCount updates the count information
func (t *Terminal) UpdateCount(cnt int, final bool, failedCommand *string) {
	t.mutex.Lock()
	t.count = cnt
	if t.hasLoadActions && t.reading && final {
		t.triggerLoad = true
	}
	t.reading = !final
	t.failed = failedCommand
	t.mutex.Unlock()
	t.reqBox.Set(reqInfo, nil)
	if final {
		t.reqBox.Set(reqRefresh, nil)
	}
}

func (t *Terminal) changeHeader(header string) bool {
	lines := strings.Split(strings.TrimSuffix(header, "\n"), "\n")
	needFullRedraw := len(t.header0) != len(lines)
	t.header0 = lines
	return needFullRedraw
}

// UpdateHeader updates the header
func (t *Terminal) UpdateHeader(header []string) {
	t.mutex.Lock()
	t.header = header
	t.mutex.Unlock()
	t.reqBox.Set(reqHeader, nil)
}

// UpdateProgress updates the search progress
func (t *Terminal) UpdateProgress(progress float32) {
	t.mutex.Lock()
	newProgress := int(progress * 100)
	changed := t.progress != newProgress
	t.progress = newProgress
	t.mutex.Unlock()

	if changed {
		t.reqBox.Set(reqInfo, nil)
	}
}

// UpdateList updates Merger to display the list
func (t *Terminal) UpdateList(merger *Merger) {
	t.mutex.Lock()
	var prevIndex int32 = -1
	reset := t.revision != merger.Revision()
	if !reset && t.track != trackDisabled {
		if t.merger.Length() > 0 {
			prevIndex = t.merger.Get(t.cy).item.Index()
		} else if merger.Length() > 0 {
			prevIndex = merger.First().item.Index()
		}
	}
	t.progress = 100
	t.merger = merger
	if reset {
		t.selected = make(map[int32]selectedItem)
		t.revision = merger.Revision()
		t.version++
	}
	if t.triggerLoad {
		t.triggerLoad = false
		t.eventChan <- tui.Load.AsEvent()
	}
	if prevIndex >= 0 {
		pos := t.cy - t.offset
		count := t.merger.Length()
		i := t.merger.FindIndex(prevIndex)
		if i >= 0 {
			t.cy = i
			t.offset = t.cy - pos
		} else if t.track == trackCurrent {
			t.track = trackDisabled
			t.cy = pos
			t.offset = 0
		} else if t.cy > count {
			// Try to keep the vertical position when the list shrinks
			t.cy = count - util.Min(count, t.maxItems()) + pos
		}
	}
	if !t.reading {
		switch t.merger.Length() {
		case 0:
			zero := tui.Zero.AsEvent()
			if _, prs := t.keymap[zero]; prs {
				t.eventChan <- zero
			}
		case 1:
			one := tui.One.AsEvent()
			if _, prs := t.keymap[one]; prs {
				t.eventChan <- one
			}
		}
	}
	t.mutex.Unlock()
	t.reqBox.Set(reqInfo, nil)
	t.reqBox.Set(reqList, nil)
}

func (t *Terminal) output() bool {
	if t.printQuery {
		t.printer(string(t.input))
	}
	if len(t.expect) > 0 {
		t.printer(t.pressed)
	}
	found := len(t.selected) > 0
	if !found {
		current := t.currentItem()
		if current != nil {
			t.printer(current.AsString(t.ansi))
			found = true
		}
	} else {
		for _, sel := range t.sortSelected() {
			t.printer(sel.item.AsString(t.ansi))
		}
	}
	return found
}

func (t *Terminal) sortSelected() []selectedItem {
	sels := make([]selectedItem, 0, len(t.selected))
	for _, sel := range t.selected {
		sels = append(sels, sel)
	}
	sort.Sort(byTimeOrder(sels))
	return sels
}

func (t *Terminal) displayWidth(runes []rune) int {
	width, _ := util.RunesWidth(runes, 0, t.tabstop, math.MaxInt32)
	return width
}

const (
	minWidth  = 4
	minHeight = 3
)

func calculateSize(base int, size sizeSpec, occupied int, minSize int, pad int) int {
	max := base - occupied
	if max < minSize {
		max = minSize
	}
	if size.percent {
		return util.Constrain(int(float64(base)*0.01*size.size), minSize, max)
	}
	return util.Constrain(int(size.size)+pad, minSize, max)
}

func (t *Terminal) adjustMarginAndPadding() (int, int, [4]int, [4]int) {
	screenWidth := t.tui.MaxX()
	screenHeight := t.tui.MaxY()
	marginInt := [4]int{}  // TRBL
	paddingInt := [4]int{} // TRBL
	sizeSpecToInt := func(index int, spec sizeSpec) int {
		if spec.percent {
			var max float64
			if index%2 == 0 {
				max = float64(screenHeight)
			} else {
				max = float64(screenWidth)
			}
			return int(max * spec.size * 0.01)
		}
		return int(spec.size)
	}
	for idx, sizeSpec := range t.padding {
		paddingInt[idx] = sizeSpecToInt(idx, sizeSpec)
	}

	bw := t.borderWidth
	extraMargin := [4]int{} // TRBL
	for idx, sizeSpec := range t.margin {
		switch t.borderShape {
		case tui.BorderHorizontal:
			extraMargin[idx] += 1 - idx%2
		case tui.BorderVertical:
			extraMargin[idx] += (1 + bw) * (idx % 2)
		case tui.BorderTop:
			if idx == 0 {
				extraMargin[idx]++
			}
		case tui.BorderRight:
			if idx == 1 {
				extraMargin[idx] += 1 + bw
			}
		case tui.BorderBottom:
			if idx == 2 {
				extraMargin[idx]++
			}
		case tui.BorderLeft:
			if idx == 3 {
				extraMargin[idx] += 1 + bw
			}
		case tui.BorderRounded, tui.BorderSharp, tui.BorderBold, tui.BorderBlock, tui.BorderThinBlock, tui.BorderDouble:
			extraMargin[idx] += 1 + bw*(idx%2)
		}
		marginInt[idx] = sizeSpecToInt(idx, sizeSpec) + extraMargin[idx]
	}

	adjust := func(idx1 int, idx2 int, max int, min int) {
		if min > max {
			min = max
		}
		margin := marginInt[idx1] + marginInt[idx2] + paddingInt[idx1] + paddingInt[idx2]
		if max-margin < min {
			desired := max - min
			paddingInt[idx1] = desired * paddingInt[idx1] / margin
			paddingInt[idx2] = desired * paddingInt[idx2] / margin
			marginInt[idx1] = util.Max(extraMargin[idx1], desired*marginInt[idx1]/margin)
			marginInt[idx2] = util.Max(extraMargin[idx2], desired*marginInt[idx2]/margin)
		}
	}

	minAreaWidth := minWidth
	minAreaHeight := minHeight
	if t.noInfoLine() {
		minAreaHeight -= 1
	}
	if t.needPreviewWindow() {
		minPreviewHeight := 1 + borderLines(t.previewOpts.border)
		minPreviewWidth := 5
		switch t.previewOpts.position {
		case posUp, posDown:
			minAreaHeight += minPreviewHeight
			minAreaWidth = util.Max(minPreviewWidth, minAreaWidth)
		case posLeft, posRight:
			minAreaWidth += minPreviewWidth
			minAreaHeight = util.Max(minPreviewHeight, minAreaHeight)
		}
	}
	adjust(1, 3, screenWidth, minAreaWidth)
	adjust(0, 2, screenHeight, minAreaHeight)

	return screenWidth, screenHeight, marginInt, paddingInt
}

func (t *Terminal) resizeWindows(forcePreview bool) {
	screenWidth, screenHeight, marginInt, paddingInt := t.adjustMarginAndPadding()
	width := screenWidth - marginInt[1] - marginInt[3]
	height := screenHeight - marginInt[0] - marginInt[2]

	t.prevLines = make([]itemLine, screenHeight)
	if t.border != nil {
		t.border.Close()
	}
	if t.window != nil {
		t.window.Close()
		t.window = nil
	}
	if t.pborder != nil {
		t.pborder.Close()
		t.pborder = nil
	}
	if t.pwindow != nil {
		t.pwindow.Close()
		t.pwindow = nil
	}
	// Reset preview version so that full redraw occurs
	t.previewed.version = 0

	bw := t.borderWidth
	switch t.borderShape {
	case tui.BorderHorizontal:
		t.border = t.tui.NewWindow(
			marginInt[0]-1, marginInt[3], width, height+2,
			false, tui.MakeBorderStyle(tui.BorderHorizontal, t.unicode))
	case tui.BorderVertical:
		t.border = t.tui.NewWindow(
			marginInt[0], marginInt[3]-(1+bw), width+(1+bw)*2, height,
			false, tui.MakeBorderStyle(tui.BorderVertical, t.unicode))
	case tui.BorderTop:
		t.border = t.tui.NewWindow(
			marginInt[0]-1, marginInt[3], width, height+1,
			false, tui.MakeBorderStyle(tui.BorderTop, t.unicode))
	case tui.BorderBottom:
		t.border = t.tui.NewWindow(
			marginInt[0], marginInt[3], width, height+1,
			false, tui.MakeBorderStyle(tui.BorderBottom, t.unicode))
	case tui.BorderLeft:
		t.border = t.tui.NewWindow(
			marginInt[0], marginInt[3]-(1+bw), width+(1+bw), height,
			false, tui.MakeBorderStyle(tui.BorderLeft, t.unicode))
	case tui.BorderRight:
		t.border = t.tui.NewWindow(
			marginInt[0], marginInt[3], width+(1+bw), height,
			false, tui.MakeBorderStyle(tui.BorderRight, t.unicode))
	case tui.BorderRounded, tui.BorderSharp, tui.BorderBold, tui.BorderBlock, tui.BorderThinBlock, tui.BorderDouble:
		t.border = t.tui.NewWindow(
			marginInt[0]-1, marginInt[3]-(1+bw), width+(1+bw)*2, height+2,
			false, tui.MakeBorderStyle(t.borderShape, t.unicode))
	}

	// Add padding to margin
	for idx, val := range paddingInt {
		marginInt[idx] += val
	}
	width -= paddingInt[1] + paddingInt[3]
	height -= paddingInt[0] + paddingInt[2]

	// Set up preview window
	noBorder := tui.MakeBorderStyle(tui.BorderNone, t.unicode)
	if forcePreview || t.needPreviewWindow() {
		var resizePreviewWindows func(previewOpts *previewOpts)
		resizePreviewWindows = func(previewOpts *previewOpts) {
			t.activePreviewOpts = previewOpts
			if previewOpts.size.size == 0 {
				return
			}
			hasThreshold := previewOpts.threshold > 0 && previewOpts.alternative != nil
			createPreviewWindow := func(y int, x int, w int, h int) {
				pwidth := w
				pheight := h
				var previewBorder tui.BorderStyle
				if previewOpts.border == tui.BorderNone {
					previewBorder = tui.MakeTransparentBorder()
				} else {
					previewBorder = tui.MakeBorderStyle(previewOpts.border, t.unicode)
				}
				t.pborder = t.tui.NewWindow(y, x, w, h, true, previewBorder)
				switch previewOpts.border {
				case tui.BorderSharp, tui.BorderRounded, tui.BorderBold, tui.BorderBlock, tui.BorderThinBlock, tui.BorderDouble:
					pwidth -= (1 + bw) * 2
					pheight -= 2
					x += 1 + bw
					y += 1
				case tui.BorderLeft:
					pwidth -= 1 + bw
					x += 1 + bw
				case tui.BorderRight:
					pwidth -= 1 + bw
				case tui.BorderTop:
					pheight -= 1
					y += 1
				case tui.BorderBottom:
					pheight -= 1
				case tui.BorderHorizontal:
					pheight -= 2
					y += 1
				case tui.BorderVertical:
					pwidth -= (1 + bw) * 2
					x += 1 + bw
				}
				if len(t.scrollbar) > 0 && !previewOpts.border.HasRight() {
					// Need a column to show scrollbar
					pwidth -= 1
				}
				pwidth = util.Max(0, pwidth)
				pheight = util.Max(0, pheight)
				t.pwindow = t.tui.NewWindow(y, x, pwidth, pheight, true, noBorder)
			}
			verticalPad := 2
			minPreviewHeight := 3
			switch previewOpts.border {
			case tui.BorderNone, tui.BorderVertical, tui.BorderLeft, tui.BorderRight:
				verticalPad = 0
				minPreviewHeight = 1
			case tui.BorderTop, tui.BorderBottom:
				verticalPad = 1
				minPreviewHeight = 2
			}
			switch previewOpts.position {
			case posUp, posDown:
				pheight := calculateSize(height, previewOpts.size, minHeight, minPreviewHeight, verticalPad)
				if hasThreshold && pheight < previewOpts.threshold {
					t.activePreviewOpts = previewOpts.alternative
					if forcePreview {
						previewOpts.alternative.hidden = false
					}
					if !previewOpts.alternative.hidden {
						resizePreviewWindows(previewOpts.alternative)
					}
					return
				}
				if forcePreview {
					previewOpts.hidden = false
				}
				if previewOpts.hidden {
					return
				}
				// Put scrollbar closer to the right border for consistent look
				if t.borderShape.HasRight() {
					width++
				}
				if previewOpts.position == posUp {
					t.window = t.tui.NewWindow(
						marginInt[0]+pheight, marginInt[3], width, height-pheight, false, noBorder)
					createPreviewWindow(marginInt[0], marginInt[3], width, pheight)
				} else {
					t.window = t.tui.NewWindow(
						marginInt[0], marginInt[3], width, height-pheight, false, noBorder)
					createPreviewWindow(marginInt[0]+height-pheight, marginInt[3], width, pheight)
				}
			case posLeft, posRight:
				pwidth := calculateSize(width, previewOpts.size, minWidth, 5, 4)
				if hasThreshold && pwidth < previewOpts.threshold {
					t.activePreviewOpts = previewOpts.alternative
					if forcePreview {
						previewOpts.alternative.hidden = false
					}
					if !previewOpts.alternative.hidden {
						resizePreviewWindows(previewOpts.alternative)
					}
					return
				}
				if forcePreview {
					previewOpts.hidden = false
				}
				if previewOpts.hidden {
					return
				}
				if previewOpts.position == posLeft {
					// Put scrollbar closer to the right border for consistent look
					if t.borderShape.HasRight() {
						width++
					}
					// Add a 1-column margin between the preview window and the main window
					t.window = t.tui.NewWindow(
						marginInt[0], marginInt[3]+pwidth+1, width-pwidth-1, height, false, noBorder)
					createPreviewWindow(marginInt[0], marginInt[3], pwidth, height)
				} else {
					t.window = t.tui.NewWindow(
						marginInt[0], marginInt[3], width-pwidth, height, false, noBorder)
					// NOTE: fzf --preview 'cat {}' --preview-window border-left --border
					x := marginInt[3] + width - pwidth
					if !previewOpts.border.HasRight() && t.borderShape.HasRight() {
						pwidth++
					}
					createPreviewWindow(marginInt[0], x, pwidth, height)
				}
			}
		}
		resizePreviewWindows(&t.previewOpts)
	} else {
		t.activePreviewOpts = &t.previewOpts
	}

	// Without preview window
	if t.window == nil {
		if t.borderShape.HasRight() {
			// Put scrollbar closer to the right border for consistent look
			width++
		}
		t.window = t.tui.NewWindow(
			marginInt[0],
			marginInt[3],
			width,
			height, false, noBorder)
	}

	// Print border label
	t.printLabel(t.border, t.borderLabel, t.borderLabelOpts, t.borderLabelLen, t.borderShape, false)
	t.printLabel(t.pborder, t.previewLabel, t.previewLabelOpts, t.previewLabelLen, t.previewOpts.border, false)

	for i := 0; i < t.window.Height(); i++ {
		t.window.MoveAndClear(i, 0)
	}
}

func (t *Terminal) printLabel(window tui.Window, render labelPrinter, opts labelOpts, length int, borderShape tui.BorderShape, redrawBorder bool) {
	if window == nil {
		return
	}

	switch borderShape {
	case tui.BorderHorizontal, tui.BorderTop, tui.BorderBottom, tui.BorderRounded, tui.BorderSharp, tui.BorderBold, tui.BorderBlock, tui.BorderThinBlock, tui.BorderDouble:
		if redrawBorder {
			window.DrawHBorder()
		}
		if render == nil {
			return
		}
		var col int
		if opts.column == 0 {
			col = util.Max(0, (window.Width()-length)/2)
		} else if opts.column < 0 {
			col = util.Max(0, window.Width()+opts.column+1-length)
		} else {
			col = util.Min(opts.column-1, window.Width()-length)
		}
		row := 0
		if borderShape == tui.BorderBottom || opts.bottom {
			row = window.Height() - 1
		}
		window.Move(row, col)
		render(window, window.Width())
	}
}

func (t *Terminal) move(y int, x int, clear bool) {
	h := t.window.Height()

	switch t.layout {
	case layoutDefault:
		y = h - y - 1
	case layoutReverseList:
		n := 2 + t.visibleHeaderLines()
		if t.noInfoLine() {
			n--
		}
		if y < n {
			y = h - y - 1
		} else {
			y -= n
		}
	}

	if clear {
		t.window.MoveAndClear(y, x)
	} else {
		t.window.Move(y, x)
	}
}

func (t *Terminal) truncateQuery() {
	t.input, _ = t.trimRight(t.input, maxPatternLength)
	t.cx = util.Constrain(t.cx, 0, len(t.input))
}

func (t *Terminal) updatePromptOffset() ([]rune, []rune) {
	maxWidth := util.Max(1, t.window.Width()-t.promptLen-1)

	_, overflow := t.trimLeft(t.input[:t.cx], maxWidth)
	minOffset := int(overflow)
	maxOffset := minOffset + (maxWidth-util.Max(0, maxWidth-t.cx))/2
	t.xoffset = util.Constrain(t.xoffset, minOffset, maxOffset)
	before, _ := t.trimLeft(t.input[t.xoffset:t.cx], maxWidth)
	beforeLen := t.displayWidth(before)
	after, _ := t.trimRight(t.input[t.cx:], maxWidth-beforeLen)
	afterLen := t.displayWidth(after)
	t.queryLen = [2]int{beforeLen, afterLen}
	return before, after
}

func (t *Terminal) promptLine() int {
	if t.headerFirst {
		max := t.window.Height() - 1
		if !t.noInfoLine() {
			max--
		}
		return util.Min(t.visibleHeaderLines(), max)
	}
	return 0
}

func (t *Terminal) placeCursor() {
	t.move(t.promptLine(), t.promptLen+t.queryLen[0], false)
}

func (t *Terminal) printPrompt() {
	t.move(t.promptLine(), 0, true)
	t.prompt()

	before, after := t.updatePromptOffset()
	color := tui.ColInput
	if t.paused {
		color = tui.ColDisabled
	}
	t.window.CPrint(color, string(before))
	t.window.CPrint(color, string(after))
}

func (t *Terminal) trimMessage(message string, maxWidth int) string {
	if len(message) <= maxWidth {
		return message
	}
	runes, _ := t.trimRight([]rune(message), maxWidth-2)
	return string(runes) + strings.Repeat(".", util.Constrain(maxWidth, 0, 2))
}

func (t *Terminal) printInfo() {
	pos := 0
	line := t.promptLine()
	printSpinner := func() {
		if t.reading {
			duration := int64(spinnerDuration)
			idx := (time.Now().UnixNano() % (duration * int64(len(t.spinner)))) / duration
			t.window.CPrint(tui.ColSpinner, t.spinner[idx])
		} else {
			t.window.Print(" ") // Clear spinner
		}
	}
	switch t.infoStyle {
	case infoDefault:
		t.move(line+1, 0, t.separatorLen == 0)
		printSpinner()
		t.move(line+1, 2, false)
		pos = 2
	case infoRight:
		t.move(line+1, 0, false)
	case infoInlineRight:
		pos = t.promptLen + t.queryLen[0] + t.queryLen[1] + 1
		t.move(line, pos, true)
	case infoInline:
		pos = t.promptLen + t.queryLen[0] + t.queryLen[1] + 1
		str := t.infoSep
		maxWidth := t.window.Width() - pos
		width := util.StringWidth(str)
		if width > maxWidth {
			trimmed, _ := t.trimRight([]rune(str), maxWidth)
			str = string(trimmed)
			width = maxWidth
		}
		t.move(line, pos, t.separatorLen == 0)
		if t.reading {
			t.window.CPrint(tui.ColSpinner, str)
		} else {
			t.window.CPrint(tui.ColPrompt, str)
		}
		pos += width
	case infoHidden:
		return
	}

	found := t.merger.Length()
	total := util.Max(found, t.count)
	output := fmt.Sprintf("%d/%d", found, total)
	if t.toggleSort {
		if t.sort {
			output += " +S"
		} else {
			output += " -S"
		}
	}
	if t.track != trackDisabled {
		output += " +T"
	}
	if t.multi > 0 {
		if t.multi == maxMulti {
			output += fmt.Sprintf(" (%d)", len(t.selected))
		} else {
			output += fmt.Sprintf(" (%d/%d)", len(t.selected), t.multi)
		}
	}
	if t.progress > 0 && t.progress < 100 {
		output += fmt.Sprintf(" (%d%%)", t.progress)
	}
	if t.failed != nil && t.count == 0 {
		output = fmt.Sprintf("[Command failed: %s]", *t.failed)
	}

	printSeparator := func(fillLength int, pad bool) {
		// --------_
		if t.separatorLen > 0 {
			t.separator(t.window, fillLength)
			t.window.Print(" ")
		} else if pad {
			t.window.Print(strings.Repeat(" ", fillLength+1))
		}
	}
	if t.infoStyle == infoRight {
		maxWidth := t.window.Width()
		if t.reading {
			// Need space for spinner and a margin column
			maxWidth -= 2
		}
		output = t.trimMessage(output, maxWidth)
		fillLength := t.window.Width() - len(output) - 2
		if t.reading {
			if fillLength >= 2 {
				printSeparator(fillLength-2, true)
			}
			printSpinner()
			t.window.Print(" ")
		} else if fillLength >= 0 {
			printSeparator(fillLength, true)
		}
		t.window.CPrint(tui.ColInfo, output)
		return
	}

	if t.infoStyle == infoInlineRight {
		pos = util.Max(pos, t.window.Width()-util.StringWidth(output)-3)
		if pos >= t.window.Width() {
			return
		}
		t.move(line, pos, false)
		printSpinner()
		t.window.Print(" ")
		pos += 2
	}

	maxWidth := t.window.Width() - pos
	output = t.trimMessage(output, maxWidth)
	t.window.CPrint(tui.ColInfo, output)
	fillLength := maxWidth - len(output) - 2
	if fillLength > 0 {
		t.window.CPrint(tui.ColSeparator, " ")
		printSeparator(fillLength, false)
	}
}

func (t *Terminal) printHeader() {
	if t.visibleHeaderLines() == 0 {
		return
	}
	max := t.window.Height()
	if t.headerFirst {
		max--
		if !t.noInfoLine() {
			max--
		}
	}
	var state *ansiState
	needReverse := false
	switch t.layout {
	case layoutDefault, layoutReverseList:
		needReverse = true
	}
	for idx, lineStr := range append(append([]string{}, t.header0...), t.header...) {
		line := idx
		if needReverse && idx < len(t.header0) {
			line = len(t.header0) - idx - 1
		}
		if !t.headerFirst {
			line++
			if !t.noInfoLine() {
				line++
			}
		}
		if line >= max {
			continue
		}
		trimmed, colors, newState := extractColor(lineStr, state, nil)
		state = newState
		item := &Item{
			text:   util.ToChars([]byte(trimmed)),
			colors: colors}

		t.move(line, 0, true)
		t.window.Print("  ")
		t.printHighlighted(Result{item: item},
			tui.ColHeader, tui.ColHeader, false, false)
	}
}

func (t *Terminal) printList() {
	t.constrain()
	barLength, barStart := t.getScrollbar()

	maxy := t.maxItems()
	count := t.merger.Length() - t.offset
	for j := 0; j < maxy; j++ {
		i := j
		if t.layout == layoutDefault {
			i = maxy - 1 - j
		}
		line := i + 2 + t.visibleHeaderLines()
		if t.noInfoLine() {
			line--
		}
		if i < count {
			t.printItem(t.merger.Get(i+t.offset), line, i, i == t.cy-t.offset, i >= barStart && i < barStart+barLength)
		} else if t.prevLines[i] != emptyLine || t.prevLines[i].offset != line {
			t.prevLines[i] = emptyLine
			t.move(line, 0, true)
		}
	}
}

func (t *Terminal) printItem(result Result, line int, i int, current bool, bar bool) {
	item := result.item
	_, selected := t.selected[item.Index()]
	label := ""
	if t.jumping != jumpDisabled {
		if i < len(t.jumpLabels) {
			// Striped
			current = i%2 == 0
			label = t.jumpLabels[i:i+1] + strings.Repeat(" ", t.pointerLen-1)
		}
	} else if current {
		label = t.pointer
	}

	// Avoid unnecessary redraw
	newLine := itemLine{offset: line, current: current, selected: selected, label: label,
		result: result, queryLen: len(t.input), width: 0, bar: bar}
	prevLine := t.prevLines[i]
	forceRedraw := prevLine.offset != newLine.offset
	printBar := func() {
		if len(t.scrollbar) > 0 && (bar != prevLine.bar || forceRedraw) {
			t.prevLines[i].bar = bar
			t.move(line, t.window.Width()-1, true)
			if bar {
				t.window.CPrint(tui.ColScrollbar, t.scrollbar)
			}
		}
	}

	if !forceRedraw &&
		prevLine.current == newLine.current &&
		prevLine.selected == newLine.selected &&
		prevLine.label == newLine.label &&
		prevLine.queryLen == newLine.queryLen &&
		prevLine.result == newLine.result {
		printBar()
		return
	}

	t.move(line, 0, forceRedraw)
	if current {
		if len(label) == 0 {
			t.window.CPrint(tui.ColCurrentCursorEmpty, t.pointerEmpty)
		} else {
			t.window.CPrint(tui.ColCurrentCursor, label)
		}
		if selected {
			t.window.CPrint(tui.ColCurrentSelected, t.marker)
		} else {
			t.window.CPrint(tui.ColCurrentSelectedEmpty, t.markerEmpty)
		}
		newLine.width = t.printHighlighted(result, tui.ColCurrent, tui.ColCurrentMatch, true, true)
	} else {
		if len(label) == 0 {
			t.window.CPrint(tui.ColCursorEmpty, t.pointerEmpty)
		} else {
			t.window.CPrint(tui.ColCursor, label)
		}
		if selected {
			t.window.CPrint(tui.ColSelected, t.marker)
		} else {
			t.window.Print(t.markerEmpty)
		}
		newLine.width = t.printHighlighted(result, tui.ColNormal, tui.ColMatch, false, true)
	}
	fillSpaces := prevLine.width - newLine.width
	if fillSpaces > 0 {
		t.window.Print(strings.Repeat(" ", fillSpaces))
	}
	printBar()
	t.prevLines[i] = newLine
}

func (t *Terminal) trimRight(runes []rune, width int) ([]rune, bool) {
	// We start from the beginning to handle tab characters
	_, overflowIdx := util.RunesWidth(runes, 0, t.tabstop, width)
	if overflowIdx >= 0 {
		return runes[:overflowIdx], true
	}
	return runes, false
}

func (t *Terminal) displayWidthWithLimit(runes []rune, prefixWidth int, limit int) int {
	width, _ := util.RunesWidth(runes, prefixWidth, t.tabstop, limit)
	return width
}

func (t *Terminal) trimLeft(runes []rune, width int) ([]rune, int32) {
	width = util.Max(0, width)
	var trimmed int32
	// Assume that each rune takes at least one column on screen
	if len(runes) > width+2 {
		diff := len(runes) - width - 2
		trimmed = int32(diff)
		runes = runes[diff:]
	}

	currentWidth := t.displayWidth(runes)

	for currentWidth > width && len(runes) > 0 {
		runes = runes[1:]
		trimmed++
		currentWidth = t.displayWidthWithLimit(runes, 2, width)
	}
	return runes, trimmed
}

func (t *Terminal) overflow(runes []rune, max int) bool {
	return t.displayWidthWithLimit(runes, 0, max) > max
}

func (t *Terminal) printHighlighted(result Result, colBase tui.ColorPair, colMatch tui.ColorPair, current bool, match bool) int {
	item := result.item

	// Overflow
	text := make([]rune, item.text.Length())
	copy(text, item.text.ToRunes())
	matchOffsets := []Offset{}
	var pos *[]int
	if match && t.merger.pattern != nil {
		_, matchOffsets, pos = t.merger.pattern.MatchItem(item, true, t.slab)
	}
	charOffsets := matchOffsets
	if pos != nil {
		charOffsets = make([]Offset, len(*pos))
		for idx, p := range *pos {
			offset := Offset{int32(p), int32(p + 1)}
			charOffsets[idx] = offset
		}
		sort.Sort(ByOrder(charOffsets))
	}
	var maxe int
	for _, offset := range charOffsets {
		maxe = util.Max(maxe, int(offset[1]))
	}

	offsets := result.colorOffsets(charOffsets, t.theme, colBase, colMatch, current)
	maxWidth := t.window.Width() - (t.pointerLen + t.markerLen + 1)
	ellipsis, ellipsisWidth := util.Truncate(t.ellipsis, maxWidth/2)
	maxe = util.Constrain(maxe+util.Min(maxWidth/2-ellipsisWidth, t.hscrollOff), 0, len(text))
	displayWidth := t.displayWidthWithLimit(text, 0, maxWidth)
	if displayWidth > maxWidth {
		transformOffsets := func(diff int32, rightTrim bool) {
			for idx, offset := range offsets {
				b, e := offset.offset[0], offset.offset[1]
				el := int32(len(ellipsis))
				b += el - diff
				e += el - diff
				b = util.Max32(b, el)
				if rightTrim {
					e = util.Min32(e, int32(maxWidth-ellipsisWidth))
				}
				offsets[idx].offset[0] = b
				offsets[idx].offset[1] = util.Max32(b, e)
			}
		}
		if t.hscroll {
			if t.keepRight && pos == nil {
				trimmed, diff := t.trimLeft(text, maxWidth-ellipsisWidth)
				transformOffsets(diff, false)
				text = append(ellipsis, trimmed...)
			} else if !t.overflow(text[:maxe], maxWidth-ellipsisWidth) {
				// Stri..
				text, _ = t.trimRight(text, maxWidth-ellipsisWidth)
				text = append(text, ellipsis...)
			} else {
				// Stri..
				rightTrim := false
				if t.overflow(text[maxe:], ellipsisWidth) {
					text = append(text[:maxe], ellipsis...)
					rightTrim = true
				}
				// ..ri..
				var diff int32
				text, diff = t.trimLeft(text, maxWidth-ellipsisWidth)

				// Transform offsets
				transformOffsets(diff, rightTrim)
				text = append(ellipsis, text...)
			}
		} else {
			text, _ = t.trimRight(text, maxWidth-ellipsisWidth)
			text = append(text, ellipsis...)

			for idx, offset := range offsets {
				offsets[idx].offset[0] = util.Min32(offset.offset[0], int32(maxWidth-len(ellipsis)))
				offsets[idx].offset[1] = util.Min32(offset.offset[1], int32(maxWidth))
			}
		}
		displayWidth = t.displayWidthWithLimit(text, 0, displayWidth)
	}

	t.printColoredString(t.window, text, offsets, colBase)
	return displayWidth
}

func (t *Terminal) printColoredString(window tui.Window, text []rune, offsets []colorOffset, colBase tui.ColorPair) {
	var index int32
	var substr string
	var prefixWidth int
	maxOffset := int32(len(text))
	for _, offset := range offsets {
		b := util.Constrain32(offset.offset[0], index, maxOffset)
		e := util.Constrain32(offset.offset[1], index, maxOffset)

		substr, prefixWidth = t.processTabs(text[index:b], prefixWidth)
		window.CPrint(colBase, substr)

		if b < e {
			substr, prefixWidth = t.processTabs(text[b:e], prefixWidth)
			window.CPrint(offset.color, substr)
		}

		index = e
		if index >= maxOffset {
			break
		}
	}
	if index < maxOffset {
		substr, _ = t.processTabs(text[index:], prefixWidth)
		window.CPrint(colBase, substr)
	}
}

func (t *Terminal) renderPreviewSpinner() {
	numLines := len(t.previewer.lines)
	spin := t.previewer.spinner
	if len(spin) > 0 || t.previewer.scrollable {
		maxWidth := t.pwindow.Width()
		if !t.previewer.scrollable {
			if maxWidth > 0 {
				t.pwindow.Move(0, maxWidth-1)
				t.pwindow.CPrint(tui.ColPreviewSpinner, spin)
			}
		} else {
			offsetString := fmt.Sprintf("%d/%d", t.previewer.offset+1, numLines)
			if len(spin) > 0 {
				spin += " "
				maxWidth -= 2
			}
			offsetRunes, _ := t.trimRight([]rune(offsetString), maxWidth)
			pos := maxWidth - t.displayWidth(offsetRunes)
			t.pwindow.Move(0, pos)
			if maxWidth > 0 {
				t.pwindow.CPrint(tui.ColPreviewSpinner, spin)
				t.pwindow.CPrint(tui.ColInfo.WithAttr(tui.Reverse), string(offsetRunes))
			}
		}
	}
}

func (t *Terminal) renderPreviewArea(unchanged bool) {
	if unchanged {
		t.pwindow.MoveAndClear(0, 0) // Clear scroll offset display
	} else {
		t.previewed.filled = false
		t.pwindow.Erase()
	}

	height := t.pwindow.Height()
	header := []string{}
	body := t.previewer.lines
	headerLines := t.previewOpts.headerLines
	// Do not enable preview header lines if it's value is too large
	if headerLines > 0 && headerLines < util.Min(len(body), height) {
		header = t.previewer.lines[0:headerLines]
		body = t.previewer.lines[headerLines:]
		// Always redraw header
		t.renderPreviewText(height, header, 0, false)
		t.pwindow.MoveAndClear(t.pwindow.Y(), 0)
	}
	t.renderPreviewText(height, body, -t.previewer.offset+headerLines, unchanged)

	if !unchanged {
		t.pwindow.FinishFill()
	}

	if len(t.scrollbar) == 0 {
		return
	}

	effectiveHeight := height - headerLines
	barLength, barStart := getScrollbar(len(body), effectiveHeight, util.Min(len(body)-effectiveHeight, t.previewer.offset-headerLines))
	t.renderPreviewScrollbar(headerLines, barLength, barStart)
}

func (t *Terminal) renderPreviewText(height int, lines []string, lineNo int, unchanged bool) {
	maxWidth := t.pwindow.Width()
	var ansi *ansiState
	spinnerRedraw := t.pwindow.Y() == 0
	for _, line := range lines {
		var lbg tui.Color = -1
		if ansi != nil {
			ansi.lbg = -1
		}
		line = strings.TrimRight(line, "\r\n")
		if lineNo >= height || t.pwindow.Y() == height-1 && t.pwindow.X() > 0 {
			t.previewed.filled = true
			t.previewer.scrollable = true
			break
		} else if lineNo >= 0 {
			if spinnerRedraw && lineNo > 0 {
				spinnerRedraw = false
				y := t.pwindow.Y()
				x := t.pwindow.X()
				t.renderPreviewSpinner()
				t.pwindow.Move(y, x)
			}
			var fillRet tui.FillReturn
			prefixWidth := 0
			_, _, ansi = extractColor(line, ansi, func(str string, ansi *ansiState) bool {
				trimmed := []rune(str)
				isTrimmed := false
				if !t.previewOpts.wrap {
					trimmed, isTrimmed = t.trimRight(trimmed, maxWidth-t.pwindow.X())
				}
				str, width := t.processTabs(trimmed, prefixWidth)
				if width > prefixWidth {
					prefixWidth = width
					if t.theme.Colored && ansi != nil && ansi.colored() {
						lbg = ansi.lbg
						fillRet = t.pwindow.CFill(ansi.fg, ansi.bg, ansi.attr, str)
					} else {
						fillRet = t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), tui.AttrRegular, str)
					}
				}
				return !isTrimmed &&
					(fillRet == tui.FillContinue || t.previewOpts.wrap && fillRet == tui.FillNextLine)
			})
			t.previewer.scrollable = t.previewer.scrollable || t.pwindow.Y() == height-1 && t.pwindow.X() == t.pwindow.Width()
			if fillRet == tui.FillNextLine {
				continue
			} else if fillRet == tui.FillSuspend {
				t.previewed.filled = true
				break
			}
			if unchanged && lineNo == 0 {
				break
			}
			if lbg >= 0 {
				t.pwindow.CFill(-1, lbg, tui.AttrRegular,
					strings.Repeat(" ", t.pwindow.Width()-t.pwindow.X())+"\n")
			} else {
				t.pwindow.Fill("\n")
			}
		}
		lineNo++
	}
}

func (t *Terminal) renderPreviewScrollbar(yoff int, barLength int, barStart int) {
	height := t.pwindow.Height()
	w := t.pborder.Width()
	redraw := false
	if len(t.previewer.bar) != height {
		redraw = true
		t.previewer.bar = make([]bool, height)
	}
	xshift := -1 - t.borderWidth
	if !t.previewOpts.border.HasRight() {
		xshift = -1
	}
	yshift := 1
	if !t.previewOpts.border.HasTop() {
		yshift = 0
	}
	for i := yoff; i < height; i++ {
		x := w + xshift
		y := i + yshift

		// Avoid unnecessary redraws
		bar := i >= yoff+barStart && i < yoff+barStart+barLength
		if !redraw && bar == t.previewer.bar[i] && !t.tui.NeedScrollbarRedraw() {
			continue
		}

		t.previewer.bar[i] = bar
		t.pborder.Move(y, x)
		if i >= yoff+barStart && i < yoff+barStart+barLength {
			t.pborder.CPrint(tui.ColPreviewScrollbar, t.previewScrollbar)
		} else {
			t.pborder.CPrint(tui.ColPreviewScrollbar, " ")
		}
	}
}

func (t *Terminal) printPreview() {
	if !t.hasPreviewWindow() || t.pwindow.Height() == 0 {
		return
	}
	numLines := len(t.previewer.lines)
	height := t.pwindow.Height()
	unchanged := (t.previewed.filled || numLines == t.previewed.numLines) &&
		t.previewer.version == t.previewed.version &&
		t.previewer.offset == t.previewed.offset
	t.previewer.scrollable = t.previewer.offset > 0 || numLines > height
	t.renderPreviewArea(unchanged)
	t.renderPreviewSpinner()
	t.previewed.numLines = numLines
	t.previewed.version = t.previewer.version
	t.previewed.offset = t.previewer.offset
}

func (t *Terminal) printPreviewDelayed() {
	if !t.hasPreviewWindow() || len(t.previewer.lines) > 0 && t.previewed.version == t.previewer.version {
		return
	}

	t.previewer.scrollable = false
	t.renderPreviewArea(true)

	message := t.trimMessage("Loading ..", t.pwindow.Width())
	pos := t.pwindow.Width() - len(message)
	t.pwindow.Move(0, pos)
	t.pwindow.CPrint(tui.ColInfo.WithAttr(tui.Reverse), message)
}

func (t *Terminal) processTabs(runes []rune, prefixWidth int) (string, int) {
	var strbuf strings.Builder
	l := prefixWidth
	gr := uniseg.NewGraphemes(string(runes))
	for gr.Next() {
		rs := gr.Runes()
		str := string(rs)
		var w int
		if len(rs) == 1 && rs[0] == '\t' {
			w = t.tabstop - l%t.tabstop
			strbuf.WriteString(strings.Repeat(" ", w))
		} else {
			w = util.StringWidth(str)
			strbuf.WriteString(str)
		}
		l += w
	}
	return strbuf.String(), l
}

func (t *Terminal) printAll() {
	t.resizeWindows(false)
	t.printList()
	t.printPrompt()
	t.printInfo()
	t.printHeader()
	t.printPreview()
}

func (t *Terminal) refresh() {
	t.placeCursor()
	if !t.suppress {
		windows := make([]tui.Window, 0, 4)
		if t.borderShape != tui.BorderNone {
			windows = append(windows, t.border)
		}
		if t.hasPreviewWindow() {
			if t.pborder != nil {
				windows = append(windows, t.pborder)
			}
			windows = append(windows, t.pwindow)
		}
		windows = append(windows, t.window)
		t.tui.RefreshWindows(windows)
	}
}

func (t *Terminal) delChar() bool {
	if len(t.input) > 0 && t.cx < len(t.input) {
		t.input = append(t.input[:t.cx], t.input[t.cx+1:]...)
		return true
	}
	return false
}

func findLastMatch(pattern string, str string) int {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return -1
	}
	locs := rx.FindAllStringIndex(str, -1)
	if locs == nil {
		return -1
	}
	prefix := []rune(str[:locs[len(locs)-1][0]])
	return len(prefix)
}

func findFirstMatch(pattern string, str string) int {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return -1
	}
	loc := rx.FindStringIndex(str)
	if loc == nil {
		return -1
	}
	prefix := []rune(str[:loc[0]])
	return len(prefix)
}

func copySlice(slice []rune) []rune {
	ret := make([]rune, len(slice))
	copy(ret, slice)
	return ret
}

func (t *Terminal) rubout(pattern string) {
	pcx := t.cx
	after := t.input[t.cx:]
	t.cx = findLastMatch(pattern, string(t.input[:t.cx])) + 1
	t.yanked = copySlice(t.input[t.cx:pcx])
	t.input = append(t.input[:t.cx], after...)
}

func keyMatch(key tui.Event, event tui.Event) bool {
	return event.Type == key.Type && event.Char == key.Char ||
		key.Type == tui.DoubleClick && event.Type == tui.Mouse && event.MouseEvent.Double
}

func parsePlaceholder(match string) (bool, string, placeholderFlags) {
	flags := placeholderFlags{}

	if match[0] == '\\' {
		// Escaped placeholder pattern
		return true, match[1:], flags
	}

	skipChars := 1
	for _, char := range match[1:] {
		switch char {
		case '+':
			flags.plus = true
			skipChars++
		case 's':
			flags.preserveSpace = true
			skipChars++
		case 'n':
			flags.number = true
			skipChars++
		case 'f':
			flags.file = true
			skipChars++
		case 'q':
			flags.query = true
			// query flag is not skipped
		default:
			break
		}
	}

	matchWithoutFlags := "{" + match[skipChars:]

	return false, matchWithoutFlags, flags
}

func hasPreviewFlags(template string) (slot bool, plus bool, query bool) {
	for _, match := range placeholder.FindAllString(template, -1) {
		_, _, flags := parsePlaceholder(match)
		if flags.plus {
			plus = true
		}
		if flags.query {
			query = true
		}
		slot = true
	}
	return
}

func writeTemporaryFile(data []string, printSep string) string {
	f, err := os.CreateTemp("", "fzf-preview-*")
	if err != nil {
		errorExit("Unable to create temporary file")
	}
	defer f.Close()

	f.WriteString(strings.Join(data, printSep))
	f.WriteString(printSep)
	activeTempFiles = append(activeTempFiles, f.Name())
	return f.Name()
}

func cleanTemporaryFiles() {
	for _, filename := range activeTempFiles {
		os.Remove(filename)
	}
	activeTempFiles = []string{}
}

func (t *Terminal) replacePlaceholder(template string, forcePlus bool, input string, list []*Item) string {
	return replacePlaceholder(
		template, t.ansi, t.delimiter, t.printsep, forcePlus, input, list)
}

func (t *Terminal) evaluateScrollOffset() int {
	if t.pwindow == nil {
		return 0
	}

	// We only need the current item to calculate the scroll offset
	offsetExpr := offsetTrimCharsRegex.ReplaceAllString(
		t.replacePlaceholder(t.previewOpts.scroll, false, "", []*Item{t.currentItem(), nil}), "")

	atoi := func(s string) int {
		n, e := strconv.Atoi(s)
		if e != nil {
			return 0
		}
		return n
	}

	base := -1
	height := util.Max(0, t.pwindow.Height()-t.previewOpts.headerLines)
	for _, component := range offsetComponentRegex.FindAllString(offsetExpr, -1) {
		if strings.HasPrefix(component, "-/") {
			component = component[1:]
		}
		if component[0] == '/' {
			denom := atoi(component[1:])
			if denom != 0 {
				base -= height / denom
			}
			break
		}
		base += atoi(component)
	}
	return util.Max(0, base)
}

func replacePlaceholder(template string, stripAnsi bool, delimiter Delimiter, printsep string, forcePlus bool, query string, allItems []*Item) string {
	current := allItems[:1]
	selected := allItems[1:]
	if current[0] == nil {
		current = []*Item{}
	}
	if selected[0] == nil {
		selected = []*Item{}
	}

	// replace placeholders one by one
	return placeholder.ReplaceAllStringFunc(template, func(match string) string {
		escaped, match, flags := parsePlaceholder(match)

		// this function implements the effects a placeholder has on items
		var replace func(*Item) string

		// placeholder types (escaped, query type, item type, token type)
		switch {
		case escaped:
			return match
		case match == "{q}":
			return quoteEntry(query)
		case match == "{}":
			replace = func(item *Item) string {
				switch {
				case flags.number:
					n := int(item.text.Index)
					if n < 0 {
						return ""
					}
					return strconv.Itoa(n)
				case flags.file:
					return item.AsString(stripAnsi)
				default:
					return quoteEntry(item.AsString(stripAnsi))
				}
			}
		default:
			// token type and also failover (below)
			rangeExpressions := strings.Split(match[1:len(match)-1], ",")
			ranges := make([]Range, len(rangeExpressions))
			for idx, s := range rangeExpressions {
				r, ok := ParseRange(&s) // ellipsis (x..y) and shorthand (x..x) range syntax
				if !ok {
					// Invalid expression, just return the original string in the template
					return match
				}
				ranges[idx] = r
			}

			replace = func(item *Item) string {
				tokens := Tokenize(item.AsString(stripAnsi), delimiter)
				trans := Transform(tokens, ranges)
				str := joinTokens(trans)

				// trim the last delimiter
				if delimiter.str != nil {
					str = strings.TrimSuffix(str, *delimiter.str)
				} else if delimiter.regex != nil {
					delims := delimiter.regex.FindAllStringIndex(str, -1)
					// make sure the delimiter is at the very end of the string
					if len(delims) > 0 && delims[len(delims)-1][1] == len(str) {
						str = str[:delims[len(delims)-1][0]]
					}
				}

				if !flags.preserveSpace {
					str = strings.TrimSpace(str)
				}
				if !flags.file {
					str = quoteEntry(str)
				}
				return str
			}
		}

		// apply 'replace' function over proper set of items and return result

		items := current
		if flags.plus || forcePlus {
			items = selected
		}
		replacements := make([]string, len(items))

		for idx, item := range items {
			replacements[idx] = replace(item)
		}

		if flags.file {
			return writeTemporaryFile(replacements, printsep)
		}
		return strings.Join(replacements, " ")
	})
}

func (t *Terminal) redraw() {
	t.tui.Clear()
	t.tui.Refresh()
	t.printAll()
}

func (t *Terminal) executeCommand(template string, forcePlus bool, background bool, capture bool, firstLineOnly bool) string {
	line := ""
	valid, list := t.buildPlusList(template, forcePlus)
	// 'capture' is used for transform-* and we don't want to
	// return an empty string in those cases
	if !valid && !capture {
		return line
	}
	command := t.replacePlaceholder(template, forcePlus, string(t.input), list)
	cmd := util.ExecCommand(command, false)
	cmd.Env = t.environ()
	t.executing.Set(true)
	if !background {
		cmd.Stdin = tui.TtyIn()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		t.tui.Pause(true)
		cmd.Run()
		t.tui.Resume(true, false)
		t.redraw()
		t.refresh()
	} else {
		if capture {
			out, _ := cmd.StdoutPipe()
			reader := bufio.NewReader(out)
			cmd.Start()
			if firstLineOnly {
				line, _ = reader.ReadString('\n')
				line = strings.TrimRight(line, "\r\n")
			} else {
				bytes, _ := io.ReadAll(reader)
				line = string(bytes)
			}
			cmd.Wait()
		} else {
			cmd.Run()
		}
	}
	t.executing.Set(false)
	cleanTemporaryFiles()
	return line
}

func (t *Terminal) hasPreviewer() bool {
	return t.previewBox != nil
}

func (t *Terminal) needPreviewWindow() bool {
	return t.hasPreviewer() && len(t.previewOpts.command) > 0 && t.previewOpts.Visible()
}

// Check if previewer is currently in action (invisible previewer with size 0 or visible previewer)
func (t *Terminal) canPreview() bool {
	return t.hasPreviewer() && (!t.previewOpts.Visible() && !t.previewOpts.hidden || t.hasPreviewWindow())
}

func (t *Terminal) hasPreviewWindow() bool {
	return t.pwindow != nil
}

func (t *Terminal) currentItem() *Item {
	cnt := t.merger.Length()
	if t.cy >= 0 && cnt > 0 && cnt > t.cy {
		return t.merger.Get(t.cy).item
	}
	return nil
}

func (t *Terminal) buildPlusList(template string, forcePlus bool) (bool, []*Item) {
	current := t.currentItem()
	slot, plus, query := hasPreviewFlags(template)
	if !(!slot || query || (forcePlus || plus) && len(t.selected) > 0) {
		return current != nil, []*Item{current, current}
	}

	// We would still want to update preview window even if there is no match if
	//   1. the command template contains {q}
	//   2. or it contains {+} and we have more than one item already selected.
	// To do so, we pass an empty Item instead of nil to trigger an update.
	if current == nil {
		current = &minItem
	}

	var sels []*Item
	if len(t.selected) == 0 {
		sels = []*Item{current, current}
	} else {
		sels = make([]*Item, len(t.selected)+1)
		sels[0] = current
		for i, sel := range t.sortSelected() {
			sels[i+1] = sel.item
		}
	}
	return true, sels
}

func (t *Terminal) selectItem(item *Item) bool {
	if len(t.selected) >= t.multi {
		return false
	}
	if _, found := t.selected[item.Index()]; found {
		return true
	}

	t.selected[item.Index()] = selectedItem{time.Now(), item}
	t.version++

	return true
}

func (t *Terminal) selectItemChanged(item *Item) bool {
	if _, found := t.selected[item.Index()]; found {
		return false
	}
	return t.selectItem(item)
}

func (t *Terminal) deselectItem(item *Item) {
	delete(t.selected, item.Index())
	t.version++
}

func (t *Terminal) deselectItemChanged(item *Item) bool {
	if _, found := t.selected[item.Index()]; found {
		t.deselectItem(item)
		return true
	}
	return false
}

func (t *Terminal) toggleItem(item *Item) bool {
	if _, found := t.selected[item.Index()]; !found {
		return t.selectItem(item)
	}
	t.deselectItem(item)
	return true
}

func (t *Terminal) killPreview(code int) {
	select {
	case t.killChan <- code:
	default:
		if code != exitCancel {
			t.eventBox.Set(EvtQuit, code)
		}
	}
}

func (t *Terminal) cancelPreview() {
	t.killPreview(exitCancel)
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
	// prof := profile.Start(profile.ProfilePath("/tmp/"))
	fitpad := <-t.startChan
	fit := fitpad.fit
	if fit >= 0 {
		pad := fitpad.pad
		t.tui.Resize(func(termHeight int) int {
			contentHeight := fit + t.extraLines()
			if t.needPreviewWindow() {
				if t.previewOpts.aboveOrBelow() {
					if t.previewOpts.size.percent {
						newContentHeight := int(float64(contentHeight) * 100. / (100. - t.previewOpts.size.size))
						contentHeight = util.Max(contentHeight+1+borderLines(t.previewOpts.border), newContentHeight)
					} else {
						contentHeight += int(t.previewOpts.size.size) + borderLines(t.previewOpts.border)
					}
				} else {
					// Minimum height if preview window can appear
					contentHeight = util.Max(contentHeight, 1+borderLines(t.previewOpts.border))
				}
			}
			return util.Min(termHeight, contentHeight+pad)
		})
	}
	{ // Late initialization
		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			for s := range intChan {
				// Don't quit by SIGINT while executing because it should be for the executing command and not for fzf itself
				if !(s == os.Interrupt && t.executing.Get()) {
					t.reqBox.Set(reqQuit, nil)
				}
			}
		}()

		contChan := make(chan os.Signal, 1)
		notifyOnCont(contChan)
		go func() {
			for {
				<-contChan
				t.reqBox.Set(reqReinit, nil)
			}
		}()

		resizeChan := make(chan os.Signal, 1)
		notifyOnResize(resizeChan) // Non-portable
		go func() {
			for {
				<-resizeChan
				t.reqBox.Set(reqFullRedraw, nil)
			}
		}()

		t.mutex.Lock()
		t.initFunc()
		t.resizeWindows(false)
		t.printPrompt()
		t.printInfo()
		t.printHeader()
		t.refresh()
		t.mutex.Unlock()
		go func() {
			timer := time.NewTimer(t.initDelay)
			<-timer.C
			t.reqBox.Set(reqRefresh, nil)
		}()

		// Keep the spinner spinning
		go func() {
			for {
				t.mutex.Lock()
				reading := t.reading
				t.mutex.Unlock()
				time.Sleep(spinnerDuration)
				if reading {
					t.reqBox.Set(reqInfo, nil)
				}
			}
		}()
	}

	if t.hasPreviewer() {
		go func() {
			var version int64
			for {
				var items []*Item
				var commandTemplate string
				var pwindow tui.Window
				initialOffset := 0
				t.previewBox.Wait(func(events *util.Events) {
					for req, value := range *events {
						switch req {
						case reqPreviewEnqueue:
							request := value.(previewRequest)
							commandTemplate = request.template
							initialOffset = request.scrollOffset
							items = request.list
							pwindow = request.pwindow
						}
					}
					events.Clear()
				})
				version++
				// We don't display preview window if no match
				if items[0] != nil {
					_, query := t.Input()
					command := t.replacePlaceholder(commandTemplate, false, string(query), items)
					cmd := util.ExecCommand(command, true)
					env := t.environ()
					if pwindow != nil {
						height := pwindow.Height()
						lines := fmt.Sprintf("LINES=%d", height)
						columns := fmt.Sprintf("COLUMNS=%d", pwindow.Width())
						env = append(env, lines)
						env = append(env, "FZF_PREVIEW_"+lines)
						env = append(env, columns)
						env = append(env, "FZF_PREVIEW_"+columns)
					}
					cmd.Env = env

					out, _ := cmd.StdoutPipe()
					cmd.Stderr = cmd.Stdout
					reader := bufio.NewReader(out)
					eofChan := make(chan bool)
					finishChan := make(chan bool, 1)
					err := cmd.Start()
					if err == nil {
						reapChan := make(chan bool)
						lineChan := make(chan eachLine)
						// Goroutine 1 reads process output
						go func() {
							for {
								line, err := reader.ReadString('\n')
								lineChan <- eachLine{line, err}
								if err != nil {
									break
								}
							}
							eofChan <- true
						}()

						// Goroutine 2 periodically requests rendering
						rendered := util.NewAtomicBool(false)
						go func(version int64) {
							lines := []string{}
							spinner := makeSpinner(t.unicode)
							spinnerIndex := -1 // Delay initial rendering by an extra tick
							ticker := time.NewTicker(previewChunkDelay)
							offset := initialOffset
						Loop:
							for {
								select {
								case <-ticker.C:
									if len(lines) > 0 && len(lines) >= initialOffset {
										if spinnerIndex >= 0 {
											spin := spinner[spinnerIndex%len(spinner)]
											t.reqBox.Set(reqPreviewDisplay, previewResult{version, lines, offset, spin})
											rendered.Set(true)
											offset = -1
										}
										spinnerIndex++
									}
								case eachLine := <-lineChan:
									line := eachLine.line
									err := eachLine.err
									if len(line) > 0 {
										clearIndex := strings.Index(line, clearCode)
										if clearIndex >= 0 {
											lines = []string{}
											line = line[clearIndex+len(clearCode):]
											version--
											offset = 0
										}
										lines = append(lines, line)
									}
									if err != nil {
										t.reqBox.Set(reqPreviewDisplay, previewResult{version, lines, offset, ""})
										rendered.Set(true)
										break Loop
									}
								}
							}
							ticker.Stop()
							reapChan <- true
						}(version)

						// Goroutine 3 is responsible for cancelling running preview command
						go func(version int64) {
							timer := time.NewTimer(previewDelayed)
						Loop:
							for {
								select {
								case <-timer.C:
									t.reqBox.Set(reqPreviewDelayed, version)
								case code := <-t.killChan:
									if code != exitCancel {
										util.KillCommand(cmd)
										t.eventBox.Set(EvtQuit, code)
									} else {
										// We can immediately kill a long-running preview program
										// once we started rendering its partial output
										delay := previewCancelWait
										if rendered.Get() {
											delay = 0
										}
										timer := time.NewTimer(delay)
										select {
										case <-timer.C:
											util.KillCommand(cmd)
										case <-finishChan:
										}
										timer.Stop()
									}
									break Loop
								case <-finishChan:
									break Loop
								}
							}
							timer.Stop()
							reapChan <- true
						}(version)

						<-eofChan          // Goroutine 1 finished
						cmd.Wait()         // NOTE: We should not call Wait before EOF
						finishChan <- true // Tell Goroutine 3 to stop
						<-reapChan         // Goroutine 2 and 3 finished
						<-reapChan
					} else {
						// Failed to start the command. Report the error immediately.
						t.reqBox.Set(reqPreviewDisplay, previewResult{version, []string{err.Error()}, 0, ""})
					}

					cleanTemporaryFiles()
				} else {
					t.reqBox.Set(reqPreviewDisplay, previewResult{version, nil, 0, ""})
				}
			}
		}()
	}

	refreshPreview := func(command string) {
		if len(command) > 0 && t.canPreview() {
			_, list := t.buildPlusList(command, false)
			t.cancelPreview()
			t.previewBox.Set(reqPreviewEnqueue, previewRequest{command, t.pwindow, t.evaluateScrollOffset(), list})
		}
	}

	go func() {
		var focusedIndex int32 = minItem.Index()
		var version int64 = -1
		running := true
		code := exitError
		exit := func(getCode func() int) {
			t.tui.Close()
			code = getCode()
			if code <= exitNoMatch && t.history != nil {
				t.history.append(string(t.input))
			}
			running = false
			t.mutex.Unlock()
		}

		for running {
			t.reqBox.Wait(func(events *util.Events) {
				defer events.Clear()
				t.mutex.Lock()
				for req, value := range *events {
					switch req {
					case reqPrompt:
						t.printPrompt()
						if t.noInfoLine() {
							t.printInfo()
						}
					case reqInfo:
						t.printInfo()
					case reqList:
						t.printList()
						var currentIndex int32 = minItem.Index()
						currentItem := t.currentItem()
						if currentItem != nil {
							currentIndex = currentItem.Index()
						}
						focusChanged := focusedIndex != currentIndex
						if focusChanged && t.track == trackCurrent {
							t.track = trackDisabled
							t.printInfo()
						}
						if onFocus, prs := t.keymap[tui.Focus.AsEvent()]; prs && focusChanged {
							t.serverChan <- onFocus
						}
						if focusChanged || version != t.version {
							version = t.version
							focusedIndex = currentIndex
							refreshPreview(t.previewOpts.command)
						}
					case reqJump:
						if t.merger.Length() == 0 {
							t.jumping = jumpDisabled
						}
						t.printList()
					case reqHeader:
						t.printHeader()
					case reqRefresh:
						t.suppress = false
					case reqRedrawBorderLabel:
						t.printLabel(t.border, t.borderLabel, t.borderLabelOpts, t.borderLabelLen, t.borderShape, true)
					case reqRedrawPreviewLabel:
						t.printLabel(t.pborder, t.previewLabel, t.previewLabelOpts, t.previewLabelLen, t.previewOpts.border, true)
					case reqReinit:
						t.tui.Resume(t.fullscreen, t.sigstop)
						t.redraw()
					case reqFullRedraw:
						wasHidden := t.pwindow == nil
						t.redraw()
						if wasHidden && t.hasPreviewWindow() {
							refreshPreview(t.previewOpts.command)
						}
					case reqClose:
						exit(func() int {
							if t.output() {
								return exitOk
							}
							return exitNoMatch
						})
						return
					case reqPreviewDisplay:
						result := value.(previewResult)
						if t.previewer.version != result.version {
							t.previewer.version = result.version
							t.previewer.following.Force(t.previewOpts.follow)
							if t.previewer.following.Enabled() {
								t.previewer.offset = 0
							}
						}
						t.previewer.lines = result.lines
						t.previewer.spinner = result.spinner
						if t.previewer.following.Enabled() {
							t.previewer.offset = util.Max(t.previewer.offset, len(t.previewer.lines)-(t.pwindow.Height()-t.previewOpts.headerLines))
						} else if result.offset >= 0 {
							t.previewer.offset = util.Constrain(result.offset, t.previewOpts.headerLines, len(t.previewer.lines)-1)
						}
						t.printPreview()
					case reqPreviewRefresh:
						t.printPreview()
					case reqPreviewDelayed:
						t.previewer.version = value.(int64)
						t.printPreviewDelayed()
					case reqPrintQuery:
						exit(func() int {
							t.printer(string(t.input))
							return exitOk
						})
						return
					case reqQuit:
						exit(func() int { return exitInterrupt })
						return
					}
				}
				t.refresh()
				t.mutex.Unlock()
			})
		}
		// prof.Stop()
		t.killPreview(code)
	}()

	looping := true
	_, startEvent := t.keymap[tui.Start.AsEvent()]

	needBarrier := true
	barrier := make(chan bool)
	go func() {
		for {
			<-barrier
			t.eventChan <- t.tui.GetChar()
		}
	}()
	previewDraggingPos := -1
	barDragging := false
	pbarDragging := false
	wasDown := false
	for looping {
		var newCommand *string
		var reloadSync bool
		changed := false
		beof := false
		queryChanged := false

		var event tui.Event
		actions := []*action{}
		if startEvent {
			event = tui.Start.AsEvent()
			startEvent = false
		} else {
			if needBarrier {
				barrier <- true
			}
			select {
			case event = <-t.eventChan:
				needBarrier = !event.Is(tui.Load, tui.One, tui.Zero)
			case actions = <-t.serverChan:
				event = tui.Invalid.AsEvent()
				needBarrier = false
			}
		}

		t.mutex.Lock()
		previousInput := t.input
		previousCx := t.cx
		events := []util.EventType{}
		req := func(evts ...util.EventType) {
			for _, event := range evts {
				events = append(events, event)
				if event == reqClose || event == reqQuit {
					looping = false
				}
			}
		}
		updatePreviewWindow := func(forcePreview bool) {
			t.resizeWindows(forcePreview)
			req(reqPrompt, reqList, reqInfo, reqHeader)
		}
		toggle := func() bool {
			current := t.currentItem()
			if current != nil && t.toggleItem(current) {
				req(reqInfo)
				return true
			}
			return false
		}
		scrollPreviewTo := func(newOffset int) {
			if !t.previewer.scrollable {
				return
			}
			numLines := len(t.previewer.lines)
			headerLines := t.previewOpts.headerLines
			if t.previewOpts.cycle {
				offsetRange := numLines - headerLines
				newOffset = ((newOffset-headerLines)+offsetRange)%offsetRange + headerLines
			}
			newOffset = util.Constrain(newOffset, headerLines, numLines-1)
			if t.previewer.offset != newOffset {
				t.previewer.offset = newOffset
				t.previewer.following.Set(t.previewer.offset >= numLines-(t.pwindow.Height()-headerLines))
				req(reqPreviewRefresh)
			}
		}
		scrollPreviewBy := func(amount int) {
			scrollPreviewTo(t.previewer.offset + amount)
		}
		for key, ret := range t.expect {
			if keyMatch(key, event) {
				t.pressed = ret
				t.reqBox.Set(reqClose, nil)
				t.mutex.Unlock()
				return
			}
		}

		actionsFor := func(eventType tui.EventType) []*action {
			return t.keymap[eventType.AsEvent()]
		}

		var doAction func(*action) bool
		doActions := func(actions []*action) bool {
			for _, action := range actions {
				if !doAction(action) {
					return false
				}
			}
			return true
		}
		doAction = func(a *action) bool {
			switch a.t {
			case actIgnore:
			case actBecome:
				valid, list := t.buildPlusList(a.a, false)
				if valid {
					command := t.replacePlaceholder(a.a, false, string(t.input), list)
					shell := os.Getenv("SHELL")
					if len(shell) == 0 {
						shell = "sh"
					}
					shellPath, err := exec.LookPath(shell)
					if err == nil {
						t.tui.Close()
						if t.history != nil {
							t.history.append(string(t.input))
						}
						/*
							FIXME: It is not at all clear why this is required.
							The following command will report 'not a tty', unless we open
							/dev/tty *twice* after closing the standard input for 'reload'
							in Reader.terminate().
								: | fzf --bind 'start:reload:ls' --bind 'enter:become:tty'
						*/
						tui.TtyIn()
						util.SetStdin(tui.TtyIn())
						syscall.Exec(shellPath, []string{shell, "-c", command}, os.Environ())
					}
				}
			case actExecute, actExecuteSilent:
				t.executeCommand(a.a, false, a.t == actExecuteSilent, false, false)
			case actExecuteMulti:
				t.executeCommand(a.a, true, false, false, false)
			case actInvalid:
				t.mutex.Unlock()
				return false
			case actTogglePreview, actShowPreview, actHidePreview:
				var act bool
				switch a.t {
				case actShowPreview:
					act = !t.hasPreviewWindow() && len(t.previewOpts.command) > 0
				case actHidePreview:
					act = t.hasPreviewWindow()
				case actTogglePreview:
					act = t.hasPreviewWindow() || len(t.previewOpts.command) > 0
				}
				if act {
					t.activePreviewOpts.Toggle()
					updatePreviewWindow(false)
					if t.canPreview() {
						valid, list := t.buildPlusList(t.previewOpts.command, false)
						if valid {
							t.cancelPreview()
							t.previewBox.Set(reqPreviewEnqueue,
								previewRequest{t.previewOpts.command, t.pwindow, t.evaluateScrollOffset(), list})
						}
					}
				}
			case actTogglePreviewWrap:
				if t.hasPreviewWindow() {
					t.previewOpts.wrap = !t.previewOpts.wrap
					// Reset preview version so that full redraw occurs
					t.previewed.version = 0
					req(reqPreviewRefresh)
				}
			case actTransformPrompt:
				prompt := t.executeCommand(a.a, false, true, true, true)
				t.prompt, t.promptLen = t.parsePrompt(prompt)
				req(reqPrompt)
			case actTransformQuery:
				query := t.executeCommand(a.a, false, true, true, true)
				t.input = []rune(query)
				t.cx = len(t.input)
			case actToggleSort:
				t.sort = !t.sort
				changed = true
			case actPreviewTop:
				if t.hasPreviewWindow() {
					scrollPreviewTo(0)
				}
			case actPreviewBottom:
				if t.hasPreviewWindow() {
					scrollPreviewTo(len(t.previewer.lines) - t.pwindow.Height())
				}
			case actPreviewUp:
				if t.hasPreviewWindow() {
					scrollPreviewBy(-1)
				}
			case actPreviewDown:
				if t.hasPreviewWindow() {
					scrollPreviewBy(1)
				}
			case actPreviewPageUp:
				if t.hasPreviewWindow() {
					scrollPreviewBy(-t.pwindow.Height())
				}
			case actPreviewPageDown:
				if t.hasPreviewWindow() {
					scrollPreviewBy(t.pwindow.Height())
				}
			case actPreviewHalfPageUp:
				if t.hasPreviewWindow() {
					scrollPreviewBy(-t.pwindow.Height() / 2)
				}
			case actPreviewHalfPageDown:
				if t.hasPreviewWindow() {
					scrollPreviewBy(t.pwindow.Height() / 2)
				}
			case actBeginningOfLine:
				t.cx = 0
			case actBackwardChar:
				if t.cx > 0 {
					t.cx--
				}
			case actPrintQuery:
				req(reqPrintQuery)
			case actChangeQuery:
				t.input = []rune(a.a)
				t.cx = len(t.input)
			case actTransformHeader:
				header := t.executeCommand(a.a, false, true, true, false)
				if t.changeHeader(header) {
					req(reqFullRedraw)
				} else {
					req(reqHeader)
				}
			case actChangeHeader:
				if t.changeHeader(a.a) {
					req(reqFullRedraw)
				} else {
					req(reqHeader)
				}
			case actChangeBorderLabel:
				if t.border != nil {
					t.borderLabel, t.borderLabelLen = t.ansiLabelPrinter(a.a, &tui.ColBorderLabel, false)
					req(reqRedrawBorderLabel)
				}
			case actChangePreviewLabel:
				if t.pborder != nil {
					t.previewLabel, t.previewLabelLen = t.ansiLabelPrinter(a.a, &tui.ColPreviewLabel, false)
					req(reqRedrawPreviewLabel)
				}
			case actTransformBorderLabel:
				if t.border != nil {
					label := t.executeCommand(a.a, false, true, true, true)
					t.borderLabel, t.borderLabelLen = t.ansiLabelPrinter(label, &tui.ColBorderLabel, false)
					req(reqRedrawBorderLabel)
				}
			case actTransformPreviewLabel:
				if t.pborder != nil {
					label := t.executeCommand(a.a, false, true, true, true)
					t.previewLabel, t.previewLabelLen = t.ansiLabelPrinter(label, &tui.ColPreviewLabel, false)
					req(reqRedrawPreviewLabel)
				}
			case actChangePrompt:
				t.prompt, t.promptLen = t.parsePrompt(a.a)
				req(reqPrompt)
			case actPreview:
				updatePreviewWindow(true)
				refreshPreview(a.a)
			case actRefreshPreview:
				refreshPreview(t.previewOpts.command)
			case actReplaceQuery:
				current := t.currentItem()
				if current != nil {
					t.input = current.text.ToRunes()
					t.cx = len(t.input)
				}
			case actAbort:
				req(reqQuit)
			case actDeleteChar:
				t.delChar()
			case actDeleteCharEOF:
				if !t.delChar() && t.cx == 0 {
					req(reqQuit)
				}
			case actEndOfLine:
				t.cx = len(t.input)
			case actCancel:
				if len(t.input) == 0 {
					req(reqQuit)
				} else {
					t.yanked = t.input
					t.input = []rune{}
					t.cx = 0
				}
			case actBackwardDeleteCharEOF:
				if len(t.input) == 0 {
					req(reqQuit)
				} else if t.cx > 0 {
					t.input = append(t.input[:t.cx-1], t.input[t.cx:]...)
					t.cx--
				}
			case actForwardChar:
				if t.cx < len(t.input) {
					t.cx++
				}
			case actBackwardDeleteChar:
				beof = len(t.input) == 0
				if t.cx > 0 {
					t.input = append(t.input[:t.cx-1], t.input[t.cx:]...)
					t.cx--
				}
			case actSelectAll:
				if t.multi > 0 {
					for i := 0; i < t.merger.Length(); i++ {
						if !t.selectItem(t.merger.Get(i).item) {
							break
						}
					}
					req(reqList, reqInfo)
				}
			case actDeselectAll:
				if t.multi > 0 {
					for i := 0; i < t.merger.Length() && len(t.selected) > 0; i++ {
						t.deselectItem(t.merger.Get(i).item)
					}
					req(reqList, reqInfo)
				}
			case actClose:
				if t.hasPreviewWindow() {
					t.activePreviewOpts.Toggle()
					updatePreviewWindow(false)
				} else {
					req(reqQuit)
				}
			case actSelect:
				current := t.currentItem()
				if t.multi > 0 && current != nil && t.selectItemChanged(current) {
					req(reqList, reqInfo)
				}
			case actDeselect:
				current := t.currentItem()
				if t.multi > 0 && current != nil && t.deselectItemChanged(current) {
					req(reqList, reqInfo)
				}
			case actToggle:
				if t.multi > 0 && t.merger.Length() > 0 && toggle() {
					req(reqList)
				}
			case actToggleAll:
				if t.multi > 0 {
					prevIndexes := make(map[int]struct{})
					for i := 0; i < t.merger.Length() && len(t.selected) > 0; i++ {
						item := t.merger.Get(i).item
						if _, found := t.selected[item.Index()]; found {
							prevIndexes[i] = struct{}{}
							t.deselectItem(item)
						}
					}

					for i := 0; i < t.merger.Length(); i++ {
						if _, found := prevIndexes[i]; !found {
							item := t.merger.Get(i).item
							if !t.selectItem(item) {
								break
							}
						}
					}
					req(reqList, reqInfo)
				}
			case actToggleIn:
				if t.layout != layoutDefault {
					return doAction(&action{t: actToggleUp})
				}
				return doAction(&action{t: actToggleDown})
			case actToggleOut:
				if t.layout != layoutDefault {
					return doAction(&action{t: actToggleDown})
				}
				return doAction(&action{t: actToggleUp})
			case actToggleDown:
				if t.multi > 0 && t.merger.Length() > 0 && toggle() {
					t.vmove(-1, true)
					req(reqList)
				}
			case actToggleUp:
				if t.multi > 0 && t.merger.Length() > 0 && toggle() {
					t.vmove(1, true)
					req(reqList)
				}
			case actDown:
				t.vmove(-1, true)
				req(reqList)
			case actUp:
				t.vmove(1, true)
				req(reqList)
			case actAccept:
				req(reqClose)
			case actAcceptNonEmpty:
				if len(t.selected) > 0 || t.merger.Length() > 0 || !t.reading && t.count == 0 {
					req(reqClose)
				}
			case actClearScreen:
				req(reqFullRedraw)
			case actClearQuery:
				t.input = []rune{}
				t.cx = 0
			case actClearSelection:
				if t.multi > 0 {
					t.selected = make(map[int32]selectedItem)
					t.version++
					req(reqList, reqInfo)
				}
			case actFirst:
				t.vset(0)
				req(reqList)
			case actLast:
				t.vset(t.merger.Length() - 1)
				req(reqList)
			case actPosition:
				if n, e := strconv.Atoi(a.a); e == nil {
					if n > 0 {
						n--
					} else if n < 0 {
						n += t.merger.Length()
					}
					t.vset(n)
					req(reqList)
				}
			case actPut:
				str := []rune(a.a)
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], str...), suffix...)
				t.cx += len(str)
			case actUnixLineDiscard:
				beof = len(t.input) == 0
				if t.cx > 0 {
					t.yanked = copySlice(t.input[:t.cx])
					t.input = t.input[t.cx:]
					t.cx = 0
				}
			case actUnixWordRubout:
				beof = len(t.input) == 0
				if t.cx > 0 {
					t.rubout("\\s\\S")
				}
			case actBackwardKillWord:
				beof = len(t.input) == 0
				if t.cx > 0 {
					t.rubout(t.wordRubout)
				}
			case actYank:
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], t.yanked...), suffix...)
				t.cx += len(t.yanked)
			case actPageUp:
				t.vmove(t.maxItems()-1, false)
				req(reqList)
			case actPageDown:
				t.vmove(-(t.maxItems() - 1), false)
				req(reqList)
			case actHalfPageUp:
				t.vmove(t.maxItems()/2, false)
				req(reqList)
			case actHalfPageDown:
				t.vmove(-(t.maxItems() / 2), false)
				req(reqList)
			case actJump:
				t.jumping = jumpEnabled
				req(reqJump)
			case actJumpAccept:
				t.jumping = jumpAcceptEnabled
				req(reqJump)
			case actBackwardWord:
				t.cx = findLastMatch(t.wordRubout, string(t.input[:t.cx])) + 1
			case actForwardWord:
				t.cx += findFirstMatch(t.wordNext, string(t.input[t.cx:])) + 1
			case actKillWord:
				ncx := t.cx +
					findFirstMatch(t.wordNext, string(t.input[t.cx:])) + 1
				if ncx > t.cx {
					t.yanked = copySlice(t.input[t.cx:ncx])
					t.input = append(t.input[:t.cx], t.input[ncx:]...)
				}
			case actKillLine:
				if t.cx < len(t.input) {
					t.yanked = copySlice(t.input[t.cx:])
					t.input = t.input[:t.cx]
				}
			case actRune:
				prefix := copySlice(t.input[:t.cx])
				t.input = append(append(prefix, event.Char), t.input[t.cx:]...)
				t.cx++
			case actPrevHistory:
				if t.history != nil {
					t.history.override(string(t.input))
					t.input = trimQuery(t.history.previous())
					t.cx = len(t.input)
				}
			case actNextHistory:
				if t.history != nil {
					t.history.override(string(t.input))
					t.input = trimQuery(t.history.next())
					t.cx = len(t.input)
				}
			case actToggleSearch:
				t.paused = !t.paused
				changed = !t.paused
				req(reqPrompt)
			case actToggleTrack:
				switch t.track {
				case trackEnabled:
					t.track = trackDisabled
				case trackDisabled:
					t.track = trackEnabled
				}
				req(reqInfo)
			case actToggleHeader:
				t.headerVisible = !t.headerVisible
				req(reqList, reqInfo, reqPrompt, reqHeader)
			case actTrack:
				if t.track == trackDisabled {
					t.track = trackCurrent
				}
				req(reqInfo)
			case actEnableSearch:
				t.paused = false
				changed = true
				req(reqPrompt)
			case actDisableSearch:
				t.paused = true
				req(reqPrompt)
			case actSigStop:
				p, err := os.FindProcess(os.Getpid())
				if err == nil {
					t.sigstop = true
					t.tui.Clear()
					t.tui.Pause(t.fullscreen)
					notifyStop(p)
					t.mutex.Unlock()
					return false
				}
			case actMouse:
				me := event.MouseEvent
				mx, my := me.X, me.Y
				clicked := !wasDown && me.Down
				wasDown = me.Down
				if !me.Down {
					barDragging = false
					pbarDragging = false
					previewDraggingPos = -1
				}

				// Scrolling
				if me.S != 0 {
					if t.window.Enclose(my, mx) && t.merger.Length() > 0 {
						if t.multi > 0 && me.Mod {
							toggle()
						}
						t.vmove(me.S, true)
						req(reqList)
					} else if t.hasPreviewWindow() && t.pwindow.Enclose(my, mx) {
						scrollPreviewBy(-me.S)
					}
					break
				}

				// Preview dragging
				if me.Down && (previewDraggingPos >= 0 || clicked && t.hasPreviewWindow() && t.pwindow.Enclose(my, mx)) {
					if previewDraggingPos > 0 {
						scrollPreviewBy(previewDraggingPos - my)
					}
					previewDraggingPos = my
					break
				}

				// Preview scrollbar dragging
				headerLines := t.previewOpts.headerLines
				pbarDragging = me.Down && (pbarDragging || clicked && t.hasPreviewWindow() && my >= t.pwindow.Top()+headerLines && my < t.pwindow.Top()+t.pwindow.Height() && mx == t.pwindow.Left()+t.pwindow.Width())
				if pbarDragging {
					effectiveHeight := t.pwindow.Height() - headerLines
					numLines := len(t.previewer.lines) - headerLines
					barLength, _ := getScrollbar(numLines, effectiveHeight, util.Min(numLines-effectiveHeight, t.previewer.offset-headerLines))
					if barLength > 0 {
						y := my - t.pwindow.Top() - headerLines - barLength/2
						y = util.Constrain(y, 0, effectiveHeight-barLength)
						// offset = (total - maxItems) * barStart / (maxItems - barLength)
						t.previewer.offset = headerLines + int(math.Ceil(float64(y)*float64(numLines-effectiveHeight)/float64(effectiveHeight-barLength)))
						t.previewer.following.Set(t.previewer.offset >= numLines-effectiveHeight)
						req(reqPreviewRefresh)
					}
					break
				}

				// Ignored
				if !t.window.Enclose(my, mx) && !barDragging {
					break
				}

				// Translate coordinates
				mx -= t.window.Left()
				my -= t.window.Top()
				min := 2 + t.visibleHeaderLines()
				if t.noInfoLine() {
					min--
				}
				h := t.window.Height()
				switch t.layout {
				case layoutDefault:
					my = h - my - 1
				case layoutReverseList:
					if my < h-min {
						my += min
					} else {
						my = h - my - 1
					}
				}

				// Scrollbar dragging
				barDragging = me.Down && (barDragging || clicked && my >= min && mx == t.window.Width()-1)
				if barDragging {
					barLength, barStart := t.getScrollbar()
					if barLength > 0 {
						maxItems := t.maxItems()
						if newBarStart := util.Constrain(my-min-barLength/2, 0, maxItems-barLength); newBarStart != barStart {
							total := t.merger.Length()
							prevOffset := t.offset
							// barStart = (maxItems - barLength) * t.offset / (total - maxItems)
							t.offset = int(math.Ceil(float64(newBarStart) * float64(total-maxItems) / float64(maxItems-barLength)))
							t.cy = t.offset + t.cy - prevOffset
							req(reqList)
						}
					}
					break
				}

				// Double-click on an item
				if me.Double && mx < t.window.Width()-1 {
					// Double-click
					if my >= min {
						if t.vset(t.offset+my-min) && t.cy < t.merger.Length() {
							return doActions(actionsFor(tui.DoubleClick))
						}
					}
					break
				}

				if me.Down {
					mx = util.Constrain(mx-t.promptLen, 0, len(t.input))
					if my == t.promptLine() && mx >= 0 {
						// Prompt
						t.cx = mx + t.xoffset
					} else if my >= min {
						// List
						if t.vset(t.offset+my-min) && t.multi > 0 && me.Mod {
							toggle()
						}
						req(reqList)
						if me.Left {
							return doActions(actionsFor(tui.LeftClick))
						}
						return doActions(actionsFor(tui.RightClick))
					}
				}
			case actReload, actReloadSync:
				t.failed = nil

				valid, list := t.buildPlusList(a.a, false)
				if !valid {
					// We run the command even when there's no match
					// 1. If the template doesn't have any slots
					// 2. If the template has {q}
					slot, _, query := hasPreviewFlags(a.a)
					valid = !slot || query
				}
				if valid {
					command := t.replacePlaceholder(a.a, false, string(t.input), list)
					newCommand = &command
					reloadSync = a.t == actReloadSync
					t.reading = true
				}
			case actUnbind:
				keys := parseKeyChords(a.a, "PANIC")
				for key := range keys {
					delete(t.keymap, key)
				}
			case actRebind:
				keys := parseKeyChords(a.a, "PANIC")
				for key := range keys {
					if originalAction, found := t.keymapOrg[key]; found {
						t.keymap[key] = originalAction
					}
				}
			case actChangePreview:
				if t.previewOpts.command != a.a {
					t.previewOpts.command = a.a
					updatePreviewWindow(false)
					refreshPreview(t.previewOpts.command)
				}
			case actChangePreviewWindow:
				currentPreviewOpts := t.previewOpts

				// Reset preview options and apply the additional options
				t.previewOpts = t.initialPreviewOpts

				// Split window options
				tokens := strings.Split(a.a, "|")
				if len(tokens[0]) > 0 && t.initialPreviewOpts.hidden {
					t.previewOpts.hidden = false
				}
				parsePreviewWindow(&t.previewOpts, tokens[0])
				if len(tokens) > 1 {
					a.a = strings.Join(append(tokens[1:], tokens[0]), "|")
				}

				// Full redraw
				if !currentPreviewOpts.sameLayout(t.previewOpts) {
					wasHidden := t.pwindow == nil
					updatePreviewWindow(false)
					if wasHidden && t.hasPreviewWindow() {
						refreshPreview(t.previewOpts.command)
					} else {
						req(reqPreviewRefresh)
					}
				} else if !currentPreviewOpts.sameContentLayout(t.previewOpts) {
					t.previewed.version = 0
					req(reqPreviewRefresh)
				}

				// Adjust scroll offset
				if t.hasPreviewWindow() && currentPreviewOpts.scroll != t.previewOpts.scroll {
					scrollPreviewTo(t.evaluateScrollOffset())
				}

				// Resume following
				t.previewer.following.Force(t.previewOpts.follow)
			case actNextSelected, actPrevSelected:
				if len(t.selected) > 0 {
					total := t.merger.Length()
					for i := 1; i < total; i++ {
						y := (t.cy + i) % total
						if t.layout == layoutDefault && a.t == actNextSelected ||
							t.layout != layoutDefault && a.t == actPrevSelected {
							y = (t.cy - i + total) % total
						}
						if _, found := t.selected[t.merger.Get(y).item.Index()]; found {
							t.vset(y)
							req(reqList)
							break
						}
					}
				}
			}
			return true
		}

		if t.jumping == jumpDisabled || len(actions) > 0 {
			// Break out of jump mode if any action is submitted to the server
			if t.jumping != jumpDisabled {
				t.jumping = jumpDisabled
				req(reqList)
			}
			if len(actions) == 0 {
				actions = t.keymap[event.Comparable()]
			}
			if len(actions) == 0 && event.Type == tui.Rune {
				doAction(&action{t: actRune})
			} else if !doActions(actions) {
				continue
			}
			t.truncateQuery()
			queryChanged = string(previousInput) != string(t.input)
			changed = changed || queryChanged
			if onChanges, prs := t.keymap[tui.Change.AsEvent()]; queryChanged && prs {
				if !doActions(onChanges) {
					continue
				}
			}
			if onEOFs, prs := t.keymap[tui.BackwardEOF.AsEvent()]; beof && prs {
				if !doActions(onEOFs) {
					continue
				}
			}
		} else {
			if event.Type == tui.Rune {
				if idx := strings.IndexRune(t.jumpLabels, event.Char); idx >= 0 && idx < t.maxItems() && idx < t.merger.Length() {
					t.cy = idx + t.offset
					if t.jumping == jumpAcceptEnabled {
						req(reqClose)
					}
				}
			}
			t.jumping = jumpDisabled
			req(reqList)
		}

		if queryChanged && t.canPreview() && len(t.previewOpts.command) > 0 {
			_, _, q := hasPreviewFlags(t.previewOpts.command)
			if q {
				t.version++
			}
		}

		if queryChanged || t.cx != previousCx {
			req(reqPrompt)
		}

		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if changed || newCommand != nil {
			t.eventBox.Set(EvtSearchNew, searchRequest{sort: t.sort, sync: reloadSync, command: newCommand, changed: changed})
		}
		for _, event := range events {
			t.reqBox.Set(event, nil)
		}
	}
}

func (t *Terminal) constrain() {
	// count of items to display allowed by filtering
	count := t.merger.Length()
	// count of lines can be displayed
	height := t.maxItems()

	t.cy = util.Constrain(t.cy, 0, count-1)

	minOffset := util.Max(t.cy-height+1, 0)
	maxOffset := util.Max(util.Min(count-height, t.cy), 0)
	t.offset = util.Constrain(t.offset, minOffset, maxOffset)
	if t.scrollOff == 0 {
		return
	}

	scrollOff := util.Min(height/2, t.scrollOff)
	for {
		prevOffset := t.offset
		if t.cy-t.offset < scrollOff {
			t.offset = util.Max(minOffset, t.offset-1)
		}
		if t.cy-t.offset >= height-scrollOff {
			t.offset = util.Min(maxOffset, t.offset+1)
		}
		if t.offset == prevOffset {
			break
		}
	}
}

func (t *Terminal) vmove(o int, allowCycle bool) {
	if t.layout != layoutDefault {
		o *= -1
	}
	dest := t.cy + o
	if t.cycle && allowCycle {
		max := t.merger.Length() - 1
		if dest > max {
			if t.cy == max {
				dest = 0
			}
		} else if dest < 0 {
			if t.cy == 0 {
				dest = max
			}
		}
	}
	t.vset(dest)
}

func (t *Terminal) vset(o int) bool {
	t.cy = util.Constrain(o, 0, t.merger.Length()-1)
	return t.cy == o
}

func (t *Terminal) maxItems() int {
	max := t.window.Height() - 2 - t.visibleHeaderLines()
	if t.noInfoLine() {
		max++
	}
	return util.Max(max, 0)
}
