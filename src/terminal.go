package fzf

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	C "github.com/junegunn/fzf/src/curses"
	"github.com/junegunn/fzf/src/util"

	"github.com/junegunn/go-runewidth"
)

type jumpMode int

const (
	jumpDisabled jumpMode = iota
	jumpEnabled
	jumpAcceptEnabled
)

// Terminal represents terminal input/output
type Terminal struct {
	initDelay  time.Duration
	inlineInfo bool
	prompt     string
	reverse    bool
	hscroll    bool
	hscrollOff int
	cx         int
	cy         int
	offset     int
	yanked     []rune
	input      []rune
	multi      bool
	sort       bool
	toggleSort bool
	expect     map[int]string
	keymap     map[int]actionType
	execmap    map[int]string
	pressed    string
	printQuery bool
	history    *History
	cycle      bool
	header     []string
	header0    []string
	ansi       bool
	margin     [4]string
	marginInt  [4]int
	count      int
	progress   int
	reading    bool
	jumping    jumpMode
	jumpLabels string
	merger     *Merger
	selected   map[int32]selectedItem
	reqBox     *util.EventBox
	eventBox   *util.EventBox
	mutex      sync.Mutex
	initFunc   func()
	suppress   bool
	startChan  chan bool
}

type selectedItem struct {
	at   time.Time
	text *string
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
var _runeWidths = make(map[rune]int)
var _tabStop int

const (
	reqPrompt util.EventType = iota
	reqInfo
	reqHeader
	reqList
	reqJump
	reqRefresh
	reqRedraw
	reqClose
	reqPrintQuery
	reqQuit
)

type actionType int

const (
	actIgnore actionType = iota
	actInvalid
	actRune
	actMouse
	actBeginningOfLine
	actAbort
	actAccept
	actBackwardChar
	actBackwardDeleteChar
	actBackwardWord
	actCancel
	actClearScreen
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
	actJump
	actJumpAccept
	actPrintQuery
	actToggleSort
	actPreviousHistory
	actNextHistory
	actExecute
	actExecuteMulti
)

func defaultKeymap() map[int]actionType {
	keymap := make(map[int]actionType)
	keymap[C.Invalid] = actInvalid
	keymap[C.CtrlA] = actBeginningOfLine
	keymap[C.CtrlB] = actBackwardChar
	keymap[C.CtrlC] = actAbort
	keymap[C.CtrlG] = actAbort
	keymap[C.CtrlQ] = actAbort
	keymap[C.ESC] = actAbort
	keymap[C.CtrlD] = actDeleteCharEOF
	keymap[C.CtrlE] = actEndOfLine
	keymap[C.CtrlF] = actForwardChar
	keymap[C.CtrlH] = actBackwardDeleteChar
	keymap[C.BSpace] = actBackwardDeleteChar
	keymap[C.Tab] = actToggleDown
	keymap[C.BTab] = actToggleUp
	keymap[C.CtrlJ] = actDown
	keymap[C.CtrlK] = actUp
	keymap[C.CtrlL] = actClearScreen
	keymap[C.CtrlM] = actAccept
	keymap[C.CtrlN] = actDown
	keymap[C.CtrlP] = actUp
	keymap[C.CtrlU] = actUnixLineDiscard
	keymap[C.CtrlW] = actUnixWordRubout
	keymap[C.CtrlY] = actYank

	keymap[C.AltB] = actBackwardWord
	keymap[C.SLeft] = actBackwardWord
	keymap[C.AltF] = actForwardWord
	keymap[C.SRight] = actForwardWord
	keymap[C.AltD] = actKillWord
	keymap[C.AltBS] = actBackwardKillWord

	keymap[C.Up] = actUp
	keymap[C.Down] = actDown
	keymap[C.Left] = actBackwardChar
	keymap[C.Right] = actForwardChar

	keymap[C.Home] = actBeginningOfLine
	keymap[C.End] = actEndOfLine
	keymap[C.Del] = actDeleteChar
	keymap[C.PgUp] = actPageUp
	keymap[C.PgDn] = actPageDown

	keymap[C.Rune] = actRune
	keymap[C.Mouse] = actMouse
	keymap[C.DoubleClick] = actAccept
	return keymap
}

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := []rune(opts.Query)
	var header []string
	if opts.Reverse {
		header = opts.Header
	} else {
		header = reverseStringArray(opts.Header)
	}
	_tabStop = opts.Tabstop
	var delay time.Duration
	if opts.Tac {
		delay = initialDelayTac
	} else {
		delay = initialDelay
	}
	return &Terminal{
		initDelay:  delay,
		inlineInfo: opts.InlineInfo,
		prompt:     opts.Prompt,
		reverse:    opts.Reverse,
		hscroll:    opts.Hscroll,
		hscrollOff: opts.HscrollOff,
		cx:         len(input),
		cy:         0,
		offset:     0,
		yanked:     []rune{},
		input:      input,
		multi:      opts.Multi,
		sort:       opts.Sort > 0,
		toggleSort: opts.ToggleSort,
		expect:     opts.Expect,
		keymap:     opts.Keymap,
		execmap:    opts.Execmap,
		pressed:    "",
		printQuery: opts.PrintQuery,
		history:    opts.History,
		margin:     opts.Margin,
		marginInt:  [4]int{0, 0, 0, 0},
		cycle:      opts.Cycle,
		header:     header,
		header0:    header,
		ansi:       opts.Ansi,
		reading:    true,
		jumping:    jumpDisabled,
		jumpLabels: opts.JumpLabels,
		merger:     EmptyMerger,
		selected:   make(map[int32]selectedItem),
		reqBox:     util.NewEventBox(),
		eventBox:   eventBox,
		mutex:      sync.Mutex{},
		suppress:   true,
		startChan:  make(chan bool, 1),
		initFunc: func() {
			C.Init(opts.Theme, opts.Black, opts.Mouse)
		}}
}

