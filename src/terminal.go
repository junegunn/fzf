package fzf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

func init() {
	placeholder = regexp.MustCompile("\\\\?(?:{\\+?[0-9,-.]*}|{q})")
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
	initDelay  time.Duration
	inlineInfo bool
	prompt     string
	promptLen  int
	reverse    bool
	fullscreen bool
	hscroll    bool
	hscrollOff int
	wordRubout string
	wordNext   string
	cx         int
	cy         int
	offset     int
	yanked     []rune
	input      []rune
	multi      bool
	sort       bool
	toggleSort bool
	delimiter  Delimiter
	expect     map[int]string
	keymap     map[int][]Action
	pressed    string
	printQuery bool
	history    *History
	cycle      bool
	header     []string
	header0    []string
	ansi       bool
	tabstop    int
	margin     [4]SizeSpec
	strong     tui.Attr
	bordered   bool
	cleanExit  bool
	border     tui.Window
	window     tui.Window
	pborder    tui.Window
	pwindow    tui.Window
	count      int
	progress   int
	reading    bool
	success    bool
	jumping    jumpMode
	jumpLabels string
	printer    func(string)
	merger     *Merger
	selected   map[int32]selectedItem
	version    int64
	reqBox     *util.EventBox
	preview    PreviewOpts
	previewer  previewer
	previewBox *util.EventBox
	eventBox   *util.EventBox
	mutex      sync.Mutex
	initFunc   func()
	prevLines  []itemLine
	suppress   bool
	startChan  chan bool
	slab       *util.Slab
	theme      *tui.ColorTheme
	tui        tui.Renderer
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

var _spinner = []string{`-`, `\`, `|`, `/`, `-`, `\`, `|`, `/`}

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

type Action struct {
	Type    ActionType
	Command Command
}

type ActionType int

const (
	ActionTypeIgnore ActionType = iota
	ActionTypeInvalid
	ActionTypeRune
	ActionTypeMouse
	ActionTypeBeginningOfLine
	ActionTypeAbort
	ActionTypeAccept
	ActionTypeBackwardChar
	ActionTypeBackwardDeleteChar
	ActionTypeBackwardWord
	ActionTypeCancel
	ActionTypeClearScreen
	ActionTypeDeleteChar
	ActionTypeDeleteCharEOF
	ActionTypeEndOfLine
	ActionTypeForwardChar
	ActionTypeForwardWord
	ActionTypeKillLine
	ActionTypeKillWord
	ActionTypeUnixLineDiscard
	ActionTypeUnixWordRubout
	ActionTypeYank
	ActionTypeBackwardKillWord
	ActionTypeSelectAll
	ActionTypeDeselectAll
	ActionTypeToggle
	ActionTypeToggleAll
	ActionTypeToggleDown
	ActionTypeToggleUp
	ActionTypeToggleIn
	ActionTypeToggleOut
	ActionTypeDown
	ActionTypeUp
	ActionTypePageUp
	ActionTypePageDown
	ActionTypeHalfPageUp
	ActionTypeHalfPageDown
	ActionTypeJump
	ActionTypeJumpAccept
	ActionTypePrintQuery
	ActionTypeToggleSort
	ActionTypeTogglePreview
	ActionTypeTogglePreviewWrap
	ActionTypePreviewUp
	ActionTypePreviewDown
	ActionTypePreviewPageUp
	ActionTypePreviewPageDown
	ActionTypePreviousHistory
	ActionTypeNextHistory
	ActionTypeExecute
	ActionTypeExecuteSilent
	ActionTypeExecuteMulti // Deprecated
	ActionTypeSigStop
	ActionTypeTop
)

func toActions(types ...ActionType) []Action {
	actions := make([]Action, len(types))
	for idx, t := range types {
		actions[idx] = Action{Type: t, Command: nil}
	}
	return actions
}

func defaultKeymap() map[int][]Action {
	keymap := make(map[int][]Action)
	keymap[tui.Invalid] = toActions(ActionTypeInvalid)
	keymap[tui.Resize] = toActions(ActionTypeClearScreen)
	keymap[tui.CtrlA] = toActions(ActionTypeBeginningOfLine)
	keymap[tui.CtrlB] = toActions(ActionTypeBackwardChar)
	keymap[tui.CtrlC] = toActions(ActionTypeAbort)
	keymap[tui.CtrlG] = toActions(ActionTypeAbort)
	keymap[tui.CtrlQ] = toActions(ActionTypeAbort)
	keymap[tui.ESC] = toActions(ActionTypeAbort)
	keymap[tui.CtrlD] = toActions(ActionTypeDeleteCharEOF)
	keymap[tui.CtrlE] = toActions(ActionTypeEndOfLine)
	keymap[tui.CtrlF] = toActions(ActionTypeForwardChar)
	keymap[tui.CtrlH] = toActions(ActionTypeBackwardDeleteChar)
	keymap[tui.BSpace] = toActions(ActionTypeBackwardDeleteChar)
	keymap[tui.Tab] = toActions(ActionTypeToggleDown)
	keymap[tui.BTab] = toActions(ActionTypeToggleUp)
	keymap[tui.CtrlJ] = toActions(ActionTypeDown)
	keymap[tui.CtrlK] = toActions(ActionTypeUp)
	keymap[tui.CtrlL] = toActions(ActionTypeClearScreen)
	keymap[tui.CtrlM] = toActions(ActionTypeAccept)
	keymap[tui.CtrlN] = toActions(ActionTypeDown)
	keymap[tui.CtrlP] = toActions(ActionTypeUp)
	keymap[tui.CtrlU] = toActions(ActionTypeUnixLineDiscard)
	keymap[tui.CtrlW] = toActions(ActionTypeUnixWordRubout)
	keymap[tui.CtrlY] = toActions(ActionTypeYank)
	if !util.IsWindows() {
		keymap[tui.CtrlZ] = toActions(ActionTypeSigStop)
	}

	keymap[tui.AltB] = toActions(ActionTypeBackwardWord)
	keymap[tui.SLeft] = toActions(ActionTypeBackwardWord)
	keymap[tui.AltF] = toActions(ActionTypeForwardWord)
	keymap[tui.SRight] = toActions(ActionTypeForwardWord)
	keymap[tui.AltD] = toActions(ActionTypeKillWord)
	keymap[tui.AltBS] = toActions(ActionTypeBackwardKillWord)

	keymap[tui.Up] = toActions(ActionTypeUp)
	keymap[tui.Down] = toActions(ActionTypeDown)
	keymap[tui.Left] = toActions(ActionTypeBackwardChar)
	keymap[tui.Right] = toActions(ActionTypeForwardChar)

	keymap[tui.Home] = toActions(ActionTypeBeginningOfLine)
	keymap[tui.End] = toActions(ActionTypeEndOfLine)
	keymap[tui.Del] = toActions(ActionTypeDeleteChar)
	keymap[tui.PgUp] = toActions(ActionTypePageUp)
	keymap[tui.PgDn] = toActions(ActionTypePageDown)

	keymap[tui.Rune] = toActions(ActionTypeRune)
	keymap[tui.Mouse] = toActions(ActionTypeMouse)
	keymap[tui.DoubleClick] = toActions(ActionTypeAccept)
	return keymap
}

func trimQuery(query string) []rune {
	return []rune(strings.Replace(query, "\t", " ", -1))
}

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := trimQuery(opts.Query)
	var header []string
	if opts.Reverse {
		header = opts.Header
	} else {
		header = reverseStringArray(opts.Header)
	}
	var delay time.Duration
	if opts.Tac {
		delay = initialDelayTac
	} else {
		delay = initialDelay
	}
	var previewBox *util.EventBox
	if opts.Preview.Command != nil {
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
			if previewBox != nil && (opts.Preview.Position == WindowPositionUp || opts.Preview.Position == WindowPositionDown) {
				effectiveMinHeight *= 2
			}
			if opts.InlineInfo {
				effectiveMinHeight -= 1
			}
			if opts.Bordered {
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
	t := Terminal{
		initDelay:  delay,
		inlineInfo: opts.InlineInfo,
		reverse:    opts.Reverse,
		fullscreen: fullscreen,
		hscroll:    opts.Hscroll,
		hscrollOff: opts.HscrollOff,
		wordRubout: wordRubout,
		wordNext:   wordNext,
		cx:         len(input),
		cy:         0,
		offset:     0,
		yanked:     []rune{},
		input:      input,
		multi:      opts.Multi,
		sort:       opts.Sort > 0,
		toggleSort: opts.ToggleSort,
		delimiter:  opts.Delimiter,
		expect:     opts.Expect,
		keymap:     opts.Keymap,
		pressed:    "",
		printQuery: opts.PrintQuery,
		history:    opts.History,
		margin:     opts.Margin,
		bordered:   opts.Bordered,
		cleanExit:  opts.ClearOnExit,
		strong:     strongAttr,
		cycle:      opts.Cycle,
		header:     header,
		header0:    header,
		ansi:       opts.Ansi,
		tabstop:    opts.Tabstop,
		reading:    true,
		success:    true,
		jumping:    jumpDisabled,
		jumpLabels: opts.JumpLabels,
		printer:    opts.Printer,
		merger:     EmptyMerger,
		selected:   make(map[int32]selectedItem),
		reqBox:     util.NewEventBox(),
		preview:    opts.Preview,
		previewer:  previewer{"", 0, 0, previewBox != nil && !opts.Preview.Hidden},
		previewBox: previewBox,
		eventBox:   eventBox,
		mutex:      sync.Mutex{},
		suppress:   true,
		slab:       util.MakeSlab(slab16Size, slab32Size),
		theme:      opts.Theme,
		startChan:  make(chan bool, 1),
		tui:        renderer,
		initFunc:   func() { renderer.Init() }}
	t.prompt, t.promptLen = t.processTabs([]rune(opts.Prompt), 0)
	return &t
}

// Input returns current query string
func (t *Terminal) Input() []rune {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return copySlice(t.input)
}

// UpdateCount updates the count information
func (t *Terminal) UpdateCount(cnt int, final bool, success bool) {
	t.mutex.Lock()
	t.count = cnt
	t.reading = !final
	t.success = success
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
func (t *Terminal) UpdateList(merger *Merger) {
	t.mutex.Lock()
	t.progress = 100
	t.merger = merger
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

func calculateSize(base int, size SizeSpec, margin int, minSize int) int {
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
		if t.bordered && idx%2 == 0 {
			marginInt[idx] += 1
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

	previewVisible := t.isPreviewEnabled() && t.preview.Size.size > 0
	minAreaWidth := minWidth
	minAreaHeight := minHeight
	if previewVisible {
		switch t.preview.Position {
		case WindowPositionUp, WindowPositionDown:
			minAreaHeight *= 2
		case WindowPositionLeft, WindowPositionRight:
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
	if t.bordered {
		t.border = t.tui.NewWindow(
			marginInt[0]-1,
			marginInt[3],
			width,
			height+2, tui.BorderHorizontal)
	}
	if previewVisible {
		createPreviewWindow := func(y int, x int, w int, h int) {
			t.pborder = t.tui.NewWindow(y, x, w, h, tui.BorderAround)
			pwidth := w - 4
			// ncurses auto-wraps the line when the cursor reaches the right-end of
			// the window. To prevent unintended line-wraps, we use the width one
			// column larger than the desired value.
			if !t.preview.Wrap && t.tui.DoesAutoWrap() {
				pwidth += 1
			}
			t.pwindow = t.tui.NewWindow(y+1, x+2, pwidth, h-2, tui.BorderNone)
			os.Setenv("FZF_PREVIEW_HEIGHT", strconv.Itoa(h-2))
		}
		switch t.preview.Position {
		case WindowPositionUp:
			pheight := calculateSize(height, t.preview.Size, minHeight, 3)
			t.window = t.tui.NewWindow(
				marginInt[0]+pheight, marginInt[3], width, height-pheight, tui.BorderNone)
			createPreviewWindow(marginInt[0], marginInt[3], width, pheight)
		case WindowPositionDown:
			pheight := calculateSize(height, t.preview.Size, minHeight, 3)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3], width, height-pheight, tui.BorderNone)
			createPreviewWindow(marginInt[0]+height-pheight, marginInt[3], width, pheight)
		case WindowPositionLeft:
			pwidth := calculateSize(width, t.preview.Size, minWidth, 5)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3]+pwidth, width-pwidth, height, tui.BorderNone)
			createPreviewWindow(marginInt[0], marginInt[3], pwidth, height)
		case WindowPositionRight:
			pwidth := calculateSize(width, t.preview.Size, minWidth, 5)
			t.window = t.tui.NewWindow(
				marginInt[0], marginInt[3], width-pwidth, height, tui.BorderNone)
			createPreviewWindow(marginInt[0], marginInt[3]+width-pwidth, pwidth, height)
		}
	} else {
		t.window = t.tui.NewWindow(
			marginInt[0],
			marginInt[3],
			width,
			height, tui.BorderNone)
	}
	if !t.tui.IsOptimized() {
		for i := 0; i < t.window.Height(); i++ {
			t.window.MoveAndClear(i, 0)
		}
	}
	t.truncateQuery()
}

func (t *Terminal) move(y int, x int, clear bool) {
	if !t.reverse {
		y = t.window.Height() - y - 1
	}

	if clear {
		t.window.MoveAndClear(y, x)
	} else {
		t.window.Move(y, x)
	}
}

func (t *Terminal) placeCursor() {
	t.move(0, t.promptLen+t.displayWidth(t.input[:t.cx]), false)
}

func (t *Terminal) printPrompt() {
	t.move(0, 0, true)
	t.window.CPrint(tui.ColPrompt, t.strong, t.prompt)
	t.window.CPrint(tui.ColNormal, t.strong, string(t.input))
}

func (t *Terminal) printInfo() {
	pos := 0
	if t.inlineInfo {
		pos = t.promptLen + t.displayWidth(t.input) + 1
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
	} else {
		t.move(1, 0, true)
		if t.reading {
			duration := int64(spinnerDuration)
			idx := (time.Now().UnixNano() % (duration * int64(len(_spinner)))) / duration
			t.window.CPrint(tui.ColSpinner, t.strong, _spinner[idx])
		}
		t.move(1, 2, false)
		pos = 2
	}

	output := fmt.Sprintf("%d/%d", t.merger.Length(), t.count)
	if t.toggleSort {
		if t.sort {
			output += " +S"
		} else {
			output += " -S"
		}
	}
	if t.multi && len(t.selected) > 0 {
		output += fmt.Sprintf(" (%d)", len(t.selected))
	}
	if t.progress > 0 && t.progress < 100 {
		output += fmt.Sprintf(" (%d%%)", t.progress)
	}
	if !t.success && t.count == 0 {
		output += " [ERROR]"
	}
	if pos+len(output) <= t.window.Width() {
		t.window.CPrint(tui.ColInfo, 0, output)
	}
}

func (t *Terminal) printHeader() {
	if len(t.header) == 0 {
		return
	}
	max := t.window.Height()
	var state *ansiState
	for idx, lineStr := range t.header {
		line := idx + 2
		if t.inlineInfo {
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
			tui.AttrRegular, tui.ColHeader, tui.ColDefault, false, false)
	}
}

func (t *Terminal) printList() {
	t.constrain()

	maxy := t.maxItems()
	count := t.merger.Length() - t.offset
	for j := 0; j < maxy; j++ {
		i := j
		if !t.reverse {
			i = maxy - 1 - j
		}
		line := i + 2 + len(t.header)
		if t.inlineInfo {
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
	label := " "
	if t.jumping != jumpDisabled {
		if i < len(t.jumpLabels) {
			// Striped
			current = i%2 == 0
			label = t.jumpLabels[i : i+1]
		}
	} else if current {
		label = ">"
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

	// Optimized renderer can simply erase to the end of the window
	t.move(line, 0, t.tui.IsOptimized())
	t.window.CPrint(tui.ColCursor, t.strong, label)
	if current {
		if selected {
			t.window.CPrint(tui.ColSelected, t.strong, ">")
		} else {
			t.window.CPrint(tui.ColCurrent, t.strong, " ")
		}
		newLine.width = t.printHighlighted(result, t.strong, tui.ColCurrent, tui.ColCurrentMatch, true, true)
	} else {
		if selected {
			t.window.CPrint(tui.ColSelected, t.strong, ">")
		} else {
			t.window.Print(" ")
		}
		newLine.width = t.printHighlighted(result, 0, tui.ColNormal, tui.ColMatch, false, true)
	}
	if !t.tui.IsOptimized() {
		fillSpaces := prevLine.width - newLine.width
		if fillSpaces > 0 {
			t.window.Print(strings.Repeat(" ", fillSpaces))
		}
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
		if t.hscroll {
			// Stri..
			if !t.overflow(text[:maxe], maxWidth-2) {
				text, _ = t.trimRight(text, maxWidth-2)
				text = append(text, []rune("..")...)
			} else {
				// Stri..
				if t.overflow(text[maxe:], 2) {
					text = append(text[:maxe], []rune("..")...)
				}
				// ..ri..
				var diff int32
				text, diff = t.trimLeft(text, maxWidth-2)

				// Transform offsets
				for idx, offset := range offsets {
					b, e := offset.offset[0], offset.offset[1]
					b += 2 - diff
					e += 2 - diff
					b = util.Max32(b, 2)
					offsets[idx].offset[0] = b
					offsets[idx].offset[1] = util.Max32(b, e)
				}
				text = append([]rune(".."), text...)
			}
		} else {
			text, _ = t.trimRight(text, maxWidth-2)
			text = append(text, []rune("..")...)

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

func numLinesMax(str string, max int) int {
	lines := 0
	for lines < max {
		idx := strings.Index(str, "\n")
		if idx < 0 {
			break
		}
		str = str[idx+1:]
		lines++
	}
	return lines
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
	var ansi *ansiState
	for {
		line, err := reader.ReadString('\n')
		eof := err == io.EOF
		if !eof {
			line = line[:len(line)-1]
		}
		lineNo++
		if lineNo > height ||
			t.pwindow.Y() == height-1 && t.pwindow.X() > 0 {
			break
		} else if lineNo > 0 {
			var fillRet tui.FillReturn
			_, _, ansi = extractColor(line, ansi, func(str string, ansi *ansiState) bool {
				trimmed := []rune(str)
				if !t.preview.Wrap {
					trimmed, _ = t.trimRight(trimmed, maxWidth-t.pwindow.X())
				}
				str, _ = t.processTabs(trimmed, 0)
				if t.theme != nil && ansi != nil && ansi.colored() {
					fillRet = t.pwindow.CFill(ansi.fg, ansi.bg, ansi.attr, str)
				} else {
					fillRet = t.pwindow.Fill(str)
				}
				return fillRet == tui.FillContinue
			})
			switch fillRet {
			case tui.FillNextLine:
				continue
			case tui.FillSuspend:
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
	if !t.suppress {
		windows := make([]tui.Window, 0, 4)
		if t.bordered {
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

func quoteEntry(entry string) string {
	if util.IsWindows() {
		return strconv.Quote(strings.Replace(entry, "\"", "\\\"", -1))
	}
	return "'" + strings.Replace(entry, "'", "'\\''", -1) + "'"
}

func (t *Terminal) redraw() {
	t.tui.Clear()
	t.tui.Refresh()
	t.printAll()
}

func (t *Terminal) executeCommand(command Command, forcePlus bool, background bool) {
	valid, list := t.buildPlusList(command, forcePlus)
	if !valid {
		return
	}
	if !background {
		t.tui.Pause(true)
		command.Execute(true, t.ansi, t.delimiter, forcePlus, string(t.input), list)
		t.tui.Resume(true)
		t.redraw()
		t.refresh()
	} else {
		command.Execute(false, t.ansi, t.delimiter, forcePlus, string(t.input), list)
	}
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
	if cnt > 0 && cnt > t.cy {
		return t.merger.Get(t.cy).item
	}
	return nil
}

func (t *Terminal) buildPlusList(command Command, forcePlus bool) (bool, []*Item) {
	current := t.currentItem()
	if !forcePlus && !command.HasPlusFlag() || len(t.selected) == 0 {
		return current != nil, []*Item{current, current}
	}
	sels := make([]*Item, len(t.selected)+1)
	sels[0] = current
	for i, sel := range t.sortSelected() {
		sels[i+1] = sel.item
	}
	return true, sels
}

func (t *Terminal) truncateQuery() {
	maxPatternLength := util.Max(1, t.window.Width()-t.promptLen-1)
	t.input, _ = t.trimRight(t.input, maxPatternLength)
	t.cx = util.Constrain(t.cx, 0, len(t.input))
}

func (t *Terminal) selectItem(item *Item) {
	t.selected[item.Index()] = selectedItem{time.Now(), item}
	t.version++
}

func (t *Terminal) deselectItem(item *Item) {
	delete(t.selected, item.Index())
	t.version++
}

func (t *Terminal) toggleItem(item *Item) {
	if _, found := t.selected[item.Index()]; !found {
		t.selectItem(item)
	} else {
		t.deselectItem(item)
	}
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
	// prof := profile.Start(profile.ProfilePath("/tmp/"))
	<-t.startChan
	{ // Late initialization
		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, os.Kill, syscall.SIGTERM)
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
		t.placeCursor()
		t.refresh()
		t.printInfo()
		t.printHeader()
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
				if !reading {
					break
				}
				time.Sleep(spinnerDuration)
				t.reqBox.Set(reqInfo, nil)
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
					out := t.preview.Command.GetPreview(t.ansi, t.delimiter, string(t.input), request)
					t.reqBox.Set(reqPreviewDisplay, string(out))
				} else {
					t.reqBox.Set(reqPreviewDisplay, "")
				}
			}
		}()
	}

	exit := func(getCode func() int) {
		if !t.cleanExit && t.fullscreen && t.inlineInfo {
			t.placeCursor()
		}
		t.tui.Close()
		code := getCode()
		if code <= exitNoMatch && t.history != nil {
			t.history.append(string(t.input))
		}
		// prof.Stop()
		os.Exit(code)
	}

	go func() {
		var focused *Item
		var version int64
		for {
			t.reqBox.Wait(func(events *util.Events) {
				defer events.Clear()
				t.mutex.Lock()
				for req, value := range *events {
					switch req {
					case reqPrompt:
						t.printPrompt()
						if t.inlineInfo {
							t.printInfo()
						}
					case reqInfo:
						t.printInfo()
					case reqList:
						t.printList()
						currentFocus := t.currentItem()
						if currentFocus != focused || version != t.version {
							version = t.version
							focused = currentFocus
							if t.isPreviewEnabled() {
								_, list := t.buildPlusList(t.preview.Command, false)
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
				t.placeCursor()
				t.mutex.Unlock()
			})
			t.refresh()
		}
	}()

	looping := true
	for looping {
		event := t.tui.GetChar()

		t.mutex.Lock()
		previousInput := t.input
		events := []util.EventType{reqPrompt}
		req := func(evts ...util.EventType) {
			for _, event := range evts {
				events = append(events, event)
				if event == reqClose || event == reqQuit {
					looping = false
				}
			}
		}
		toggle := func() {
			if t.cy < t.merger.Length() {
				t.toggleItem(t.merger.Get(t.cy).item)
				req(reqInfo)
			}
		}
		scrollPreview := func(amount int) {
			t.previewer.offset = util.Constrain(
				t.previewer.offset+amount, 0, t.previewer.lines-1)
			req(reqPreviewRefresh)
		}
		for key, ret := range t.expect {
			if keyMatch(key, event) {
				t.pressed = ret
				t.reqBox.Set(reqClose, nil)
				t.mutex.Unlock()
				return
			}
		}

		var doAction func(Action, int) bool
		doActions := func(actions []Action, mapkey int) bool {
			for _, action := range actions {
				if !doAction(action, mapkey) {
					return false
				}
			}
			return true
		}
		doAction = func(a Action, mapkey int) bool {
			switch a.Type {
			case ActionTypeIgnore:
			case ActionTypeExecute, ActionTypeExecuteSilent:
				t.executeCommand(a.Command, false, a.Type == ActionTypeExecuteSilent)
			case ActionTypeExecuteMulti:
				t.executeCommand(a.Command, true, false)
			case ActionTypeInvalid:
				t.mutex.Unlock()
				return false
			case ActionTypeTogglePreview:
				if t.hasPreviewer() {
					t.previewer.enabled = !t.previewer.enabled
					t.tui.Clear()
					t.resizeWindows()
					if t.previewer.enabled {
						valid, list := t.buildPlusList(t.preview.Command, false)
						if valid {
							t.previewBox.Set(reqPreviewEnqueue, list)
						}
					}
					req(reqList, reqInfo, reqHeader)
				}
			case ActionTypeTogglePreviewWrap:
				if t.hasPreviewWindow() {
					t.preview.Wrap = !t.preview.Wrap
					req(reqPreviewRefresh)
				}
			case ActionTypeToggleSort:
				t.sort = !t.sort
				t.eventBox.Set(EvtSearchNew, t.sort)
				t.mutex.Unlock()
				return false
			case ActionTypePreviewUp:
				if t.hasPreviewWindow() {
					scrollPreview(-1)
				}
			case ActionTypePreviewDown:
				if t.hasPreviewWindow() {
					scrollPreview(1)
				}
			case ActionTypePreviewPageUp:
				if t.hasPreviewWindow() {
					scrollPreview(-t.pwindow.Height())
				}
			case ActionTypePreviewPageDown:
				if t.hasPreviewWindow() {
					scrollPreview(t.pwindow.Height())
				}
			case ActionTypeBeginningOfLine:
				t.cx = 0
			case ActionTypeBackwardChar:
				if t.cx > 0 {
					t.cx--
				}
			case ActionTypePrintQuery:
				req(reqPrintQuery)
			case ActionTypeAbort:
				req(reqQuit)
			case ActionTypeDeleteChar:
				t.delChar()
			case ActionTypeDeleteCharEOF:
				if !t.delChar() && t.cx == 0 {
					req(reqQuit)
				}
			case ActionTypeEndOfLine:
				t.cx = len(t.input)
			case ActionTypeCancel:
				if len(t.input) == 0 {
					req(reqQuit)
				} else {
					t.yanked = t.input
					t.input = []rune{}
					t.cx = 0
				}
			case ActionTypeForwardChar:
				if t.cx < len(t.input) {
					t.cx++
				}
			case ActionTypeBackwardDeleteChar:
				if t.cx > 0 {
					t.input = append(t.input[:t.cx-1], t.input[t.cx:]...)
					t.cx--
				}
			case ActionTypeSelectAll:
				if t.multi {
					for i := 0; i < t.merger.Length(); i++ {
						t.selectItem(t.merger.Get(i).item)
					}
					req(reqList, reqInfo)
				}
			case ActionTypeDeselectAll:
				if t.multi {
					t.selected = make(map[int32]selectedItem)
					t.version++
					req(reqList, reqInfo)
				}
			case ActionTypeToggle:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					req(reqList)
				}
			case ActionTypeToggleAll:
				if t.multi {
					for i := 0; i < t.merger.Length(); i++ {
						t.toggleItem(t.merger.Get(i).item)
					}
					req(reqList, reqInfo)
				}
			case ActionTypeToggleIn:
				if t.reverse {
					return doAction(Action{Type: ActionTypeToggleUp}, mapkey)
				}
				return doAction(Action{Type: ActionTypeToggleDown}, mapkey)
			case ActionTypeToggleOut:
				if t.reverse {
					return doAction(Action{Type: ActionTypeToggleDown}, mapkey)
				}
				return doAction(Action{Type: ActionTypeToggleUp}, mapkey)
			case ActionTypeToggleDown:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					t.vmove(-1, true)
					req(reqList)
				}
			case ActionTypeToggleUp:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					t.vmove(1, true)
					req(reqList)
				}
			case ActionTypeDown:
				t.vmove(-1, true)
				req(reqList)
			case ActionTypeUp:
				t.vmove(1, true)
				req(reqList)
			case ActionTypeAccept:
				req(reqClose)
			case ActionTypeClearScreen:
				req(reqRedraw)
			case ActionTypeTop:
				t.vset(0)
				req(reqList)
			case ActionTypeUnixLineDiscard:
				if t.cx > 0 {
					t.yanked = copySlice(t.input[:t.cx])
					t.input = t.input[t.cx:]
					t.cx = 0
				}
			case ActionTypeUnixWordRubout:
				if t.cx > 0 {
					t.rubout("\\s\\S")
				}
			case ActionTypeBackwardKillWord:
				if t.cx > 0 {
					t.rubout(t.wordRubout)
				}
			case ActionTypeYank:
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], t.yanked...), suffix...)
				t.cx += len(t.yanked)
			case ActionTypePageUp:
				t.vmove(t.maxItems()-1, false)
				req(reqList)
			case ActionTypePageDown:
				t.vmove(-(t.maxItems() - 1), false)
				req(reqList)
			case ActionTypeHalfPageUp:
				t.vmove(t.maxItems()/2, false)
				req(reqList)
			case ActionTypeHalfPageDown:
				t.vmove(-(t.maxItems() / 2), false)
				req(reqList)
			case ActionTypeJump:
				t.jumping = jumpEnabled
				req(reqJump)
			case ActionTypeJumpAccept:
				t.jumping = jumpAcceptEnabled
				req(reqJump)
			case ActionTypeBackwardWord:
				t.cx = findLastMatch(t.wordRubout, string(t.input[:t.cx])) + 1
			case ActionTypeForwardWord:
				t.cx += findFirstMatch(t.wordNext, string(t.input[t.cx:])) + 1
			case ActionTypeKillWord:
				ncx := t.cx +
					findFirstMatch(t.wordNext, string(t.input[t.cx:])) + 1
				if ncx > t.cx {
					t.yanked = copySlice(t.input[t.cx:ncx])
					t.input = append(t.input[:t.cx], t.input[ncx:]...)
				}
			case ActionTypeKillLine:
				if t.cx < len(t.input) {
					t.yanked = copySlice(t.input[t.cx:])
					t.input = t.input[:t.cx]
				}
			case ActionTypeRune:
				prefix := copySlice(t.input[:t.cx])
				t.input = append(append(prefix, event.Char), t.input[t.cx:]...)
				t.cx++
			case ActionTypePreviousHistory:
				if t.history != nil {
					t.history.override(string(t.input))
					t.input = trimQuery(t.history.previous())
					t.cx = len(t.input)
				}
			case ActionTypeNextHistory:
				if t.history != nil {
					t.history.override(string(t.input))
					t.input = trimQuery(t.history.next())
					t.cx = len(t.input)
				}
			case ActionTypeSigStop:
				p, err := os.FindProcess(os.Getpid())
				if err == nil {
					t.tui.Clear()
					t.tui.Pause(t.fullscreen)
					notifyStop(p)
					t.mutex.Unlock()
					return false
				}
			case ActionTypeMouse:
				me := event.MouseEvent
				mx, my := me.X, me.Y
				if me.S != 0 {
					// Scroll
					if t.window.Enclose(my, mx) && t.merger.Length() > 0 {
						if t.multi && me.Mod {
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
					if !t.reverse {
						my = t.window.Height() - my - 1
					}
					min := 2 + len(t.header)
					if t.inlineInfo {
						min--
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
							t.cx = mx
						} else if my >= min {
							// List
							if t.vset(t.offset+my-min) && t.multi && me.Mod {
								toggle()
							}
							req(reqList)
						}
					}
				}
			}
			return true
		}
		changed := false
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
			changed = string(previousInput) != string(t.input)
			if onChanges, prs := t.keymap[tui.Change]; changed && prs {
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
		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if changed {
			t.eventBox.Set(EvtSearchNew, t.sort)
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
	if t.reverse {
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
	if t.inlineInfo {
		max++
	}
	return util.Max(max, 0)
}
