package fzf

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"

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
	    {\+?s?f?r?RANGE(?:,RANGE)*}             # token type
	    {q[:s?RANGE]}                           # query type
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
var passThroughBeginRegex *regexp.Regexp
var passThroughEndTmuxRegex *regexp.Regexp
var ttyin *os.File

const clearCode string = "\x1b[2J"

// Number of maximum focus events to process synchronously
const maxFocusEvents = 10000

// execute-silent and transform* actions will block user input for this duration.
// After this duration, users can press CTRL-C to terminate the command.
const blockDuration = 1 * time.Second

func init() {
	placeholder = regexp.MustCompile(`\\?(?:{[+*sfr]*[0-9,-.]*}|{q(?::s?[0-9,-.]+)?}|{fzf:(?:query|action|prompt)}|{[+*]?f?nf?})`)
	whiteSuffix = regexp.MustCompile(`\s*$`)
	offsetComponentRegex = regexp.MustCompile(`([+-][0-9]+)|(-?/[1-9][0-9]*)`)
	offsetTrimCharsRegex = regexp.MustCompile(`[^0-9/+-]`)

	// Parts of the preview output that should be passed through to the terminal
	// * https://github.com/tmux/tmux/wiki/FAQ#what-is-the-passthrough-escape-sequence-and-how-do-i-use-it
	// * https://sw.kovidgoyal.net/kitty/graphics-protocol
	// * https://en.wikipedia.org/wiki/Sixel
	// * https://iterm2.com/documentation-images.html
	/*
		passThroughRegex = regexp.MustCompile(`
			  \x1bPtmux;\x1b\x1b  .*?  [^\x1b]\x1b\\
			| \x1b(_G|P[0-9;]*q)  .*?  \x1b\\\r?
			| \x1b]1337;          .*?  (\a|\x1b\\)
		`)
	*/
	passThroughBeginRegex = regexp.MustCompile(`\x1bPtmux;\x1b\x1b|\x1b(_G|P[0-9;]*q)|\x1b]1337;`)
	passThroughEndTmuxRegex = regexp.MustCompile(`[^\x1b]\x1b\\`)
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

type commandSpec struct {
	command   string
	tempFiles []string
}

type quitSignal struct {
	code int
	err  error
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
	xw         [2]int
}

type previewed struct {
	version   int64
	numLines  int
	offset    int
	filled    bool
	image     bool
	wipe      bool
	wireframe bool
}

type eachLine struct {
	line string
	err  error
}

type itemLine struct {
	valid     bool
	firstLine int
	numLines  int
	cy        int
	current   bool
	selected  bool
	label     string
	queryLen  int
	width     int
	hasBar    bool
	result    Result
	empty     bool
	other     bool
	hidden    bool
}

func (t *Terminal) inListWindow() bool {
	return t.window != t.inputWindow && t.window != t.headerWindow && t.window != t.headerLinesWindow && t.window != t.footerWindow
}

func (t *Terminal) markEmptyLine(line int) {
	if t.inListWindow() {
		t.prevLines[line] = itemLine{valid: true, firstLine: line, empty: true}
	}
}

func (t *Terminal) markOtherLine(line int) {
	if t.inListWindow() {
		t.prevLines[line] = itemLine{valid: true, firstLine: line, other: true}
	}
}

type fitpad struct {
	fit int
	pad int
}

type labelPrinter func(tui.Window, int)

type markerClass int

const (
	markerSingle markerClass = iota
	markerTop
	markerMiddle
	markerBottom
)

type StatusItem struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type Status struct {
	Reading    bool         `json:"reading"`
	Progress   int          `json:"progress"`
	Query      string       `json:"query"`
	Position   int          `json:"position"`
	Sort       bool         `json:"sort"`
	TotalCount int          `json:"totalCount"`
	MatchCount int          `json:"matchCount"`
	Current    *StatusItem  `json:"current"`
	Matches    []StatusItem `json:"matches"`
	Selected   []StatusItem `json:"selected"`
}

type versionedCallback struct {
	version  int64
	callback func()
}

type runningCmd struct {
	cmd       *exec.Cmd
	tempFiles []string
}

// Terminal represents terminal input/output
type Terminal struct {
	initDelay          time.Duration
	infoCommand        string
	infoStyle          infoStyle
	infoPrefix         string
	wrap               bool
	wrapSign           string
	wrapSignWidth      int
	ghost              string
	separator          labelPrinter
	separatorLen       int
	spinner            []string
	promptString       string
	prompt             func()
	promptLen          int
	borderLabel        labelPrinter
	borderLabelLen     int
	borderLabelOpts    labelOpts
	previewLabel       labelPrinter
	previewLabelLen    int
	previewLabelOpts   labelOpts
	inputLabel         labelPrinter
	inputLabelLen      int
	inputLabelOpts     labelOpts
	headerLabel        labelPrinter
	headerLabelLen     int
	headerLabelOpts    labelOpts
	footerLabel        labelPrinter
	footerLabelLen     int
	footerLabelOpts    labelOpts
	gutterReverse      bool
	gutterRawReverse   bool
	pointer            string
	pointerLen         int
	pointerEmpty       string
	pointerEmptyRaw    string
	marker             string
	markerLen          int
	markerEmpty        string
	markerMultiLine    [3]string
	queryLen           [2]int
	layout             layoutType
	fullscreen         bool
	keepRight          bool
	hscroll            bool
	hscrollOff         int
	scrollOff          int
	gap                int
	gapLine            labelPrinter
	gapLineLen         int
	wordRubout         string
	wordNext           string
	subWordRubout      string
	subWordNext        string
	cx                 int
	cy                 int
	offset             int
	xoffset            int
	yanked             []rune
	input              []rune
	inputOverride      *[]rune
	pasting            *[]rune
	multi              int
	multiLine          bool
	sort               bool
	toggleSort         bool
	track              trackOption
	delimiter          Delimiter
	expect             map[tui.Event]string
	keymap             map[tui.Event][]*action
	keymapOrg          map[tui.Event][]*action
	pressed            string
	printQueue         []string
	printQuery         bool
	history            *History
	cycle              bool
	highlightLine      bool
	headerVisible      bool
	headerFirst        bool
	headerLines        int
	header             []string
	header0            []string
	footer             []string
	ellipsis           string
	scrollbar          string
	previewScrollbar   string
	ansi               bool
	freezeLeft         int
	freezeRight        int
	nthAttr            tui.Attr
	nth                []Range
	nthCurrent         []Range
	acceptNth          func([]Token, int32) string
	tabstop            int
	margin             [4]sizeSpec
	padding            [4]sizeSpec
	unicode            bool
	listenAddr         *listenAddress
	listenPort         *int
	listener           net.Listener
	listenUnsafe       bool
	borderShape        tui.BorderShape
	listBorderShape    tui.BorderShape
	inputBorderShape   tui.BorderShape
	headerBorderShape  tui.BorderShape
	headerLinesShape   tui.BorderShape
	footerBorderShape  tui.BorderShape
	listLabel          labelPrinter
	listLabelLen       int
	listLabelOpts      labelOpts
	cleanExit          bool
	executor           *util.Executor
	paused             bool
	inputless          bool
	border             tui.Window
	window             tui.Window
	inputWindow        tui.Window
	inputBorder        tui.Window
	headerWindow       tui.Window
	headerBorder       tui.Window
	headerLinesWindow  tui.Window
	headerLinesBorder  tui.Window
	footerWindow       tui.Window
	footerBorder       tui.Window
	wborder            tui.Window
	pborder            tui.Window
	pwindow            tui.Window
	borderWidth        int
	count              int
	progress           int
	hasStartActions    bool
	hasResultActions   bool
	hasFocusActions    bool
	hasLoadActions     bool
	hasResizeActions   bool
	triggerLoad        bool
	reading            bool
	running            *util.AtomicBool
	failed             *string
	jumping            jumpMode
	jumpLabels         string
	printer            func(string)
	printsep           string
	merger             *Merger
	passMerger         *Merger
	resultMerger       *Merger
	matchMap           map[int32]Result
	selected           map[int32]selectedItem
	version            int64
	revision           revision
	bgVersion          int64
	runningCmds        *util.ConcurrentSet[*runningCmd]
	reqBox             *util.EventBox
	initialPreviewOpts previewOpts
	previewOpts        previewOpts
	activePreviewOpts  *previewOpts
	previewer          previewer
	previewed          previewed
	previewBox         *util.EventBox
	eventBox           *util.EventBox
	mutex              sync.Mutex
	uiMutex            sync.Mutex
	initFunc           func() error
	prevLines          []itemLine
	suppress           bool
	startChan          chan fitpad
	killChan           chan bool
	serverInputChan    chan []*action
	callbackChan       chan versionedCallback
	bgQueue            map[action][]func(bool)
	bgSemaphore        chan struct{}
	bgSemaphores       map[action]chan struct{}
	keyChan            chan tui.Event
	eventChan          chan tui.Event
	slab               *util.Slab
	theme              *tui.ColorTheme
	tui                tui.Renderer
	ttyDefault         string
	ttyin              *os.File
	executing          *util.AtomicBool
	termSize           tui.TermSize
	lastAction         actionType
	lastKey            string
	lastFocus          int32
	areaLines          int
	areaColumns        int
	forcePreview       bool
	clickHeaderLine    int
	clickHeaderColumn  int
	clickFooterLine    int
	clickFooterColumn  int
	proxyScript        string
	numLinesCache      map[int32]numLinesCacheValue
	raw                bool
}

type numLinesCacheValue struct {
	atMost   int
	numLines int
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

// EventTypes are listed in the order of their priority.
const (
	reqResize util.EventType = iota
	reqReinit
	reqFullRedraw
	reqRedraw

	reqJump
	reqPrompt
	reqInfo
	reqHeader
	reqFooter
	reqList
	reqRedrawInputLabel
	reqRedrawHeaderLabel
	reqRedrawFooterLabel
	reqRedrawListLabel
	reqRedrawBorderLabel
	reqRedrawPreviewLabel

	reqPreviewReady
	reqPreviewEnqueue
	reqPreviewDisplay
	reqPreviewRefresh
	reqPreviewDelayed

	reqActivate
	reqClose
	reqPrintQuery
	reqBecome
	reqQuit
	reqFatal
)

func isTerminalEvent(et util.EventType) bool {
	switch et {
	case reqClose, reqPrintQuery, reqBecome, reqQuit, reqFatal:
		return true
	}
	return false
}

type action struct {
	t actionType
	a string
}

//go:generate stringer -type=actionType
type actionType int

const (
	actIgnore actionType = iota
	actStart
	actClick
	actInvalid
	actBracketedPasteBegin
	actBracketedPasteEnd
	actChar
	actMouse
	actBeginningOfLine
	actAbort
	actAccept
	actAcceptNonEmpty
	actAcceptOrPrintQuery
	actBackwardChar
	actBackwardDeleteChar
	actBackwardDeleteCharEof
	actBackwardWord
	actBackwardSubWord
	actCancel

	actChangeBorderLabel
	actChangeGhost
	actChangeHeader
	actChangeFooter
	actChangeHeaderLabel
	actChangeFooterLabel
	actChangeInputLabel
	actChangeListLabel
	actChangeMulti
	actChangeNth
	actChangePointer
	actChangePreview
	actChangePreviewLabel
	actChangePreviewWindow
	actChangePrompt
	actChangeQuery

	actClearScreen
	actClearQuery
	actClearSelection
	actClose
	actDeleteChar
	actDeleteCharEof
	actEndOfLine
	actFatal
	actForwardChar
	actForwardWord
	actForwardSubWord
	actKillLine
	actKillWord
	actKillSubWord
	actUnixLineDiscard
	actUnixWordRubout
	actYank
	actBackwardKillWord
	actBackwardKillSubWord
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
	actToggleTrackCurrent
	actToggleHeader
	actToggleWrap
	actToggleMultiLine
	actToggleHscroll
	actToggleRaw
	actEnableRaw
	actDisableRaw
	actTrackCurrent
	actToggleInput
	actHideInput
	actShowInput
	actUntrackCurrent
	actDown
	actDownMatch
	actUp
	actUpMatch
	actPageUp
	actPageDown
	actPosition
	actHalfPageUp
	actHalfPageDown
	actOffsetUp
	actOffsetDown
	actOffsetMiddle
	actJump
	actJumpAccept // XXX Deprecated in favor of jump:accept binding
	actPrintQuery // XXX Deprecated (not very useful, just use --print-query)
	actRefreshPreview
	actReplaceQuery
	actToggleSort
	actShowPreview
	actHidePreview
	actTogglePreview
	actTogglePreviewWrap

	actTransform
	actTransformBorderLabel
	actTransformGhost
	actTransformHeader
	actTransformFooter
	actTransformHeaderLabel
	actTransformFooterLabel
	actTransformInputLabel
	actTransformListLabel
	actTransformNth
	actTransformPointer
	actTransformPreviewLabel
	actTransformPrompt
	actTransformQuery
	actTransformSearch

	actTrigger

	actBgTransform
	actBgTransformBorderLabel
	actBgTransformGhost
	actBgTransformHeader
	actBgTransformFooter
	actBgTransformHeaderLabel
	actBgTransformFooterLabel
	actBgTransformInputLabel
	actBgTransformListLabel
	actBgTransformNth
	actBgTransformPointer
	actBgTransformPreviewLabel
	actBgTransformPrompt
	actBgTransformQuery
	actBgTransformSearch

	actBgCancel

	actSearch
	actPreview
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
	actPrint
	actPut
	actNextHistory
	actNextSelected
	actExecute
	actExecuteSilent
	actExecuteMulti // Deprecated
	actSigStop
	actBest
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
	actToggleBind
	actBecome
	actShowHeader
	actHideHeader
	actBell
	actExclude
	actExcludeMulti
	actAsync
)

func (a actionType) Name() string {
	return util.ToKebabCase(a.String()[3:])
}

func processExecution(action actionType) bool {
	switch action {
	case actTransform,
		actTransformBorderLabel,
		actTransformGhost,
		actTransformHeader,
		actTransformFooter,
		actTransformHeaderLabel,
		actTransformFooterLabel,
		actTransformInputLabel,
		actTransformListLabel,
		actTransformNth,
		actTransformPointer,
		actTransformPreviewLabel,
		actTransformPrompt,
		actTransformQuery,
		actTransformSearch,
		actBgTransform,
		actBgTransformBorderLabel,
		actBgTransformGhost,
		actBgTransformHeader,
		actBgTransformFooter,
		actBgTransformHeaderLabel,
		actBgTransformFooterLabel,
		actBgTransformInputLabel,
		actBgTransformListLabel,
		actBgTransformNth,
		actBgTransformPointer,
		actBgTransformPreviewLabel,
		actBgTransformPrompt,
		actBgTransformQuery,
		actBgTransformSearch,
		actPreview,
		actChangePreview,
		actRefreshPreview,
		actExecute,
		actExecuteSilent,
		actExecuteMulti,
		actReload,
		actReloadSync,
		actBecome:
		return true
	}
	return false
}

type placeholderFlags struct {
	plus          bool
	asterisk      bool
	preserveSpace bool
	number        bool
	forceUpdate   bool
	file          bool
	raw           bool
}

type searchRequest struct {
	sort     bool
	sync     bool
	nth      *[]Range
	command  *commandSpec
	environ  []string
	changed  bool
	denylist []int32
	revision revision
}

type previewRequest struct {
	template     string
	scrollOffset int
	list         [3][]*Item // current, select, and all matched items
	env          []string
	query        string
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

	add(tui.Fatal, actFatal)
	add(tui.Invalid, actInvalid)
	add(tui.BracketedPasteBegin, actBracketedPasteBegin)
	add(tui.BracketedPasteEnd, actBracketedPasteEnd)
	add(tui.CtrlA, actBeginningOfLine)
	add(tui.CtrlB, actBackwardChar)
	add(tui.CtrlC, actAbort)
	add(tui.CtrlG, actAbort)
	add(tui.CtrlQ, actAbort)
	add(tui.Esc, actAbort)
	add(tui.CtrlD, actDeleteCharEof)
	add(tui.CtrlE, actEndOfLine)
	add(tui.CtrlF, actForwardChar)
	add(tui.Backspace, actBackwardDeleteChar)
	add(tui.CtrlBackspace, actBackwardDeleteChar)
	add(tui.Tab, actToggleDown)
	add(tui.ShiftTab, actToggleUp)
	add(tui.CtrlJ, actDown)
	add(tui.CtrlK, actUp)
	add(tui.CtrlL, actClearScreen)
	add(tui.Enter, actAccept)
	add(tui.CtrlN, actDownMatch)
	add(tui.CtrlP, actUpMatch)
	add(tui.AltDown, actDownMatch)
	add(tui.AltUp, actUpMatch)
	add(tui.CtrlU, actUnixLineDiscard)
	add(tui.CtrlW, actUnixWordRubout)
	add(tui.CtrlY, actYank)
	if !util.IsWindows() {
		add(tui.CtrlZ, actSigStop)
	}
	add(tui.CtrlSlash, actToggleWrap)
	addEvent(tui.AltKey('/'), actToggleWrap)

	addEvent(tui.AltKey('b'), actBackwardWord)
	add(tui.ShiftLeft, actBackwardWord)
	addEvent(tui.AltKey('f'), actForwardWord)
	add(tui.ShiftRight, actForwardWord)
	addEvent(tui.AltKey('d'), actKillWord)
	add(tui.AltBackspace, actBackwardKillWord)

	add(tui.Up, actUp)
	add(tui.Down, actDown)
	add(tui.Left, actBackwardChar)
	add(tui.Right, actForwardChar)

	add(tui.Home, actBeginningOfLine)
	add(tui.End, actEndOfLine)
	add(tui.Delete, actDeleteChar)
	add(tui.PageUp, actPageUp)
	add(tui.PageDown, actPageDown)

	add(tui.ShiftUp, actPreviewUp)
	add(tui.ShiftDown, actPreviewDown)

	add(tui.Mouse, actMouse)
	add(tui.LeftClick, actClick)
	add(tui.RightClick, actToggle)
	add(tui.SLeftClick, actToggle)
	add(tui.SRightClick, actToggle)

	add(tui.ScrollUp, actUp)
	add(tui.ScrollDown, actDown)
	keymap[tui.SScrollUp.AsEvent()] = toActions(actToggle, actUp)
	keymap[tui.SScrollDown.AsEvent()] = toActions(actToggle, actDown)

	add(tui.PreviewScrollUp, actPreviewUp)
	add(tui.PreviewScrollDown, actPreviewDown)
	return keymap
}

func trimQuery(query string) []rune {
	return []rune(strings.ReplaceAll(query, "\t", " "))
}

func mayTriggerPreview(opts *Options) bool {
	if opts.ListenAddr != nil {
		return true
	}
	for _, actions := range opts.Keymap {
		for _, action := range actions {
			switch action.t {
			case actPreview, actChangePreview, actTransform, actBgTransform:
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
	size := opts.Height.size
	if opts.Height.percent {
		if opts.Height.inverse {
			size = 100 - size
		}
		return max(int(size*float64(termHeight)/100.0), opts.MinHeight)
	}
	if opts.Height.inverse {
		size = float64(termHeight) - size
	}
	return int(size)
}

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox, executor *util.Executor) (*Terminal, error) {
	input := trimQuery(opts.Query)
	var delay time.Duration
	if opts.Sync {
		delay = 0
	} else if opts.Tac {
		delay = initialDelayTac
	} else {
		delay = initialDelay
	}
	var previewBox *util.EventBox
	// We need to start the previewer even when --preview option is not specified
	// * if HTTP server is enabled
	// * if 'preview' or 'change-preview' action is bound to a key
	// * if 'transform' action is bound to a key
	if len(opts.Preview.command) > 0 || mayTriggerPreview(opts) {
		previewBox = util.NewEventBox()
	}
	var renderer tui.Renderer
	fullscreen := !opts.Height.auto && (opts.Height.size == 0 || opts.Height.percent && opts.Height.size == 100)
	var err error
	// Reuse ttyin if available to avoid having multiple file descriptors open
	// when you run fzf multiple times in your Go program. Closing it is known to
	// cause problems with 'become' action and invalid terminal state after exit.
	if ttyin == nil {
		if ttyin, err = tui.TtyIn(opts.TtyDefault); err != nil {
			return nil, err
		}
	}
	if fullscreen {
		if tui.HasFullscreenRenderer() {
			renderer = tui.NewFullscreenRenderer(opts.Theme, opts.Black, opts.Mouse)
		} else {
			renderer, err = tui.NewLightRenderer(opts.TtyDefault, ttyin, opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit,
				true, func(h int) int { return h })
		}
	} else {
		maxHeightFunc := func(termHeight int) int {
			// Minimum height required to render fzf excluding margin and padding
			effectiveMinHeight := minHeight
			if previewBox != nil && opts.Preview.aboveOrBelow() {
				effectiveMinHeight += 1 + borderLines(opts.Preview.Border())
			}
			if opts.noSeparatorLine() {
				effectiveMinHeight--
			}
			effectiveMinHeight += borderLines(opts.BorderShape)
			return min(termHeight, max(evaluateHeight(opts, termHeight), effectiveMinHeight))
		}
		renderer, err = tui.NewLightRenderer(opts.TtyDefault, ttyin, opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit, false, maxHeightFunc)
	}
	if err != nil {
		return nil, err
	}
	if opts.Inputless {
		renderer.HideCursor()
	}
	wordRubout := "[^\\pL\\pN][\\pL\\pN]"
	wordNext := "[\\pL\\pN][^\\pL\\pN]|(.$)"
	subWordRubout := "[a-z][A-Z]|[^\\pL\\pN][\\pL\\pN]"
	subWordNext := "[a-z][A-Z]|[\\pL\\pN][^\\pL\\pN]|(.$)"
	if opts.FileWord {
		sep := regexp.QuoteMeta(string(os.PathSeparator))
		wordRubout = fmt.Sprintf("%s[^%s]", sep, sep)
		wordNext = fmt.Sprintf("[^%s]%s|(.$)", sep, sep)
	}
	keymapCopy := maps.Clone(opts.Keymap)

	em := EmptyMerger(revision{})
	t := Terminal{
		initDelay:          delay,
		infoCommand:        opts.InfoCommand,
		infoStyle:          opts.InfoStyle,
		infoPrefix:         opts.InfoPrefix,
		ghost:              opts.Ghost,
		separator:          nil,
		spinner:            makeSpinner(opts.Unicode),
		promptString:       opts.Prompt,
		queryLen:           [2]int{0, 0},
		layout:             opts.Layout,
		fullscreen:         fullscreen,
		keepRight:          opts.KeepRight,
		hscroll:            opts.Hscroll,
		hscrollOff:         opts.HscrollOff,
		scrollOff:          opts.ScrollOff,
		pointer:            *opts.Pointer,
		pointerLen:         uniseg.StringWidth(*opts.Pointer),
		marker:             *opts.Marker,
		markerLen:          uniseg.StringWidth(*opts.Marker),
		markerMultiLine:    *opts.MarkerMulti,
		wordRubout:         wordRubout,
		wordNext:           wordNext,
		subWordRubout:      subWordRubout,
		subWordNext:        subWordNext,
		cx:                 len(input),
		cy:                 0,
		offset:             0,
		xoffset:            0,
		yanked:             []rune{},
		input:              input,
		multi:              opts.Multi,
		multiLine:          opts.ReadZero && opts.MultiLine,
		wrap:               opts.Wrap,
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
		listenAddr:         opts.ListenAddr,
		listenUnsafe:       opts.Unsafe,
		borderShape:        opts.BorderShape,
		listBorderShape:    opts.ListBorderShape,
		inputBorderShape:   opts.InputBorderShape,
		headerBorderShape:  opts.HeaderBorderShape,
		headerLinesShape:   opts.HeaderLinesShape,
		footerBorderShape:  opts.FooterBorderShape,
		borderWidth:        1,
		listLabel:          nil,
		listLabelOpts:      opts.ListLabel,
		borderLabel:        nil,
		borderLabelOpts:    opts.BorderLabel,
		previewLabel:       nil,
		previewLabelOpts:   opts.PreviewLabel,
		inputLabel:         nil,
		inputLabelOpts:     opts.InputLabel,
		headerLabel:        nil,
		headerLabelOpts:    opts.HeaderLabel,
		footerLabel:        nil,
		footerLabelOpts:    opts.FooterLabel,
		cleanExit:          opts.ClearOnExit,
		executor:           executor,
		paused:             opts.Phony,
		inputless:          opts.Inputless,
		cycle:              opts.Cycle,
		highlightLine:      opts.CursorLine,
		headerVisible:      true,
		headerFirst:        opts.HeaderFirst,
		headerLines:        opts.HeaderLines,
		gap:                opts.Gap,
		header:             []string{},
		footer:             opts.Footer,
		header0:            opts.Header,
		ansi:               opts.Ansi,
		freezeLeft:         opts.FreezeLeft,
		freezeRight:        opts.FreezeRight,
		nthAttr:            opts.Theme.Nth.Attr,
		nth:                opts.Nth,
		nthCurrent:         opts.Nth,
		tabstop:            opts.Tabstop,
		raw:                opts.Raw,
		hasStartActions:    false,
		hasResultActions:   false,
		hasFocusActions:    false,
		hasLoadActions:     false,
		triggerLoad:        false,
		reading:            true,
		running:            util.NewAtomicBool(true),
		failed:             nil,
		jumping:            jumpDisabled,
		jumpLabels:         opts.JumpLabels,
		printer:            opts.Printer,
		printsep:           opts.PrintSep,
		proxyScript:        opts.ProxyScript,
		merger:             em,
		passMerger:         em,
		resultMerger:       em,
		matchMap:           make(map[int32]Result),
		selected:           make(map[int32]selectedItem),
		runningCmds:        util.NewConcurrentSet[*runningCmd](),
		reqBox:             util.NewEventBox(),
		initialPreviewOpts: opts.Preview,
		previewOpts:        opts.Preview,
		activePreviewOpts:  &opts.Preview,
		previewer:          previewer{0, []string{}, 0, false, true, disabledState, "", []bool{}, [2]int{0, 0}},
		previewed:          previewed{0, 0, 0, false, false, false, false},
		previewBox:         previewBox,
		eventBox:           eventBox,
		mutex:              sync.Mutex{},
		uiMutex:            sync.Mutex{},
		suppress:           true,
		slab:               util.MakeSlab(slab16Size, slab32Size),
		theme:              opts.Theme,
		startChan:          make(chan fitpad, 1),
		killChan:           make(chan bool),
		serverInputChan:    make(chan []*action, 100),
		callbackChan:       make(chan versionedCallback, maxBgProcesses),
		bgQueue:            make(map[action][]func(bool)),
		bgSemaphore:        make(chan struct{}, maxBgProcesses),
		bgSemaphores:       make(map[action]chan struct{}),
		keyChan:            make(chan tui.Event),
		eventChan:          make(chan tui.Event, 6), // start | (load + result + zero|one) | (focus) | (resize)
		tui:                renderer,
		ttyDefault:         opts.TtyDefault,
		ttyin:              ttyin,
		initFunc:           func() error { return renderer.Init() },
		executing:          util.NewAtomicBool(false),
		lastAction:         actStart,
		lastFocus:          minItem.Index(),
		numLinesCache:      make(map[int32]numLinesCacheValue)}
	if opts.AcceptNth != nil {
		t.acceptNth = opts.AcceptNth(t.delimiter)
	}

	baseTheme := opts.BaseTheme
	if baseTheme == nil {
		baseTheme = renderer.DefaultTheme()
	}
	// This should be called before accessing tui.Color*
	tui.InitTheme(opts.Theme, baseTheme, opts.Bold, opts.Black, opts.InputBorderShape.Visible(), opts.HeaderBorderShape.Visible())

	// Gutter character
	var gutterChar, gutterRawChar string
	if opts.Gutter != nil {
		gutterChar = *opts.Gutter
	} else if t.unicode {
		gutterChar = "▌"
	} else {
		gutterChar = " "
		t.gutterReverse = true
	}

	if opts.GutterRaw != nil {
		gutterRawChar = *opts.GutterRaw
	} else if t.unicode {
		gutterRawChar = "▖"
	} else {
		gutterRawChar = ":"
		t.gutterRawReverse = false
	}

	t.prompt, t.promptLen = t.parsePrompt(opts.Prompt)
	// Pre-calculated empty pointer and marker signs
	if t.pointerLen == 0 {
		t.pointerEmpty = ""
		t.pointerEmptyRaw = ""
	} else {
		t.pointerEmpty = gutterChar + strings.Repeat(" ", max(0, t.pointerLen-1))
		t.pointerEmptyRaw = gutterRawChar + strings.Repeat(" ", max(0, t.pointerLen-1))
	}
	t.markerEmpty = strings.Repeat(" ", t.markerLen)

	// Labels
	t.listLabel, t.listLabelLen = t.ansiLabelPrinter(opts.ListLabel.label, &tui.ColListLabel, false)
	t.borderLabel, t.borderLabelLen = t.ansiLabelPrinter(opts.BorderLabel.label, &tui.ColBorderLabel, false)
	t.previewLabel, t.previewLabelLen = t.ansiLabelPrinter(opts.PreviewLabel.label, &tui.ColPreviewLabel, false)
	t.inputLabel, t.inputLabelLen = t.ansiLabelPrinter(opts.InputLabel.label, &tui.ColInputLabel, false)
	t.headerLabel, t.headerLabelLen = t.ansiLabelPrinter(opts.HeaderLabel.label, &tui.ColHeaderLabel, false)
	t.footerLabel, t.footerLabelLen = t.ansiLabelPrinter(opts.FooterLabel.label, &tui.ColFooterLabel, false)

	// Determine border shape
	if t.borderShape == tui.BorderLine {
		if t.fullscreen {
			t.borderShape = tui.BorderNone
		} else {
			t.borderShape = tui.BorderTop
		}
	}

	// Determine input border shape
	if t.inputBorderShape == tui.BorderLine {
		if t.layout == layoutReverse {
			t.inputBorderShape = tui.BorderBottom
		} else {
			t.inputBorderShape = tui.BorderTop
		}
	}

	// Determine header border shape
	if t.headerBorderShape == tui.BorderLine {
		if t.layout == layoutReverse {
			t.headerBorderShape = tui.BorderBottom
		} else {
			t.headerBorderShape = tui.BorderTop
		}
	}

	// Determine header lines border shape
	if t.headerLinesShape == tui.BorderLine {
		if t.layout == layoutDefault {
			t.headerLinesShape = tui.BorderTop
		} else {
			t.headerLinesShape = tui.BorderBottom
		}
	}

	// Determine footer border shape
	if t.footerBorderShape == tui.BorderLine {
		if t.layout == layoutReverse {
			t.footerBorderShape = tui.BorderTop
		} else {
			t.footerBorderShape = tui.BorderBottom
		}
	}

	// Disable separator by default if input border is set
	if opts.Separator == nil && !t.inputBorderShape.Visible() || opts.Separator != nil && len(*opts.Separator) > 0 {
		bar := "─"
		if opts.Separator != nil {
			bar = *opts.Separator
		} else if !t.unicode {
			bar = "-"
		}
		t.separator, t.separatorLen = t.ansiLabelPrinter(bar, &tui.ColSeparator, true)
	}

	// Gap line
	if t.gap > 0 && len(*opts.GapLine) > 0 {
		t.gapLine, t.gapLineLen = t.ansiLabelPrinter(*opts.GapLine, &tui.ColGapLine, true)
	}

	if opts.Ellipsis != nil {
		t.ellipsis = *opts.Ellipsis
	} else if t.unicode {
		t.ellipsis = "··"
	} else {
		t.ellipsis = ".."
	}

	if t.unicode {
		t.wrapSign = "↳ "
		t.borderWidth = uniseg.StringWidth("│")
	} else {
		t.wrapSign = "> "
	}
	if opts.WrapSign != nil {
		t.wrapSign = *opts.WrapSign
	}
	t.wrapSign, t.wrapSignWidth = t.processTabs([]rune(t.wrapSign), 0)
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

	var resizeActions []*action
	resizeActions, t.hasResizeActions = t.keymap[tui.Resize.AsEvent()]
	if t.tui.ShouldEmitResizeEvent() {
		t.keymap[tui.Resize.AsEvent()] = append(toActions(actClearScreen), resizeActions...)
	}
	_, t.hasStartActions = t.keymap[tui.Start.AsEvent()]
	_, t.hasResultActions = t.keymap[tui.Result.AsEvent()]
	_, t.hasFocusActions = t.keymap[tui.Focus.AsEvent()]
	_, t.hasLoadActions = t.keymap[tui.Load.AsEvent()]

	if t.listenAddr != nil {
		listener, port, err := startHttpServer(*t.listenAddr, t.serverInputChan, t.dumpStatus)
		if err != nil {
			return nil, err
		}
		t.listener = listener
		if port > 0 {
			t.listenPort = &port
		}
	}

	if t.hasStartActions {
		t.eventChan <- tui.Start.AsEvent()
	}

	return &t, nil
}

func (t *Terminal) deferActivation() bool {
	return t.initDelay == 0 && (t.hasStartActions || t.hasLoadActions || t.hasResultActions || t.hasFocusActions)
}

func (t *Terminal) environ() []string {
	return t.environImpl(false)
}

func (t *Terminal) environForPreview() []string {
	return t.environImpl(true)
}

func (t *Terminal) environImpl(forPreview bool) []string {
	env := os.Environ()
	if t.listenAddr != nil && len(t.listenAddr.sock) > 0 {
		env = append(env, "FZF_SOCK="+t.listenAddr.sock)
	}
	if t.listenPort != nil {
		env = append(env, fmt.Sprintf("FZF_PORT=%d", *t.listenPort))
	}
	env = append(env, "FZF_QUERY="+string(t.input))
	env = append(env, "FZF_ACTION="+t.lastAction.Name())
	env = append(env, "FZF_KEY="+t.lastKey)
	env = append(env, "FZF_PROMPT="+string(t.promptString))
	env = append(env, "FZF_GHOST="+string(t.ghost))
	env = append(env, "FZF_POINTER="+string(t.pointer))
	env = append(env, "FZF_PREVIEW_LABEL="+t.previewLabelOpts.label)
	env = append(env, "FZF_BORDER_LABEL="+t.borderLabelOpts.label)
	env = append(env, "FZF_LIST_LABEL="+t.listLabelOpts.label)
	env = append(env, "FZF_INPUT_LABEL="+t.inputLabelOpts.label)
	env = append(env, "FZF_HEADER_LABEL="+t.headerLabelOpts.label)
	direction := "down"
	if t.layout == layoutDefault {
		direction = "up"
	}
	env = append(env, "FZF_DIRECTION="+direction)
	if len(t.nthCurrent) > 0 {
		env = append(env, "FZF_NTH="+RangesToString(t.nthCurrent))
	}
	if t.raw {
		val := "0"
		if t.isCurrentItemMatch() {
			val = "1"
		}
		env = append(env, "FZF_RAW="+val)
	}
	inputState := "enabled"
	if t.inputless {
		inputState = "hidden"
	} else if t.paused {
		inputState = "disabled"
	}
	env = append(env, "FZF_INPUT_STATE="+inputState)
	env = append(env, fmt.Sprintf("FZF_TOTAL_COUNT=%d", t.count))
	env = append(env, fmt.Sprintf("FZF_MATCH_COUNT=%d", t.resultMerger.Length()))
	env = append(env, fmt.Sprintf("FZF_SELECT_COUNT=%d", len(t.selected)))
	env = append(env, fmt.Sprintf("FZF_LINES=%d", t.areaLines))
	env = append(env, fmt.Sprintf("FZF_COLUMNS=%d", t.areaColumns))
	env = append(env, fmt.Sprintf("FZF_POS=%d", min(t.merger.Length(), t.cy+1)))
	env = append(env, fmt.Sprintf("FZF_CLICK_HEADER_LINE=%d", t.clickHeaderLine))
	env = append(env, fmt.Sprintf("FZF_CLICK_HEADER_COLUMN=%d", t.clickHeaderColumn))
	env = append(env, fmt.Sprintf("FZF_CLICK_FOOTER_LINE=%d", t.clickFooterLine))
	env = append(env, fmt.Sprintf("FZF_CLICK_FOOTER_COLUMN=%d", t.clickFooterColumn))
	env = t.addClickHeaderWord(env)
	env = t.addClickFooterWord(env)

	// Add preview environment variables if preview is enabled
	pwindowSize := t.pwindowSize()
	if pwindowSize.Lines > 0 {
		lines := fmt.Sprintf("LINES=%d", pwindowSize.Lines)
		columns := fmt.Sprintf("COLUMNS=%d", pwindowSize.Columns)
		if forPreview {
			env = append(env, lines)
			env = append(env, columns)
		}
		env = append(env, "FZF_PREVIEW_"+lines)
		env = append(env, "FZF_PREVIEW_"+columns)
		env = append(env, fmt.Sprintf("FZF_PREVIEW_TOP=%d", t.tui.Top()+t.pwindow.Top()))
		env = append(env, fmt.Sprintf("FZF_PREVIEW_LEFT=%d", t.pwindow.Left()))
	}

	return env
}

func borderLines(shape tui.BorderShape) int {
	lines := 0
	if shape.HasTop() {
		lines++
	}
	if shape.HasBottom() {
		lines++
	}
	return lines
}

func borderColumns(shape tui.BorderShape, borderWidth int) int {
	columns := 0
	if shape.HasLeft() {
		columns += 1 + borderWidth
	}
	if shape.HasRight() {
		columns += 1 + borderWidth
	}
	return columns
}

func (t *Terminal) visibleHeaderLines() int {
	if !t.headerVisible {
		return 0
	}
	return len(t.header0) + t.headerLines
}

func (t *Terminal) visibleHeaderLinesInList() int {
	if t.headerWindow != nil || t.headerLinesWindow != nil {
		return 0
	}
	return t.visibleHeaderLines()
}

func (t *Terminal) visibleInputLinesInList() int {
	if t.inputWindow != nil || t.inputless {
		return 0
	}
	if t.noSeparatorLine() {
		return 1
	}
	return 2
}

// Extra number of lines needed to display fzf
func (t *Terminal) extraLines() int {
	extra := 0
	if !t.inputless {
		extra++
		if !t.noSeparatorLine() {
			extra++
		}
		if t.inputBorderShape.Visible() {
			extra += borderLines(t.inputBorderShape)
		}
	}
	if t.listBorderShape.Visible() {
		extra += borderLines(t.listBorderShape)
	}
	if t.headerVisible {
		if t.hasHeaderWindow() {
			extra += borderLines(t.headerBorderShape)
		}
		extra += len(t.header0)
		if w, shape := t.determineHeaderLinesShape(); w {
			extra += borderLines(shape)
		}
		extra += t.headerLines
	}
	if len(t.footer) > 0 {
		extra += borderLines(t.footerBorderShape)
		extra += len(t.footer)
	}
	return extra
}

func (t *Terminal) MaxFitAndPad() (int, int) {
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
	if colors == nil && !strings.ContainsRune(text, '\t') {
		length := util.StringWidth(text)
		if length == 0 {
			return nil, 0
		}
		printFn := func(window tui.Window, limit int) {
			ellipsis := []rune{}
			ellipsisWidth := 0
			if !fill {
				ellipsis, ellipsisWidth = util.Truncate(t.ellipsis, limit)
			}
			if length > limit {
				trimmedRunes, _ := t.trimRight(runes, limit-ellipsisWidth)
				window.CPrint(*color, string(trimmedRunes)+string(ellipsis))
			} else if fill {
				window.CPrint(*color, util.RepeatToFill(text, length, limit))
			} else {
				window.CPrint(*color, text)
			}
		}
		return printFn, length
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
			offsets = result.colorOffsets(nil, nil, t.theme, *color, *color, t.nthAttr, false)
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

// Temporarily switch 'window' so that we can use the existing windows with
// a different window
func (t *Terminal) withWindow(w tui.Window, f func()) {
	prevWindow := t.window
	if w != nil {
		t.window = w
	}
	f()
	t.window = prevWindow
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
		blankState := ansiOffset{[2]int32{int32(loc[0]), int32(loc[1])}, ansiState{tui.ColPrompt.Fg(), tui.ColPrompt.Bg(), tui.AttrClear, -1, nil}}
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
		wrap := t.wrap
		t.wrap = false
		t.withWindow(t.inputWindow, func() {
			line := t.promptLine()
			preTask := func(markerClass) int {
				return 1
			}
			t.printHighlighted(
				Result{item: item}, tui.ColPrompt, tui.ColPrompt, false, false, false, line, line, true, preTask, nil)
		})
		t.wrap = wrap
	}
	_, promptLen := t.processTabs([]rune(trimmed), 0)

	return output, promptLen
}

func (t *Terminal) noSeparatorLine() bool {
	return t.inputless || noSeparatorLine(t.infoStyle, t.separatorLen > 0)
}

func getScrollbar(perLine int, total int, height int, offset int) (int, int) {
	if total == 0 || total*perLine <= height {
		return 0, 0
	}
	barLength := max(1, height*height/(total*perLine))
	var barStart int
	if total == height {
		barStart = 0
	} else {
		barStart = min(height-barLength, (height*perLine-barLength)*offset/(total*perLine-height))
	}
	return barLength, barStart
}

func (t *Terminal) barCol() int {
	if len(t.scrollbar) == 0 && !t.listBorderShape.HasRight() && !t.borderShape.HasRight() && !t.hasPreviewWindowOnRight() {
		return 0
	}
	return 1
}

func (t *Terminal) wrapCols() int {
	if !t.wrap {
		return 0 // No wrap
	}
	return max(t.window.Width()-(t.pointerLen+t.markerLen+t.barCol()), 1)
}

func (t *Terminal) clearNumLinesCache() {
	t.numLinesCache = make(map[int32]numLinesCacheValue)
}

// Number of lines the item takes including the gap
func (t *Terminal) numItemLines(item *Item, atMost int) (int, bool) {
	var numLines int
	if !t.wrap && !t.multiLine {
		numLines = 1 + t.gap
		return numLines, numLines > atMost
	}
	if cached, prs := t.numLinesCache[item.Index()]; prs {
		// Can we use this cache? Let's be conservative.
		if cached.atMost <= atMost {
			return cached.numLines, false
		}
	}
	var overflow bool
	if !t.wrap && t.multiLine {
		numLines, overflow = item.text.NumLines(atMost)
	} else {
		var lines [][]rune
		lines, overflow = item.text.Lines(t.multiLine, atMost, t.wrapCols(), t.wrapSignWidth, t.tabstop)
		numLines = len(lines)
	}
	numLines += t.gap
	if !overflow {
		t.numLinesCache[item.Index()] = numLinesCacheValue{atMost, numLines}
	}
	return numLines, overflow || numLines > atMost
}

func (t *Terminal) itemLines(item *Item, atMost int) ([][]rune, bool) {
	if !t.wrap && !t.multiLine {
		text := make([]rune, item.text.Length())
		copy(text, item.text.ToRunes())
		return [][]rune{text}, false
	}
	return item.text.Lines(t.multiLine, atMost, t.wrapCols(), t.wrapSignWidth, t.tabstop)
}

// Estimate the average number of lines per item. Instead of going through all
// items, we only check a few items around the current cursor position.
func (t *Terminal) avgNumLines() int {
	if !t.wrap && !t.multiLine {
		return 1
	}

	maxItems := t.maxItems()
	numLines := 0
	count := 0
	total := t.merger.Length()
	offset := max(0, min(t.offset, total-maxItems-1))
	for idx := 0; idx < maxItems && idx+offset < total; idx++ {
		result := t.merger.Get(idx + offset)
		lines, _ := t.numItemLines(result.item, maxItems)
		numLines += lines
		count++
	}
	if count == 0 {
		return 1
	}
	return numLines / count
}

func (t *Terminal) getScrollbar() (int, int) {
	return getScrollbar(t.avgNumLines(), t.merger.Length(), t.maxItems(), t.offset)
}

// Input returns current query string
func (t *Terminal) Input() (bool, []rune) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	paused := t.paused
	src := t.input
	if t.inputOverride != nil {
		paused = false
		src = *t.inputOverride
	}
	return paused, copySlice(src)
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
	suppressed := t.suppress
	t.mutex.Unlock()
	t.reqBox.Set(reqInfo, nil)

	// We want to defer activating the interface when --sync is used and any of
	// start, load, or result events are bound
	if suppressed && final && !t.deferActivation() {
		t.reqBox.Set(reqActivate, nil)
	}
}

func (t *Terminal) changeHeader(header string) bool {
	var lines []string
	if len(header) > 0 {
		lines = strings.Split(strings.TrimSuffix(header, "\n"), "\n")
	}
	needFullRedraw := len(t.header0) != len(lines)
	t.header0 = lines
	t.clickHeaderLine = 0
	t.clickHeaderColumn = 0
	return needFullRedraw
}

func (t *Terminal) changeFooter(footer string) {
	var lines []string
	if len(footer) > 0 {
		lines = strings.Split(strings.TrimSuffix(footer, "\n"), "\n")
	}
	t.footer = lines
	t.clickFooterLine = 0
	t.clickFooterColumn = 0
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
func (t *Terminal) UpdateList(result MatchResult) {
	merger := result.merger
	t.mutex.Lock()
	prevIndex := minItem.Index()
	newRevision := merger.Revision()
	if t.revision.compatible(newRevision) && t.track != trackDisabled {
		if t.merger.Length() > 0 {
			prevIndex = t.currentIndex()
		} else if merger.Length() > 0 {
			prevIndex = merger.First().item.Index()
		}
	}
	t.progress = 100
	t.merger = merger
	t.resultMerger = merger
	t.passMerger = result.passMerger
	if t.raw {
		t.merger = result.passMerger
		t.matchMap = t.resultMerger.ToMap()
	} else {
		t.merger = result.merger
		t.matchMap = make(map[int32]Result)
	}
	if t.revision != newRevision {
		if !t.revision.compatible(newRevision) {
			// Reloaded: clear selection
			t.selected = make(map[int32]selectedItem)
			t.clearNumLinesCache()
		} else {
			// Trimmed by --tail: filter selection by index
			filtered := make(map[int32]selectedItem)
			minIndex := merger.minIndex
			maxIndex := merger.maxIndex
			for k, v := range t.selected {
				var included bool
				if maxIndex > minIndex {
					included = k >= minIndex && k < maxIndex
				} else if maxIndex < minIndex { // int32 overflow [==>   <==]
					included = k >= minIndex || k < maxIndex
				}
				if included {
					filtered[k] = v
				}
			}
			t.selected = filtered
		}
		t.revision = newRevision
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
		} else if t.track.Current() {
			t.track = trackDisabled
			t.cy = pos
			t.offset = 0
		} else if t.cy > count {
			// Try to keep the vertical position when the list shrinks
			t.cy = count - min(count, t.maxItems()) + pos
		}
	}
	needActivation := false
	if !t.reading {
		switch t.resultMerger.Length() {
		case 0:
			zero := tui.Zero.AsEvent()
			if _, prs := t.keymap[zero]; prs {
				t.eventChan <- zero
			}
			// --sync, only 'focus' is bound, but no items to focus
			needActivation = t.suppress && !t.hasResultActions && !t.hasLoadActions && t.hasFocusActions
		case 1:
			one := tui.One.AsEvent()
			if _, prs := t.keymap[one]; prs {
				t.eventChan <- one
			}
		}
	}
	if t.hasResultActions {
		t.eventChan <- tui.Result.AsEvent()
	}
	t.mutex.Unlock()
	t.reqBox.Set(reqInfo, nil)
	t.reqBox.Set(reqList, nil)
	if needActivation {
		t.reqBox.Set(reqActivate, nil)
	}
}

func (t *Terminal) output() bool {
	if t.printQuery {
		t.printer(string(t.input))
	}
	if len(t.expect) > 0 {
		t.printer(t.pressed)
	}
	for _, s := range t.printQueue {
		t.printer(s)
	}
	transform := func(item *Item) string {
		return item.AsString(t.ansi)
	}
	if t.acceptNth != nil {
		transform = func(item *Item) string {
			return item.acceptNth(t.ansi, t.delimiter, t.acceptNth)
		}
	}
	found := len(t.selected) > 0
	if !found {
		current := t.currentItem()
		if current != nil {
			t.printer(transform(current))
			found = true
		}
	} else {
		for _, sel := range t.sortSelected() {
			t.printer(transform(sel.item))
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

func (t *Terminal) displayWidthWithPrefix(str string, prefixWidth int) int {
	width, _ := util.RunesWidth([]rune(str), prefixWidth, t.tabstop, math.MaxInt32)
	return width
}

const (
	minWidth  = 4
	minHeight = 3
)

func calculateSize(base int, size sizeSpec, occupied int, minSize int) int {
	max := base - occupied
	if max < minSize {
		max = minSize
	}
	if size.percent {
		return util.Constrain(int(float64(base)*0.01*size.size), minSize, max)
	}
	return util.Constrain(int(size.size)+minSize-1, minSize, max)
}

func (t *Terminal) minPreviewSize(opts *previewOpts) (int, int) {
	minPreviewWidth := 1 + borderColumns(opts.Border(), t.borderWidth)
	minPreviewHeight := 1 + borderLines(opts.Border())

	switch opts.position {
	case posLeft, posRight:
		if len(t.scrollbar) > 0 && !opts.Border().HasRight() {
			// Need a column to show scrollbar
			minPreviewWidth++
		}
	}

	return minPreviewWidth, minPreviewHeight
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

	adjust := func(idx1 int, idx2 int, maximum int, minimum int) {
		if minimum > maximum {
			minimum = maximum
		}
		margin := marginInt[idx1] + marginInt[idx2] + paddingInt[idx1] + paddingInt[idx2]
		if maximum-margin < minimum {
			desired := maximum - minimum
			paddingInt[idx1] = desired * paddingInt[idx1] / margin
			paddingInt[idx2] = desired * paddingInt[idx2] / margin
			marginInt[idx1] = max(extraMargin[idx1], desired*marginInt[idx1]/margin)
			marginInt[idx2] = max(extraMargin[idx2], desired*marginInt[idx2]/margin)
		}
	}

	minAreaWidth := minWidth
	minAreaHeight := minHeight
	if t.inputless {
		minAreaHeight--
	}
	if t.noSeparatorLine() {
		minAreaHeight--
	}
	if t.needPreviewWindow() {
		minPreviewWidth, minPreviewHeight := t.minPreviewSize(t.activePreviewOpts)
		switch t.activePreviewOpts.position {
		case posUp, posDown:
			minAreaHeight += minPreviewHeight
			minAreaWidth = max(minPreviewWidth, minAreaWidth)
		case posLeft, posRight:
			minAreaWidth += minPreviewWidth
			minAreaHeight = max(minPreviewHeight, minAreaHeight)
		}
	}
	adjust(1, 3, screenWidth, minAreaWidth)
	adjust(0, 2, screenHeight, minAreaHeight)

	return screenWidth, screenHeight, marginInt, paddingInt
}

func (t *Terminal) forceRerenderList() {
	t.prevLines = make([]itemLine, len(t.prevLines))
}

func (t *Terminal) hasHeaderWindow() bool {
	if !t.headerVisible {
		return false
	}
	if t.hasHeaderLinesWindow() {
		return len(t.header0) > 0
	}
	if t.headerBorderShape.Visible() {
		return len(t.header0)+t.headerLines > 0
	}
	return t.inputBorderShape.Visible()
}

func (t *Terminal) hasHeaderLinesWindow() bool {
	w, _ := t.determineHeaderLinesShape()
	return w
}

func (t *Terminal) determineHeaderLinesShape() (bool, tui.BorderShape) {
	if !t.headerVisible || t.headerLines == 0 {
		return false, tui.BorderNone
	}

	// --header-lines-border is set
	if t.headerLinesShape != tui.BorderUndefined {
		return true, t.headerLinesShape
	}

	// --header-lines-border is not set, determine if we should use
	// the style of --header-border
	shape := tui.BorderNone
	if len(t.header0) == 0 {
		shape = t.headerBorderShape
	}
	if shape == tui.BorderNone {
		shape = tui.BorderPhantom
	}

	// --layout reverse-list is set
	if t.layout == layoutReverseList {
		return true, shape
	}

	// Use header window instead
	if len(t.header0) == 0 {
		return false, t.headerBorderShape
	}

	// We have both types of headers, and we want to separate the two
	if t.headerFirst {
		return true, shape
	}

	return false, tui.BorderNone
}

func (t *Terminal) resizeWindows(forcePreview bool, redrawBorder bool) {
	t.clearNumLinesCache()
	t.forcePreview = forcePreview
	screenWidth, screenHeight, marginInt, paddingInt := t.adjustMarginAndPadding()
	width := screenWidth - marginInt[1] - marginInt[3]
	height := screenHeight - marginInt[0] - marginInt[2]

	t.prevLines = make([]itemLine, screenHeight)
	if t.border != nil && redrawBorder {
		t.border = nil
	}
	if t.window != nil {
		t.window = nil
	}
	if t.wborder != nil {
		t.wborder = nil
	}
	if t.headerWindow != nil {
		t.headerWindow = nil
	}
	if t.headerBorder != nil {
		t.headerBorder = nil
	}
	if t.footerWindow != nil {
		t.footerWindow = nil
	}
	if t.footerBorder != nil {
		t.footerBorder = nil
	}
	if t.headerLinesWindow != nil {
		t.headerLinesWindow = nil
	}
	if t.headerLinesBorder != nil {
		t.headerLinesBorder = nil
	}
	if t.inputWindow != nil {
		t.inputWindow = nil
	}
	if t.inputBorder != nil {
		t.inputBorder = nil
	}
	if t.pborder != nil {
		t.pborder = nil
	}
	hadPreviewWindow := t.hasPreviewWindow()
	if hadPreviewWindow {
		t.pwindow = nil
	}
	// Reset preview version so that full redraw occurs
	t.previewed.version = 0

	bw := t.borderWidth
	offsets := [4]int{} // TRWH
	if t.borderShape.HasTop() {
		offsets[0] -= 1
		offsets[3] += 1
	}
	if t.borderShape.HasRight() {
		offsets[2] += 1 + bw
	}
	if t.borderShape.HasBottom() {
		offsets[3] += 1
	}
	if t.borderShape.HasLeft() {
		offsets[1] -= 1 + bw
		offsets[2] += 1 + bw
	}
	if t.border == nil && t.borderShape.Visible() {
		t.border = t.tui.NewWindow(
			marginInt[0]+offsets[0], marginInt[3]+offsets[1], width+offsets[2], height+offsets[3],
			tui.WindowBase, tui.MakeBorderStyle(t.borderShape, t.unicode), true)
	}

	// Add padding to margin
	for idx, val := range paddingInt {
		marginInt[idx] += val
	}
	width -= paddingInt[1] + paddingInt[3]
	height -= paddingInt[0] + paddingInt[2]

	// Adjust position and size of the list window if input border is set
	inputBorderHeight := 0
	availableLines := height

	shift := 0
	shrink := 0
	hasHeaderWindow := t.hasHeaderWindow()
	hasFooterWindow := len(t.footer) > 0
	hasHeaderLinesWindow, headerLinesShape := t.determineHeaderLinesShape()
	hasInputWindow := !t.inputless && (t.inputBorderShape.Visible() || hasHeaderWindow || hasHeaderLinesWindow)
	inputWindowHeight := 2
	if t.noSeparatorLine() {
		inputWindowHeight--
	}
	if hasInputWindow {
		inputBorderHeight = util.Constrain(borderLines(t.inputBorderShape)+inputWindowHeight, 0, availableLines)
		if t.layout == layoutReverse {
			shift = inputBorderHeight
			shrink = inputBorderHeight
		} else {
			shrink = inputBorderHeight
		}
		availableLines -= inputBorderHeight
	} else if !t.inputless {
		availableLines -= inputWindowHeight
	}

	// FIXME: Needed?
	if t.needPreviewWindow() {
		_, minPreviewHeight := t.minPreviewSize(t.activePreviewOpts)
		switch t.activePreviewOpts.position {
		case posUp, posDown:
			availableLines -= minPreviewHeight
		}
	}

	// Adjust position and size of the list window if header border is set
	headerBorderHeight := 0
	if hasHeaderWindow {
		headerWindowHeight := t.visibleHeaderLines()
		if hasHeaderLinesWindow {
			headerWindowHeight -= t.headerLines
		}
		headerBorderHeight = util.Constrain(borderLines(t.headerBorderShape)+headerWindowHeight, 0, availableLines)
		if t.layout == layoutReverse {
			shift += headerBorderHeight
			shrink += headerBorderHeight
		} else {
			shrink += headerBorderHeight
		}
		availableLines -= headerBorderHeight
	}

	headerLinesHeight := 0
	if hasHeaderLinesWindow {
		headerLinesHeight = util.Constrain(borderLines(headerLinesShape)+t.headerLines, 0, availableLines)
		if t.layout != layoutDefault {
			shift += headerLinesHeight
			shrink += headerLinesHeight
		} else {
			shrink += headerLinesHeight
		}
		availableLines -= headerLinesHeight
	}

	footerBorderHeight := 0
	if hasFooterWindow {
		// Footer lines should not take all available lines
		footerBorderHeight = util.Constrain(borderLines(t.footerBorderShape)+len(t.footer), 0, availableLines)
		shrink += footerBorderHeight
		if t.layout != layoutReverse {
			shift += footerBorderHeight
		}
		availableLines -= footerBorderHeight
	}

	// Set up list border
	hasListBorder := t.listBorderShape.Visible()
	innerWidth := width
	innerHeight := height
	innerMarginInt := marginInt
	innerBorderFn := func(top int, left int, width int, height int) {
		if hasListBorder {
			t.wborder = t.tui.NewWindow(
				top+shift, left, width, height-shrink, tui.WindowList, tui.MakeBorderStyle(t.listBorderShape, t.unicode), false)
		}
	}
	if hasListBorder {
		if t.listBorderShape.HasTop() {
			innerHeight--
			innerMarginInt[0]++
		}
		if t.listBorderShape.HasBottom() {
			innerHeight--
		}
		if t.listBorderShape.HasLeft() {
			innerWidth -= 2
			innerMarginInt[3] += 2
		}
		if t.listBorderShape.HasRight() {
			innerWidth--
		}
	}

	t.areaLines = height
	t.areaColumns = width

	// If none of the inner borders has the right side, but the outer border does, increase the list width by 1 column
	listStickToRight := t.borderShape.HasRight() && !t.listBorderShape.HasRight() && !t.inputBorderShape.HasRight() &&
		(!t.headerVisible || !t.headerBorderShape.HasRight() || t.visibleHeaderLines() == 0)

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
				shape := previewOpts.Border()
				previewBorder := tui.MakeBorderStyle(shape, t.unicode)
				t.pborder = t.tui.NewWindow(y, x, w, h, tui.WindowPreview, previewBorder, false)
				pwidth -= borderColumns(shape, bw)
				pheight -= borderLines(shape)
				if shape.HasLeft() {
					x += 1 + bw
				}
				if shape.HasTop() {
					y += 1
				}
				if len(t.scrollbar) > 0 && !shape.HasRight() {
					// Need a column to show scrollbar
					pwidth -= 1
				}
				t.pwindow = t.tui.NewWindow(y, x, pwidth, pheight, tui.WindowPreview, noBorder, true)
				t.pwindow.SetWrapSign(t.wrapSign, t.wrapSignWidth)
				if !hadPreviewWindow {
					t.pwindow.Erase()
				}
			}
			minPreviewWidth, minPreviewHeight := t.minPreviewSize(previewOpts)
			switch previewOpts.position {
			case posUp, posDown:
				minWindowHeight := minHeight
				if t.inputless {
					minWindowHeight--
				}
				if t.noSeparatorLine() {
					minWindowHeight--
				}
				pheight := calculateSize(height, previewOpts.size, minWindowHeight, minPreviewHeight)
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

				listStickToRight = listStickToRight && !previewOpts.Border().HasRight()
				if listStickToRight {
					innerWidth++
					width++
				}

				pheight = util.Constrain(pheight, minPreviewHeight, availableLines)

				if previewOpts.position == posUp {
					innerBorderFn(marginInt[0]+pheight, marginInt[3], width, height-pheight)
					t.window = t.tui.NewWindow(
						innerMarginInt[0]+pheight+shift, innerMarginInt[3], innerWidth, innerHeight-pheight-shrink, tui.WindowList, noBorder, true)
					createPreviewWindow(marginInt[0], marginInt[3], width, pheight)
				} else {
					innerBorderFn(marginInt[0], marginInt[3], width, height-pheight)
					t.window = t.tui.NewWindow(
						innerMarginInt[0]+shift, innerMarginInt[3], innerWidth, innerHeight-pheight-shrink, tui.WindowList, noBorder, true)
					createPreviewWindow(marginInt[0]+height-pheight, marginInt[3], width, pheight)
				}
			case posLeft, posRight:
				minListWidth := minWidth
				if t.listBorderShape.HasLeft() {
					minListWidth += 2
				}
				if t.listBorderShape.HasRight() {
					minListWidth++
				}
				pwidth := calculateSize(width, previewOpts.size, minListWidth, minPreviewWidth)
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
					if t.borderShape.HasRight() && !hasListBorder {
						innerWidth++
					}
					// Add a 1-column margin between the preview window and the main window
					m := 0
					if !hasListBorder {
						m = 1
					}
					t.window = t.tui.NewWindow(
						innerMarginInt[0]+shift, innerMarginInt[3]+pwidth+m, innerWidth-pwidth-m, innerHeight-shrink, tui.WindowList, noBorder, true)

					// Clear characters on the margin
					// fzf --bind 'space:toggle-preview' --preview ':' --preview-window left,1
					if !hasListBorder {
						for y := 0; y < innerHeight; y++ {
							t.window.Move(y, -1)
							t.window.Print(" ")
						}
					}
					// fzf --bind 'space:toggle-preview' --preview ':' --preview-window left,1,border-none
					if !previewOpts.Border().HasRight() {
						for y := 0; y < innerHeight; y++ {
							t.window.Move(y, -2)
							t.window.Print(" ")
						}
					}

					innerBorderFn(marginInt[0], marginInt[3]+pwidth, width-pwidth, height)
					createPreviewWindow(marginInt[0], marginInt[3], pwidth, height)
				} else {
					// NOTE: Relaxed condition for the following cases
					//  fzf --preview 'seq 500' --preview-window border-left --border
					//  fzf --preview 'seq 500' --preview-window border-left --border --list-border
					//  fzf --preview 'seq 500' --preview-window border-left --border --input-border
					listStickToRight = t.borderShape.HasRight() && !previewOpts.Border().HasRight()
					if listStickToRight {
						innerWidth++
						width++
					}
					innerBorderFn(marginInt[0], marginInt[3], width-pwidth, height)
					t.window = t.tui.NewWindow(
						innerMarginInt[0]+shift, innerMarginInt[3], innerWidth-pwidth, innerHeight-shrink, tui.WindowList, noBorder, true)
					x := marginInt[3] + width - pwidth
					createPreviewWindow(marginInt[0], x, pwidth, height)
				}
			}
		}
		resizePreviewWindows(&t.previewOpts)

		if t.borderShape.HasRight() && !listStickToRight {
			// Need to clear the extra margin between the borders
			// fzf --preview 'seq 1000' --preview-window border-left --bind space:change-preview-window:border-rounded --border vertical
			// fzf --preview 'seq 1000' --preview-window up,hidden --bind space:toggle-preview --border vertical
			y := 0
			if t.borderShape.HasTop() {
				y++
			}
			maxY := t.border.Height()
			if t.borderShape.HasBottom() {
				maxY--
			}
			for ; y < maxY; y++ {
				t.border.Move(y, t.border.Width()-2)
				t.border.Print(" ")
			}
		}
	} else {
		t.activePreviewOpts = &t.previewOpts
	}

	// Without preview window
	if t.window == nil {
		if listStickToRight {
			// Put scrollbar closer to the right border for consistent look
			innerWidth++
			width++
		}
		innerBorderFn(marginInt[0], marginInt[3], width, height)
		t.window = t.tui.NewWindow(
			innerMarginInt[0]+shift,
			innerMarginInt[3],
			innerWidth,
			innerHeight-shrink, tui.WindowList, noBorder, true)
	}

	if len(t.scrollbar) == 0 {
		for y := 0; y < t.window.Height(); y++ {
			t.window.Move(y, t.window.Width()-1)
			t.window.Print(" ")
		}
	}

	createInnerWindow := func(b tui.Window, shape tui.BorderShape, windowType tui.WindowType, shift int) tui.Window {
		top := b.Top()
		left := b.Left() + shift
		if shape.HasTop() {
			top++
		}
		if shape.HasLeft() {
			left += t.borderWidth + 1
		}
		width := b.Width() - borderColumns(shape, t.borderWidth) - shift
		if shape.HasRight() {
			width++
		}
		// Make sure that the width does not exceed the list width
		width = min(t.window.Width()+t.headerIndentImpl(0, shape), width)
		height := b.Height() - borderLines(shape)
		return t.tui.NewWindow(top, left, width, height, windowType, noBorder, true)
	}

	// Set up input border
	w := t.wborder
	if t.wborder == nil {
		w = t.window
	}

	if hasInputWindow {
		var btop int
		if (hasHeaderWindow || hasHeaderLinesWindow) && t.headerFirst {
			switch t.layout {
			case layoutDefault:
				btop = w.Top() + w.Height()
				// If both headers are present, the header lines are displayed with the list
				if hasHeaderWindow && hasHeaderLinesWindow {
					btop += headerLinesHeight
				}
			case layoutReverse:
				btop = w.Top() - inputBorderHeight
				if hasHeaderWindow && hasHeaderLinesWindow {
					btop -= headerLinesHeight
				}
			case layoutReverseList:
				btop = w.Top() + w.Height()
			}
		} else {
			switch t.layout {
			case layoutDefault:
				btop = w.Top() + w.Height() + headerBorderHeight + headerLinesHeight
			case layoutReverse:
				btop = w.Top() - shrink + footerBorderHeight
			case layoutReverseList:
				btop = w.Top() + w.Height() + headerBorderHeight
			}
		}
		shift := 0
		if !t.inputBorderShape.HasLeft() && t.listBorderShape.HasLeft() {
			shift += t.borderWidth + 1
		}
		t.inputBorder = t.tui.NewWindow(
			btop,
			w.Left(),
			w.Width(),
			inputBorderHeight, tui.WindowInput, tui.MakeBorderStyle(t.inputBorderShape, t.unicode), true)
		if shift > 0 && !t.inputBorderShape.Visible() {
			// Small box on the left to erase the residue
			// e.g.
			//  fzf --list-border --header-border --bind 'space:change-header(hello),enter:change-header()'
			t.tui.NewWindow(btop, w.Left(), shift, inputBorderHeight, tui.WindowInput, noBorder, false).Erase()
		}
		t.inputWindow = createInnerWindow(t.inputBorder, t.inputBorderShape, tui.WindowInput, shift)
	}

	// Set up header border
	if hasHeaderWindow {
		var btop int
		if hasInputWindow && t.headerFirst {
			if t.layout == layoutReverse {
				btop = w.Top() - shrink + footerBorderHeight
			} else if t.layout == layoutReverseList {
				btop = w.Top() + w.Height() + inputBorderHeight
			} else {
				btop = w.Top() + w.Height() + inputBorderHeight + headerLinesHeight
			}
		} else {
			if t.layout == layoutReverse {
				btop = w.Top() - headerBorderHeight - headerLinesHeight
			} else if t.layout == layoutReverseList {
				btop = w.Top() + w.Height()
			} else {
				btop = w.Top() + w.Height() + headerLinesHeight
			}
		}
		t.headerBorder = t.tui.NewWindow(
			btop,
			w.Left(),
			w.Width(),
			headerBorderHeight, tui.WindowHeader, tui.MakeBorderStyle(t.headerBorderShape, t.unicode), true)
		t.headerWindow = createInnerWindow(t.headerBorder, t.headerBorderShape, tui.WindowHeader, 0)
	}

	// Set up header lines border
	if hasHeaderLinesWindow {
		var btop int
		// NOTE: We still have to handle --header-first here in case
		// --header-lines-border is set. Can't we just use header window instead
		// with the style? So we can display header label.
		//   fzf --header-lines 3 --header-label hello --header-border
		//   fzf --header-lines 3 --header-label hello --header-lines-border
		headerFirst := t.headerFirst && len(t.header0) == 0

		if headerFirst {
			if t.layout == layoutDefault {
				btop = w.Top() + w.Height() + inputBorderHeight
			} else if t.layout == layoutReverse {
				btop = w.Top() - headerLinesHeight - inputBorderHeight
			} else {
				btop = w.Top() - headerLinesHeight
			}
		} else {
			if t.layout != layoutDefault {
				btop = w.Top() - headerLinesHeight
			} else {
				btop = w.Top() + w.Height()
			}
		}
		t.headerLinesBorder = t.tui.NewWindow(
			btop,
			w.Left(),
			w.Width(),
			headerLinesHeight, tui.WindowHeader, tui.MakeBorderStyle(headerLinesShape, t.unicode), true)
		t.headerLinesWindow = createInnerWindow(t.headerLinesBorder, headerLinesShape, tui.WindowHeader, 0)
	}

	// Set up footer
	if hasFooterWindow {
		var btop int
		if t.layout == layoutReverse {
			btop = w.Top() + w.Height()
		} else if t.layout == layoutReverseList {
			btop = w.Top() - footerBorderHeight - headerLinesHeight
		} else {
			btop = w.Top() - footerBorderHeight
		}
		t.footerBorder = t.tui.NewWindow(
			btop,
			w.Left(),
			w.Width(),
			footerBorderHeight, tui.WindowFooter, tui.MakeBorderStyle(t.footerBorderShape, t.unicode), true)
		t.footerWindow = createInnerWindow(t.footerBorder, t.footerBorderShape, tui.WindowFooter, 0)
	}

	// Print border label
	t.printLabel(t.wborder, t.listLabel, t.listLabelOpts, t.listLabelLen, t.listBorderShape, false)
	t.printLabel(t.border, t.borderLabel, t.borderLabelOpts, t.borderLabelLen, t.borderShape, false)
	t.printLabel(t.pborder, t.previewLabel, t.previewLabelOpts, t.previewLabelLen, t.activePreviewOpts.Border(), false)
	t.printLabel(t.inputBorder, t.inputLabel, t.inputLabelOpts, t.inputLabelLen, t.inputBorderShape, false)
	t.printLabel(t.headerBorder, t.headerLabel, t.headerLabelOpts, t.headerLabelLen, t.headerBorderShape, false)
	t.printLabel(t.footerBorder, t.footerLabel, t.footerLabelOpts, t.footerLabelLen, t.footerBorderShape, false)
}

func (t *Terminal) printLabel(window tui.Window, render labelPrinter, opts labelOpts, length int, borderShape tui.BorderShape, redrawBorder bool) {
	if window == nil {
		return
	}

	if window.Height() == 0 {
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
			col = max(0, (window.Width()-length)/2)
		} else if opts.column < 0 {
			col = max(0, window.Width()+opts.column+1-length)
		} else {
			col = min(opts.column-1, window.Width()-length)
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
		if !t.inListWindow() && t.window != t.headerLinesWindow {
			// From bottom to top
			y = h - y - 1
		} else {
			/*
			 * List 1
			 * List 2
			 * Header 1
			 * Header 2
			 * Input 2
			 * Input 1
			 */
			i := t.visibleInputLinesInList()
			n := t.visibleHeaderLinesInList()
			if i > 0 && y < i {
				y = h - y - 1
			} else if n > 0 && y < i+n {
				y = h - y - 1
			} else {
				// Top to bottom
				y -= n + i
			}
		}
	}

	if clear {
		t.window.MoveAndClear(y, x)
	} else {
		t.window.Move(y, x)
	}
}

func (t *Terminal) truncateQuery() {
	// We're limiting the length of the query not to make fzf unresponsive when
	// the user accidentally pastes a huge chunk of text. Therefore, we're not
	// interested in the exact display width of the query. We just limit the
	// number of runes.
	t.input = t.input[:min(len(t.input), maxPatternLength)]
	t.cx = util.Constrain(t.cx, 0, len(t.input))
}

func (t *Terminal) updatePromptOffset() ([]rune, []rune) {
	w := t.window
	if t.inputWindow != nil {
		w = t.inputWindow
	}
	maxWidth := max(1, w.Width()-t.promptLen-1)

	_, overflow := t.trimLeft(t.input[:t.cx], maxWidth, 0)
	minOffset := int(overflow)
	maxOffset := minOffset + (maxWidth-max(0, maxWidth-t.cx))/2
	t.xoffset = util.Constrain(t.xoffset, minOffset, maxOffset)
	before, _ := t.trimLeft(t.input[t.xoffset:t.cx], maxWidth, 0)
	beforeLen := t.displayWidth(before)
	after, _ := t.trimRight(t.input[t.cx:], maxWidth-beforeLen)
	afterLen := t.displayWidth(after)
	t.queryLen = [2]int{beforeLen, afterLen}
	return before, after
}

func (t *Terminal) promptLine() int {
	if t.inputWindow != nil {
		return 0
	}
	if t.headerFirst {
		max := t.window.Height() - 1
		if max <= 0 { // Extremely short terminal
			return 0
		}
		if !t.noSeparatorLine() {
			max--
		}
		return min(t.visibleHeaderLinesInList(), max)
	}
	return 0
}

func (t *Terminal) placeCursor() {
	if t.inputless {
		return
	}
	x := t.promptLen + t.queryLen[0]
	if t.inputWindow != nil {
		y := t.inputWindow.Height() - 1
		if t.layout == layoutReverse {
			y = 0
		}
		x = min(x, t.inputWindow.Width()-1)
		t.inputWindow.Move(y, x)
		return
	}
	x = min(x, t.window.Width()-1)
	t.move(t.promptLine(), x, false)
}

func (t *Terminal) printPrompt() {
	if t.inputless {
		return
	}
	w := t.window
	if t.inputWindow != nil {
		w = t.inputWindow
	}
	if w.Height() == 0 {
		return
	}
	t.prompt()

	before, after := t.updatePromptOffset()
	if len(before) == 0 && len(after) == 0 && len(t.ghost) > 0 {
		maxWidth := max(1, w.Width()-t.promptLen-1)
		runes, _ := t.trimRight([]rune(t.ghost), maxWidth)
		w.CPrint(tui.ColGhost, string(runes))
		return
	}

	color := tui.ColInput
	if t.paused {
		color = tui.ColDisabled
	}
	w.CPrint(color, string(before))
	w.CPrint(color, string(after))
}

func (t *Terminal) trimMessage(message string, maxWidth int) string {
	if len(message) <= maxWidth {
		return message
	}
	runes, _ := t.trimRight([]rune(message), maxWidth-2)
	return string(runes) + strings.Repeat(".", util.Constrain(maxWidth, 0, 2))
}

func (t *Terminal) printInfo() {
	if t.inputless {
		return
	}
	t.withWindow(t.inputWindow, func() { t.printInfoImpl() })
}

func (t *Terminal) printInfoImpl() {
	if t.window.Width() <= 1 || t.window.Height() == 0 {
		return
	}
	pos := 0
	line := t.promptLine()
	maxHeight := t.window.Height()
	move := func(y int, x int, clear bool) bool {
		if y < 0 || y >= maxHeight {
			return false
		}
		t.move(y, x, clear)
		t.markOtherLine(y)
		return true
	}
	printSpinner := func() {
		if t.reading {
			duration := int64(spinnerDuration)
			idx := (time.Now().UnixNano() % (duration * int64(len(t.spinner)))) / duration
			t.window.CPrint(tui.ColSpinner, t.spinner[idx])
		} else {
			t.window.Print(" ") // Clear spinner
		}
	}
	printInfoPrefix := func() {
		str := t.infoPrefix
		maxWidth := t.window.Width() - pos
		width := util.StringWidth(str)
		if width > maxWidth {
			trimmed, _ := t.trimRight([]rune(str), maxWidth)
			str = string(trimmed)
			width = maxWidth
		}
		move(line, pos, t.separatorLen == 0)
		if t.reading {
			t.window.CPrint(tui.ColSpinner, str)
		} else {
			t.window.CPrint(tui.ColPrompt, str)
		}
		pos += width
	}
	printSeparator := func(fillLength int, pad bool) {
		if t.separatorLen > 0 {
			t.separator(t.window, fillLength)
			t.window.Print(" ")
		} else if pad {
			t.window.Print(strings.Repeat(" ", fillLength+1))
		}
	}

	if t.infoStyle == infoHidden {
		if t.separatorLen > 0 {
			if !move(line+1, 0, false) {
				return
			}
			printSeparator(t.window.Width()-1, false)
		}
		return
	}

	found := t.resultMerger.Length()
	total := max(found, t.count)
	output := fmt.Sprintf("%d/%d", found, total)
	if t.toggleSort {
		if t.sort {
			output += " +S"
		} else {
			output += " -S"
		}
	}
	if t.track.Global() {
		output += " +T"
	} else if t.track.Current() {
		output += " +t"
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
	var outputPrinter labelPrinter
	outputLen := len(output)
	if t.infoCommand != "" {
		output = t.executeCommand(t.infoCommand, false, true, true, true, output)
		outputPrinter, outputLen = t.ansiLabelPrinter(output, &tui.ColInfo, false)
	}

	shiftLen := t.queryLen[0] + t.queryLen[1] + 1
	if shiftLen == 1 && len(t.ghost) > 0 {
		shiftLen = util.StringWidth(t.ghost)
	}
	switch t.infoStyle {
	case infoDefault:
		if !move(line+1, 0, t.separatorLen == 0) {
			return
		}
		printSpinner()
		t.window.Print(" ") // Margin
		pos = 2
	case infoRight:
		if !move(line+1, 0, false) {
			return
		}
	case infoInlineRight:
		pos = t.promptLen + shiftLen
	case infoInline:
		pos = t.promptLen + shiftLen
		printInfoPrefix()
	}

	if t.infoStyle == infoRight {
		maxWidth := t.window.Width() - 1
		if t.reading {
			// Need space for spinner and a margin column
			maxWidth -= 2
		}
		var fillLength int
		if outputPrinter == nil {
			output = t.trimMessage(output, maxWidth)
			fillLength = t.window.Width() - len(output) - 2
		} else {
			fillLength = t.window.Width() - outputLen - 2
		}
		if t.reading {
			if fillLength >= 2 {
				printSeparator(fillLength-2, true)
			}
			printSpinner()
			t.window.Print(" ")
		} else if fillLength >= 0 {
			printSeparator(fillLength, true)
		}
		if outputPrinter == nil {
			t.window.CPrint(tui.ColInfo, output)
		} else {
			outputPrinter(t.window, maxWidth-1)
		}
		if fillLength >= 0 {
			t.window.Print(" ") // Margin
		}
		return
	}

	if t.infoStyle == infoInlineRight {
		if len(t.infoPrefix) == 0 {
			move(line, pos, false)
			newPos := max(pos, t.window.Width()-outputLen-3)
			t.window.Print(strings.Repeat(" ", newPos-pos))
			pos = newPos
			if pos < t.window.Width() {
				printSpinner()
				pos++
			}
			if pos < t.window.Width()-1 {
				t.window.Print(" ")
				pos++
			}
		} else {
			pos = max(pos, t.window.Width()-outputLen-util.StringWidth(t.infoPrefix)-1)
			printInfoPrefix()
		}
	}

	maxWidth := t.window.Width() - pos - 1
	if outputPrinter == nil {
		output = t.trimMessage(output, maxWidth)
		t.window.CPrint(tui.ColInfo, output)
	} else {
		outputPrinter(t.window, maxWidth)
	}
	if t.infoStyle == infoInline && outputLen < maxWidth-1 && t.reading {
		t.window.Print(" ")
		printSpinner()
		outputLen += 2
	}

	if t.infoStyle == infoInlineRight {
		if t.separatorLen > 0 {
			if !move(line+1, 0, false) {
				return
			}
			printSeparator(t.window.Width()-1, false)
		}
		return
	}

	fillLength := maxWidth - outputLen - 1
	if fillLength > 0 {
		t.window.CPrint(tui.ColSeparator, " ")
		printSeparator(fillLength, false)
	}
}

func (t *Terminal) resizeIfNeeded() bool {
	// Check if input border is used and input has changed
	if t.inputBorderShape.Visible() && t.inputWindow == nil && !t.inputless || t.inputWindow != nil && t.inputless {
		t.printAll()
		return true
	}

	// Check footer window
	if len(t.footer) > 0 && (t.footerWindow == nil || t.footerWindow.Height() != len(t.footer)) ||
		len(t.footer) == 0 && t.footerWindow != nil {
		t.printAll()
		return true
	}

	// Check if the header borders are used and header has changed
	allHeaderLines := t.visibleHeaderLines()
	primaryHeaderLines := allHeaderLines
	needHeaderWindow := t.hasHeaderWindow()
	needHeaderLinesWindow := t.hasHeaderLinesWindow()
	if needHeaderLinesWindow {
		primaryHeaderLines -= t.headerLines
	}
	// FIXME: Full redraw is triggered if there are too many lines in the header
	// so that the header window cannot display all of them.
	if (needHeaderWindow && t.headerWindow == nil) ||
		(!needHeaderWindow && t.headerWindow != nil) ||
		(needHeaderWindow && t.headerWindow != nil && primaryHeaderLines != t.headerWindow.Height()) ||
		(needHeaderLinesWindow && t.headerLinesWindow == nil) ||
		(!needHeaderLinesWindow && t.headerLinesWindow != nil) ||
		(needHeaderLinesWindow && t.headerLinesWindow != nil && t.headerLines != t.headerLinesWindow.Height()) {
		t.printAll()
		return true
	}
	return false
}

func (t *Terminal) printHeader() {
	if !t.headerVisible {
		return
	}

	t.withWindow(t.headerWindow, func() {
		var lines []string
		if !t.hasHeaderLinesWindow() {
			lines = t.header
		}
		t.printHeaderImpl(t.headerWindow, t.headerBorderShape, t.header0, lines)
	})
	if w, shape := t.determineHeaderLinesShape(); w {
		t.withWindow(t.headerLinesWindow, func() {
			t.printHeaderImpl(t.headerLinesWindow, shape, nil, t.header)
		})
	}
}

func (t *Terminal) printFooter() {
	if len(t.footer) == 0 {
		return
	}
	indentSize := t.headerIndent(t.footerBorderShape)
	indent := strings.Repeat(" ", indentSize)
	max := min(len(t.footer), t.footerWindow.Height())

	// Wrapping is not supported for footer
	wrap := t.wrap
	t.wrap = false
	t.withWindow(t.footerWindow, func() {
		var state *ansiState
		for idx, lineStr := range t.footer[:max] {
			line := idx
			if t.layout != layoutReverse {
				line = max - idx - 1
			}
			trimmed, colors, newState := extractColor(lineStr, state, nil)
			state = newState
			item := &Item{
				text:   util.ToChars([]byte(trimmed)),
				colors: colors}

			t.printHighlighted(Result{item: item},
				tui.ColFooter, tui.ColFooter, false, false, false, line, line, true,
				func(markerClass) int {
					t.footerWindow.Print(indent)
					return indentSize
				}, nil)
		}
	})
	t.wrap = wrap
}

func (t *Terminal) headerIndent(borderShape tui.BorderShape) int {
	return t.headerIndentImpl(t.pointerLen+t.markerLen, borderShape)
}

func (t *Terminal) headerIndentImpl(base int, borderShape tui.BorderShape) int {
	indentSize := base
	if t.listBorderShape.HasLeft() {
		indentSize += 1 + t.borderWidth
	}
	if borderShape.HasLeft() {
		indentSize -= 1 + t.borderWidth
		if indentSize < 0 {
			indentSize = 0
		}
	}
	return indentSize
}

func (t *Terminal) printHeaderImpl(window tui.Window, borderShape tui.BorderShape, lines1 []string, lines2 []string) {
	max := t.window.Height()
	if !t.inputless && t.inputWindow == nil && window == nil && t.headerFirst {
		max--
		if !t.noSeparatorLine() {
			max--
		}
	}
	var state *ansiState
	needReverse := false
	switch t.layout {
	case layoutDefault, layoutReverseList:
		needReverse = true
	}
	// Wrapping is not supported for header
	wrap := t.wrap

	// Align header with the list
	//   fzf --header-lines 3 --style full --no-list-border
	//   fzf --header-lines 3 --style full --no-header-border
	//   fzf --header-lines 3 --style full --no-header-border --no-input-border
	indentSize := t.pointerLen + t.markerLen
	if window != nil {
		indentSize = t.headerIndent(borderShape)
	}
	indent := strings.Repeat(" ", indentSize)
	t.wrap = false
	for idx, lineStr := range append(append([]string{}, lines1...), lines2...) {
		line := idx
		if needReverse && idx < len(lines1) {
			line = len(lines1) - idx - 1
		}
		if !t.inputless && t.inputWindow == nil && window == nil && !t.headerFirst {
			line++
			if !t.noSeparatorLine() {
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

		t.printHighlighted(Result{item: item},
			tui.ColHeader, tui.ColHeader, false, false, false, line, line, true,
			func(markerClass) int {
				t.window.Print(indent)
				return indentSize
			}, nil)
	}
	t.wrap = wrap
}

func (t *Terminal) canSpanMultiLines() bool {
	return (t.multiLine || t.wrap || t.gap > 0) && t.inListWindow()
}

func (t *Terminal) renderBar(line int, barRange [2]int) {
	// If the screen is not filled with the list in non-multi-line mode,
	// scrollbar is not visible at all. But in multi-line mode, we may need
	// to redraw the scrollbar character at the end.
	if t.canSpanMultiLines() {
		t.prevLines[line].hasBar = t.printBar(line, true, barRange)
	}
}

func (t *Terminal) renderEmptyLine(line int, barRange [2]int) {
	t.move(line, 0, true)
	t.markEmptyLine(line)
	t.renderBar(line, barRange)
}

func (t *Terminal) gutter(current bool, alt bool) {
	var color tui.ColorPair
	if current {
		color = tui.ColCurrentCursorEmpty
	} else if !t.raw && t.gutterReverse || t.raw && t.gutterRawReverse {
		if alt {
			color = tui.ColAltCursorEmpty
		} else {
			color = tui.ColCursorEmpty
		}
	} else {
		if alt {
			color = tui.ColAltCursorEmptyChar
		} else {
			color = tui.ColCursorEmptyChar
		}
	}
	gutter := t.pointerEmpty
	if t.raw {
		gutter = t.pointerEmptyRaw
	}
	t.window.CPrint(color, gutter)
}

func (t *Terminal) renderGapLine(line int, barRange [2]int, drawLine bool) {
	t.move(line, 0, false)
	t.gutter(false, false)
	t.window.Print(t.markerEmpty)
	x := t.pointerLen + t.markerLen

	width := t.window.Width() - x - 1
	if drawLine && t.gapLine != nil {
		t.gapLine(t.window, width)
	} else {
		t.move(line, x, true)
	}
	t.markOtherLine(line)
	t.renderBar(line, barRange)
	t.prevLines[line].width = width
}

func (t *Terminal) printList() {
	t.constrain()
	barLength, barStart := t.getScrollbar()

	maxy := t.maxItems() - 1
	count := t.merger.Length() - t.offset

	// Start line
	startLine := t.promptLines() + t.visibleHeaderLinesInList()
	maxy += startLine

	barRange := [2]int{startLine + barStart, startLine + barStart + barLength}
	for line, itemCount := startLine, 0; line <= maxy; line, itemCount = line+1, itemCount+1 {
		if itemCount < count {
			item := t.merger.Get(itemCount + t.offset)
			current := itemCount == t.cy-t.offset
			line = t.printItem(item, line, maxy, itemCount, current, barRange)
		} else if !t.prevLines[line].empty {
			t.renderEmptyLine(line, barRange)
		}
	}
}

func (t *Terminal) printBar(lineNum int, forceRedraw bool, barRange [2]int) bool {
	hasBar := lineNum >= barRange[0] && lineNum < barRange[1]
	if (hasBar != t.prevLines[lineNum].hasBar || forceRedraw) && t.window.Width() > 0 {
		if len(t.scrollbar) > 0 {
			t.move(lineNum, t.window.Width()-1, true)
			if hasBar {
				t.window.CPrint(tui.ColScrollbar, t.scrollbar)
			}
		}
	}
	return hasBar
}

func (t *Terminal) printItem(result Result, line int, maxLine int, index int, current bool, barRange [2]int) int {
	item := result.item
	matched := true
	var matchResult Result
	if t.raw {
		if matchResult, matched = t.matchMap[item.Index()]; matched {
			result = matchResult
		}
	}

	_, selected := t.selected[item.Index()]
	label := ""
	extraWidth := 0
	alt := false
	altBg := t.theme.AltBg
	selectedBg := selected && t.theme.SelectedBg != t.theme.ListBg
	if t.jumping != jumpDisabled {
		if index < len(t.jumpLabels) {
			// Striped
			if !altBg.IsColorDefined() {
				altBg = t.theme.DarkBg
				alt = index%2 == 0
			} else {
				alt = index%2 == 1
			}
			label = t.jumpLabels[index:index+1] + strings.Repeat(" ", max(0, t.pointerLen-1))
			if t.pointerLen == 0 {
				extraWidth = 1
			}
		}
	} else {
		if current {
			label = t.pointer
		}
		alt = !selectedBg && altBg.IsColorDefined() && index%2 == 1
	}

	// Avoid unnecessary redraw
	numLines, _ := t.numItemLines(item, maxLine-line+1)
	newLine := itemLine{valid: true, firstLine: line, numLines: numLines, cy: index + t.offset, current: current, selected: selected, label: label,
		result: result, queryLen: len(t.input), width: 0, hasBar: line >= barRange[0] && line < barRange[1], hidden: !matched}
	prevLine := t.prevLines[line]
	forceRedraw := !prevLine.valid || prevLine.other || prevLine.firstLine != newLine.firstLine
	printBar := func(lineNum int, forceRedraw bool) bool {
		return t.printBar(lineNum, forceRedraw, barRange)
	}

	if !forceRedraw &&
		prevLine.hidden == newLine.hidden &&
		prevLine.numLines == newLine.numLines &&
		prevLine.current == newLine.current &&
		prevLine.selected == newLine.selected &&
		prevLine.label == newLine.label &&
		prevLine.queryLen == newLine.queryLen &&
		prevLine.result == newLine.result {
		t.prevLines[line].hasBar = printBar(line, false)
		return line + numLines - 1
	}

	maxWidth := t.window.Width() - (t.pointerLen + t.markerLen + t.barCol())
	postTask := func(lineNum int, width int, wrapped bool, forceRedraw bool, lbg tui.ColorPair) {
		width += extraWidth
		if (current || selected || alt) && t.highlightLine || lbg.IsFullBgMarker() {
			color := tui.ColSelected
			if lbg.IsFullBgMarker() {
				color = lbg
			} else if current {
				color = tui.ColCurrent
			} else if alt {
				color = color.WithBg(altBg)
			}
			fillSpaces := maxWidth - width
			if wrapped {
				fillSpaces -= t.wrapSignWidth
			}
			if fillSpaces > 0 {
				t.window.CPrint(color, strings.Repeat(" ", fillSpaces))
			}
			newLine.width = maxWidth
		} else {
			var fillSpaces int
			if forceRedraw {
				fillSpaces = maxWidth - width
			} else {
				fillSpaces = t.prevLines[lineNum].width - width
			}
			if wrapped {
				fillSpaces -= t.wrapSignWidth
			}
			if fillSpaces > 0 {
				t.window.Print(strings.Repeat(" ", fillSpaces))
			}
			newLine.width = width
			if wrapped {
				newLine.width += t.wrapSignWidth
			}
		}
		// When width is 0, line is completely cleared. We need to redraw scrollbar
		newLine.hasBar = printBar(lineNum, forceRedraw || width == 0)
		t.prevLines[lineNum] = newLine
	}

	var finalLineNum int
	markerFor := func(markerClass markerClass) string {
		marker := t.marker
		switch markerClass {
		case markerTop:
			marker = t.markerMultiLine[0]
		case markerMiddle:
			marker = t.markerMultiLine[1]
		case markerBottom:
			marker = t.markerMultiLine[2]
		}
		return marker
	}
	indentSize := t.pointerLen + t.markerLen
	if current {
		preTask := func(marker markerClass) int {
			w := t.window.Width() - t.pointerLen
			if w < 0 {
				return indentSize
			}
			if len(label) == 0 {
				t.gutter(true, false)
			} else {
				t.window.CPrint(tui.ColCurrentCursor, label)
			}
			if w-t.markerLen < 0 {
				return indentSize
			}
			if selected {
				t.window.CPrint(tui.ColCurrentMarker, markerFor(marker))
			} else {
				t.window.CPrint(tui.ColCurrentSelectedEmpty, t.markerEmpty)
			}
			return indentSize
		}
		finalLineNum = t.printHighlighted(result, tui.ColCurrent, tui.ColCurrentMatch, true, true, !matched, line, maxLine, forceRedraw, preTask, postTask)
	} else {
		preTask := func(marker markerClass) int {
			w := t.window.Width() - t.pointerLen
			if w < 0 {
				return indentSize
			}
			if len(label) == 0 {
				t.gutter(false, index%2 == 1)
			} else {
				t.window.CPrint(tui.ColCursor, label)
			}
			if w-t.markerLen < 0 {
				return indentSize
			}
			if selected {
				t.window.CPrint(tui.ColMarker, markerFor(marker))
			} else {
				t.window.Print(t.markerEmpty)
			}
			return indentSize
		}
		var base, match tui.ColorPair
		if selected {
			base = tui.ColSelected
			match = tui.ColSelectedMatch
		} else {
			base = tui.ColNormal
			match = tui.ColMatch
		}
		if alt {
			base = base.WithBg(altBg)
			match = match.WithBg(altBg)
		}
		finalLineNum = t.printHighlighted(result, base, match, false, true, !matched, line, maxLine, forceRedraw, preTask, postTask)
	}
	for i := 0; i < t.gap && finalLineNum < maxLine; i++ {
		finalLineNum++
		t.renderGapLine(finalLineNum, barRange, i == t.gap-1)
	}
	return finalLineNum
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

func (t *Terminal) trimLeft(runes []rune, width int, ellipsisWidth int) ([]rune, int32) {
	width = max(0, width)
	var trimmed int32
	// Assume that each rune takes at least one column on screen
	if len(runes) > width {
		diff := len(runes) - width
		trimmed = int32(diff)
		runes = runes[diff:]
	}

	currentWidth := t.displayWidth(runes)

	for currentWidth > width-ellipsisWidth && len(runes) > 0 {
		runes = runes[1:]
		trimmed++
		currentWidth = t.displayWidthWithLimit(runes, ellipsisWidth, width)
	}
	return runes, trimmed
}

func (t *Terminal) overflow(runes []rune, max int) bool {
	return t.displayWidthWithLimit(runes, 0, max) > max
}

func (t *Terminal) printHighlighted(result Result, colBase tui.ColorPair, colMatch tui.ColorPair, current bool, match bool, hidden bool, lineNum int, maxLineNum int, forceRedraw bool, preTask func(markerClass) int, postTask func(int, int, bool, bool, tui.ColorPair)) int {
	var displayWidth int
	item := result.item
	matchOffsets := []Offset{}
	var pos *[]int
	if match && t.resultMerger.pattern != nil {
		_, matchOffsets, pos = t.resultMerger.pattern.MatchItem(item, true, t.slab)
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

	// When postTask is nil, we're printing header lines. No need to care about nth.
	var nthOffsets []Offset
	if postTask != nil {
		wholeCovered := len(t.nthCurrent) == 0
		for _, nth := range t.nthCurrent {
			// Do we still want to apply a different style when the current nth
			// covers the whole string? Probably not. And we can simplify the logic.
			if nth.IsFull() {
				wholeCovered = true
				break
			}
		}
		if wholeCovered && t.nthAttr&tui.AttrRegular > 0 {
			// But if 'nth' is set to 'regular', it's a sign that you're applying
			// a different style to the rest of the string. e.g. 'nth:regular,fg:dim'
			// In this case, we still need to apply it to clear the style.
			colBase = colBase.WithAttr(t.nthAttr)
		}
		if !wholeCovered && t.nthAttr > 0 {
			var tokens []Token
			if item.transformed != nil && item.transformed.revision == t.resultMerger.revision {
				tokens = item.transformed.tokens
			} else {
				tokens = Transform(Tokenize(item.text.ToString(), t.delimiter), t.nthCurrent)
			}
			nthOffsets = make([]Offset, len(tokens))
			for i, token := range tokens {
				start := token.prefixLength
				length := token.text.Length() - token.text.TrailingWhitespaces()
				end := start + int32(length)
				nthOffsets[i] = Offset{int32(start), int32(end)}
			}
			sort.Sort(ByOrder(nthOffsets))
		}
	}
	allOffsets := result.colorOffsets(charOffsets, nthOffsets, t.theme, colBase, colMatch, t.nthAttr, hidden)

	// Determine split offset for horizontal scrolling with freeze
	splitOffset1 := -1
	splitOffset2 := -1
	if t.hscroll && !t.wrap {
		var tokens []Token
		if t.freezeLeft > 0 || t.freezeRight > 0 {
			tokens = Tokenize(item.text.ToString(), t.delimiter)
		}

		// 0 | 1 | 2 | 3 | 4 | 5
		// ------>       <------
		if t.freezeLeft > 0 {
			if len(tokens) > 0 {
				token := tokens[min(t.freezeLeft, len(tokens))-1]
				splitOffset1 = int(token.prefixLength) + token.text.Length() - token.text.TrailingWhitespaces()
			}
		}
		if t.freezeRight > 0 {
			index := max(t.freezeLeft-1, len(tokens)-t.freezeRight-1)
			if index < 0 {
				splitOffset2 = 0
			} else if index >= t.freezeLeft {
				token := tokens[index]
				delimiter := strings.TrimLeftFunc(GetLastDelimiter(token.text.ToString(), t.delimiter), unicode.IsSpace)
				splitOffset2 = int(token.prefixLength) + token.text.Length() - len([]rune(delimiter))
			}
			splitOffset2 = max(splitOffset2, splitOffset1)
		}
	}

	maxLines := 1
	if t.canSpanMultiLines() {
		maxLines = maxLineNum - lineNum + 1
	}
	lines, overflow := t.itemLines(item, maxLines)
	numItemLines := len(lines)

	finalLineNum := lineNum
	topCutoff := false
	skipLines := 0
	wrapped := false
	if t.canSpanMultiLines() {
		// Cut off the upper lines in the 'default' layout
		if t.layout == layoutDefault && !current && maxLines == numItemLines && overflow {
			lines, _ = t.itemLines(item, math.MaxInt)

			// To see if the first visible line is wrapped, we need to check the last cut-off line
			prevLine := lines[len(lines)-maxLines-1]
			if len(prevLine) == 0 || prevLine[len(prevLine)-1] != '\n' {
				wrapped = true
			}

			skipLines = len(lines) - maxLines
			topCutoff = true
		}
	}
	from := 0
	for lineOffset := 0; lineOffset < len(lines) && (lineNum <= maxLineNum || maxLineNum == 0); lineOffset++ {
		line := lines[lineOffset]
		finalLineNum = lineNum
		offsets := []colorOffset{}
		lbg := tui.NoColorPair()
		for idx, offset := range allOffsets {
			lineEnd := int32(from + len(line))
			if offset.offset[0] >= lineEnd {
				if offset.IsFullBgMarker(lineEnd) {
					lbg = offset.color
				}
				allOffsets = allOffsets[idx:]
				break
			}

			if offset.offset[0] < int32(from) {
				continue
			}

			if offset.offset[1] < lineEnd {
				offset.offset[0] -= int32(from)
				offset.offset[1] -= int32(from)
				offsets = append(offsets, offset)
			} else {
				if idx < len(allOffsets)-1 {
					next := allOffsets[idx+1]
					if next.IsFullBgMarker(lineEnd) {
						lbg = next.color
						idx++
					}
				}
				dupe := offset
				dupe.offset[0] = lineEnd

				offset.offset[0] -= int32(from)
				offset.offset[1] = lineEnd
				offsets = append(offsets, offset)

				allOffsets = append([]colorOffset{dupe}, allOffsets[idx+1:]...)
				break
			}
		}
		splitOffsetLeft := 0
		if splitOffset1 >= 0 && splitOffset1 > from && splitOffset1 < from+len(line) {
			splitOffsetLeft = splitOffset1 - from
		}
		splitOffsetRight := -1
		if splitOffset2 >= 0 && splitOffset2 >= from && splitOffset2 < from+len(line) {
			splitOffsetRight = splitOffset2 - from
		}
		from += len(line)
		if lineOffset < skipLines {
			continue
		}
		actualLineOffset := lineOffset - skipLines

		var maxEnd int
		for _, offset := range offsets {
			if offset.match {
				maxEnd = max(maxEnd, int(offset.offset[1]))
			}
		}

		actualLineNum := lineNum
		if t.layout == layoutDefault {
			actualLineNum = (lineNum - actualLineOffset) + (numItemLines - actualLineOffset) - 1
		}
		t.move(actualLineNum, 0, forceRedraw && postTask == nil)

		indentSize := t.pointerLen + t.markerLen
		if preTask != nil {
			var marker markerClass
			if numItemLines == 1 {
				if !overflow {
					marker = markerSingle
				} else if topCutoff {
					marker = markerBottom
				} else {
					marker = markerTop
				}
			} else {
				if actualLineOffset == 0 { // First line
					if topCutoff {
						marker = markerMiddle
					} else {
						marker = markerTop
					}
				} else if actualLineOffset == numItemLines-1 { // Last line
					if topCutoff || !overflow {
						marker = markerBottom
					} else {
						marker = markerMiddle
					}
				} else {
					marker = markerMiddle
				}
			}

			indentSize = preTask(marker)
		}

		maxWidth := t.window.Width() - (indentSize + t.barCol())
		wasWrapped := false
		if wrapped {
			wrapSign := t.wrapSign
			if maxWidth < t.wrapSignWidth {
				runes, _ := util.Truncate(wrapSign, maxWidth)
				wrapSign = string(runes)
				maxWidth = 0
			} else {
				maxWidth -= t.wrapSignWidth
			}
			t.window.CPrint(colBase.WithAttr(tui.Dim), wrapSign)
			wrapped = false
			wasWrapped = true
		}

		if len(line) > 0 && line[len(line)-1] == '\n' && lineOffset < len(lines)-1 {
			line = line[:len(line)-1]
		} else {
			wrapped = true
		}

		frozenLeft := line[:splitOffsetLeft]
		middle := line[splitOffsetLeft:]
		frozenRight := []rune{}
		if splitOffsetRight >= splitOffsetLeft {
			middle = line[splitOffsetLeft:splitOffsetRight]
			frozenRight = line[splitOffsetRight:]
		}
		displayWidthSum := 0
		todo := [3]func(){}
		for fidx, runes := range [][]rune{frozenLeft, frozenRight, middle} {
			if len(runes) == 0 {
				continue
			}
			shift := 0
			maxe := maxEnd
			offs := make([]colorOffset, len(offsets))
			for idx := range offsets {
				offs[idx] = offsets[idx]
				if fidx == 1 && splitOffsetRight > 0 {
					shift = splitOffsetRight
				} else if fidx == 2 && splitOffsetLeft > 0 {
					shift = splitOffsetLeft
				}
				offs[idx].offset[0] -= int32(shift)
				offs[idx].offset[1] -= int32(shift)
			}
			maxe -= shift
			ellipsis, ellipsisWidth := util.Truncate(t.ellipsis, maxWidth)
			adjustedMaxWidth := maxWidth
			if fidx < 2 {
				// For frozen parts, reserve space for the ellipsis in the middle part
				adjustedMaxWidth -= ellipsisWidth
			}
			displayWidth = t.displayWidthWithLimit(runes, 0, adjustedMaxWidth)
			if !t.wrap && displayWidth > adjustedMaxWidth {
				maxe = util.Constrain(maxe+min(maxWidth/2-ellipsisWidth, t.hscrollOff), 0, len(runes))
				transformOffsets := func(diff int32, rightTrim bool) {
					for idx, offset := range offs {
						b, e := offset.offset[0], offset.offset[1]
						el := int32(len(ellipsis))
						b += el - diff
						e += el - diff
						b = max(b, el)
						if rightTrim {
							e = min(e, int32(maxWidth-ellipsisWidth))
						}
						offs[idx].offset[0] = b
						offs[idx].offset[1] = max(b, e)
					}
				}
				if t.hscroll {
					if fidx == 1 || fidx == 2 && t.keepRight && pos == nil {
						trimmed, diff := t.trimLeft(runes, maxWidth, ellipsisWidth)
						transformOffsets(diff, false)
						runes = append(ellipsis, trimmed...)
					} else if fidx == 0 || !t.overflow(runes[:maxe], maxWidth-ellipsisWidth) {
						// Stri..
						runes, _ = t.trimRight(runes, maxWidth-ellipsisWidth)
						runes = append(runes, ellipsis...)
					} else {
						// Stri..
						rightTrim := false
						if t.overflow(runes[maxe:], ellipsisWidth) {
							runes = append(runes[:maxe], ellipsis...)
							rightTrim = true
						}
						// ..ri..
						var diff int32
						runes, diff = t.trimLeft(runes, maxWidth, ellipsisWidth)

						// Transform offsets
						transformOffsets(diff, rightTrim)
						runes = append(ellipsis, runes...)
					}
				} else {
					runes, _ = t.trimRight(runes, maxWidth-ellipsisWidth)
					runes = append(runes, ellipsis...)

					for idx, offset := range offs {
						offs[idx].offset[0] = min(offset.offset[0], int32(maxWidth-len(ellipsis)))
						offs[idx].offset[1] = min(offset.offset[1], int32(maxWidth))
					}
				}
				displayWidth = t.displayWidthWithLimit(runes, 0, maxWidth)
			}
			displayWidthSum += displayWidth

			if maxWidth > 0 {
				color := colBase
				if hidden {
					color = color.WithFg(t.theme.Nomatch)
				}
				todo[fidx] = func() {
					t.printColoredString(t.window, runes, offs, color)
				}
			} else {
				break
			}
			maxWidth -= displayWidth
		}
		if todo[0] != nil {
			todo[0]()
		}
		if todo[2] != nil {
			todo[2]()
		}
		if todo[1] != nil {
			todo[1]()
		}
		if postTask != nil {
			postTask(actualLineNum, displayWidthSum, wasWrapped, forceRedraw, lbg)
		} else {
			t.markOtherLine(actualLineNum)
		}
		lineNum += 1
	}

	return finalLineNum
}

func (t *Terminal) printColoredString(window tui.Window, text []rune, offsets []colorOffset, colBase tui.ColorPair) {
	var index int32
	var substr string
	var prefixWidth int
	maxOffset := int32(len(text))
	var url *url
	for _, offset := range offsets {
		b := util.Constrain(offset.offset[0], index, maxOffset)
		e := util.Constrain(offset.offset[1], index, maxOffset)
		if url != nil && offset.url != url {
			url = nil
			window.LinkEnd()
		}

		substr, prefixWidth = t.processTabs(text[index:b], prefixWidth)
		window.CPrint(colBase, substr)

		if b < e {
			substr, prefixWidth = t.processTabs(text[b:e], prefixWidth)
			if url == nil && offset.url != nil {
				url = offset.url
				window.LinkBegin(url.uri, url.params)
			}
			window.CPrint(offset.color, substr)
		}

		index = e
		if index >= maxOffset {
			break
		}
	}
	if url != nil {
		window.LinkEnd()
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
		if !t.previewer.scrollable || !t.activePreviewOpts.info {
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
	if t.previewed.wipe && t.previewed.version != t.previewer.version {
		t.previewed.wipe = false
		t.pwindow.Erase()
	} else if unchanged {
		t.pwindow.MoveAndClear(0, 0) // Clear scroll offset display
	} else {
		t.previewed.filled = false
		// We don't erase the window here to avoid flickering during scroll.
		// However, tcell renderer uses double-buffering technique and there's no
		// flickering. So we just erase the window and make the rest of the code
		// simpler.
		if !t.pwindow.EraseMaybe() {
			t.pwindow.DrawBorder()
			t.pwindow.Move(0, 0)
		}
	}

	height := t.pwindow.Height()
	body := t.previewer.lines
	headerLines := t.activePreviewOpts.headerLines
	// Do not enable preview header lines if it's value is too large
	if headerLines > 0 && headerLines < min(len(body), height) {
		header := t.previewer.lines[0:headerLines]
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
	barLength, barStart := getScrollbar(1, len(body), effectiveHeight, min(len(body)-effectiveHeight, t.previewer.offset-headerLines))
	t.renderPreviewScrollbar(headerLines, barLength, barStart)
}

func (t *Terminal) makeImageBorder(width int, top bool) string {
	tl := "┌"
	tr := "┐"
	v := "╎"
	h := "╌"
	if !t.unicode {
		tl = "+"
		tr = "+"
		h = "-"
		v = "|"
	}
	repeat := max(0, width-2)
	if top {
		return tl + strings.Repeat(h, repeat) + tr
	}
	return v + strings.Repeat(" ", repeat) + v
}

func findPassThrough(line string) []int {
	loc := passThroughBeginRegex.FindStringIndex(line)
	if loc == nil {
		return nil
	}

	rest := line[loc[0]:]
	after := line[loc[1]:]
	if strings.HasPrefix(rest, "\x1bPtmux") { // Tmux
		eloc := passThroughEndTmuxRegex.FindStringIndex(after)
		if eloc == nil {
			return nil
		}
		return []int{loc[0], loc[1] + eloc[1]}
	} else if strings.HasPrefix(rest, "\x1b]1337;") { // iTerm2
		index := loc[1]
		for {
			after := line[index:]
			pos := strings.IndexAny(after, "\x1b\a")
			if pos < 0 {
				return nil
			}
			if after[pos] == '\a' {
				return []int{loc[0], index + pos + 1}
			}
			if pos < len(after)-1 && after[pos+1] == '\\' {
				return []int{loc[0], index + pos + 2}
			}
			index += pos + 1
		}
	}
	// Kitty
	pos := strings.Index(after, "\x1b\\")
	if pos < 0 {
		return nil
	}
	if pos < len(after)-2 && after[pos+2] == '\r' {
		return []int{loc[0], loc[1] + pos + 3}
	}
	return []int{loc[0], loc[1] + pos + 2}
}

func extractPassThroughs(line string) ([]string, string) {
	passThroughs := []string{}
	transformed := ""
	index := 0
	for {
		rest := line[index:]
		loc := findPassThrough(rest)
		if loc == nil {
			transformed += rest
			break
		}
		passThroughs = append(passThroughs, rest[loc[0]:loc[1]])
		transformed += line[index : index+loc[0]]
		index += loc[1]
	}

	return passThroughs, transformed
}

func (t *Terminal) renderPreviewText(height int, lines []string, lineNo int, unchanged bool) {
	maxWidth := t.pwindow.Width()
	var ansi *ansiState
	spinnerRedraw := t.pwindow.Y() == 0
	wiped := false
	image := false
	wireframe := false
	var index int
	var line string
Loop:
	for index, line = range lines {
		var lbg tui.Color = -1
		if ansi != nil {
			ansi.lbg = -1
		}

		passThroughs, line := extractPassThroughs(line)
		line = strings.TrimLeft(strings.TrimRight(line, "\r\n"), "\r")

		if lineNo >= height || t.pwindow.Y() == height-1 && t.pwindow.X() > 0 {
			t.previewed.filled = true
			t.previewer.scrollable = true
			break
		} else if lineNo >= 0 {
			x := t.pwindow.X()
			y := t.pwindow.Y()
			if spinnerRedraw && lineNo > 0 {
				spinnerRedraw = false
				t.renderPreviewSpinner()
				t.pwindow.Move(y, x)
			}
			for idx, passThrough := range passThroughs {
				// Handling Sixel/iTerm image
				requiredLines := 0
				isSixel := strings.HasPrefix(passThrough, "\x1bP")
				isItermImage := strings.HasPrefix(passThrough, "\x1b]1337;")
				isImage := isSixel || isItermImage
				if isImage {
					t.previewed.wipe = true
					// NOTE: We don't have a good way to get the height of an iTerm image,
					// so we assume that it requires the full height of the preview
					// window.
					requiredLines = height

					if isSixel && t.termSize.PxHeight > 0 {
						rows := strings.Count(passThrough, "-")
						requiredLines = int(math.Ceil(float64(rows*6*t.termSize.Lines) / float64(t.termSize.PxHeight)))
					}
				}

				// Render wireframe when the image cannot be displayed entirely
				if requiredLines > 0 && y+requiredLines > height {
					top := true
					for ; y < height; y++ {
						t.pwindow.MoveAndClear(y, 0)
						t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), tui.AttrRegular, t.makeImageBorder(maxWidth, top))
						top = false
					}
					wireframe = true
					t.previewed.filled = true
					t.previewer.scrollable = true
					break Loop
				}

				// Clear previous wireframe or any other text
				if (t.previewed.wireframe || isImage && !t.previewed.image) && !wiped {
					wiped = true
					for i := y + 1; i < height; i++ {
						t.pwindow.MoveAndClear(i, 0)
					}
				}
				image = image || isImage
				if idx == 0 {
					t.pwindow.MoveAndClear(y, x)
				} else {
					t.pwindow.Move(y, x)
				}
				t.tui.PassThrough(passThrough)

				if requiredLines > 0 {
					if y+requiredLines == height {
						t.pwindow.Move(height-1, maxWidth-1)
						t.previewed.filled = true
						break Loop
					}
					t.pwindow.MoveAndClear(y+requiredLines, 0)
				}
			}

			if len(passThroughs) > 0 && len(line) == 0 {
				continue
			}

			var fillRet tui.FillReturn
			prefixWidth := 0
			var url *url
			_, _, ansi = extractColor(line, ansi, func(str string, ansi *ansiState) bool {
				trimmed := []rune(str)
				isTrimmed := false
				if !t.activePreviewOpts.wrap {
					trimmed, isTrimmed = t.trimRight(trimmed, maxWidth-t.pwindow.X())
				}
				if url == nil && ansi != nil && ansi.url != nil {
					url = ansi.url
					t.pwindow.LinkBegin(url.uri, url.params)
				}
				if url != nil && (ansi == nil || ansi.url == nil) {
					url = nil
					t.pwindow.LinkEnd()
				}
				if ansi != nil {
					lbg = ansi.lbg
				} else {
					lbg = -1
				}
				str, width := t.processTabs(trimmed, prefixWidth)
				if width > prefixWidth {
					prefixWidth = width
					colored := ansi != nil && ansi.colored()
					if t.theme.Colored && colored {
						fillRet = t.pwindow.CFill(ansi.fg, ansi.bg, ansi.attr, str)
					} else {
						attr := tui.AttrRegular
						if colored {
							attr = ansi.attr
						}
						fillRet = t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), attr, str)
					}
				}
				return !isTrimmed &&
					(fillRet == tui.FillContinue || t.activePreviewOpts.wrap && fillRet == tui.FillNextLine)
			})
			if url != nil {
				t.pwindow.LinkEnd()
			}
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
			if t.theme.Colored && lbg >= 0 {
				fillRet = t.pwindow.CFill(-1, lbg, tui.AttrRegular,
					strings.Repeat(" ", t.pwindow.Width()-t.pwindow.X())+"\n")
			} else {
				fillRet = t.pwindow.Fill("\n")
			}
			if fillRet == tui.FillSuspend {
				t.previewed.filled = true
				break
			}
		}
		lineNo++
	}
	t.previewer.scrollable = t.previewer.scrollable || index < len(lines)-1
	t.previewed.image = image
	t.previewed.wireframe = wireframe
}

func (t *Terminal) renderPreviewScrollbar(yoff int, barLength int, barStart int) {
	height := t.pwindow.Height()
	w := t.pborder.Width()
	xw := [2]int{t.pwindow.Left(), t.pwindow.Width()}
	redraw := false
	if len(t.previewer.bar) != height || t.previewer.xw != xw {
		redraw = true
		t.previewer.bar = make([]bool, height)
		t.previewer.xw = xw
	}
	xshift := -1 - t.borderWidth
	if !t.activePreviewOpts.Border().HasRight() {
		xshift = -1
	}
	yshift := 1
	if !t.activePreviewOpts.Border().HasTop() {
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
	t.previewer.scrollable = t.previewer.offset > t.activePreviewOpts.headerLines || numLines > height
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
	t.resizeWindows(t.forcePreview, true)
	t.printList()
	t.printPrompt()
	t.printInfo()
	t.printHeader()
	t.printFooter()
	t.printPreview()
}

func (t *Terminal) flush() {
	t.placeCursor()
	if !t.suppress {
		windows := make([]tui.Window, 0, 9)
		if t.border != nil {
			windows = append(windows, t.border)
		}
		if t.pborder != nil {
			windows = append(windows, t.pborder)
		}
		if t.pwindow != nil {
			windows = append(windows, t.pwindow)
		}
		if t.wborder != nil {
			windows = append(windows, t.wborder)
		}
		if t.window != nil {
			windows = append(windows, t.window)
		}
		if t.headerBorder != nil {
			windows = append(windows, t.headerBorder)
		}
		if t.headerWindow != nil {
			windows = append(windows, t.headerWindow)
		}
		if t.headerLinesBorder != nil {
			windows = append(windows, t.headerLinesBorder)
		}
		if t.headerLinesWindow != nil {
			windows = append(windows, t.headerLinesWindow)
		}
		if t.inputBorder != nil {
			windows = append(windows, t.inputBorder)
		}
		if t.inputWindow != nil {
			windows = append(windows, t.inputWindow)
		}
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

	if strings.HasPrefix(match, "{fzf:") {
		// {fzf:*} are not determined by the current item
		flags.forceUpdate = true
		return false, match, flags
	}

	trimmed := ""
	for _, char := range match[1:] {
		switch char {
		case '*':
			flags.asterisk = true
		case '+':
			flags.plus = true
		case 's':
			flags.preserveSpace = true
		case 'n':
			flags.number = true
		case 'f':
			flags.file = true
		case 'r':
			flags.raw = true
		case 'q':
			flags.forceUpdate = true
			trimmed += string(char)
		default:
			trimmed += string(char)
		}
	}

	matchWithoutFlags := "{" + trimmed

	return false, matchWithoutFlags, flags
}

func hasPreviewFlags(template string) (slot bool, plus bool, asterisk bool, forceUpdate bool) {
	for _, match := range placeholder.FindAllString(template, -1) {
		escaped, _, flags := parsePlaceholder(match)
		if escaped {
			continue
		}
		slot = true
		plus = plus || flags.plus
		asterisk = asterisk || flags.asterisk
		forceUpdate = forceUpdate || flags.forceUpdate
	}
	return
}

type replacePlaceholderParams struct {
	template   string
	stripAnsi  bool
	delimiter  Delimiter
	printsep   string
	forcePlus  bool
	query      string
	allItems   [3][]*Item // current, select, and all matched items
	lastAction actionType
	prompt     string
	executor   *util.Executor
}

func (t *Terminal) replacePlaceholderInInitialCommand(template string) (string, []string) {
	return t.replacePlaceholder(template, false, string(t.input), [3][]*Item{nil, nil, nil})
}

func (t *Terminal) replacePlaceholder(template string, forcePlus bool, input string, list [3][]*Item) (string, []string) {
	return replacePlaceholder(replacePlaceholderParams{
		template:   template,
		stripAnsi:  t.ansi,
		delimiter:  t.delimiter,
		printsep:   t.printsep,
		forcePlus:  forcePlus,
		query:      input,
		allItems:   list,
		lastAction: t.lastAction,
		prompt:     t.promptString,
		executor:   t.executor,
	})
}

func (t *Terminal) evaluateScrollOffset() int {
	if t.pwindow == nil {
		return 0
	}

	// We only need the current item to calculate the scroll offset
	current := []*Item{t.currentItem()}
	if current[0] == nil {
		current = nil
	}
	replaced, tempFiles := t.replacePlaceholder(t.activePreviewOpts.scroll, false, "", [3][]*Item{current, nil, nil})
	removeFiles(tempFiles)
	offsetExpr := offsetTrimCharsRegex.ReplaceAllString(replaced, "")

	atoi := func(s string) int {
		n, e := strconv.Atoi(s)
		if e != nil {
			return 0
		}
		return n
	}

	base := -1
	height := max(0, t.pwindow.Height()-t.activePreviewOpts.headerLines)
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
	return max(0, base)
}

func replacePlaceholder(params replacePlaceholderParams) (string, []string) {
	tempFiles := []string{}
	current := params.allItems[0]
	selected := params.allItems[1]
	matched := params.allItems[2]

	// replace placeholders one by one
	replaced := placeholder.ReplaceAllStringFunc(params.template, func(match string) string {
		escaped, match, flags := parsePlaceholder(match)

		// this function implements the effects a placeholder has on items
		var replace func(*Item) string

		// placeholder types (escaped, query type, item type, token type)
		switch {
		case escaped:
			return match
		case match == "{q}" || match == "{fzf:query}":
			return params.executor.QuoteEntry(params.query)
		case strings.HasPrefix(match, "{q:"):
			if nth, err := splitNth(match[3 : len(match)-1]); err == nil {
				elems, prefixLength := awkTokenizer(params.query)
				tokens := withPrefixLengths(elems, prefixLength)
				trans := Transform(tokens, nth)
				result := JoinTokens(trans)
				if !flags.preserveSpace {
					result = strings.TrimSpace(result)
				}
				return params.executor.QuoteEntry(result)
			}

			return match
		case match == "{}":
			replace = func(item *Item) string {
				switch {
				case flags.number:
					n := item.text.Index
					if n == minItem.Index() {
						// NOTE: Item index should normally be positive, but if there's no
						// match, it will be set to math.MinInt32, and we don't want to
						// show that value. However, int32 can overflow, especially when
						// `--tail` is used with an endless input stream, and the index of
						// an item actually can be math.MinInt32. In that case, you're
						// getting an incorrect value, but we're going to ignore that for
						// now.
						return "''"
					}
					return strconv.Itoa(int(n))
				case flags.file || flags.raw:
					return item.AsString(params.stripAnsi)
				default:
					return params.executor.QuoteEntry(item.AsString(params.stripAnsi))
				}
			}
		case match == "{fzf:action}":
			return params.lastAction.Name()
		case match == "{fzf:prompt}":
			return params.executor.QuoteEntry(params.prompt)
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
				tokens := Tokenize(item.AsString(params.stripAnsi), params.delimiter)
				trans := Transform(tokens, ranges)
				str := JoinTokens(trans)

				// trim the last delimiter
				if params.delimiter.str != nil {
					str = strings.TrimSuffix(str, *params.delimiter.str)
				} else if params.delimiter.regex != nil {
					delims := params.delimiter.regex.FindAllStringIndex(str, -1)
					// make sure the delimiter is at the very end of the string
					if len(delims) > 0 && delims[len(delims)-1][1] == len(str) {
						str = str[:delims[len(delims)-1][0]]
					}
				}

				if !flags.preserveSpace {
					str = strings.TrimSpace(str)
				}
				if !flags.file && !flags.raw {
					str = params.executor.QuoteEntry(str)
				}
				return str
			}
		}

		// apply 'replace' function over proper set of items and return result

		items := current
		if flags.asterisk {
			items = matched
		} else if flags.plus || params.forcePlus {
			items = selected
		}
		replacements := make([]string, len(items))

		for idx, item := range items {
			replacements[idx] = replace(item)
		}

		if flags.file {
			file := WriteTemporaryFile(replacements, params.printsep)
			tempFiles = append(tempFiles, file)
			return file
		}
		return strings.Join(replacements, " ")
	})

	return replaced, tempFiles
}

func (t *Terminal) fullRedraw() {
	t.tui.Clear()
	t.tui.Refresh()
	t.printAll()
}

func (t *Terminal) captureLine(template string) string {
	return t.executeCommand(template, false, true, true, true, "")
}

func (t *Terminal) captureLines(template string) string {
	return t.executeCommand(template, false, true, true, false, "")
}

func (t *Terminal) captureAsync(a action, firstLineOnly bool, callback func(string)) {
	_, list := t.buildPlusList(a.a, false)
	command, tempFiles := t.replacePlaceholder(a.a, false, string(t.input), list)
	version := t.bgVersion
	cmd := t.executor.ExecCommand(command, true)
	cmd.Env = t.environ()
	item := func(proceed bool) {
		if proceed {
			out, _ := cmd.StdoutPipe()
			reader := bufio.NewReader(out)
			var output string
			if err := cmd.Start(); err == nil {
				runningCmd := runningCmd{cmd, tempFiles}
				t.runningCmds.Add(&runningCmd)
				if firstLineOnly {
					output, _ = reader.ReadString('\n')
					output = strings.TrimRight(output, "\r\n")
				} else {
					bytes, _ := io.ReadAll(reader)
					output = string(bytes)
				}
				cmd.Wait()
				t.runningCmds.Remove(&runningCmd)
			}
			t.callbackChan <- versionedCallback{version, func() { callback(output) }}
		}
		removeFiles(tempFiles)

	}
	queue, prs := t.bgQueue[a]
	if !prs {
		queue = []func(bool){}
	}
	queue = append(queue, item)
	t.bgQueue[a] = queue
}

func (t *Terminal) dispatchAsync() {
Loop:
	for a, queue := range t.bgQueue {
		delete(t.bgQueue, a)
		if len(queue) == 0 {
			continue
		}

		semaphore, prs := t.bgSemaphores[a]
		if !prs {
			semaphore = make(chan struct{}, maxBgProcessesPerAction)
			t.bgSemaphores[a] = semaphore
		}
		for _, item := range queue {
			select {
			// Acquire local semaphore
			case semaphore <- struct{}{}:
			default:
				// Failed to acquire local semaphore, putting only the last one back to the queue
				for _, item := range queue[:len(queue)-1] {
					item(false)
				}
				t.bgQueue[a] = queue[len(queue)-1:]
				continue Loop
			}
			todo := item
			go func() {
				// Acquire global semaphore
				t.bgSemaphore <- struct{}{}

				todo(true)
				// Release local semaphore
				<-semaphore
				// Release global semaphore
				<-t.bgSemaphore
			}()
		}
	}
}

func (t *Terminal) executeCommand(template string, forcePlus bool, background bool, capture bool, firstLineOnly bool, info string) string {
	line := ""
	valid, list := t.buildPlusList(template, forcePlus)
	// 'capture' is used for transform-* and we don't want to
	// return an empty string in those cases
	if !valid && !capture {
		return line
	}
	command, tempFiles := t.replacePlaceholder(template, forcePlus, string(t.input), list)
	cmd := t.executor.ExecCommand(command, false)
	cmd.Env = t.environ()
	if len(info) > 0 {
		cmd.Env = append(cmd.Env, "FZF_INFO="+info)
	}
	t.executing.Set(true)
	if !background {
		// Open a separate handle for tty input
		if in, _ := tui.TtyIn(t.ttyDefault); in != nil {
			cmd.Stdin = in
			if in != os.Stdin {
				defer in.Close()
			}
		}

		cmd.Stdout = os.Stdout
		if !util.IsTty(os.Stdout) {
			if out, _ := tui.TtyOut(t.ttyDefault); out != nil {
				cmd.Stdout = out
				defer out.Close()
			}
		}

		cmd.Stderr = os.Stderr
		if !util.IsTty(os.Stderr) {
			if out, _ := tui.TtyOut(t.ttyDefault); out != nil {
				cmd.Stderr = out
				defer out.Close()
			}
		}

		t.mutex.Unlock()
		if len(info) == 0 {
			t.uiMutex.Lock()
		}
		t.tui.Pause(true)
		cmd.Run()
		t.tui.Resume(true, false)
		t.mutex.Lock()
		// NOTE: Using t.reqBox.Set(reqFullRedraw...) instead can cause a deadlock
		t.fullRedraw()
		t.flush()
	} else {
		t.mutex.Unlock()
		if len(info) == 0 {
			t.uiMutex.Lock()
		}
		paused := atomic.Int32{}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(blockDuration):
				if paused.CompareAndSwap(0, 1) {
					t.tui.Pause(false)
				}
			}
		}()
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
		cancel()
		if paused.CompareAndSwap(1, 2) {
			t.tui.Resume(false, false)
		}
		t.mutex.Lock()

		// Redraw prompt in case the user has typed something after blockDuration
		if paused.Load() > 0 {
			// NOTE: Using t.reqBox.Set(reqXXX...) instead can cause a deadlock
			t.printPrompt()
			if t.infoStyle == infoInline || t.infoStyle == infoInlineRight {
				t.printInfo()
			}
		}
	}
	if len(info) == 0 {
		t.uiMutex.Unlock()
	}
	t.executing.Set(false)
	removeFiles(tempFiles)
	return line
}

func (t *Terminal) hasPreviewer() bool {
	return t.previewBox != nil
}

func (t *Terminal) needPreviewWindow() bool {
	return t.hasPreviewer() && len(t.previewOpts.command) > 0 && t.activePreviewOpts.Visible()
}

// Check if previewer is currently in action (invisible previewer with size 0 or visible previewer)
func (t *Terminal) canPreview() bool {
	return t.hasPreviewer() && (!t.activePreviewOpts.Visible() && !t.activePreviewOpts.hidden || t.hasPreviewWindow())
}

func (t *Terminal) hasPreviewWindow() bool {
	return t.pwindow != nil
}

func (t *Terminal) hasPreviewWindowOnRight() bool {
	return t.hasPreviewWindow() && t.activePreviewOpts.position == posRight
}

func (t *Terminal) currentItem() *Item {
	cnt := t.merger.Length()
	if t.cy >= 0 && cnt > 0 && cnt > t.cy {
		return t.merger.Get(t.cy).item
	}
	return nil
}

func (t *Terminal) isCurrentItemMatch() bool {
	cnt := t.merger.Length()
	if t.cy >= 0 && cnt > 0 && cnt > t.cy {
		if !t.raw {
			return true
		}
		item := t.merger.Get(t.cy).item
		return t.isItemMatch(item)
	}
	return false
}

func (t *Terminal) isItemMatch(item *Item) bool {
	_, matched := t.matchMap[item.Index()]
	return matched
}

func (t *Terminal) filterSelected() {
	filtered := make(map[int32]selectedItem)
	for k, v := range t.selected {
		if t.isItemMatch(v.item) {
			filtered[k] = v
		}
	}
	t.selected = filtered
}

func (t *Terminal) buildPlusList(template string, forcePlus bool) (bool, [3][]*Item) {
	current := t.currentItem()
	slot, plus, asterisk, forceUpdate := hasPreviewFlags(template)
	if !(!slot || forceUpdate || asterisk || (forcePlus || plus) && len(t.selected) > 0) {
		if current == nil {
			// Invalid
			return false, [3][]*Item{nil, nil, nil}
		}
		return true, [3][]*Item{{current}, {current}, nil}
	}

	// We would still want to update preview window even if there is no match if
	//   1. the command template contains {q}
	//   2. or it contains {+} and we have more than one item already selected.
	// To do so, we pass an empty Item instead of nil to trigger an update.
	if current == nil {
		current = &minItem
	}

	var all []*Item
	if asterisk {
		cnt := t.merger.Length()
		all = make([]*Item, cnt)
		for i := range cnt {
			all[i] = t.merger.Get(i).item
		}
	}

	var sels []*Item
	if len(t.selected) == 0 {
		sels = []*Item{current}
	} else if len(t.selected) > 0 {
		sels = make([]*Item, len(t.selected))
		for i, sel := range t.sortSelected() {
			sels[i] = sel.item
		}
	}
	return true, [3][]*Item{{current}, sels, all}
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

func (t *Terminal) killPreview() {
	select {
	case t.killChan <- true:
	default:
	}
}

func (t *Terminal) cancelPreview() {
	select {
	case t.killChan <- false:
	default:
	}
}

func (t *Terminal) pwindowSize() tui.TermSize {
	if t.pwindow == nil {
		return tui.TermSize{}
	}
	size := tui.TermSize{Lines: t.pwindow.Height(), Columns: t.pwindow.Width()}

	if t.termSize.PxWidth > 0 {
		size.PxWidth = size.Columns * t.termSize.PxWidth / t.termSize.Columns
		size.PxHeight = size.Lines * t.termSize.PxHeight / t.termSize.Lines
	}
	return size
}

func (t *Terminal) currentIndex() int32 {
	if currentItem := t.currentItem(); currentItem != nil {
		return currentItem.Index()
	}
	return minItem.Index()
}

func (t *Terminal) addClickHeaderWord(env []string) []string {
	/*
	 * echo $'HL1\nHL2' | fzf --header-lines 3 --header $'H1\nH2' --header-lines-border --bind 'click-header:preview:env | grep FZF_CLICK'
	 *
	 *   REVERSE      DEFAULT      REVERSE-LIST
	 *   H1      1            1    HL1     1
	 *   H2      2    HL2     2    HL2     2
	 *   -------      HL1     3            3
	 *   HL1     3    -------      -------
	 *   HL2     4    H1      4    H1      4
	 *           5    H2      5    H2      5
	 */
	clickHeaderLine := t.clickHeaderLine - 1
	if clickHeaderLine < 0 {
		// Never clicked on the header
		return env
	}

	// NOTE: t.header is padded with empty strings so that its size is equal to t.headerLines
	nthBase := 0
	headers := [2][]string{t.header, t.header0}
	if t.layout == layoutReverse {
		headers[0], headers[1] = headers[1], headers[0]
	}
	var trimmedLine string
	var words []Token
	var lineNum int
	for lineNum = 0; lineNum <= clickHeaderLine; lineNum++ {
		currentLine := lineNum == clickHeaderLine
		var line string
		if lineNum < len(headers[0]) {
			index := lineNum
			if t.layout == layoutDefault {
				index = len(headers[0]) - index - 1
			}
			line = headers[0][index]
		} else if lineNum-len(headers[0]) < len(headers[1]) {
			line = headers[1][lineNum-len(headers[0])]
		}
		if currentLine && len(line) == 0 {
			return env
		}

		// NOTE: We can't expand tabs here because the delimiter can contain tabs.
		trimmedLine, _, _ = extractColor(line, nil, nil)
		words = Tokenize(trimmedLine, t.delimiter)
		if currentLine {
			break
		} else {
			// TODO: Counting can be incorrect when the delimiter contains new line
			// characters, and there are empty lines in the header.
			nthBase += len(words)
		}
	}

	colNum := t.clickHeaderColumn - 1
	prefixWidth, prefixLength := 0, 0
	for idx, token := range words {
		prefixWidth += t.displayWidthWithPrefix(trimmedLine[prefixLength:token.prefixLength], prefixWidth)
		prefixLength = int(token.prefixLength)

		word, _ := t.processTabs(token.text.ToRunes(), prefixWidth)
		trimmed := strings.TrimRightFunc(word, unicode.IsSpace)
		trimWidth := t.displayWidthWithPrefix(trimmed, prefixWidth)

		// Find the position of the first non-space character in the word
		minPos := strings.IndexFunc(trimmed, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		if colNum >= minPos && colNum >= prefixWidth && colNum < prefixWidth+trimWidth {
			env = append(env, fmt.Sprintf("FZF_CLICK_HEADER_WORD=%s", trimmed))
			nth := fmt.Sprintf("FZF_CLICK_HEADER_NTH=%d", nthBase+idx+1)
			if lineNum == len(t.header)+len(t.header0)-1 && idx == len(words)-1 {
				nth += ".."
			}
			env = append(env, nth)
			return env
		}
	}
	return env
}

func (t *Terminal) addClickFooterWord(env []string) []string {
	clickFooterLine := t.clickFooterLine - 1
	if clickFooterLine < 0 || clickFooterLine >= len(t.footer) {
		// Never clicked on the footer
		return env
	}

	// NOTE: Unlike in click-header, we don't use --delimiter here, since we're
	// only interested in the word, not nth. Does this make sense?
	trimmed, _, _ := extractColor(t.footer[clickFooterLine], nil, nil)
	trimmed, _ = t.processTabs([]rune(trimmed), 0)
	words := Tokenize(trimmed, Delimiter{})
	colNum := t.clickFooterColumn - 1
	for _, token := range words {
		prefixWidth := int(token.prefixLength)
		word := token.text.ToString()
		trimmed := strings.TrimRightFunc(word, unicode.IsSpace)
		trimWidth := t.displayWidthWithPrefix(trimmed, prefixWidth)

		// Find the position of the first non-space character in the word
		minPos := strings.IndexFunc(trimmed, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		if colNum >= minPos && colNum >= prefixWidth && colNum < prefixWidth+trimWidth {
			env = append(env, fmt.Sprintf("FZF_CLICK_FOOTER_WORD=%s", trimmed))
			return env
		}
	}
	return env
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() error {
	// prof := profile.Start(profile.ProfilePath("/tmp/"))
	fitpad := <-t.startChan
	fit := fitpad.fit
	if fit >= 0 {
		pad := fitpad.pad
		t.tui.Resize(func(termHeight int) int {
			contentHeight := fit + t.extraLines()
			if t.needPreviewWindow() {
				if t.activePreviewOpts.aboveOrBelow() {
					if t.activePreviewOpts.size.percent {
						newContentHeight := int(float64(contentHeight) * 100. / (100. - t.activePreviewOpts.size.size))
						contentHeight = max(contentHeight+1+borderLines(t.activePreviewOpts.Border()), newContentHeight)
					} else {
						contentHeight += int(t.activePreviewOpts.size.size) + borderLines(t.activePreviewOpts.Border())
					}
				} else {
					// Minimum height if preview window can appear
					contentHeight = max(contentHeight, 1+borderLines(t.activePreviewOpts.Border()))
				}
			}
			return min(termHeight, contentHeight+pad)
		})
	}

	// Context
	ctx, cancel := context.WithCancel(context.Background())

	{ // Late initialization
		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case s := <-intChan:
					// Don't quit by SIGINT while executing because it should be for the executing command and not for fzf itself
					if !(s == os.Interrupt && t.executing.Get()) {
						t.reqBox.Set(reqQuit, nil)
					}
				}
			}
		}()

		if !t.tui.ShouldEmitResizeEvent() {
			resizeChan := make(chan os.Signal, 1)
			notifyOnResize(resizeChan) // Non-portable
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-resizeChan:
						t.reqBox.Set(reqResize, nil)
					}
				}
			}()
		}

		t.mutex.Lock()
		if err := t.initFunc(); err != nil {
			t.mutex.Unlock()
			cancel()
			t.eventBox.Set(EvtQuit, quitSignal{ExitError, err})
			return err
		}
		t.termSize = t.tui.Size()
		t.resizeWindows(false, false)
		t.window.Erase()
		t.mutex.Unlock()

		t.reqBox.Set(reqPrompt, nil)
		t.reqBox.Set(reqInfo, nil)
		t.reqBox.Set(reqHeader, nil)
		t.reqBox.Set(reqFooter, nil)
		if t.initDelay > 0 {
			go func() {
				timer := time.NewTimer(t.initDelay)
				<-timer.C
				t.reqBox.Set(reqActivate, nil)
			}()
		}

		// Keep the spinner spinning
		go func() {
			for t.running.Get() {
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
			stop := false
			t.previewBox.WaitFor(reqPreviewReady)
			for {
				requested := false
				var items [3][]*Item
				var commandTemplate string
				var env []string
				var query string
				initialOffset := 0
				t.previewBox.Wait(func(events *util.Events) {
					for req, value := range *events {
						switch req {
						case reqQuit:
							stop = true
							return
						case reqPreviewEnqueue:
							request := value.(previewRequest)
							commandTemplate = request.template
							initialOffset = request.scrollOffset
							items = request.list
							env = request.env
							query = request.query
							requested = true
						}
					}
					events.Clear()
				})
				if stop {
					break
				}
				if !requested {
					continue
				}
				version++
				// We don't display preview window if no match
				if items[0] != nil {
					command, tempFiles := t.replacePlaceholder(commandTemplate, false, query, items)
					cmd := t.executor.ExecCommand(command, true)
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
								case <-ctx.Done():
									break Loop
								case <-timer.C:
									t.reqBox.Set(reqPreviewDelayed, version)
								case immediately := <-t.killChan:
									if immediately {
										util.KillCommand(cmd)
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
						removeFiles(tempFiles)
					} else {
						// Failed to start the command. Report the error immediately.
						t.reqBox.Set(reqPreviewDisplay, previewResult{version, []string{err.Error()}, 0, ""})
					}
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
			t.previewBox.Set(reqPreviewEnqueue, previewRequest{command, t.evaluateScrollOffset(), list, t.environForPreview(), string(t.input)})
		}
	}

	go func() { // Render loop
		var focusedIndex = minItem.Index()
		var version int64 = -1
		running := true
		code := ExitError
		exit := func(getCode func() int) {
			if t.hasPreviewer() {
				t.previewBox.Set(reqQuit, nil)
			}
			if t.listener != nil {
				t.listener.Close()
			}
			t.tui.Close()
			code = getCode()
			if code <= ExitNoMatch && t.history != nil {
				t.history.append(string(t.input))
			}
			t.runningCmds.ForEach(func(cmd *runningCmd) {
				util.KillCommand(cmd.cmd)
				removeFiles(cmd.tempFiles)
			})
			running = false
			t.mutex.Unlock()
		}

		for running {
			t.reqBox.Wait(func(events *util.Events) {
				defer events.Clear()

				// Sort events.
				// e.g. Make sure that reqPrompt is processed before reqInfo
				keys := make([]int, 0, len(*events))
				for key := range *events {
					keys = append(keys, int(key))
				}
				sort.Ints(keys)

				// t.uiMutex must be locked first to avoid deadlock. Execute actions
				// will 1. unlock t.mutex to allow GET endpoint and 2. lock t.uiMutex
				// to block rendering during the execution.
				//
				// T1           T2 (good)       |  T1            T2 (bad)
				//               L t.uiMutex    |
				//  L t.mutex                   |   L t.mutex
				//  U t.mutex                   |   U t.mutex
				//               L t.mutex      |                 L t.mutex
				//               U t.mutex      |   L t.uiMutex
				//               U t.uiMutex    |                 L t.uiMutex!!
				//  L t.uiMutex                 |
				//                              |   L t.mutex!!
				//  L t.mutex                   |   U t.uiMutex
				//  U t.uiMutex                 |
				t.uiMutex.Lock()
				t.mutex.Lock()
				info := false
				header := false
				footer := false
				for _, key := range keys {
					req := util.EventType(key)
					value := (*events)[req]
					switch req {
					case reqPrompt:
						t.printPrompt()
						if t.infoStyle == infoInline || t.infoStyle == infoInlineRight {
							info = true
						}
					case reqInfo:
						info = true
					case reqList:
						t.printList()
						currentIndex := t.currentIndex()
						if t.track.Current() && t.track.index != currentIndex {
							t.track = trackDisabled
							info = true
						}
						focusChanged := focusedIndex != currentIndex
						if (t.hasFocusActions || t.infoCommand != "") && focusChanged && currentIndex != t.lastFocus {
							t.lastFocus = currentIndex
							t.eventChan <- tui.Focus.AsEvent()
							if t.infoCommand != "" {
								info = true
							}
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
						header = true
					case reqFooter:
						footer = true
					case reqActivate:
						t.suppress = false
						if t.hasPreviewer() {
							t.previewBox.Set(reqPreviewReady, nil)
						}
					case reqRedrawInputLabel:
						t.printLabel(t.inputBorder, t.inputLabel, t.inputLabelOpts, t.inputLabelLen, t.inputBorderShape, true)
					case reqRedrawHeaderLabel:
						t.printLabel(t.headerBorder, t.headerLabel, t.headerLabelOpts, t.headerLabelLen, t.headerBorderShape, true)
					case reqRedrawFooterLabel:
						t.printLabel(t.footerBorder, t.footerLabel, t.footerLabelOpts, t.footerLabelLen, t.footerBorderShape, true)
					case reqRedrawListLabel:
						t.printLabel(t.wborder, t.listLabel, t.listLabelOpts, t.listLabelLen, t.listBorderShape, true)
					case reqRedrawBorderLabel:
						t.printLabel(t.border, t.borderLabel, t.borderLabelOpts, t.borderLabelLen, t.borderShape, true)
					case reqRedrawPreviewLabel:
						t.printLabel(t.pborder, t.previewLabel, t.previewLabelOpts, t.previewLabelLen, t.activePreviewOpts.Border(), true)
					case reqReinit, reqResize, reqFullRedraw, reqRedraw:
						if req == reqReinit {
							t.tui.Resume(t.fullscreen, true)
						}
						if req == reqResize {
							t.termSize = t.tui.Size()
						}
						wasHidden := t.pwindow == nil
						if req == reqRedraw {
							t.printAll()
						} else {
							t.fullRedraw()
						}
						if wasHidden && t.hasPreviewWindow() {
							refreshPreview(t.previewOpts.command)
						}
						if req == reqResize && t.hasResizeActions {
							t.eventChan <- tui.Resize.AsEvent()
						}
					case reqClose:
						exit(func() int {
							if t.output() {
								return ExitOk
							}
							return ExitNoMatch
						})
						return
					case reqPreviewDisplay:
						result := value.(previewResult)
						if t.previewer.version != result.version {
							t.previewer.version = result.version
							t.previewer.following.Force(t.activePreviewOpts.follow)
							if t.previewer.following.Enabled() {
								t.previewer.offset = 0
							}
						}
						t.previewer.lines = result.lines
						t.previewer.spinner = result.spinner
						if t.hasPreviewWindow() && t.previewer.following.Enabled() {
							t.previewer.offset = max(t.previewer.offset, len(t.previewer.lines)-(t.pwindow.Height()-t.activePreviewOpts.headerLines))
						} else if result.offset >= 0 {
							t.previewer.offset = util.Constrain(result.offset, t.activePreviewOpts.headerLines, len(t.previewer.lines)-1)
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
							return ExitOk
						})
						return
					case reqBecome:
						exit(func() int { return ExitBecome })
						return
					case reqQuit:
						exit(func() int { return ExitInterrupt })
						return
					case reqFatal:
						exit(func() int { return ExitError })
						return
					}
				}
				if (info || header || footer) && !t.resizeIfNeeded() {
					if info {
						t.printInfo()
					}
					if header {
						t.printHeader()
					}
					if footer {
						t.printFooter()
					}
				}
				t.flush()
				t.mutex.Unlock()
				t.uiMutex.Unlock()
			})
		}

		t.eventBox.Set(EvtQuit, quitSignal{code, nil})
		t.running.Set(false)
		t.killPreview()
		cancel()
	}()

	looping := true
	barrier := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-barrier:
			}
			select {
			case <-ctx.Done():
				return
			case t.keyChan <- t.tui.GetChar(t.listenAddr != nil):
			}
		}
	}()
	previewDraggingPos := -1
	barDragging := false
	pbarDragging := false
	pborderDragging := -1
	wasDown := false
	pmx, pmy := -1, -1
	needBarrier := true

	// If an action is bound to 'start', we're going to process it before reading
	// user input.
	if !t.hasStartActions {
		barrier <- true
		needBarrier = false
	}

	// These variables are defined outside the loop to be accessible from closures
	events := []util.EventType{}
	changed := false
	var newNth *[]Range
	req := func(evts ...util.EventType) {
		for _, event := range evts {
			events = append(events, event)
			if isTerminalEvent(event) {
				looping = false
			}
		}
	}

	// The main event loop
	for loopIndex := int64(0); looping; loopIndex++ {
		var newCommand *commandSpec
		var reloadSync bool
		events = []util.EventType{}
		changed = false
		newNth = nil
		beof := false
		queryChanged := false
		denylist := []int32{}

		// Special handling of --sync. Activate the interface on the second tick.
		if loopIndex == 1 && t.deferActivation() {
			t.reqBox.Set(reqActivate, nil)
		}

		if loopIndex > 0 && needBarrier {
			barrier <- true
			needBarrier = false
		}

		var event tui.Event
		actions := []*action{}
		callbacks := []versionedCallback{}
		select {
		case event = <-t.keyChan:
			needBarrier = true
		case event = <-t.eventChan:
			// Drain channel to process all queued events at once without rendering
			// the intermediate states
		Drain:
			for {
				if eventActions, prs := t.keymap[event]; prs {
					actions = append(actions, eventActions...)
				}
				for {
					select {
					case event = <-t.eventChan:
						continue Drain
					default:
						break Drain
					}
				}
			}
		case serverActions := <-t.serverInputChan:
			event = tui.Invalid.AsEvent()
			if t.listenAddr == nil || t.listenAddr.IsLocal() || t.listenUnsafe {
				actions = serverActions
			} else {
				for _, action := range serverActions {
					if !processExecution(action.t) {
						actions = append(actions, action)
					}
				}
			}
			for _, action := range actions {
				if action.t == actExecute {
					t.tui.CancelGetChar()
					break
				}
			}

		case callback := <-t.callbackChan:
			event = tui.Invalid.AsEvent()
			actions = append(actions, &action{t: actAsync})
			callbacks = append(callbacks, callback)
		DrainCallback:
			for {
				select {
				case callback = <-t.callbackChan:
					callbacks = append(callbacks, callback)
					continue DrainCallback
				default:
					break DrainCallback
				}
			}
		}

		t.mutex.Lock()
		for key, ret := range t.expect {
			if keyMatch(key, event) {
				t.pressed = ret
				t.mutex.Unlock()
				t.reqBox.Set(reqClose, nil)
				return nil
			}
		}
		triggering := map[tui.Event]struct{}{}
		previousInput := t.input
		previousCx := t.cx
		previousVersion := t.version
		t.lastKey = event.KeyName()
		updatePreviewWindow := func(forcePreview bool) {
			t.resizeWindows(forcePreview, false)
			req(reqPrompt, reqList, reqInfo, reqHeader, reqFooter)
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
			headerLines := t.activePreviewOpts.headerLines
			if t.activePreviewOpts.cycle {
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

		actionsFor := func(eventType tui.EventType) []*action {
			return t.keymap[eventType.AsEvent()]
		}

		var doAction func(*action) bool
		doActions := func(actions []*action) bool {
			for iter := 0; iter <= maxFocusEvents; iter++ {
				currentIndex := t.currentIndex()
				for _, action := range actions {
					if !doAction(action) {
						return false
					}
					// A terminal action performed. We should stop processing more.
					if !looping {
						break
					}
				}

				if onFocus, prs := t.keymap[tui.Focus.AsEvent()]; prs && iter < maxFocusEvents {
					if newIndex := t.currentIndex(); newIndex != currentIndex {
						t.lastFocus = newIndex
						if t.infoCommand != "" {
							req(reqInfo)
						}
						actions = onFocus
						continue
					}
				}
				break
			}
			return true
		}
		doAction = func(a *action) bool {
			// Keep track of the current query before the action is executed,
			// so we can restore it when the input section is hidden (--no-input).
			// * By doing this, we don't have to add a conditional branch to each
			//   query modifying action.
			// * We restore the query after each action instead of after a set of
			//   actions to allow changing the query even when the input is hidden
			//     e.g. fzf --no-input --bind 'space:show-input+change-query(foo)+hide-input'
			currentInput := t.input
			capture := func(firstLineOnly bool, callback func(string)) {
				if a.t >= actBgTransform {
					// bg-transform-*
					t.captureAsync(*a, firstLineOnly, callback)
				} else if a.t >= actTransform {
					// transform-*
					if firstLineOnly {
						callback(t.captureLine(a.a))
					} else {
						callback(t.captureLines(a.a))
					}
				} else {
					// change-*
					callback(a.a)
				}
			}
		Action:
			switch a.t {
			case actIgnore, actStart, actClick:
			case actAsync:
				for _, callback := range callbacks {
					if t.bgVersion == callback.version {
						callback.callback()
					}
				}
			case actBecome:
				valid, list := t.buildPlusList(a.a, false)
				if valid {
					// We do not remove temp files in this case
					command, _ := t.replacePlaceholder(a.a, false, string(t.input), list)
					t.tui.Close()
					if t.history != nil {
						t.history.append(string(t.input))
					}

					if len(t.proxyScript) > 0 {
						data := strings.Join(append([]string{command}, t.environ()...), "\x00")
						os.WriteFile(t.proxyScript+becomeSuffix, []byte(data), 0600)
						req(reqBecome)
					} else {
						t.executor.Become(t.ttyin, t.environ(), command)
					}
				}
			case actBell:
				t.tui.Bell()
			case actExcludeMulti:
				if len(t.selected) > 0 {
					for _, item := range t.sortSelected() {
						denylist = append(denylist, item.item.Index())
					}
					// Clear selected items
					t.selected = make(map[int32]selectedItem)
					t.version++
				} else {
					item := t.currentItem()
					if item != nil {
						denylist = append(denylist, item.Index())
					}
				}
				changed = true
			case actExclude:
				if item := t.currentItem(); item != nil {
					denylist = append(denylist, item.Index())
					t.deselectItem(item)
					changed = true
				}
			case actExecute, actExecuteSilent:
				t.executeCommand(a.a, false, a.t == actExecuteSilent, false, false, "")
			case actExecuteMulti:
				t.executeCommand(a.a, true, false, false, false, "")
			case actInvalid:
				t.mutex.Unlock()
				return false
			case actBracketedPasteBegin:
				current := []rune(t.input)
				t.pasting = &current
			case actBracketedPasteEnd:
				if t.pasting != nil {
					queryChanged = string(t.input) != string(*t.pasting)
					t.pasting = nil
				}
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
								previewRequest{t.previewOpts.command, t.evaluateScrollOffset(), list, t.environForPreview(), string(t.input)})
						}
					} else {
						// Discard the preview content so that it won't accidentally appear
						// when preview window is re-enabled and previewDelay is triggered
						t.previewer.lines = nil

						// Also kill the preview process if it's still running
						t.cancelPreview()
					}
				}
			case actTogglePreviewWrap:
				if t.hasPreviewWindow() {
					t.activePreviewOpts.wrap = !t.activePreviewOpts.wrap
					// Reset preview version so that full redraw occurs
					t.previewed.version = 0
					req(reqPreviewRefresh)
				}
			case actTransformPrompt, actBgTransformPrompt:
				capture(true, func(prompt string) {
					t.promptString = prompt
					t.prompt, t.promptLen = t.parsePrompt(prompt)
					req(reqPrompt)
				})
			case actTransformQuery, actBgTransformQuery:
				capture(true, func(query string) {
					t.input = []rune(query)
					t.cx = len(t.input)
				})
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
				t.xoffset = 0
			case actBackwardChar:
				if t.cx > 0 {
					t.cx--
				}
			case actPrintQuery:
				req(reqPrintQuery)
			case actChangeMulti:
				multi := t.multi
				if a.a == "" {
					multi = maxMulti
				} else if n, e := strconv.Atoi(a.a); e == nil && n >= 0 {
					multi = n
				}
				if t.multi > 0 && multi != t.multi {
					t.selected = make(map[int32]selectedItem)
					t.version++
				}
				t.multi = multi
				req(reqList, reqInfo)
			case actChangeNth, actTransformNth, actBgTransformNth:
				capture(true, func(expr string) {
					// Split nth expression
					tokens := strings.Split(expr, "|")
					if nth, err := splitNth(tokens[0]); err == nil || len(expr) == 0 {
						// Changed
						newNth = &nth
					} else {
						// The default
						newNth = &t.nth
					}
					// Cycle
					if len(tokens) > 1 {
						a.a = strings.Join(append(tokens[1:], tokens[0]), "|")
					}
					if !compareRanges(t.nthCurrent, *newNth) {
						changed = true
						t.nthCurrent = *newNth
						t.forceRerenderList()
					}
				})
			case actChangeQuery:
				t.input = []rune(a.a)
				t.cx = len(t.input)
			case actChangeHeader, actTransformHeader, actBgTransformHeader:
				capture(false, func(header string) {
					// When a dedicated header window is not used, we may need to
					// update other elements as well.
					if t.changeHeader(header) {
						req(reqList, reqPrompt, reqInfo)
					}
					req(reqHeader)
				})
			case actChangeFooter, actTransformFooter, actBgTransformFooter:
				capture(false, func(footer string) {
					t.changeFooter(footer)
					req(reqFooter)
				})
			case actChangeHeaderLabel, actTransformHeaderLabel, actBgTransformHeaderLabel:
				capture(true, func(label string) {
					t.headerLabelOpts.label = label
					t.headerLabel, t.headerLabelLen = t.ansiLabelPrinter(label, &tui.ColHeaderLabel, false)
					req(reqRedrawHeaderLabel)
				})
			case actChangeFooterLabel, actTransformFooterLabel, actBgTransformFooterLabel:
				capture(true, func(label string) {
					t.footerLabelOpts.label = label
					t.footerLabel, t.footerLabelLen = t.ansiLabelPrinter(label, &tui.ColFooterLabel, false)
					req(reqRedrawFooterLabel)
				})
			case actChangeInputLabel, actTransformInputLabel, actBgTransformInputLabel:
				capture(true, func(label string) {
					t.inputLabelOpts.label = label
					if t.inputBorder != nil {
						t.inputLabel, t.inputLabelLen = t.ansiLabelPrinter(label, &tui.ColInputLabel, false)
						req(reqRedrawInputLabel)
					}
				})
			case actChangeListLabel, actTransformListLabel, actBgTransformListLabel:
				capture(true, func(label string) {
					t.listLabelOpts.label = label
					if t.wborder != nil {
						t.listLabel, t.listLabelLen = t.ansiLabelPrinter(label, &tui.ColListLabel, false)
						req(reqRedrawListLabel)
					}
				})
			case actChangeBorderLabel, actTransformBorderLabel, actBgTransformBorderLabel:
				capture(true, func(label string) {
					t.borderLabelOpts.label = label
					if t.border != nil {
						t.borderLabel, t.borderLabelLen = t.ansiLabelPrinter(label, &tui.ColBorderLabel, false)
						req(reqRedrawBorderLabel)
					}
				})
			case actChangePreviewLabel, actTransformPreviewLabel, actBgTransformPreviewLabel:
				capture(true, func(label string) {
					t.previewLabelOpts.label = label
					if t.pborder != nil {
						t.previewLabel, t.previewLabelLen = t.ansiLabelPrinter(label, &tui.ColPreviewLabel, false)
						req(reqRedrawPreviewLabel)
					}
				})
			case actTransform, actBgTransform:
				capture(false, func(body string) {
					if actions, err := parseSingleActionList(strings.Trim(body, "\r\n")); err == nil {
						// NOTE: We're not properly passing the return value here
						doActions(actions)
					}
				})
			case actBgCancel:
				t.bgVersion++
				t.runningCmds.ForEach(func(cmd *runningCmd) {
					util.KillCommand(cmd.cmd)
				})
			case actChangePrompt:
				t.promptString = a.a
				t.prompt, t.promptLen = t.parsePrompt(a.a)
				req(reqPrompt)
			case actPreview:
				if !t.hasPreviewWindow() {
					updatePreviewWindow(true)
				}
				refreshPreview(a.a)
			case actRefreshPreview:
				refreshPreview(t.previewOpts.command)
			case actReplaceQuery:
				current := t.currentItem()
				if current != nil {
					t.input = current.text.ToRunes()
					t.cx = len(t.input)
				}
			case actFatal:
				req(reqFatal)
			case actAbort:
				req(reqQuit)
			case actDeleteChar:
				t.delChar()
			case actDeleteCharEof:
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
			case actBackwardDeleteCharEof:
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
					// Limit the scope only to the matching items
					for i := 0; i < t.resultMerger.Length(); i++ {
						if !t.selectItem(t.resultMerger.Get(i).item) {
							break
						}
					}
					req(reqList, reqInfo)
				}
			case actDeselectAll:
				if t.multi > 0 {
					// Also limit the scope only to the matching items, while this may
					// not be straightforward in raw mode.
					for i := 0; i < t.resultMerger.Length() && len(t.selected) > 0; i++ {
						t.deselectItem(t.resultMerger.Get(i).item)
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
					for i := 0; i < t.resultMerger.Length() && len(t.selected) > 0; i++ {
						item := t.resultMerger.Get(i).item
						if _, found := t.selected[item.Index()]; found {
							prevIndexes[i] = struct{}{}
							t.deselectItem(item)
						}
					}

					for i := 0; i < t.resultMerger.Length(); i++ {
						if _, found := prevIndexes[i]; !found {
							item := t.resultMerger.Get(i).item
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
			case actDown, actDownMatch, actUp, actUpMatch:
				dir := -1
				if a.t == actUp || a.t == actUpMatch {
					dir = 1
				}
				if t.raw && (a.t == actDownMatch || a.t == actUpMatch) {
					if t.resultMerger.Length() > 0 {
						prevCy := t.cy
						for t.vmove(dir, true) && !t.isCurrentItemMatch() {
						}
						if !t.isCurrentItemMatch() {
							t.vset(prevCy)
						}
					}
				} else {
					t.vmove(dir, true)
				}
				req(reqList)
			case actToggleRaw, actEnableRaw, actDisableRaw:
				prevRaw := t.raw
				newRaw := t.raw
				switch a.t {
				case actEnableRaw:
					newRaw = true
				case actDisableRaw:
					newRaw = false
				case actToggleRaw:
					newRaw = !t.raw
				}
				if prevRaw == newRaw {
					break
				}
				prevPos := t.cy - t.offset
				prevIndex := t.currentIndex()
				if newRaw {
					// Build matchMap if not available
					if len(t.matchMap) == 0 {
						t.matchMap = t.resultMerger.ToMap()
					}
					t.merger = t.passMerger
				} else {
					// Find the closest matching item
					if !t.isCurrentItemMatch() && t.resultMerger.Length() > 1 {
						distance := 0
					Loop:
						for {
							distance++
							checks := 0
							for _, cy := range []int{t.cy + distance, t.cy - distance} {
								if cy >= 0 && cy < t.merger.Length() {
									checks++
									item := t.merger.Get(cy).item
									if t.isItemMatch(item) {
										prevIndex = item.Index()
										break Loop
									}
								}
							}
							if checks == 0 {
								break
							}
						}
					}

					t.merger = t.resultMerger

					// Need to remove non-matching items from the selection
					if t.multi > 0 && len(t.selected) > 0 {
						t.filterSelected()
						req(reqInfo)
					}
				}
				t.raw = newRaw

				// Try to retain position
				if prevIndex != minItem.Index() {
					t.cy = max(0, t.merger.FindIndex(prevIndex))
					t.offset = t.cy - prevPos
				}

				// List needs to be rerendered
				t.forceRerenderList()
				req(reqList)
			case actAccept:
				req(reqClose)
			case actAcceptNonEmpty:
				if len(t.selected) > 0 || t.merger.Length() > 0 || !t.reading && t.count == 0 {
					req(reqClose)
				}
			case actAcceptOrPrintQuery:
				if len(t.selected) > 0 || t.merger.Length() > 0 {
					req(reqClose)
				} else {
					req(reqPrintQuery)
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
			case actFirst, actBest:
				if t.raw && a.t == actBest {
					if t.resultMerger.Length() > 0 {
						t.vset(t.merger.FindIndex(t.resultMerger.Get(0).item.Index()))
					}
				} else {
					t.vset(0)
				}
				t.constrain()
				req(reqList)
			case actLast:
				t.vset(t.merger.Length() - 1)
				t.constrain()
				req(reqList)
			case actPosition:
				if n, e := strconv.Atoi(a.a); e == nil {
					if n > 0 {
						n--
					} else if n < 0 {
						n += t.merger.Length()
					}
					t.vset(n)
					t.constrain()
					req(reqList)
				}
			case actPut:
				str := []rune(a.a)
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], str...), suffix...)
				t.cx += len(str)
			case actPrint:
				t.printQueue = append(t.printQueue, a.a)
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
			case actBackwardKillSubWord:
				beof = len(t.input) == 0
				if t.cx > 0 {
					t.rubout(t.subWordRubout)
				}
			case actYank:
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], t.yanked...), suffix...)
				t.cx += len(t.yanked)
			case actPageUp, actPageDown, actHalfPageUp, actHalfPageDown:
				// Calculate the number of lines to move
				maxItems := t.maxItems()
				linesToMove := maxItems - 1
				if a.t == actHalfPageUp || a.t == actHalfPageDown {
					linesToMove = maxItems / 2
				}
				// Move at least one line even in a very short window
				linesToMove = max(1, linesToMove)

				// Determine the direction of the movement
				direction := -1
				if a.t == actPageUp || a.t == actHalfPageUp {
					direction = 1
				}

				// In non-default layout, items are listed from top to bottom
				if t.layout != layoutDefault {
					direction *= -1
				}

				// We can simply add the number of lines to the current position in
				// single-line mode
				if !t.canSpanMultiLines() {
					t.vset(t.cy + direction*linesToMove)
					req(reqList)
					break
				}

				// But in multi-line mode, we need to carefully limit the amount of
				// vertical movement so that items are not skipped. In order to do
				// this, we calculate the minimum or maximum offset based on the
				// direction of the movement and the number of lines of the items
				// around the current scroll offset.
				var minOffset, maxOffset, lineSum int
				if direction > 0 {
					maxOffset = t.offset
					for ; maxOffset < t.merger.Length(); maxOffset++ {
						itemLines, _ := t.numItemLines(t.merger.Get(maxOffset).item, maxItems)
						lineSum += itemLines
						if lineSum >= maxItems {
							break
						}
					}
				} else {
					minOffset = t.offset
					for ; minOffset >= 0 && minOffset < t.merger.Length(); minOffset-- {
						itemLines, _ := t.numItemLines(t.merger.Get(minOffset).item, maxItems)
						lineSum += itemLines
						if lineSum >= maxItems {
							if lineSum > maxItems {
								minOffset++
							}
							break
						}
					}
				}

				for i := 0; i < linesToMove; i++ {
					cy, offset := t.cy, t.offset
					t.vset(cy + direction)
					t.constrain()
					if cy == t.cy {
						break
					}
					if i > 0 && (direction > 0 && t.offset > maxOffset ||
						direction < 0 && t.offset < minOffset) {
						t.cy, t.offset = cy, offset
						break
					}
				}
				req(reqList)
			case actOffsetUp, actOffsetDown:
				diff := 1
				if a.t == actOffsetDown {
					diff = -1
				}
				if t.layout != layoutDefault {
					diff *= -1
				}
				t.offset += diff
				before := t.offset
				t.constrain()
				if before != t.offset {
					t.offset = before
					if t.layout != layoutDefault {
						diff *= -1
					}
					t.vmove(diff, false)
				}
				req(reqList)
			case actOffsetMiddle:
				soff := t.scrollOff
				t.scrollOff = t.window.Height()
				t.constrain()
				t.scrollOff = soff
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
			case actBackwardSubWord:
				t.cx = findLastMatch(t.subWordRubout, string(t.input[:t.cx])) + 1
			case actForwardSubWord:
				t.cx += findFirstMatch(t.subWordNext, string(t.input[t.cx:])) + 1
			case actKillWord:
				ncx := t.cx +
					findFirstMatch(t.wordNext, string(t.input[t.cx:])) + 1
				if ncx > t.cx {
					t.yanked = copySlice(t.input[t.cx:ncx])
					t.input = append(t.input[:t.cx], t.input[ncx:]...)
				}
			case actKillSubWord:
				ncx := t.cx +
					findFirstMatch(t.subWordNext, string(t.input[t.cx:])) + 1
				if ncx > t.cx {
					t.yanked = copySlice(t.input[t.cx:ncx])
					t.input = append(t.input[:t.cx], t.input[ncx:]...)
				}
			case actKillLine:
				if t.cx < len(t.input) {
					t.yanked = copySlice(t.input[t.cx:])
					t.input = t.input[:t.cx]
				}
			case actChar:
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
			case actToggleTrackCurrent:
				if t.track.Current() {
					t.track = trackDisabled
				} else if t.track.Disabled() {
					t.track = trackCurrent(t.currentIndex())
				}
				req(reqInfo)
			case actShowHeader:
				t.headerVisible = true
				req(reqList, reqInfo, reqPrompt, reqHeader)
			case actHideHeader:
				t.headerVisible = false
				req(reqList, reqInfo, reqPrompt, reqHeader)
			case actToggleHeader:
				t.headerVisible = !t.headerVisible
				req(reqList, reqInfo, reqPrompt, reqHeader)
			case actToggleWrap:
				t.wrap = !t.wrap
				t.clearNumLinesCache()
				req(reqList, reqHeader)
			case actToggleMultiLine:
				t.multiLine = !t.multiLine
				t.clearNumLinesCache()
				req(reqList)
			case actToggleHscroll:
				// Force re-rendering of the list
				t.forceRerenderList()
				t.hscroll = !t.hscroll
				req(reqList)
			case actToggleInput, actShowInput, actHideInput:
				switch a.t {
				case actToggleInput:
					t.inputless = !t.inputless
				case actShowInput:
					if !t.inputless {
						break Action
					}
					t.inputless = false
				case actHideInput:
					if t.inputless {
						break Action
					}
					t.inputless = true
				}
				t.forceRerenderList()
				if t.inputless {
					t.tui.HideCursor()
				} else {
					t.tui.ShowCursor()
				}
				req(reqList, reqInfo, reqPrompt, reqHeader)
			case actTrackCurrent:
				// Global tracking has higher priority
				if !t.track.Global() {
					t.track = trackCurrent(t.currentIndex())
				}
				req(reqInfo)
			case actUntrackCurrent:
				if t.track.Current() {
					t.track = trackDisabled
				}
				req(reqInfo)
			case actSearch:
				override := []rune(a.a)
				t.inputOverride = &override
				changed = true
			case actTransformSearch, actBgTransformSearch:
				capture(true, func(query string) {
					override := []rune(query)
					t.inputOverride = &override
					changed = true
				})
			case actEnableSearch:
				t.paused = false
				changed = true
				req(reqPrompt)
			case actDisableSearch:
				t.paused = true
				req(reqPrompt)
			case actTrigger:
				if _, chords, err := parseKeyChords(a.a, ""); err == nil {
					for _, chord := range chords {
						if _, prs := triggering[chord]; prs {
							// Avoid recursive triggering
							continue
						}
						if acts, prs := t.keymap[chord]; prs {
							triggering[chord] = struct{}{}
							doActions(acts)
							delete(triggering, chord)
						}
					}
				}
			case actSigStop:
				p, err := os.FindProcess(os.Getpid())
				if err == nil {
					t.tui.Clear()
					t.tui.Pause(t.fullscreen)
					notifyStop(p)
					t.mutex.Unlock()
					t.reqBox.Set(reqReinit, nil)
					return false
				}
			case actMouse:
				me := event.MouseEvent
				mx, my := me.X, me.Y
				click := !wasDown && me.Down
				clicked := wasDown && !me.Down && (mx == pmx && my == pmy)
				wasDown = me.Down
				if click {
					pmx, pmy = mx, my
				}
				if !me.Down {
					barDragging = false
					pbarDragging = false
					pborderDragging = -1
					previewDraggingPos = -1
					pmx, pmy = -1, -1
				}

				// Scrolling
				if me.S != 0 {
					if t.window.Enclose(my, mx) && t.merger.Length() > 0 {
						evt := tui.ScrollUp
						if me.Mod() {
							evt = tui.SScrollUp
						}
						if me.S < 0 {
							evt = tui.ScrollDown
							if me.Mod() {
								evt = tui.SScrollDown
							}
						}
						return doActions(actionsFor(evt))
					} else if t.hasPreviewWindow() && t.pwindow.Enclose(my, mx) {
						evt := tui.PreviewScrollUp
						if me.S < 0 {
							evt = tui.PreviewScrollDown
						}
						return doActions(actionsFor(evt))
					}
					break
				}

				// Preview dragging
				if me.Down && (previewDraggingPos >= 0 || click && t.hasPreviewWindow() && t.pwindow.Enclose(my, mx)) {
					if previewDraggingPos > 0 {
						scrollPreviewBy(previewDraggingPos - my)
					}
					previewDraggingPos = my
					break
				}

				// Preview scrollbar dragging
				headerLines := t.activePreviewOpts.headerLines
				pbarDragging = me.Down && (pbarDragging || click && t.hasPreviewWindow() && my >= t.pwindow.Top()+headerLines && my < t.pwindow.Top()+t.pwindow.Height() && mx == t.pwindow.Left()+t.pwindow.Width())
				if pbarDragging {
					effectiveHeight := t.pwindow.Height() - headerLines
					numLines := len(t.previewer.lines) - headerLines
					barLength, _ := getScrollbar(1, numLines, effectiveHeight, min(numLines-effectiveHeight, t.previewer.offset-headerLines))
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

				// Preview border dragging (resizing)
				if pborderDragging < 0 && click && t.hasPreviewWindow() {
					switch t.activePreviewOpts.position {
					case posUp:
						if t.pborder.Enclose(my, mx) && my == t.pborder.Top()+t.pborder.Height()-1 {
							pborderDragging = 0
						} else if t.listBorderShape.HasTop() && t.pborder.EncloseX(mx) && my == t.wborder.Top() {
							pborderDragging = 1
						}
					case posDown:
						if t.pborder.Enclose(my, mx) && my == t.pborder.Top() {
							pborderDragging = 0
						} else if t.listBorderShape.HasBottom() && t.pborder.EncloseX(mx) && my == t.wborder.Top()+t.wborder.Height()-1 {
							pborderDragging = 1
						}
					case posLeft:
						if t.pborder.Enclose(my, mx) && mx == t.pborder.Left()+t.pborder.Width()-1 {
							pborderDragging = 0
						} else if t.listBorderShape.HasLeft() && t.pborder.EncloseY(my) && mx == t.wborder.Left() {
							pborderDragging = 1
						}
					case posRight:
						if t.pborder.Enclose(my, mx) && mx == t.pborder.Left() {
							pborderDragging = 0
						} else if t.listBorderShape.HasRight() && t.pborder.EncloseY(my) && mx == t.wborder.Left()+t.wborder.Width()-1 {
							pborderDragging = 1
						}
					}
				}

				if pborderDragging >= 0 && t.hasPreviewWindow() {
					var newSize int
					var prevSize int
					switch t.activePreviewOpts.position {
					case posLeft:
						prevSize = t.pwindow.Width()
						diff := t.pborder.Width() - prevSize
						newSize = mx - t.pborder.Left() - diff + 1
					case posUp:
						prevSize = t.pwindow.Height()
						diff := t.pborder.Height() - prevSize
						newSize = my - t.pborder.Top() - diff + 1
					case posDown:
						prevSize = t.pwindow.Height()
						offset := my - t.pborder.Top()
						newSize = prevSize - offset
					case posRight:
						prevSize = t.pwindow.Width()
						offset := mx - t.pborder.Left()
						newSize = prevSize - offset
					}
					newSize -= pborderDragging
					if newSize < 1 {
						newSize = 1
					}

					if prevSize == newSize {
						break
					}

					t.activePreviewOpts.size = sizeSpec{float64(newSize), false}
					updatePreviewWindow(true)
					req(reqPreviewRefresh)
					break
				}

				// Inside the input window
				if t.inputWindow != nil && t.inputWindow.Enclose(my, mx) {
					mx -= t.inputWindow.Left()
					my -= t.inputWindow.Top()
					y := t.inputWindow.Height() - 1
					if t.layout == layoutReverse {
						y = 0
					}
					mxCons := util.Constrain(mx-t.promptLen, 0, len(t.input))
					if my == y && mxCons >= 0 {
						t.cx = mxCons + t.xoffset
					}
					break
				}

				// Inside the header window
				if clicked && t.headerVisible && t.headerWindow != nil && t.headerWindow.Enclose(my, mx) {
					mx -= t.headerWindow.Left() + t.headerIndent(t.headerBorderShape)
					my -= t.headerWindow.Top()
					if mx < 0 {
						break
					}
					t.clickHeaderLine = my + 1
					if t.layout != layoutReverse && t.headerLinesWindow != nil {
						t.clickHeaderLine += t.headerLines
					}
					t.clickHeaderColumn = mx + 1
					return doActions(actionsFor(tui.ClickHeader))
				}

				if clicked && t.headerVisible && t.headerLinesWindow != nil && t.headerLinesWindow.Enclose(my, mx) {
					_, shape := t.determineHeaderLinesShape()
					mx -= t.headerLinesWindow.Left() + t.headerIndent(shape)
					my -= t.headerLinesWindow.Top()
					if mx < 0 {
						break
					}
					t.clickHeaderLine = my + 1
					if t.layout == layoutReverse {
						t.clickHeaderLine += len(t.header0)
					}
					t.clickHeaderColumn = mx + 1
					return doActions(actionsFor(tui.ClickHeader))
				}

				// Inside the footer window
				if clicked && t.footerWindow != nil && t.footerWindow.Enclose(my, mx) {
					mx -= t.footerWindow.Left() + t.headerIndent(t.footerBorderShape)
					my -= t.footerWindow.Top()
					if mx < 0 {
						break
					}
					t.clickFooterLine = my + 1
					t.clickFooterColumn = mx + 1
					return doActions(actionsFor(tui.ClickFooter))
				}

				// Ignored
				if !t.window.Enclose(my, mx) && !barDragging {
					break
				}

				// Translate coordinates
				mx -= t.window.Left()
				my -= t.window.Top()
				min := t.promptLines() + t.visibleHeaderLinesInList()
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
				barDragging = me.Down && (barDragging || click && my >= min && mx == t.window.Width()-1)
				if barDragging {
					barLength, barStart := t.getScrollbar()
					if barLength > 0 {
						maxItems := t.maxItems()
						if newBarStart := util.Constrain(my-min-barLength/2, 0, maxItems-barLength); newBarStart != barStart {
							total := t.merger.Length()
							prevOffset := t.offset
							// barStart = (maxItems - barLength) * t.offset / (total - maxItems)
							perLine := t.avgNumLines()
							t.offset = int(math.Ceil(float64(newBarStart) * float64(total*perLine-maxItems) / float64(maxItems*perLine-barLength)))
							t.cy = t.offset + t.cy - prevOffset
							req(reqList)
						}
					}
					break
				}

				// There can be empty lines after the list in multi-line mode
				prevLine := t.prevLines[my]
				if prevLine.empty {
					break
				}

				// Double-click on an item
				cy := prevLine.cy
				if me.Double && mx < t.window.Width()-1 {
					// Double-click
					if my >= min {
						if t.vset(cy) && t.cy < t.merger.Length() {
							return doActions(actionsFor(tui.DoubleClick))
						}
					}
				}

				if me.Down {
					mxCons := util.Constrain(mx-t.promptLen, 0, len(t.input))
					if !t.inputless && t.inputWindow == nil && my == t.promptLine() && mxCons >= 0 {
						// Prompt
						t.cx = mxCons + t.xoffset
					} else if my >= min {
						t.vset(cy)
						req(reqList)
						evt := tui.RightClick
						if me.Mod() {
							evt = tui.SRightClick
						}
						if me.Left {
							evt = tui.LeftClick
							if me.Mod() {
								evt = tui.SLeftClick
							}
						}
						return doActions(actionsFor(evt))
					}
				}
				if clicked && t.headerVisible && t.headerWindow == nil {
					// Header
					numLines := t.visibleHeaderLinesInList()
					lineOffset := 0
					if !t.inputless && t.inputWindow == nil && !t.headerFirst {
						// offset for info line
						if t.noSeparatorLine() {
							lineOffset = 1
						} else {
							lineOffset = 2
						}
					}
					my -= lineOffset
					mx -= t.pointerLen + t.markerLen
					if my >= 0 && my < numLines && mx >= 0 {
						if t.layout == layoutReverse {
							t.clickHeaderLine = my + 1
						} else {
							t.clickHeaderLine = numLines - my
						}
						t.clickHeaderColumn = mx + 1
						return doActions(actionsFor(tui.ClickHeader))
					}
				}
			case actReload, actReloadSync:
				t.failed = nil

				valid, list := t.buildPlusList(a.a, false)
				if !valid {
					// We run the command even when there's no match
					// 1. If the template doesn't have any slots
					// 2. If the template has {q}
					slot, _, _, forceUpdate := hasPreviewFlags(a.a)
					valid = !slot || forceUpdate
				}
				if valid {
					command, tempFiles := t.replacePlaceholder(a.a, false, string(t.input), list)
					newCommand = &commandSpec{command, tempFiles}
					reloadSync = a.t == actReloadSync
					t.reading = true
				}
			case actUnbind:
				if keys, _, err := parseKeyChords(a.a, "PANIC"); err == nil {
					for key := range keys {
						delete(t.keymap, key)
					}
				}
			case actRebind:
				if keys, _, err := parseKeyChords(a.a, "PANIC"); err == nil {
					for key := range keys {
						if originalAction, found := t.keymapOrg[key]; found {
							t.keymap[key] = originalAction
						}
					}
				}
			case actToggleBind:
				if keys, _, err := parseKeyChords(a.a, "PANIC"); err == nil {
					for key := range keys {
						if _, bound := t.keymap[key]; bound {
							delete(t.keymap, key)
						} else if originalAction, found := t.keymapOrg[key]; found {
							t.keymap[key] = originalAction
						}
					}
				}
			case actChangeGhost, actTransformGhost, actBgTransformGhost:
				capture(true, func(ghost string) {
					t.ghost = ghost
					if len(t.input) == 0 {
						req(reqPrompt)
					}
				})
			case actChangePointer, actTransformPointer, actBgTransformPointer:
				capture(true, func(pointer string) {
					length := uniseg.StringWidth(pointer)
					if length <= 2 {
						if length != t.pointerLen {
							t.forceRerenderList()
						}
						t.pointer = pointer
						t.pointerLen = length
						t.pointerEmpty = strings.Repeat(" ", t.pointerLen)
						req(reqList)
					}
				})
			case actChangePreview:
				if t.previewOpts.command != a.a {
					t.previewOpts.command = a.a
					updatePreviewWindow(false)
					refreshPreview(t.previewOpts.command)
				}
			case actChangePreviewWindow:
				// NOTE: We intentionally use "previewOpts" instead of "activePreviewOpts" here
				currentPreviewOpts := t.previewOpts

				// Reset preview options and apply the additional options
				t.previewOpts = t.initialPreviewOpts
				t.previewOpts.command = currentPreviewOpts.command

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
				switch currentPreviewOpts.compare(t.activePreviewOpts, &t.previewOpts) {
				case previewOptsDifferentLayout:
					// Preview command can be running in the background if the size of
					// the preview window is 0 but not 'hidden'
					wasHidden := currentPreviewOpts.hidden

					// FIXME: One-time preview window can't reappear once hidden
					// fzf --bind space:preview:ls --bind 'enter:change-preview-window:down|left|up|hidden|'
					updatePreviewWindow(t.hasPreviewWindow() && !t.activePreviewOpts.hidden)
					if wasHidden && t.hasPreviewWindow() {
						// Restart
						refreshPreview(t.previewOpts.command)
					} else if t.activePreviewOpts.hidden {
						// Cancel
						t.cancelPreview()
					} else {
						// Refresh
						req(reqPreviewRefresh)
					}
				case previewOptsDifferentContentLayout:
					t.previewed.version = 0
					req(reqPreviewRefresh)
				}

				// Adjust scroll offset
				if t.hasPreviewWindow() && currentPreviewOpts.scroll != t.activePreviewOpts.scroll {
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

			if !processExecution(a.t) {
				t.lastAction = a.t
			}

			if t.inputless {
				// Always just discard the change
				t.input = currentInput
				t.cx = len(t.input)
				beof = false
			} else if string(t.input) != string(currentInput) {
				t.inputOverride = nil
			}
			return true
		}

		if t.jumping == jumpDisabled || len(actions) > 0 {
			// Break out of jump mode if any action is submitted to the server
			if t.jumping != jumpDisabled {
				t.jumping = jumpDisabled
				if acts, prs := t.keymap[tui.JumpCancel.AsEvent()]; prs && !doActions(acts) {
					continue
				}
				req(reqList)
			}
			if len(actions) == 0 {
				actions = t.keymap[event.Comparable()]
			}
			if len(actions) == 0 && event.Type == tui.Rune {
				doAction(&action{t: actChar})
			} else if !doActions(actions) {
				continue
			}
			if !t.inputless {
				t.truncateQuery()
			}
			queryChanged = queryChanged || t.pasting == nil && string(previousInput) != string(t.input)
			changed = changed || queryChanged
			if onChanges, prs := t.keymap[tui.Change.AsEvent()]; queryChanged && prs && !doActions(onChanges) {
				continue
			}
			if onEOFs, prs := t.keymap[tui.BackwardEOF.AsEvent()]; beof && prs && !doActions(onEOFs) {
				continue
			}
			if onMultis, prs := t.keymap[tui.Multi.AsEvent()]; t.version != previousVersion && prs && !doActions(onMultis) {
				continue
			}
		} else {
			jumpEvent := tui.JumpCancel
			if event.Type == tui.Rune {
				if idx := strings.IndexRune(t.jumpLabels, event.Char); idx >= 0 && idx < t.maxItems() && idx < t.merger.Length() {
					jumpEvent = tui.Jump
					t.cy = idx + t.offset
					if t.jumping == jumpAcceptEnabled {
						req(reqClose)
					}
				}
			}
			t.jumping = jumpDisabled
			if acts, prs := t.keymap[jumpEvent.AsEvent()]; prs && !doActions(acts) {
				continue
			}
			req(reqList)
		}

		if queryChanged && t.canPreview() && len(t.previewOpts.command) > 0 {
			_, _, _, forceUpdate := hasPreviewFlags(t.previewOpts.command)
			if forceUpdate {
				t.version++
			}
		}

		if queryChanged || t.cx != previousCx {
			req(reqPrompt)
		}

		reload := changed || newCommand != nil
		var reloadRequest *searchRequest
		if reload {
			reloadRequest = &searchRequest{sort: t.sort, sync: reloadSync, nth: newNth, command: newCommand, environ: t.environ(), changed: changed, denylist: denylist, revision: t.resultMerger.Revision()}
		}

		// Dispatch queued background requests
		t.dispatchAsync()

		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if reload {
			t.eventBox.Set(EvtSearchNew, *reloadRequest)
		}
		for _, event := range events {
			t.reqBox.Set(event, nil)
		}
	}
	return nil
}

func (t *Terminal) constrain() {
	// count of items to display allowed by filtering
	count := t.merger.Length()
	maxLines := t.maxItems()

	// May need to try again after adjusting the offset
	t.offset = util.Constrain(t.offset, 0, count)
	for range maxLines {
		numItems := maxLines
		// How many items can be fit on screen including the current item?
		if t.canSpanMultiLines() && t.merger.Length() > 0 {
			numItemsFound := 0
			linesSum := 0

			add := func(i int) bool {
				lines, overflow := t.numItemLines(t.merger.Get(i).item, numItems-linesSum)
				linesSum += lines
				if linesSum >= numItems {
					/*
						# Should show all 3 items
						printf "file1\0file2\0file3\0" | fzf --height=5 --read0 --bind load:last --reverse

						# Should not truncate the last item
						printf "file\n1\0file\n2\0file\n3\0" | fzf --height=5 --read0 --bind load:last --reverse
					*/
					if numItemsFound == 0 || !overflow {
						numItemsFound++
					}
					return false
				}
				numItemsFound++
				return true
			}

			for i := t.offset; i < t.merger.Length(); i++ {
				if !add(i) {
					break
				}
			}

			// We can possibly fit more items "before" the offset on screen
			if linesSum < numItems {
				for i := t.offset - 1; i >= 0; i-- {
					if !add(i) {
						break
					}
				}
			}

			numItems = numItemsFound
		}

		t.cy = util.Constrain(t.cy, 0, max(0, count-1))
		minOffset := max(t.cy-numItems+1, 0)
		maxOffset := max(min(count-numItems, t.cy), 0)
		prevOffset := t.offset
		t.offset = util.Constrain(t.offset, minOffset, maxOffset)
		if t.scrollOff > 0 {
			scrollOff := min(maxLines/2, t.scrollOff)
			newOffset := t.offset
			// 2-phase adjustment to avoid infinite loop of alternating between moving up and down
			for phase := range 2 {
				for {
					prevOffset := newOffset
					numItems := t.merger.Length()
					itemLines := 1 + t.gap
					if t.canSpanMultiLines() && t.cy < numItems {
						itemLines, _ = t.numItemLines(t.merger.Get(t.cy).item, maxLines)
					}
					linesBefore := t.cy - newOffset
					if t.canSpanMultiLines() {
						linesBefore = 0
						for i := newOffset; i < t.cy && i < numItems; i++ {
							lines, _ := t.numItemLines(t.merger.Get(i).item, maxLines-linesBefore-itemLines)
							linesBefore += lines
						}
					}
					linesAfter := maxLines - (linesBefore + itemLines)

					// Stuck in the middle, nothing to do
					if linesBefore < scrollOff && linesAfter < scrollOff {
						break
					}

					if phase == 0 && linesBefore < scrollOff {
						newOffset = max(minOffset, newOffset-1)
					} else if phase == 1 && linesAfter < scrollOff {
						newOffset = min(maxOffset, newOffset+1)
					}
					if newOffset == prevOffset {
						break
					}
				}
				t.offset = newOffset
			}
		}
		if t.offset == prevOffset {
			break
		}
	}
}

// Returns true if the cursor position is successfully updated
func (t *Terminal) vmove(o int, allowCycle bool) bool {
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
	return t.vset(dest)
}

func (t *Terminal) vset(o int) bool {
	t.cy = util.Constrain(o, 0, t.merger.Length()-1)
	return t.cy == o
}

// Number of prompt lines in the list window
func (t *Terminal) promptLines() int {
	if t.inputless {
		return 0
	}
	if t.inputWindow != nil {
		return 0
	}
	if t.noSeparatorLine() {
		return 1
	}
	return 2
}

// Number of item lines in the list window
func (t *Terminal) maxItems() int {
	maximum := t.window.Height() - t.visibleHeaderLinesInList() - t.promptLines()
	return max(maximum, 0)
}

func (t *Terminal) dumpItem(i *Item) StatusItem {
	if i == nil {
		return StatusItem{}
	}
	return StatusItem{
		Index: int(i.Index()),
		Text:  i.AsString(t.ansi),
	}
}

func (t *Terminal) tryLock(timeout time.Duration) bool {
	sleepDuration := 10 * time.Millisecond

	for {
		if t.mutex.TryLock() {
			return true
		}

		timeout -= sleepDuration
		if timeout <= 0 {
			break
		}
		time.Sleep(sleepDuration)
	}
	return false
}

func (t *Terminal) dumpStatus(params getParams) string {
	if !t.tryLock(channelTimeout) {
		return ""
	}
	defer t.mutex.Unlock()

	selectedItems := t.sortSelected()
	selected := make([]StatusItem, max(0, min(params.limit, len(selectedItems)-params.offset)))
	for i := range selected {
		selected[i] = t.dumpItem(selectedItems[i+params.offset].item)
	}

	matches := make([]StatusItem, max(0, min(params.limit, t.resultMerger.Length()-params.offset)))
	for i := range matches {
		matches[i] = t.dumpItem(t.resultMerger.Get(i + params.offset).item)
	}

	var current *StatusItem
	currentItem := t.currentItem()
	if currentItem != nil {
		item := t.dumpItem(currentItem)
		current = &item
	}

	dump := Status{
		Reading:    t.reading,
		Progress:   t.progress,
		Query:      string(t.input),
		Position:   t.cy,
		Sort:       t.sort,
		TotalCount: t.count,
		MatchCount: t.resultMerger.Length(),
		Current:    current,
		Matches:    matches,
		Selected:   selected,
	}
	bytes, _ := json.Marshal(&dump) // TODO: Errors?
	return string(bytes)
}
