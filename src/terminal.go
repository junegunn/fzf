package fzf

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	C "github.com/junegunn/fzf/src/curses"
	"github.com/junegunn/fzf/src/util"

	"github.com/junegunn/go-runewidth"
)

// Terminal represents terminal input/output
type Terminal struct {
	prompt     string
	reverse    bool
	tac        bool
	cx         int
	cy         int
	offset     int
	yanked     []rune
	input      []rune
	multi      bool
	printQuery bool
	count      int
	progress   int
	reading    bool
	merger     *Merger
	selected   map[*string]selectedItem
	reqBox     *util.EventBox
	eventBox   *util.EventBox
	mutex      sync.Mutex
	initFunc   func()
	suppress   bool
}

type selectedItem struct {
	at   time.Time
	text *string
}

type ByTimeOrder []selectedItem

func (a ByTimeOrder) Len() int {
	return len(a)
}

func (a ByTimeOrder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByTimeOrder) Less(i, j int) bool {
	return a[i].at.Before(a[j].at)
}

var _spinner = []string{`-`, `\`, `|`, `/`, `-`, `\`, `|`, `/`}

const (
	reqPrompt util.EventType = iota
	reqInfo
	reqList
	reqRefresh
	reqRedraw
	reqClose
	reqQuit
)

const (
	initialDelay    = 100 * time.Millisecond
	spinnerDuration = 200 * time.Millisecond
)

// NewTerminal returns new Terminal object
func NewTerminal(opts *Options, eventBox *util.EventBox) *Terminal {
	input := []rune(opts.Query)
	return &Terminal{
		prompt:     opts.Prompt,
		tac:        opts.Sort == 0,
		reverse:    opts.Reverse,
		cx:         displayWidth(input),
		cy:         0,
		offset:     0,
		yanked:     []rune{},
		input:      input,
		multi:      opts.Multi,
		printQuery: opts.PrintQuery,
		merger:     EmptyMerger,
		selected:   make(map[*string]selectedItem),
		reqBox:     util.NewEventBox(),
		eventBox:   eventBox,
		mutex:      sync.Mutex{},
		suppress:   true,
		initFunc: func() {
			C.Init(opts.Color, opts.Color256, opts.Black, opts.Mouse)
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

func (t *Terminal) listIndex(y int) int {
	if t.tac {
		return t.merger.Length() - y - 1
	}
	return y
}

func (t *Terminal) output() {
	if t.printQuery {
		fmt.Println(string(t.input))
	}
	if len(t.selected) == 0 {
		cnt := t.merger.Length()
		if cnt > 0 && cnt > t.cy {
			fmt.Println(t.merger.Get(t.listIndex(t.cy)).AsString())
		}
	} else {
		sels := make([]selectedItem, 0, len(t.selected))
		for _, sel := range t.selected {
			sels = append(sels, sel)
		}
		sort.Sort(ByTimeOrder(sels))
		for _, sel := range sels {
			fmt.Println(*sel.text)
		}
	}
}

func displayWidth(runes []rune) int {
	l := 0
	for _, r := range runes {
		l += runewidth.RuneWidth(r)
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
	t.move(1, 0, true)
	if t.reading {
		duration := int64(spinnerDuration)
		idx := (time.Now().UnixNano() % (duration * int64(len(_spinner)))) / duration
		C.CPrint(C.ColSpinner, true, _spinner[idx])
	}

	t.move(1, 2, false)
	output := fmt.Sprintf("%d/%d", t.merger.Length(), t.count)
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

	maxy := maxItems()
	count := t.merger.Length() - t.offset
	for i := 0; i < maxy; i++ {
		t.move(i+2, 0, true)
		if i < count {
			t.printItem(t.merger.Get(t.listIndex(i+t.offset)), i == t.cy-t.offset)
		}
	}
}

func (t *Terminal) printItem(item *Item, current bool) {
	_, selected := t.selected[item.text]
	if current {
		C.CPrint(C.ColCursor, true, ">")
		if selected {
			C.CPrint(C.ColCurrent, true, ">")
		} else {
			C.CPrint(C.ColCurrent, true, " ")
		}
		t.printHighlighted(item, true, C.ColCurrent, C.ColCurrentMatch)
	} else {
		C.CPrint(C.ColCursor, true, " ")
		if selected {
			C.CPrint(C.ColSelected, true, ">")
		} else {
			C.Print(" ")
		}
		t.printHighlighted(item, false, 0, C.ColMatch)
	}
}

func trimRight(runes []rune, width int) ([]rune, int) {
	currentWidth := displayWidth(runes)
	trimmed := 0

	for currentWidth > width && len(runes) > 0 {
		sz := len(runes)
		currentWidth -= runewidth.RuneWidth(runes[sz-1])
		runes = runes[:sz-1]
		trimmed++
	}
	return runes, trimmed
}

func trimLeft(runes []rune, width int) ([]rune, int32) {
	currentWidth := displayWidth(runes)
	var trimmed int32

	for currentWidth > width && len(runes) > 0 {
		currentWidth -= runewidth.RuneWidth(runes[0])
		runes = runes[1:]
		trimmed++
	}
	return runes, trimmed
}

func (*Terminal) printHighlighted(item *Item, bold bool, col1 int, col2 int) {
	var maxe int32
	for _, offset := range item.offsets {
		if offset[1] > maxe {
			maxe = offset[1]
		}
	}

	// Overflow
	text := []rune(*item.text)
	offsets := item.offsets
	maxWidth := C.MaxX() - 3
	fullWidth := displayWidth(text)
	if fullWidth > maxWidth {
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
			offsets = make([]Offset, len(item.offsets))
			for idx, offset := range item.offsets {
				b, e := offset[0], offset[1]
				b += 2 - diff
				e += 2 - diff
				b = util.Max32(b, 2)
				if b < e {
					offsets[idx] = Offset{b, e}
				}
			}
			text = append([]rune(".."), text...)
		}
	}

	sort.Sort(ByOrder(offsets))
	var index int32
	for _, offset := range offsets {
		b := util.Max32(index, offset[0])
		e := util.Max32(index, offset[1])
		C.CPrint(col1, bold, string(text[index:b]))
		C.CPrint(col2, bold, string(text[b:e]))
		index = e
	}
	if index < int32(len(text)) {
		C.CPrint(col1, bold, string(text[index:]))
	}
}

func (t *Terminal) printAll() {
	t.printList()
	t.printInfo()
	t.printPrompt()
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

// Loop is called to start Terminal I/O
func (t *Terminal) Loop() {
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
					case reqInfo:
						t.printInfo()
					case reqList:
						t.printList()
					case reqRefresh:
						t.suppress = false
					case reqRedraw:
						C.Clear()
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
			idx := t.listIndex(t.cy)
			if idx < t.merger.Length() {
				item := t.merger.Get(idx)
				if _, found := t.selected[item.text]; !found {
					var strptr *string
					if item.origText != nil {
						strptr = item.origText
					} else {
						strptr = item.text
					}
					t.selected[item.text] = selectedItem{time.Now(), strptr}
				} else {
					delete(t.selected, item.text)
				}
				req(reqInfo)
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
			t.vmove(maxItems() - 1)
			req(reqList)
		case C.PgDn:
			t.vmove(-(maxItems() - 1))
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
				if my >= 2 {
					if t.vset(my-2) && t.listIndex(t.cy) < t.merger.Length() {
						req(reqClose)
					}
				}
			} else if me.Down {
				if my == 0 && mx >= 0 {
					// Prompt
					t.cx = mx
				} else if my >= 2 {
					// List
					if t.vset(t.offset+my-2) && t.multi && me.Mod {
						toggle()
					}
					req(reqList)
				}
			}
		}
		changed := string(previousInput) != string(t.input)
		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if changed {
			t.eventBox.Set(EvtSearchNew, nil)
		}
		for _, event := range events {
			t.reqBox.Set(event, nil)
		}
	}
}

func (t *Terminal) constrain() {
	count := t.merger.Length()
	height := C.MaxY() - 2
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

func maxItems() int {
	return C.MaxY() - 2
}
