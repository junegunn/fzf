package fzf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

// import "github.com/pkg/profile"

var placeholder *regexp.Regexp
var activeTempFiles []string

const ellipsis string = ".."

func init() {
	placeholder = regexp.MustCompile(`\\?(?:{[+sf]*[0-9,-.]*}|{q}|{\+?f?nf?})`)
	activeTempFiles = []string{}
}

type jumpMode int

const (
	jumpDisabled jumpMode = iota
	jumpEnabled
	jumpAcceptEnabled
)

type previewer struct {
	text    string
	lines   int
	offset  int
	enabled bool
	more    bool
}

type itemLine struct {
	current  bool
	selected bool
	label    string
	queryLen int
	width    int
	result   Result
}

var emptyLine = itemLine{}

// Terminal represents terminal input/output
type Terminal struct {
	initDelay    time.Duration
	infoStyle    infoStyle
	spinner      []string
	prompt       string
	promptLen    int
	pointer      string
	pointerLen   int
	pointerEmpty string
	marker       string
	markerLen    int
	markerEmpty  string
	queryLen     [2]int
	layout       layoutType
	fullscreen   bool
	keepRight    bool
	hscroll      bool
	hscrollOff   int
	wordRubout   string
	wordNext     string
	cx           int
	cy           int
	offset       int
	xoffset      int
	yanked       []rune
	input        []rune
	multi        int
	sort         bool
	toggleSort   bool
	delimiter    Delimiter
	expect       map[int]string
	keymap       map[int][]action
	pressed      string
	printQuery   bool
	history      *History
	cycle        bool
	header       []string
	header0      []string
	ansi         bool
	tabstop      int
	margin       [4]sizeSpec
	strong       tui.Attr
	unicode      bool
	borderShape  tui.BorderShape
	cleanExit    bool
	border       tui.Window
	window       tui.Window
	pborder      tui.Window
	pwindow      tui.Window
	count        int
	progress     int
	reading      bool
	failed       *string
	jumping      jumpMode
	jumpLabels   string
	printer      func(string)
	printsep     string
	merger       *Merger
	selected     map[int32]selectedItem
	version      int64
	reqBox       *util.EventBox
	preview      previewOpts
	previewer    previewer
	previewBox   *util.EventBox
	eventBox     *util.EventBox
	mutex        sync.Mutex
	initFunc     func()
	prevLines    []itemLine
	suppress     bool
	startChan    chan bool
	killChan     chan int
	slab         *util.Slab
	theme        *tui.ColorTheme
	tui          tui.Renderer
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
	reqRedraw
	reqClose
	reqPrintQuery
	reqPreviewEnqueue
	reqPreviewDisplay
	reqPreviewRefresh
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
	actClearScreen
	actClearQuery
	actClearSelection
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
	actToggleAll
	actToggleDown
	actToggleUp
	actToggleIn
	actToggleOut
	actDown
	actUp
	actPageUp
	actPageDown
	actHalfPageUp
	actHalfPageDown
	actJump
	actJumpAccept
	actPrintQuery
	actReplaceQuery
	actToggleSort
	actTogglePreview
	actTogglePreviewWrap
	actPreviewUp
	actPreviewDown
	actPreviewPageUp
	actPreviewPageDown
	actPreviousHistory
	actNextHistory
	actExecute
	actExecuteSilent
	actExecuteMulti // Deprecated
	actSigStop
	actTop
	actReload
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
	command *string
}

func toActions(types ...actionType) []action {
	actions := make([]action, len(types))
	for idx, t := range types {
		actions[idx] = action{t: t, a: ""}
	}
	return actions
}

func defaultKeymap() map[int][]action {
	keymap := make(map[int][]action)
	keymap[tui.Invalid] = toActions(actInvalid)
	keymap[tui.Resize] = toActions(actClearScreen)
	keymap[tui.CtrlA] = toActions(actBeginningOfLine)
	keymap[tui.CtrlB] = toActions(actBackwardChar)
	keymap[tui.CtrlC] = toActions(actAbort)
	keymap[tui.CtrlG] = toActions(actAbort)
	keymap[tui.CtrlQ] = toActions(actAbort)
	keymap[tui.ESC] = toActions(actAbort)
	keymap[tui.CtrlD] = toActions(actDeleteCharEOF)
	keymap[tui.CtrlE] = toActions(actEndOfLine)
	keymap[tui.CtrlF] = toActions(actForwardChar)
	keymap[tui.CtrlH] = toActions(actBackwardDeleteChar)
	keymap[tui.BSpace] = toActions(actBackwardDeleteChar)
	keymap[tui.Tab] = toActions(actToggleDown)
	keymap[tui.BTab] = toActions(actToggleUp)
	keymap[tui.CtrlJ] = toActions(actDown)
	keymap[tui.CtrlK] = toActions(actUp)
	keymap[tui.CtrlL] = toActions(actClearScreen)
	keymap[tui.CtrlM] = toActions(actAccept)
	keymap[tui.CtrlN] = toActions(actDown)
	keymap[tui.CtrlP] = toActions(actUp)
	keymap[tui.CtrlU] = toActions(actUnixLineDiscard)
	keymap[tui.CtrlW] = toActions(actUnixWordRubout)
	keymap[tui.CtrlY] = toActions(actYank)
	if !util.IsWindows() {
		keymap[tui.CtrlZ] = toActions(actSigStop)
	}

	keymap[tui.AltB] = toActions(actBackwardWord)
	keymap[tui.SLeft] = toActions(actBackwardWord)
	keymap[tui.AltF] = toActions(actForwardWord)
	keymap[tui.SRight] = toActions(actForwardWord)
	keymap[tui.AltD] = toActions(actKillWord)
	keymap[tui.AltBS] = toActions(actBackwardKillWord)

	keymap[tui.Up] = toActions(actUp)
	keymap[tui.Down] = toActions(actDown)
	keymap[tui.Left] = toActions(actBackwardChar)
	keymap[tui.Right] = toActions(actForwardChar)

	keymap[tui.Home] = toActions(actBeginningOfLine)
	keymap[tui.End] = toActions(actEndOfLine)
	keymap[tui.Del] = toActions(actDeleteChar)
	keymap[tui.PgUp] = toActions(actPageUp)
	keymap[tui.PgDn] = toActions(actPageDown)

	keymap[tui.SUp] = toActions(actPreviewUp)
	keymap[tui.SDown] = toActions(actPreviewDown)

	keymap[tui.Rune] = toActions(actRune)
	keymap[tui.Mouse] = toActions(actMouse)
	keymap[tui.DoubleClick] = toActions(actAccept)
	keymap[tui.LeftClick] = toActions(actIgnore)
	keymap[tui.RightClick] = toActions(actToggle)
	return keymap
}

