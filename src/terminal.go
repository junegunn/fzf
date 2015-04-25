package fzf

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	C "github.com/junegunn/fzf/src/curses"
	"github.com/junegunn/fzf/src/util"

	"github.com/junegunn/go-runewidth"
)

// Terminal represents terminal input/output
type Terminal struct {
	inlineInfo bool
	prompt     string
	reverse    bool
	hscroll    bool
	cx         int
	cy         int
	offset     int
	yanked     []rune
	input      []rune
	multi      bool
	sort       bool
	toggleSort int
	expect     []int
	pressed    int
	printQuery bool
	count      int
	progress   int
	reading    bool
	merger     *Merger
	selected   map[uint32]selectedItem
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

const (
	reqPrompt util.EventType = iota
	reqInfo
	reqList
	reqRefresh
	reqRedraw
	reqClose
	reqQuit
)

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := []rune(opts.Query)
	return &Terminal{
		inlineInfo: opts.InlineInfo,
		prompt:     opts.Prompt,
		reverse:    opts.Reverse,
		hscroll:    opts.Hscroll,
		cx:         len(input),
		cy:         0,
		offset:     0,
		yanked:     []rune{},
		input:      input,
		multi:      opts.Multi,
		sort:       opts.Sort > 0,
		toggleSort: opts.ToggleSort,
		expect:     opts.Expect,
		pressed:    0,
		printQuery: opts.PrintQuery,
		merger:     EmptyMerger,
		selected:   make(map[uint32]selectedItem),
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

func (t *Terminal) output() {
	if t.printQuery {
		fmt.Println(string(t.input))
	}
	if len(t.expect) > 0 {
		if t.pressed == 0 {
			fmt.Println()
		} else if util.Between(t.pressed, C.AltA, C.AltZ) {
			fmt.Printf("alt-%c\n", t.pressed+'a'-C.AltA)
		} else if util.Between(t.pressed, C.F1, C.F4) {
			fmt.Printf("f%c\n", t.pressed+'1'-C.F1)
		} else if util.Between(t.pressed, C.CtrlA, C.CtrlZ) {
			fmt.Printf("ctrl-%c\n", t.pressed+'a'-C.CtrlA)
		} else {
			fmt.Printf("%c\n", t.pressed-C.AltZ)
		}
	}
	if len(t.selected) == 0 {
		cnt := t.merger.Length()
		if cnt > 0 && cnt > t.cy {
			fmt.Println(t.merger.Get(t.cy).AsString())
		}
	} else {
		sels := make([]selectedItem, 0, len(t.selected))
		for _, sel := range t.selected {
			sels = append(sels, sel)
		}
		sort.Sort(byTimeOrder(sels))
		for _, sel := range sels {
			fmt.Println(*sel.text)
		}
	}
}

func runeWidth(r rune, prefixWidth int) int {
	if r == '\t' {
		return 8 - prefixWidth%8
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

func (t *Terminal) move(y int, x int, clear bool) {
	maxy := C.MaxY()
	if !t.reverse {
		y = maxy - y - 1
	}

	if clear {
		C.MoveAndClear(y, x)
	} else {
		C.Move(y, x)
	}
}

func (t *Terminal) placeCursor() {
	t.move(0, len(t.prompt)+displayWidth(t.input[:t.cx]), false)
}

func (t *Terminal) printPrompt() {
	t.move(0, 0, true)
	C.CPrint(C.ColPrompt, true, t.prompt)
	C.CPrint(C.ColNormal, true, string(t.input))
}

func (t *Terminal) printInfo() {
	if t.inlineInfo {
		t.move(0, len(t.prompt)+displayWidth(t.input)+1, true)
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
	if t.toggleSort > 0 {
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

func (t *Terminal) printList() {
	t.constrain()

	maxy := t.maxItems()
	count := t.merger.Length() - t.offset
	for i := 0; i < maxy; i++ {
		var line int
		if t.inlineInfo {
			line = i + 1
		} else {
			line = i + 2
		}
		t.move(line, 0, true)
		if i < count {
			t.printItem(t.merger.Get(i+t.offset), i == t.cy-t.offset)
		}
	}
}

func (t *Terminal) printItem(item *Item, current bool) {
	_, selected := t.selected[item.index]
	if current {
		C.CPrint(C.ColCursor, true, ">")
		if selected {
			C.CPrint(C.ColCurrent, true, ">")
		} else {
			C.CPrint(C.ColCurrent, true, " ")
		}
		t.printHighlighted(item, true, C.ColCurrent, C.ColCurrentMatch, true)
	} else {
		C.CPrint(C.ColCursor, true, " ")
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
	var maxe int32
	for _, offset := range item.offsets {
		if offset[1] > maxe {
			maxe = offset[1]
		}
	}

	// Overflow
	text := []rune(*item.text)
	offsets := item.colorOffsets(col2, bold, current)
	maxWidth := C.MaxX() - 3
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
	t.printList()
	t.printPrompt()
	t.printInfo()
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
	return event.Type == key || event.Type == C.Rune && int(event.Char) == key-C.AltZ
}

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
	<-t.startChan
	{ // Late initialization
		t.mutex.Lock()
		t.initFunc()
		t.printPrompt()
		t.placeCursor()
		C.Refresh()
		t.printInfo()
		t.mutex.Unlock()
		go func() {
			timer := time.NewTimer(initialDelay)
			<-timer.C
			t.reqBox.Set(reqRefresh, nil)
		}()

		resizeChan := make(chan os.Signal, 1)
		signal.Notify(resizeChan, syscall.SIGWINCH)
		go func() {
			for {
				<-resizeChan
				t.reqBox.Set(reqRedraw, nil)
			}
		}()
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
					case reqRefresh:
						t.suppress = false
					case reqRedraw:
						C.Clear()
						C.Endwin()
						C.Refresh()
						t.printAll()
					case reqClose:
						C.Close()
						t.output()
						os.Exit(0)
					case reqQuit:
						C.Close()
						os.Exit(1)
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
		toggle := func() {
			if t.cy < t.merger.Length() {
				item := t.merger.Get(t.cy)
				if _, found := t.selected[item.index]; !found {
					var strptr *string
					if item.origText != nil {
						strptr = item.origText
					} else {
						strptr = item.text
					}
					t.selected[item.index] = selectedItem{time.Now(), strptr}
				} else {
					delete(t.selected, item.index)
				}
				req(reqInfo)
			}
		}
		for _, key := range t.expect {
			if keyMatch(key, event) {
				t.pressed = key
				req(reqClose)
				break
			}
		}
		if t.toggleSort > 0 {
			if keyMatch(t.toggleSort, event) {
				t.sort = !t.sort
				t.eventBox.Set(EvtSearchNew, t.sort)
				t.mutex.Unlock()
				continue
			}
		}
		switch event.Type {
		case C.Invalid:
			t.mutex.Unlock()
			continue
		case C.CtrlA:
			t.cx = 0
		case C.CtrlB:
			if t.cx > 0 {
				t.cx--
			}
		case C.CtrlC, C.CtrlG, C.CtrlQ, C.ESC:
			req(reqQuit)
		case C.CtrlD:
			if !t.delChar() && t.cx == 0 {
				req(reqQuit)
			}
		case C.CtrlE:
			t.cx = len(t.input)
		case C.CtrlF:
			if t.cx < len(t.input) {
				t.cx++
			}
		case C.CtrlH:
			if t.cx > 0 {
				t.input = append(t.input[:t.cx-1], t.input[t.cx:]...)
				t.cx--
			}
		case C.Tab:
			if t.multi && t.merger.Length() > 0 {
				toggle()
				t.vmove(-1)
				req(reqList)
			}
		case C.BTab:
			if t.multi && t.merger.Length() > 0 {
				toggle()
				t.vmove(1)
				req(reqList)
			}
		case C.CtrlJ, C.CtrlN:
			t.vmove(-1)
			req(reqList)
		case C.CtrlK, C.CtrlP:
			t.vmove(1)
			req(reqList)
		case C.CtrlM:
			req(reqClose)
		case C.CtrlL:
			req(reqRedraw)
		case C.CtrlU:
			if t.cx > 0 {
				t.yanked = copySlice(t.input[:t.cx])
				t.input = t.input[t.cx:]
				t.cx = 0
			}
		case C.CtrlW:
			if t.cx > 0 {
				t.rubout("\\s\\S")
			}
		case C.AltBS:
			if t.cx > 0 {
				t.rubout("[^[:alnum:]][[:alnum:]]")
			}
		case C.CtrlY:
			suffix := copySlice(t.input[t.cx:])
			t.input = append(append(t.input[:t.cx], t.yanked...), suffix...)
			t.cx += len(t.yanked)
		case C.Del:
			t.delChar()
		case C.PgUp:
			t.vmove(t.maxItems() - 1)
			req(reqList)
		case C.PgDn:
			t.vmove(-(t.maxItems() - 1))
			req(reqList)
		case C.AltB:
			t.cx = findLastMatch("[^[:alnum:]][[:alnum:]]", string(t.input[:t.cx])) + 1
		case C.AltF:
			t.cx += findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
		case C.AltD:
			ncx := t.cx +
				findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
			if ncx > t.cx {
				t.yanked = copySlice(t.input[t.cx:ncx])
				t.input = append(t.input[:t.cx], t.input[ncx:]...)
			}
		case C.Rune:
			prefix := copySlice(t.input[:t.cx])
			t.input = append(append(prefix, event.Char), t.input[t.cx:]...)
			t.cx++
		case C.Mouse:
			me := event.MouseEvent
			mx, my := util.Constrain(me.X-len(t.prompt), 0, len(t.input)), me.Y
			if !t.reverse {
				my = C.MaxY() - my - 1
			}
			min := 2
			if t.inlineInfo {
				min = 1
			}
			if me.S != 0 {
				// Scroll
				if t.merger.Length() > 0 {
					if t.multi && me.Mod {
						toggle()
					}
					t.vmove(me.S)
					req(reqList)
				}
			} else if me.Double {
				// Double-click
				if my >= min {
					if t.vset(t.offset+my-min) && t.cy < t.merger.Length() {
						req(reqClose)
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
		changed := string(previousInput) != string(t.input)
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

	if t.cy > t.offset+(height-1) {
		// Ceil
		t.offset = t.cy - (height - 1)
	} else if t.offset > t.cy {
		// Floor
		t.offset = t.cy
	}

	// Adjustment
	if count-t.offset < height {
		t.offset = util.Max(0, count-height)
		t.cy = util.Constrain(t.offset+diffpos, 0, count-1)
	}
}

func (t *Terminal) vmove(o int) {
	if t.reverse {
		t.vset(t.cy - o)
	} else {
		t.vset(t.cy + o)
	}
}

func (t *Terminal) vset(o int) bool {
	t.cy = util.Constrain(o, 0, t.merger.Length()-1)
	return t.cy == o
}

func (t *Terminal) maxItems() int {
	if t.inlineInfo {
		return C.MaxY() - 1
	} else {
		return C.MaxY() - 2
	}
}