// Input returns current query string
func (t *Terminal) Input() []rune {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return copySlice(t.input)
}

// UpdateCount updates the count information
func (t *Terminal) UpdateCount(cnt int, final bool) {
	t.mutex.Lock()
	t.count = cnt
	t.reading = !final
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
		fmt.Println(string(t.input))
	}
	if len(t.expect) > 0 {
		fmt.Println(t.pressed)
	}
	found := len(t.selected) > 0
	if !found {
		cnt := t.merger.Length()
		if cnt > 0 && cnt > t.cy {
			fmt.Println(t.merger.Get(t.cy).AsString(t.ansi))
			found = true
		}
	} else {
		for _, sel := range t.sortSelected() {
			fmt.Println(*sel.text)
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

func runeWidth(r rune, prefixWidth int) int {
	if r == '\t' {
		return _tabStop - prefixWidth%_tabStop
	} else if w, found := _runeWidths[r]; found {
		return w
	} else {
		w := runewidth.RuneWidth(r)
		_runeWidths[r] = w
		return w
	}
}

func displayWidth(runes []rune) int {
	l := 0
	for _, r := range runes {
		l += runeWidth(r, l)
	}
	return l
}

const minWidth = 16
const minHeight = 4

func (t *Terminal) calculateMargins() {
	screenWidth := C.MaxX()
	screenHeight := C.MaxY()
	for idx, str := range t.margin {
		if str == "0" {
			t.marginInt[idx] = 0
		} else if strings.HasSuffix(str, "%") {
			num, _ := strconv.ParseFloat(str[:len(str)-1], 64)
			var val float64
			if idx%2 == 0 {
				val = float64(screenHeight)
			} else {
				val = float64(screenWidth)
			}
			t.marginInt[idx] = int(val * num * 0.01)
		} else {
			num, _ := strconv.Atoi(str)
			t.marginInt[idx] = num
		}
	}
	adjust := func(idx1 int, idx2 int, max int, min int) {
		if max >= min {
			margin := t.marginInt[idx1] + t.marginInt[idx2]
			if max-margin < min {
				desired := max - min
				t.marginInt[idx1] = desired * t.marginInt[idx1] / margin
				t.marginInt[idx2] = desired * t.marginInt[idx2] / margin
			}
		}
	}
	adjust(1, 3, screenWidth, minWidth)
	adjust(0, 2, screenHeight, minHeight)
}

func (t *Terminal) move(y int, x int, clear bool) {
	x += t.marginInt[3]
	maxy := C.MaxY()
	if !t.reverse {
		y = maxy - y - 1 - t.marginInt[2]
	} else {
		y += t.marginInt[0]
	}

	if clear {
		C.MoveAndClear(y, x)
	} else {
		C.Move(y, x)
	}
}

func (t *Terminal) placeCursor() {
	t.move(0, displayWidth([]rune(t.prompt))+displayWidth(t.input[:t.cx]), false)
}

func (t *Terminal) printPrompt() {
	t.move(0, 0, true)
	C.CPrint(C.ColPrompt, true, t.prompt)
	C.CPrint(C.ColNormal, true, string(t.input))
}

func (t *Terminal) printInfo() {
	if t.inlineInfo {
		t.move(0, displayWidth([]rune(t.prompt))+displayWidth(t.input)+1, true)
		if t.reading {
			C.CPrint(C.ColSpinner, true, " < ")
		} else {
			C.CPrint(C.ColPrompt, true, " < ")
		}
	} else {
		t.move(1, 0, true)
		if t.reading {
			duration := int64(spinnerDuration)
			idx := (time.Now().UnixNano() % (duration * int64(len(_spinner)))) / duration
			C.CPrint(C.ColSpinner, true, _spinner[idx])
		}
		t.move(1, 2, false)
	}

	output := fmt.Sprintf("%d/%d", t.merger.Length(), t.count)
	if t.toggleSort {
		if t.sort {
			output += "/S"
		} else {
			output += "  "
		}
	}
	if t.multi && len(t.selected) > 0 {
		output += fmt.Sprintf(" (%d)", len(t.selected))
	}
	if t.progress > 0 && t.progress < 100 {
		output += fmt.Sprintf(" (%d%%)", t.progress)
	}
	C.CPrint(C.ColInfo, false, output)
}

func (t *Terminal) maxHeight() int {
	return C.MaxY() - t.marginInt[0] - t.marginInt[2]
}

func (t *Terminal) printHeader() {
	if len(t.header) == 0 {
		return
	}
	max := t.maxHeight()
	var state *ansiState
	for idx, lineStr := range t.header {
		line := idx + 2
		if t.inlineInfo {
			line--
		}
		if line >= max {
			continue
		}
		trimmed, colors, newState := extractColor(lineStr, state)
		state = newState
		item := &Item{
			text:   []rune(trimmed),
			colors: colors,
			rank:   buildEmptyRank(0)}

		t.move(line, 2, true)
		t.printHighlighted(item, false, C.ColHeader, 0, false)
	}
}

func (t *Terminal) printList() {
	t.constrain()

	maxy := t.maxItems()
	count := t.merger.Length() - t.offset
	for i := 0; i < maxy; i++ {
		line := i + 2 + len(t.header)
		if t.inlineInfo {
			line--
		}
		t.move(line, 0, true)
		if i < count {
			t.printItem(t.merger.Get(i+t.offset), i, i == t.cy-t.offset)
		}
	}
}

func (t *Terminal) printItem(item *Item, i int, current bool) {
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
	C.CPrint(C.ColCursor, true, label)
	if current {
		if selected {
			C.CPrint(C.ColSelected, true, ">")
		} else {
			C.CPrint(C.ColCurrent, true, " ")
		}
		t.printHighlighted(item, true, C.ColCurrent, C.ColCurrentMatch, true)
	} else {
		if selected {
			C.CPrint(C.ColSelected, true, ">")
		} else {
			C.Print(" ")
		}
		t.printHighlighted(item, false, 0, C.ColMatch, false)
	}
}

func trimRight(runes []rune, width int) ([]rune, int) {
	// We start from the beginning to handle tab characters
	l := 0
	for idx, r := range runes {
		l += runeWidth(r, l)
		if idx > 0 && l > width {
			return runes[:idx], len(runes) - idx
		}
	}
	return runes, 0
}

func displayWidthWithLimit(runes []rune, prefixWidth int, limit int) int {
	l := 0
	for _, r := range runes {
		l += runeWidth(r, l+prefixWidth)
		if l > limit {
			// Early exit
			return l
		}
	}
	return l
}

func trimLeft(runes []rune, width int) ([]rune, int32) {
	currentWidth := displayWidth(runes)
	var trimmed int32

	for currentWidth > width && len(runes) > 0 {
		runes = runes[1:]
		trimmed++
		currentWidth = displayWidthWithLimit(runes, 2, width)
	}
	return runes, trimmed
}

func (t *Terminal) printHighlighted(item *Item, bold bool, col1 int, col2 int, current bool) {
	var maxe int
	for _, offset := range item.offsets {
		maxe = util.Max(maxe, int(offset[1]))
	}

	// Overflow
	text := make([]rune, len(item.text))
	copy(text, item.text)
	offsets := item.colorOffsets(col2, bold, current)
	maxWidth := C.MaxX() - 3 - t.marginInt[1] - t.marginInt[3]
	maxe = util.Constrain(maxe+util.Min(maxWidth/2-2, t.hscrollOff), 0, len(text))
	fullWidth := displayWidth(text)
	if fullWidth > maxWidth {
		if t.hscroll {
			// Stri..
			matchEndWidth := displayWidth(text[:maxe])
			if matchEndWidth <= maxWidth-2 {
				text, _ = trimRight(text, maxWidth-2)
				text = append(text, []rune("..")...)
			} else {
				// Stri..
				if matchEndWidth < fullWidth-2 {
					text = append(text[:maxe], []rune("..")...)
				}
				// ..ri..
				var diff int32
				text, diff = trimLeft(text, maxWidth-2)

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
			text, _ = trimRight(text, maxWidth-2)
			text = append(text, []rune("..")...)

			for idx, offset := range offsets {
				offsets[idx].offset[0] = util.Min32(offset.offset[0], int32(maxWidth-2))
				offsets[idx].offset[1] = util.Min32(offset.offset[1], int32(maxWidth))
			}
		}
	}

	var index int32
	var substr string
	var prefixWidth int
	maxOffset := int32(len(text))
	for _, offset := range offsets {
		b := util.Constrain32(offset.offset[0], index, maxOffset)
		e := util.Constrain32(offset.offset[1], index, maxOffset)

		substr, prefixWidth = processTabs(text[index:b], prefixWidth)
		C.CPrint(col1, bold, substr)

		if b < e {
			substr, prefixWidth = processTabs(text[b:e], prefixWidth)
			C.CPrint(offset.color, offset.bold, substr)
		}

		index = e
		if index >= maxOffset {
			break
		}
	}
	if index < maxOffset {
		substr, _ = processTabs(text[index:], prefixWidth)
		C.CPrint(col1, bold, substr)
	}
}

func processTabs(runes []rune, prefixWidth int) (string, int) {
	var strbuf bytes.Buffer
	l := prefixWidth
	for _, r := range runes {
		w := runeWidth(r, l)
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
	t.calculateMargins()
	t.printList()
	t.printPrompt()
	t.printInfo()
	t.printHeader()
}

func (t *Terminal) refresh() {
	if !t.suppress {
		C.Refresh()
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

func keyMatch(key int, event C.Event) bool {
	return event.Type == key ||
		event.Type == C.Rune && int(event.Char) == key-C.AltZ ||
		event.Type == C.Mouse && key == C.DoubleClick && event.MouseEvent.Double
}

func quoteEntry(entry string) string {
	return fmt.Sprintf("%q", entry)
}

func executeCommand(template string, replacement string) {
	command := strings.Replace(template, "{}", replacement, -1)
	cmd := util.ExecCommand(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	C.Endwin()
	cmd.Run()
	C.Refresh()
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
	<-t.startChan
	{ // Late initialization
		intChan := make(chan os.Signal, 1)
		signal.Notify(intChan, os.Interrupt, os.Kill, syscall.SIGTERM)
		go func() {
			<-intChan
			t.reqBox.Set(reqQuit, nil)
		}()

		resizeChan := make(chan os.Signal, 1)
		signal.Notify(resizeChan, syscall.SIGWINCH)
		go func() {
			for {
				<-resizeChan
				t.reqBox.Set(reqRedraw, nil)
			}
		}()

		t.mutex.Lock()
		t.initFunc()
		t.calculateMargins()
		t.printPrompt()
		t.placeCursor()
		C.Refresh()
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

	exit := func(code int) {
		if code <= exitNoMatch && t.history != nil {
			t.history.append(string(t.input))
		}
		os.Exit(code)
	}

	go func() {
		for {
			t.reqBox.Wait(func(events *util.Events) {
				defer events.Clear()
				t.mutex.Lock()
				for req := range *events {
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
					case reqJump:
						if t.merger.Length() == 0 {
							t.jumping = jumpDisabled
						}
						t.printList()
					case reqHeader:
						t.printHeader()
					case reqRefresh:
						t.suppress = false
					case reqRedraw:
						C.Clear()
						C.Endwin()
						C.Refresh()
						t.printAll()
					case reqClose:
						C.Close()
						if t.output() {
							exit(exitOk)
						}
						exit(exitNoMatch)
					case reqPrintQuery:
						C.Close()
						fmt.Println(string(t.input))
						exit(exitOk)
					case reqQuit:
						C.Close()
						exit(exitInterrupt)
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
		event := C.GetChar()

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
		selectItem := func(item *Item) bool {
			if _, found := t.selected[item.Index()]; !found {
				t.selected[item.Index()] = selectedItem{time.Now(), item.StringPtr(t.ansi)}
				return true
			}
			return false
		}
		toggleY := func(y int) {
			item := t.merger.Get(y)
			if !selectItem(item) {
				delete(t.selected, item.Index())
			}
		}
		toggle := func() {
			if t.cy < t.merger.Length() {
				toggleY(t.cy)
				req(reqInfo)
			}
		}
		for key, ret := range t.expect {
			if keyMatch(key, event) {
				t.pressed = ret
				req(reqClose)
				break
			}
		}

		var doAction func(actionType, int) bool
		doAction = func(action actionType, mapkey int) bool {
			switch action {
			case actIgnore:
			case actExecute:
				if t.cy >= 0 && t.cy < t.merger.Length() {
					item := t.merger.Get(t.cy)
					executeCommand(t.execmap[mapkey], quoteEntry(item.AsString(t.ansi)))
				}
			case actExecuteMulti:
				if len(t.selected) > 0 {
					sels := make([]string, len(t.selected))
					for i, sel := range t.sortSelected() {
						sels[i] = quoteEntry(*sel.text)
					}
					executeCommand(t.execmap[mapkey], strings.Join(sels, " "))
				} else {
					return doAction(actExecute, mapkey)
				}
			case actInvalid:
				t.mutex.Unlock()
				return false
			case actToggleSort:
				t.sort = !t.sort
				t.eventBox.Set(EvtSearchNew, t.sort)
				t.mutex.Unlock()
				return false
			case actBeginningOfLine:
				t.cx = 0
			case actBackwardChar:
				if t.cx > 0 {
					t.cx--
				}
			case actPrintQuery:
				req(reqPrintQuery)
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
				if t.multi {
					for i := 0; i < t.merger.Length(); i++ {
						item := t.merger.Get(i)
						selectItem(item)
					}
					req(reqList, reqInfo)
				}
			case actDeselectAll:
				if t.multi {
					for i := 0; i < t.merger.Length(); i++ {
						item := t.merger.Get(i)
						delete(t.selected, item.Index())
					}
					req(reqList, reqInfo)
				}
			case actToggle:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					req(reqList)
				}
			case actToggleAll:
				if t.multi {
					for i := 0; i < t.merger.Length(); i++ {
						toggleY(i)
					}
					req(reqList, reqInfo)
				}
			case actToggleIn:
				if t.reverse {
					return doAction(actToggleUp, mapkey)
				}
				return doAction(actToggleDown, mapkey)
			case actToggleOut:
				if t.reverse {
					return doAction(actToggleDown, mapkey)
				}
				return doAction(actToggleUp, mapkey)
			case actToggleDown:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					t.vmove(-1)
					req(reqList)
				}
			case actToggleUp:
				if t.multi && t.merger.Length() > 0 {
					toggle()
					t.vmove(1)
					req(reqList)
				}
			case actDown:
				t.vmove(-1)
				req(reqList)
			case actUp:
				t.vmove(1)
				req(reqList)
			case actAccept:
				req(reqClose)
			case actClearScreen:
				req(reqRedraw)
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
					t.rubout("[^[:alnum:]][[:alnum:]]")
				}
			case actYank:
				suffix := copySlice(t.input[t.cx:])
				t.input = append(append(t.input[:t.cx], t.yanked...), suffix...)
				t.cx += len(t.yanked)
			case actPageUp:
				t.vmove(t.maxItems() - 1)
				req(reqList)
			case actPageDown:
				t.vmove(-(t.maxItems() - 1))
				req(reqList)
			case actJump:
				t.jumping = jumpEnabled
				req(reqJump)
			case actJumpAccept:
				t.jumping = jumpAcceptEnabled
				req(reqJump)
			case actBackwardWord:
				t.cx = findLastMatch("[^[:alnum:]][[:alnum:]]", string(t.input[:t.cx])) + 1
			case actForwardWord:
				t.cx += findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
			case actKillWord:
				ncx := t.cx +
					findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
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
					t.input = []rune(t.history.previous())
					t.cx = len(t.input)
				}
			case actNextHistory:
				if t.history != nil {
					t.history.override(string(t.input))
					t.input = []rune(t.history.next())
					t.cx = len(t.input)
				}
			case actMouse:
				me := event.MouseEvent
				mx, my := me.X, me.Y
				if me.S != 0 {
					// Scroll
					if t.merger.Length() > 0 {
						if t.multi && me.Mod {
							toggle()
						}
						t.vmove(me.S)
						req(reqList)
					}
				} else if mx >= t.marginInt[3] && mx < C.MaxX()-t.marginInt[1] &&
					my >= t.marginInt[0] && my < C.MaxY()-t.marginInt[2] {
					mx -= t.marginInt[3]
					my -= t.marginInt[0]
					mx = util.Constrain(mx-displayWidth([]rune(t.prompt)), 0, len(t.input))
					if !t.reverse {
						my = t.maxHeight() - my - 1
					}
					min := 2 + len(t.header)
					if t.inlineInfo {
						min--
					}
					if me.Double {
						// Double-click
						if my >= min {
							if t.vset(t.offset+my-min) && t.cy < t.merger.Length() {
								return doAction(t.keymap[C.DoubleClick], C.DoubleClick)
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
			action := t.keymap[mapkey]
			if mapkey == C.Rune {
				mapkey = int(event.Char) + int(C.AltZ)
				if act, prs := t.keymap[mapkey]; prs {
					action = act
				}
			}
			if !doAction(action, mapkey) {
				continue
			}
			changed = string(previousInput) != string(t.input)
		} else {
			if mapkey == C.Rune {
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

func (t *Terminal) vmove(o int) {
	if t.reverse {
		o *= -1
	}
	dest := t.cy + o
	if t.cycle {
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
	max := t.maxHeight() - 2 - len(t.header)
	if t.inlineInfo {
		max++
	}
	return util.Max(max, 0)
}