func trimQuery(query string) []rune {
	return []rune(strings.Replace(query, "\t", " ", -1))
}

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := trimQuery(opts.Query)
	var header []string
	switch opts.Layout {
	case layoutDefault, layoutReverseList:
		header = reverseStringArray(opts.Header)
	default:
		header = opts.Header
	}
	var delay time.Duration
	if opts.Tac {
		delay = initialDelayTac
	} else {
		delay = initialDelay
	}
	var previewBox *util.EventBox
	if len(opts.Preview.command) > 0 {
		previewBox = util.NewEventBox()
	}
	strongAttr := tui.Bold
	if !opts.Bold {
		strongAttr = tui.AttrRegular
	}
	var renderer tui.Renderer
	fullscreen := opts.Height.size == 0 || opts.Height.percent && opts.Height.size == 100
	if fullscreen {
		if tui.HasFullscreenRenderer() {
			renderer = tui.NewFullscreenRenderer(opts.Theme, opts.Black, opts.Mouse)
		} else {
			renderer = tui.NewLightRenderer(opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit,
				true, func(h int) int { return h })
		}
	} else {
		maxHeightFunc := func(termHeight int) int {
			var maxHeight int
			if opts.Height.percent {
				maxHeight = util.Max(int(opts.Height.size*float64(termHeight)/100.0), opts.MinHeight)
			} else {
				maxHeight = int(opts.Height.size)
			}

			effectiveMinHeight := minHeight
			if previewBox != nil && (opts.Preview.position == posUp || opts.Preview.position == posDown) {
				effectiveMinHeight *= 2
			}
			if opts.InfoStyle != infoDefault {
				effectiveMinHeight--
			}
			if opts.BorderShape != tui.BorderNone {
				effectiveMinHeight += 2
			}
			return util.Min(termHeight, util.Max(maxHeight, effectiveMinHeight))
		}
		renderer = tui.NewLightRenderer(opts.Theme, opts.Black, opts.Mouse, opts.Tabstop, opts.ClearOnExit, false, maxHeightFunc)
	}
	wordRubout := "[^[:alnum:]][[:alnum:]]"
	wordNext := "[[:alnum:]][^[:alnum:]]|(.$)"
	if opts.FileWord {
		sep := regexp.QuoteMeta(string(os.PathSeparator))
		wordRubout = fmt.Sprintf("%s[^%s]", sep, sep)
		wordNext = fmt.Sprintf("[^%s]%s|(.$)", sep, sep)
	}
	spinner := []string{`⠋`, `⠙`, `⠹`, `⠸`, `⠼`, `⠴`, `⠦`, `⠧`, `⠇`, `⠏`}
	if !opts.Unicode {
		spinner = []string{`-`, `\`, `|`, `/`, `-`, `\`, `|`, `/`}
	}
	t := Terminal{
		initDelay:   delay,
		infoStyle:   opts.InfoStyle,
		spinner:     spinner,
		queryLen:    [2]int{0, 0},
		layout:      opts.Layout,
		fullscreen:  fullscreen,
		keepRight:   opts.KeepRight,
		hscroll:     opts.Hscroll,
		hscrollOff:  opts.HscrollOff,
		wordRubout:  wordRubout,
		wordNext:    wordNext,
		cx:          len(input),
		cy:          0,
		offset:      0,
		xoffset:     0,
		yanked:      []rune{},
		input:       input,
		multi:       opts.Multi,
		sort:        opts.Sort > 0,
		toggleSort:  opts.ToggleSort,
		delimiter:   opts.Delimiter,
		expect:      opts.Expect,
		keymap:      opts.Keymap,
		pressed:     "",
		printQuery:  opts.PrintQuery,
		history:     opts.History,
		margin:      opts.Margin,
		unicode:     opts.Unicode,
		borderShape: opts.BorderShape,
		cleanExit:   opts.ClearOnExit,
		strong:      strongAttr,
		cycle:       opts.Cycle,
		header:      header,
		header0:     header,
		ansi:        opts.Ansi,
		tabstop:     opts.Tabstop,
		reading:     true,
		failed:      nil,
		jumping:     jumpDisabled,
		jumpLabels:  opts.JumpLabels,
		printer:     opts.Printer,
		printsep:    opts.PrintSep,
		merger:      EmptyMerger,
		selected:    make(map[int32]selectedItem),
		reqBox:      util.NewEventBox(),
		preview:     opts.Preview,
		previewer:   previewer{"", 0, 0, previewBox != nil && !opts.Preview.hidden, false},
		previewBox:  previewBox,
		eventBox:    eventBox,
		mutex:       sync.Mutex{},
		suppress:    true,
		slab:        util.MakeSlab(slab16Size, slab32Size),
		theme:       opts.Theme,
		startChan:   make(chan bool, 1),
		killChan:    make(chan int),
		tui:         renderer,
		initFunc:    func() { renderer.Init() }}
	t.prompt, t.promptLen = t.processTabs([]rune(opts.Prompt), 0)
	t.pointer, t.pointerLen = t.processTabs([]rune(opts.Pointer), 0)
	t.marker, t.markerLen = t.processTabs([]rune(opts.Marker), 0)
	// Pre-calculated empty pointer and marker signs
	t.pointerEmpty = strings.Repeat(" ", t.pointerLen)
	t.markerEmpty = strings.Repeat(" ", t.markerLen)

	return &t
}

func (t *Terminal) noInfoLine() bool {
	return t.infoStyle != infoDefault
}

// Input returns current query string
func (t *Terminal) Input() []rune {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return copySlice(t.input)
}

// UpdateCount updates the count information
func (t *Terminal) UpdateCount(cnt int, final bool, failedCommand *string) {
	t.mutex.Lock()
	t.count = cnt
	t.reading = !final
	t.failed = failedCommand
	t.mutex.Unlock()
	t.reqBox.Set(reqInfo, nil)
	if final {
		t.reqBox.Set(reqRefresh, nil)
	}
}

func reverseStringArray(input []string) []string {
	size := len(input)
	reversed := make([]string, size)
	for idx, str := range input {
		reversed[size-idx-1] = str
	}
	return reversed
}

// UpdateHeader updates the header
func (t *Terminal) UpdateHeader(header []string) {
	t.mutex.Lock()
	t.header = append(append([]string{}, t.header0...), header...)
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
func (t *Terminal) UpdateList(merger *Merger, reset bool) {
	t.mutex.Lock()
	t.progress = 100
	t.merger = merger
	if reset {
		t.selected = make(map[int32]selectedItem)
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
	l := 0
	for _, r := range runes {
		l += util.RuneWidth(r, l, t.tabstop)
	}
	return l
}

const (
	minWidth  = 16
	minHeight = 4

	maxDisplayWidthCalc = 1024
)

func calculateSize(base int, size sizeSpec, margin int, minSize int) int {
	max := base - margin
	if size.percent {
		return util.Constrain(int(float64(base)*0.01*size.size), minSize, max)
	}
	return util.Constrain(int(size.size), minSize, max)
}

func (t *Terminal) resizeWindows() {
	screenWidth := t.tui.MaxX()
	screenHeight := t.tui.MaxY()
	marginInt := [4]int{}
	t.prevLines = make([]itemLine, screenHeight)
	for idx, sizeSpec := range t.margin {
		if sizeSpec.percent {
			var max float64
			if idx%2 == 0 {
				max = float64(screenHeight)
			} else {
				max = float64(screenWidth)
			}
			marginInt[idx] = int(max * sizeSpec.size * 0.01)
		} else {
			marginInt[idx] = int(sizeSpec.size)
		}
		switch t.borderShape {
		case tui.BorderHorizontal:
			marginInt[idx] += 1 - idx%2
		case tui.BorderRounded, tui.BorderSharp:
			marginInt[idx] += 1 + idx%2
		}
	}
	adjust := func(idx1 int, idx2 int, max int, min int) {
		if max >= min {
			margin := marginInt[idx1] + marginInt[idx2]
			if max-margin < min {
				desired := max - min
				marginInt[idx1] = desired * marginInt[idx1] / margin
				marginInt[idx2] = desired * marginInt[idx2] / margin
			}
		}
	}

	previewVisible := t.isPreviewEnabled() && t.preview.size.size > 0
	minAreaWidth := minWidth
	minAreaHeight := minHeight
	if previewVisible {
		switch t.preview.position {
		case posUp, posDown:
			minAreaHeight *= 2
		case posLeft, posRight:
			minAreaWidth *= 2
		}
	}
	adjust(1, 3, screenWidth, minAreaWidth)
	adjust(0, 2, screenHeight, minAreaHeight)
	if t.border != nil {
		t.border.Close()
	}
	if t.window != nil {
		t.window.Close()
	}
	if t.pborder != nil {
		t.pborder.Close()
		t.pwindow.Close()
	}

	width := screenWidth - marginInt[1] - marginInt[3]
	height := screenHeight - marginInt[0] - marginInt[2]
	switch t.borderShape {
	case tui.BorderHorizontal:
		t.border = t.tui.NewWindow(
			marginInt[0]-1,
			marginInt[3],
			width,
			height+2,
			false, tui.MakeBorderStyle(tui.BorderHorizontal, t.unicode))
	case tui.BorderRounded, tui.BorderSharp:
		t.border = t.tui.NewWindow(
			marginInt[0]-1,
			marginInt[3]-2,
			width+4,
			height+2,
			false, tui.MakeBorderStyle(t.borderShape, t.unicode))
	}
	noBorder := tui.MakeBorderStyle(tui.BorderNone, t.unicode)
	if previewVisible {
		createPreviewWindow := func(y int, x int, w int, h int) {
			previewBorder := tui.MakeBorderStyle(tui.BorderRounded, t.unicode)
			if !t.preview.border {
				previewBorder = tui.MakeTransparentBorder()
			}
			t.pborder = t.tui.NewWindow(y, x, w, h, true, previewBorder)
			pwidth := w - 4
			// ncurses auto-wraps the line when the cursor reaches the right-end of
			// the window. To prevent unintended line-wraps, we use the width one
			// column larger than the desired value.
			if !t.preview.wrap && t.tui.DoesAutoWrap() {
				pwidth += 1
			}
			t.pwindow = t.tui.NewWindow(y+1, x+2, pwidth, h-2, true, noBorder)
		}
		switch t.preview.position {
		case posUp:
			pheight := calculateSize(height, t.preview.size, minHeight, 3)
			t.window = t.tui.NewWindow(
				marginInt[0]+pheight, marginInt[3], width, height-pheight, false, noBorder)
			createPreviewWindow(marginInt[0], marginInt[3], width, pheight)
		case posDown:
			pheight := calculateSize(height, t.preview.size, minHeight, 3)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3], width, height-pheight, false, noBorder)
			createPreviewWindow(marginInt[0]+height-pheight, marginInt[3], width, pheight)
		case posLeft:
			pwidth := calculateSize(width, t.preview.size, minWidth, 5)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3]+pwidth, width-pwidth, height, false, noBorder)
			createPreviewWindow(marginInt[0], marginInt[3], pwidth, height)
		case posRight:
			pwidth := calculateSize(width, t.preview.size, minWidth, 5)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3], width-pwidth, height, false, noBorder)
			createPreviewWindow(marginInt[0], marginInt[3]+width-pwidth, pwidth, height)
		}
	} else {
		t.window = t.tui.NewWindow(
			marginInt[0],
			marginInt[3],
			width,
			height, false, noBorder)
	}
	for i := 0; i < t.window.Height(); i++ {
		t.window.MoveAndClear(i, 0)
	}
}

func (t *Terminal) move(y int, x int, clear bool) {
	h := t.window.Height()

	switch t.layout {
	case layoutDefault:
		y = h - y - 1
	case layoutReverseList:
		n := 2 + len(t.header)
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
	maxOffset := util.Min(util.Min(len(t.input), minOffset+maxWidth), t.cx)

	t.xoffset = util.Constrain(t.xoffset, minOffset, maxOffset)
	before, _ := t.trimLeft(t.input[t.xoffset:t.cx], maxWidth)
	beforeLen := t.displayWidth(before)
	after, _ := t.trimRight(t.input[t.cx:], maxWidth-beforeLen)
	afterLen := t.displayWidth(after)
	t.queryLen = [2]int{beforeLen, afterLen}
	return before, after
}

func (t *Terminal) placeCursor() {
	t.move(0, t.promptLen+t.queryLen[0], false)
}

func (t *Terminal) printPrompt() {
	t.move(0, 0, true)
	t.window.CPrint(tui.ColPrompt, t.strong, t.prompt)

	before, after := t.updatePromptOffset()
	t.window.CPrint(tui.ColNormal, t.strong, string(before))
	t.window.CPrint(tui.ColNormal, t.strong, string(after))
}

func (t *Terminal) printInfo() {
	pos := 0
	switch t.infoStyle {
	case infoDefault:
		t.move(1, 0, true)
		if t.reading {
			duration := int64(spinnerDuration)
			idx := (time.Now().UnixNano() % (duration * int64(len(t.spinner)))) / duration
			t.window.CPrint(tui.ColSpinner, t.strong, t.spinner[idx])
		}
		t.move(1, 2, false)
		pos = 2
	case infoInline:
		pos = t.promptLen + t.queryLen[0] + t.queryLen[1] + 1
		if pos+len(" < ") > t.window.Width() {
			return
		}
		t.move(0, pos, true)
		if t.reading {
			t.window.CPrint(tui.ColSpinner, t.strong, " < ")
		} else {
			t.window.CPrint(tui.ColPrompt, t.strong, " < ")
		}
		pos += len(" < ")
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
	if len(t.selected) > 0 {
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
	maxWidth := t.window.Width() - pos
	if len(output) > maxWidth {
		outputRunes, _ := t.trimRight([]rune(output), maxWidth-2)
		output = string(outputRunes) + strings.Repeat(".", util.Constrain(maxWidth, 0, 2))
	}
	t.window.CPrint(tui.ColInfo, 0, output)
}

func (t *Terminal) printHeader() {
	if len(t.header) == 0 {
		return
	}
	max := t.window.Height()
	var state *ansiState
	for idx, lineStr := range t.header {
		line := idx + 2
		if t.noInfoLine() {
			line--
		}
		if line >= max {
			continue
		}
		trimmed, colors, newState := extractColor(lineStr, state, nil)
		state = newState
		item := &Item{
			text:   util.ToChars([]byte(trimmed)),
			colors: colors}

		t.move(line, 2, true)
		t.printHighlighted(Result{item: item},
			tui.AttrRegular, tui.ColHeader, tui.ColHeader, false, false)
	}
}

func (t *Terminal) printList() {
	t.constrain()

	maxy := t.maxItems()
	count := t.merger.Length() - t.offset
	for j := 0; j < maxy; j++ {
		i := j
		if t.layout == layoutDefault {
			i = maxy - 1 - j
		}
		line := i + 2 + len(t.header)
		if t.noInfoLine() {
			line--
		}
		if i < count {
			t.printItem(t.merger.Get(i+t.offset), line, i, i == t.cy-t.offset)
		} else if t.prevLines[i] != emptyLine {
			t.prevLines[i] = emptyLine
			t.move(line, 0, true)
		}
	}
}

func (t *Terminal) printItem(result Result, line int, i int, current bool) {
	item := result.item
	_, selected := t.selected[item.Index()]
	label := t.pointerEmpty
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
	newLine := itemLine{current: current, selected: selected, label: label,
		result: result, queryLen: len(t.input), width: 0}
	prevLine := t.prevLines[i]
	if prevLine.current == newLine.current &&
		prevLine.selected == newLine.selected &&
		prevLine.label == newLine.label &&
		prevLine.queryLen == newLine.queryLen &&
		prevLine.result == newLine.result {
		return
	}

	t.move(line, 0, false)
	if current {
		t.window.CPrint(tui.ColCurrentCursor, t.strong, label)
		if selected {
			t.window.CPrint(tui.ColCurrentSelected, t.strong, t.marker)
		} else {
			t.window.CPrint(tui.ColCurrentSelected, t.strong, t.markerEmpty)
		}
		newLine.width = t.printHighlighted(result, t.strong, tui.ColCurrent, tui.ColCurrentMatch, true, true)
	} else {
		t.window.CPrint(tui.ColCursor, t.strong, label)
		if selected {
			t.window.CPrint(tui.ColSelected, t.strong, t.marker)
		} else {
			t.window.Print(t.markerEmpty)
		}
		newLine.width = t.printHighlighted(result, 0, tui.ColNormal, tui.ColMatch, false, true)
	}
	fillSpaces := prevLine.width - newLine.width
	if fillSpaces > 0 {
		t.window.Print(strings.Repeat(" ", fillSpaces))
	}
	t.prevLines[i] = newLine
}

func (t *Terminal) trimRight(runes []rune, width int) ([]rune, int) {
	// We start from the beginning to handle tab characters
	l := 0
	for idx, r := range runes {
		l += util.RuneWidth(r, l, t.tabstop)
		if l > width {
			return runes[:idx], len(runes) - idx
		}
	}
	return runes, 0
}

func (t *Terminal) displayWidthWithLimit(runes []rune, prefixWidth int, limit int) int {
	l := 0
	for _, r := range runes {
		l += util.RuneWidth(r, l+prefixWidth, t.tabstop)
		if l > limit {
			// Early exit
			return l
		}
	}
	return l
}

func (t *Terminal) trimLeft(runes []rune, width int) ([]rune, int32) {
	if len(runes) > maxDisplayWidthCalc && len(runes) > width {
		trimmed := len(runes) - width
		return runes[trimmed:], int32(trimmed)
	}

	currentWidth := t.displayWidth(runes)
	var trimmed int32

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

func (t *Terminal) printHighlighted(result Result, attr tui.Attr, col1 tui.ColorPair, col2 tui.ColorPair, current bool, match bool) int {
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

	offsets := result.colorOffsets(charOffsets, t.theme, col2, attr, current)
	maxWidth := t.window.Width() - 3
	maxe = util.Constrain(maxe+util.Min(maxWidth/2-2, t.hscrollOff), 0, len(text))
	displayWidth := t.displayWidthWithLimit(text, 0, maxWidth)
	if displayWidth > maxWidth {
		transformOffsets := func(diff int32) {
			for idx, offset := range offsets {
				b, e := offset.offset[0], offset.offset[1]
				b += 2 - diff
				e += 2 - diff
				b = util.Max32(b, 2)
				offsets[idx].offset[0] = b
				offsets[idx].offset[1] = util.Max32(b, e)
			}
		}
		if t.hscroll {
			if t.keepRight && pos == nil {
				trimmed, diff := t.trimLeft(text, maxWidth-2)
				transformOffsets(diff)
				text = append([]rune(ellipsis), trimmed...)
			} else if !t.overflow(text[:maxe], maxWidth-2) {
				// Stri..
				text, _ = t.trimRight(text, maxWidth-2)
				text = append(text, []rune(ellipsis)...)
			} else {
				// Stri..
				if t.overflow(text[maxe:], 2) {
					text = append(text[:maxe], []rune(ellipsis)...)
				}
				// ..ri..
				var diff int32
				text, diff = t.trimLeft(text, maxWidth-2)

				// Transform offsets
				transformOffsets(diff)
				text = append([]rune(ellipsis), text...)
			}
		} else {
			text, _ = t.trimRight(text, maxWidth-2)
			text = append(text, []rune(ellipsis)...)

			for idx, offset := range offsets {
				offsets[idx].offset[0] = util.Min32(offset.offset[0], int32(maxWidth-2))
				offsets[idx].offset[1] = util.Min32(offset.offset[1], int32(maxWidth))
			}
		}
		displayWidth = t.displayWidthWithLimit(text, 0, displayWidth)
	}

	var index int32
	var substr string
	var prefixWidth int
	maxOffset := int32(len(text))
	for _, offset := range offsets {
		b := util.Constrain32(offset.offset[0], index, maxOffset)
		e := util.Constrain32(offset.offset[1], index, maxOffset)

		substr, prefixWidth = t.processTabs(text[index:b], prefixWidth)
		t.window.CPrint(col1, attr, substr)

		if b < e {
			substr, prefixWidth = t.processTabs(text[b:e], prefixWidth)
			t.window.CPrint(offset.color, offset.attr, substr)
		}

		index = e
		if index >= maxOffset {
			break
		}
	}
	if index < maxOffset {
		substr, _ = t.processTabs(text[index:], prefixWidth)
		t.window.CPrint(col1, attr, substr)
	}
	return displayWidth
}

func (t *Terminal) printPreview() {
	if !t.hasPreviewWindow() {
		return
	}
	t.pwindow.Erase()

	maxWidth := t.pwindow.Width()
	if t.tui.DoesAutoWrap() {
		maxWidth -= 1
	}
	reader := bufio.NewReader(strings.NewReader(t.previewer.text))
	lineNo := -t.previewer.offset
	height := t.pwindow.Height()
	t.previewer.more = t.previewer.offset > 0
	var ansi *ansiState
	for ; ; lineNo++ {
		line, err := reader.ReadString('\n')
		eof := err == io.EOF
		if !eof {
			line = line[:len(line)-1]
		}
		if lineNo >= height || t.pwindow.Y() == height-1 && t.pwindow.X() > 0 {
			break
		} else if lineNo >= 0 {
			var fillRet tui.FillReturn
			prefixWidth := 0
			_, _, ansi = extractColor(line, ansi, func(str string, ansi *ansiState) bool {
				trimmed := []rune(str)
				if !t.preview.wrap {
					trimmed, _ = t.trimRight(trimmed, maxWidth-t.pwindow.X())
				}
				str, width := t.processTabs(trimmed, prefixWidth)
				prefixWidth += width
				if t.theme != nil && ansi != nil && ansi.colored() {
					fillRet = t.pwindow.CFill(ansi.fg, ansi.bg, ansi.attr, str)
				} else {
					fillRet = t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), tui.AttrRegular, str)
				}
				return fillRet == tui.FillContinue
			})
			t.previewer.more = t.previewer.more || t.pwindow.Y() == height-1 && t.pwindow.X() == t.pwindow.Width()
			if fillRet == tui.FillNextLine {
				continue
			} else if fillRet == tui.FillSuspend {
				break
			}
			t.pwindow.Fill("\n")
		}
		if eof {
			break
		}
	}
	t.pwindow.FinishFill()
	if t.previewer.lines > height {
		t.previewer.more = true
		offset := fmt.Sprintf("%d/%d", t.previewer.offset+1, t.previewer.lines)
		pos := t.pwindow.Width() - len(offset)
		if t.tui.DoesAutoWrap() {
			pos -= 1
		}
		t.pwindow.Move(0, pos)
		t.pwindow.CPrint(tui.ColInfo, tui.Reverse, offset)
	}
}

func (t *Terminal) processTabs(runes []rune, prefixWidth int) (string, int) {
	var strbuf bytes.Buffer
	l := prefixWidth
	for _, r := range runes {
		w := util.RuneWidth(r, l, t.tabstop)
		l += w
		if r == '\t' {
			strbuf.WriteString(strings.Repeat(" ", w))
		} else {
			strbuf.WriteRune(r)
		}
	}
	return strbuf.String(), l
}

func (t *Terminal) printAll() {
	t.resizeWindows()
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
			windows = append(windows, t.pborder, t.pwindow)
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
	return locs[len(locs)-1][0]
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
	return loc[0]
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

func keyMatch(key int, event tui.Event) bool {
	return event.Type == key ||
		event.Type == tui.Rune && int(event.Char) == key-tui.AltZ ||
		event.Type == tui.Mouse && key == tui.DoubleClick && event.MouseEvent.Double
}

func quoteEntryCmd(entry string) string {
	escaped := strings.Replace(entry, `\`, `\\`, -1)
	escaped = `"` + strings.Replace(escaped, `"`, `\"`, -1) + `"`
	r, _ := regexp.Compile(`[&|<>()@^%!"]`)
	return r.ReplaceAllStringFunc(escaped, func(match string) string {
		return "^" + match
	})
}

func quoteEntry(entry string) string {
	if util.IsWindows() {
		return quoteEntryCmd(entry)
	}
	return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
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
	f, err := ioutil.TempFile("", "fzf-preview-*")
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

func replacePlaceholder(template string, stripAnsi bool, delimiter Delimiter, printsep string, forcePlus bool, query string, allItems []*Item) string {
	current := allItems[:1]
	selected := allItems[1:]
	if current[0] == nil {
		current = []*Item{}
	}
	if selected[0] == nil {
		selected = []*Item{}
	}
	return placeholder.ReplaceAllStringFunc(template, func(match string) string {
		escaped, match, flags := parsePlaceholder(match)

		if escaped {
			return match
		}

		// Current query
		if match == "{q}" {
			return quoteEntry(query)
		}

		items := current
		if flags.plus || forcePlus {
			items = selected
		}

		replacements := make([]string, len(items))

		if match == "{}" {
			for idx, item := range items {
				if flags.number {
					n := int(item.text.Index)
					if n < 0 {
						replacements[idx] = ""
					} else {
						replacements[idx] = strconv.Itoa(n)
					}
				} else if flags.file {
					replacements[idx] = item.AsString(stripAnsi)
				} else {
					replacements[idx] = quoteEntry(item.AsString(stripAnsi))
				}
			}
			if flags.file {
				return writeTemporaryFile(replacements, printsep)
			}
			return strings.Join(replacements, " ")
		}

		tokens := strings.Split(match[1:len(match)-1], ",")
		ranges := make([]Range, len(tokens))
		for idx, s := range tokens {
			r, ok := ParseRange(&s)
			if !ok {
				// Invalid expression, just return the original string in the template
				return match
			}
			ranges[idx] = r
		}

		for idx, item := range items {
			tokens := Tokenize(item.AsString(stripAnsi), delimiter)
			trans := Transform(tokens, ranges)
			str := joinTokens(trans)
			if delimiter.str != nil {
				str = strings.TrimSuffix(str, *delimiter.str)
			} else if delimiter.regex != nil {
				delims := delimiter.regex.FindAllStringIndex(str, -1)
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
			replacements[idx] = str
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

func (t *Terminal) executeCommand(template string, forcePlus bool, background bool) {
	valid, list := t.buildPlusList(template, forcePlus)
	if !valid {
		return
	}
	command := replacePlaceholder(template, t.ansi, t.delimiter, t.printsep, forcePlus, string(t.input), list)
	cmd := util.ExecCommand(command, false)
	if !background {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		t.tui.Pause(true)
		cmd.Run()
		t.tui.Resume(true)
		t.redraw()
		t.refresh()
	} else {
		t.tui.Pause(false)
		cmd.Run()
		t.tui.Resume(false)
	}
	cleanTemporaryFiles()
}

func (t *Terminal) hasPreviewer() bool {
	return t.previewBox != nil
}

func (t *Terminal) isPreviewEnabled() bool {
	return t.hasPreviewer() && t.previewer.enabled
}

func (t *Terminal) hasPreviewWindow() bool {
	return t.pwindow != nil && t.isPreviewEnabled()
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
	_, plus, query := hasPreviewFlags(template)
	if !(query && len(t.input) > 0 || (forcePlus || plus) && len(t.selected) > 0) {
		return current != nil, []*Item{current, current}
	}

	// We would still want to update preview window even if there is no match if
	//   1. command template contains {q} and the query string is not empty
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

func (t *Terminal) deselectItem(item *Item) {
	delete(t.selected, item.Index())
	t.version++
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
			os.Exit(code)
		}
	}
}

func (t *Terminal) cancelPreview() {
	t.killPreview(exitCancel)
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
	// prof := profile.Start(profile.ProfilePath("/tmp/"))
	<-t.startChan
	{ // Late initialization
		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-intChan
			t.reqBox.Set(reqQuit, nil)
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
				t.reqBox.Set(reqRedraw, nil)
			}
		}()

		t.mutex.Lock()
		t.initFunc()
		t.resizeWindows()
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
			for {
				var request []*Item
				t.previewBox.Wait(func(events *util.Events) {
					for req, value := range *events {
						switch req {
						case reqPreviewEnqueue:
							request = value.([]*Item)
						}
					}
					events.Clear()
				})
				// We don't display preview window if no match
				if request[0] != nil {
					command := replacePlaceholder(t.preview.command,
						t.ansi, t.delimiter, t.printsep, false, string(t.Input()), request)
					cmd := util.ExecCommand(command, true)
					if t.pwindow != nil {
						env := os.Environ()
						lines := fmt.Sprintf("LINES=%d", t.pwindow.Height())
						columns := fmt.Sprintf("COLUMNS=%d", t.pwindow.Width())
						env = append(env, lines)
						env = append(env, "FZF_PREVIEW_"+lines)
						env = append(env, columns)
						env = append(env, "FZF_PREVIEW_"+columns)
						cmd.Env = env
					}
					var out bytes.Buffer
					cmd.Stdout = &out
					cmd.Stderr = &out
					err := cmd.Start()
					if err != nil {
						out.Write([]byte(err.Error()))
					}
					finishChan := make(chan bool, 1)
					updateChan := make(chan bool)
					go func() {
						select {
						case code := <-t.killChan:
							if code != exitCancel {
								util.KillCommand(cmd)
								os.Exit(code)
							} else {
								select {
								case <-time.After(previewCancelWait):
									util.KillCommand(cmd)
									updateChan <- true
								case <-finishChan:
									updateChan <- false
								}
							}
						case <-finishChan:
							updateChan <- false
						}
					}()
					cmd.Wait()
					finishChan <- true
					if out.Len() > 0 || !<-updateChan {
						t.reqBox.Set(reqPreviewDisplay, out.String())
					}
					cleanTemporaryFiles()
				} else {
					t.reqBox.Set(reqPreviewDisplay, "")
				}
			}
		}()
	}

	exit := func(getCode func() int) {
		t.tui.Close()
		code := getCode()
		if code <= exitNoMatch && t.history != nil {
			t.history.append(string(t.input))
		}
		// prof.Stop()
		t.killPreview(code)
	}

	go func() {
		var focusedIndex int32 = minItem.Index()
		var version int64 = -1
		for {
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
						if focusedIndex != currentIndex || version != t.version {
							version = t.version
							focusedIndex = currentIndex
							if t.isPreviewEnabled() {
								_, list := t.buildPlusList(t.preview.command, false)
								t.cancelPreview()
								t.previewBox.Set(reqPreviewEnqueue, list)
							}
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
					case reqReinit:
						t.tui.Resume(t.fullscreen)
						t.redraw()
					case reqRedraw:
						t.redraw()
					case reqClose:
						exit(func() int {
							if t.output() {
								return exitOk
							}
							return exitNoMatch
						})
					case reqPreviewDisplay:
						t.previewer.text = value.(string)
						t.previewer.lines = strings.Count(t.previewer.text, "\n")
						t.previewer.offset = 0
						t.printPreview()
					case reqPreviewRefresh:
						t.printPreview()
					case reqPrintQuery:
						exit(func() int {
							t.printer(string(t.input))
							return exitOk
						})
					case reqQuit:
						exit(func() int { return exitInterrupt })
					}
				}
				t.refresh()
				t.mutex.Unlock()
			})
		}
	}()

	looping := true
	for looping {
		var newCommand *string
		changed := false
		queryChanged := false

		event := t.tui.GetChar()

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
		toggle := func() bool {
			if t.cy < t.merger.Length() && t.toggleItem(t.merger.Get(t.cy).item) {
				req(reqInfo)
				return true
			}
			return false
		}
		scrollPreview := func(amount int) {
			if !t.previewer.more {
				return
			}
			newOffset := util.Constrain(
				t.previewer.offset+amount, 0, t.previewer.lines-1)
			if t.previewer.offset != newOffset {
				t.previewer.offset = newOffset
				req(reqPreviewRefresh)
			}
		}
		for key, ret := range t.expect {
			if keyMatch(key, event) {
				t.pressed = ret
				t.reqBox.Set(reqClose, nil)
				t.mutex.Unlock()
				return
			}
		}

		var doAction func(action, int) bool
		doActions := func(actions []action, mapkey int) bool {
			for _, action := range actions {
				if !doAction(action, mapkey) {
					return false
				}
			}
			return true
		}
		doAction = func(a action, mapkey int) bool {
			switch a.t {
			case actIgnore:
			case actExecute, actExecuteSilent:
				t.executeCommand(a.a, false, a.t == actExecuteSilent)
			case actExecuteMulti:
				t.executeCommand(a.a, true, false)
			case actInvalid:
				t.mutex.Unlock()
				return false
			case actTogglePreview:
				if t.hasPreviewer() {
					t.previewer.enabled = !t.previewer.enabled
					t.tui.Clear()
					t.resizeWindows()
					if t.previewer.enabled {
						valid, list := t.buildPlusList(t.preview.command, false)
						if valid {
							t.cancelPreview()
							t.previewBox.Set(reqPreviewEnqueue, list)
						}
					}
					req(reqPrompt, reqList, reqInfo, reqHeader)
				}
			case actTogglePreviewWrap:
				if t.hasPreviewWindow() {
					t.preview.wrap = !t.preview.wrap
					req(reqPreviewRefresh)
				}
			case actToggleSort:
				t.sort = !t.sort
				changed = true
			case actPreviewUp:
				if t.hasPreviewWindow() {
					scrollPreview(-1)
				}
			case actPreviewDown:
				if t.hasPreviewWindow() {
					scrollPreview(1)
				}
			case actPreviewPageUp:
				if t.hasPreviewWindow() {
					scrollPreview(-t.pwindow.Height())
				}
			case actPreviewPageDown:
				if t.hasPreviewWindow() {
					scrollPreview(t.pwindow.Height())
				}
			case actBeginningOfLine:
				t.cx = 0
			case actBackwardChar:
				if t.cx > 0 {
					t.cx--
				}
			case actPrintQuery:
				req(reqPrintQuery)
			case actReplaceQuery:
				if t.cy >= 0 && t.cy < t.merger.Length() {
					t.input = t.merger.Get(t.cy).item.text.ToRunes()
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
					return doAction(action{t: actToggleUp}, mapkey)
				}
				return doAction(action{t: actToggleDown}, mapkey)
			case actToggleOut:
				if t.layout != layoutDefault {
					return doAction(action{t: actToggleDown}, mapkey)
				}
				return doAction(action{t: actToggleUp}, mapkey)
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
				req(reqRedraw)
			case actClearQuery:
				t.input = []rune{}
				t.cx = 0
			case actClearSelection:
				if t.multi > 0 {
					t.selected = make(map[int32]selectedItem)
					t.version++
					req(reqList, reqInfo)
				}
			case actTop:
				t.vset(0)
				req(reqList)
			case actUnixLineDiscard:
				if t.cx > 0 {
					t.yanked = copySlice(t.input[:t.cx])
					t.input = t.input[t.cx:]
					t.cx = 0
				}
			case actUnixWordRubout:
				if t.cx > 0 {
					t.rubout("\\s\\S")
				}
			case actBackwardKillWord:
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
			case actPreviousHistory:
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
			case actSigStop:
				p, err := os.FindProcess(os.Getpid())
				if err == nil {
					t.tui.Clear()
					t.tui.Pause(t.fullscreen)
					notifyStop(p)
					t.mutex.Unlock()
					return false
				}
			case actMouse:
				me := event.MouseEvent
				mx, my := me.X, me.Y
				if me.S != 0 {
					// Scroll
					if t.window.Enclose(my, mx) && t.merger.Length() > 0 {
						if t.multi > 0 && me.Mod {
							toggle()
						}
						t.vmove(me.S, true)
						req(reqList)
					} else if t.hasPreviewWindow() && t.pwindow.Enclose(my, mx) {
						scrollPreview(-me.S)
					}
				} else if t.window.Enclose(my, mx) {
					mx -= t.window.Left()
					my -= t.window.Top()
					mx = util.Constrain(mx-t.promptLen, 0, len(t.input))
					min := 2 + len(t.header)
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
					if me.Double {
						// Double-click
						if my >= min {
							if t.vset(t.offset+my-min) && t.cy < t.merger.Length() {
								return doActions(t.keymap[tui.DoubleClick], tui.DoubleClick)
							}
						}
					} else if me.Down {
						if my == 0 && mx >= 0 {
							// Prompt
							t.cx = mx + t.xoffset
						} else if my >= min {
							// List
							if t.vset(t.offset+my-min) && t.multi > 0 && me.Mod {
								toggle()
							}
							req(reqList)
							if me.Left {
								return doActions(t.keymap[tui.LeftClick], tui.LeftClick)
							}
							return doActions(t.keymap[tui.RightClick], tui.RightClick)
						}
					}
				}
			case actReload:
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
					command := replacePlaceholder(a.a,
						t.ansi, t.delimiter, t.printsep, false, string(t.input), list)
					newCommand = &command
				}
			}
			return true
		}
		mapkey := event.Type
		if t.jumping == jumpDisabled {
			actions := t.keymap[mapkey]
			if mapkey == tui.Rune {
				mapkey = int(event.Char) + int(tui.AltZ)
				if act, prs := t.keymap[mapkey]; prs {
					actions = act
				}
			}
			if !doActions(actions, mapkey) {
				continue
			}
			t.truncateQuery()
			queryChanged = string(previousInput) != string(t.input)
			changed = changed || queryChanged
			if onChanges, prs := t.keymap[tui.Change]; queryChanged && prs {
				if !doActions(onChanges, tui.Change) {
					continue
				}
			}
		} else {
			if mapkey == tui.Rune {
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

		if queryChanged {
			if t.isPreviewEnabled() {
				_, _, q := hasPreviewFlags(t.preview.command)
				if q {
					t.version++
				}
			}
		}

		if queryChanged || t.cx != previousCx {
			req(reqPrompt)
		}

		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if changed || newCommand != nil {
			t.eventBox.Set(EvtSearchNew, searchRequest{sort: t.sort, command: newCommand})
		}
		for _, event := range events {
			t.reqBox.Set(event, nil)
		}
	}
}

func (t *Terminal) constrain() {
	count := t.merger.Length()
	height := t.maxItems()
	diffpos := t.cy - t.offset

	t.cy = util.Constrain(t.cy, 0, count-1)
	t.offset = util.Constrain(t.offset, t.cy-height+1, t.cy)
	// Adjustment
	if count-t.offset < height {
		t.offset = util.Max(0, count-height)
		t.cy = util.Constrain(t.offset+diffpos, 0, count-1)
	}
	t.offset = util.Max(0, t.offset)
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
	max := t.window.Height() - 2 - len(t.header)
	if t.noInfoLine() {
		max++
	}
	return util.Max(max, 0)
}
