package fzf

import (
	"fmt"
	C "github.com/junegunn/fzf/src/curses"
	"github.com/junegunn/go-runewidth"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"
)

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
	selected   map[*string]*string
	reqBox     *EventBox
	eventBox   *EventBox
	mutex      sync.Mutex
	initFunc   func()
	suppress   bool
}

var _spinner []string = []string{`-`, `\`, `|`, `/`, `-`, `\`, `|`, `/`}

const (
	REQ_PROMPT EventType = iota
	REQ_INFO
	REQ_LIST
	REQ_REFRESH
	REQ_REDRAW
	REQ_CLOSE
	REQ_QUIT
)

const (
	INITIAL_DELAY    = 100 * time.Millisecond
	SPINNER_DURATION = 200 * time.Millisecond
)

func NewTerminal(opts *Options, eventBox *EventBox) *Terminal {
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
		selected:   make(map[*string]*string),
		reqBox:     NewEventBox(),
		eventBox:   eventBox,
		mutex:      sync.Mutex{},
		suppress:   true,
		initFunc: func() {
			C.Init(opts.Color, opts.Color256, opts.Black, opts.Mouse)
		}}
}

func (t *Terminal) Input() []rune {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return copySlice(t.input)
}

func (t *Terminal) UpdateCount(cnt int, final bool) {
	t.mutex.Lock()
	t.count = cnt
	t.reading = !final
	t.mutex.Unlock()
	t.reqBox.Set(REQ_INFO, nil)
	if final {
		t.reqBox.Set(REQ_REFRESH, nil)
	}
}

func (t *Terminal) UpdateProgress(progress float32) {
	t.mutex.Lock()
	t.progress = int(progress * 100)
	t.mutex.Unlock()
	t.reqBox.Set(REQ_INFO, nil)
}

func (t *Terminal) UpdateList(merger *Merger) {
	t.mutex.Lock()
	t.progress = 100
	t.merger = merger
	t.mutex.Unlock()
	t.reqBox.Set(REQ_INFO, nil)
	t.reqBox.Set(REQ_LIST, nil)
}

func (t *Terminal) listIndex(y int) int {
	if t.tac {
		return t.merger.Length() - y - 1
	} else {
		return y
	}
}

func (t *Terminal) output() {
	if t.printQuery {
		fmt.Println(string(t.input))
	}
	if len(t.selected) == 0 {
		if t.merger.Length() > t.cy {
			t.merger.Get(t.listIndex(t.cy)).Print()
		}
	} else {
		for ptr, orig := range t.selected {
			if orig != nil {
				fmt.Println(*orig)
			} else {
				fmt.Println(*ptr)
			}
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
	C.CPrint(C.COL_PROMPT, true, t.prompt)
	C.CPrint(C.COL_NORMAL, true, string(t.input))
}

func (t *Terminal) printInfo() {
	t.move(1, 0, true)
	if t.reading {
		duration := int64(SPINNER_DURATION)
		idx := (time.Now().UnixNano() % (duration * int64(len(_spinner)))) / duration
		C.CPrint(C.COL_SPINNER, true, _spinner[idx])
	}

	t.move(1, 2, false)
	output := fmt.Sprintf("%d/%d", t.merger.Length(), t.count)
	if t.multi && len(t.selected) > 0 {
		output += fmt.Sprintf(" (%d)", len(t.selected))
	}
	if t.progress > 0 && t.progress < 100 {
		output += fmt.Sprintf(" (%d%%)", t.progress)
	}
	C.CPrint(C.COL_INFO, false, output)
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
		C.CPrint(C.COL_CURSOR, true, ">")
		if selected {
			C.CPrint(C.COL_CURRENT, true, ">")
		} else {
			C.CPrint(C.COL_CURRENT, true, " ")
		}
		t.printHighlighted(item, true, C.COL_CURRENT, C.COL_CURRENT_MATCH)
	} else {
		C.CPrint(C.COL_CURSOR, true, " ")
		if selected {
			C.CPrint(C.COL_SELECTED, true, ">")
		} else {
			C.Print(" ")
		}
		t.printHighlighted(item, false, 0, C.COL_MATCH)
	}
}

func trimRight(runes []rune, width int) ([]rune, int) {
	currentWidth := displayWidth(runes)
	trimmed := 0

	for currentWidth > width && len(runes) > 0 {
		sz := len(runes)
		currentWidth -= runewidth.RuneWidth(runes[sz-1])
		runes = runes[:sz-1]
		trimmed += 1
	}
	return runes, trimmed
}

func trimLeft(runes []rune, width int) ([]rune, int32) {
	currentWidth := displayWidth(runes)
	var trimmed int32 = 0

	for currentWidth > width && len(runes) > 0 {
		currentWidth -= runewidth.RuneWidth(runes[0])
		runes = runes[1:]
		trimmed += 1
	}
	return runes, trimmed
}

func (*Terminal) printHighlighted(item *Item, bold bool, col1 int, col2 int) {
	var maxe int32 = 0
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
				b = Max32(b, 2)
				if b < e {
					offsets[idx] = Offset{b, e}
				}
			}
			text = append([]rune(".."), text...)
		}
	}

	sort.Sort(ByOrder(offsets))
	var index int32 = 0
	for _, offset := range offsets {
		b := Max32(index, offset[0])
		e := Max32(index, offset[1])
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
			timer := time.NewTimer(INITIAL_DELAY)
			<-timer.C
			t.reqBox.Set(REQ_REFRESH, nil)
		}()
	}

	go func() {
		for {
			t.reqBox.Wait(func(events *Events) {
				defer events.Clear()
				t.mutex.Lock()
				for req := range *events {
					switch req {
					case REQ_PROMPT:
						t.printPrompt()
					case REQ_INFO:
						t.printInfo()
					case REQ_LIST:
						t.printList()
					case REQ_REFRESH:
						t.suppress = false
					case REQ_REDRAW:
						C.Clear()
						t.printAll()
					case REQ_CLOSE:
						C.Close()
						t.output()
						os.Exit(0)
					case REQ_QUIT:
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
		events := []EventType{REQ_PROMPT}
		req := func(evts ...EventType) {
			for _, event := range evts {
				events = append(events, event)
				if event == REQ_CLOSE || event == REQ_QUIT {
					looping = false
				}
			}
		}
		toggle := func() {
			idx := t.listIndex(t.cy)
			if idx < t.merger.Length() {
				item := t.merger.Get(idx)
				if _, found := t.selected[item.text]; !found {
					t.selected[item.text] = item.origText
				} else {
					delete(t.selected, item.text)
				}
				req(REQ_INFO)
			}
		}
		switch event.Type {
		case C.INVALID:
			t.mutex.Unlock()
			continue
		case C.CTRL_A:
			t.cx = 0
		case C.CTRL_B:
			if t.cx > 0 {
				t.cx -= 1
			}
		case C.CTRL_C, C.CTRL_G, C.CTRL_Q, C.ESC:
			req(REQ_QUIT)
		case C.CTRL_D:
			if !t.delChar() && t.cx == 0 {
				req(REQ_QUIT)
			}
		case C.CTRL_E:
			t.cx = len(t.input)
		case C.CTRL_F:
			if t.cx < len(t.input) {
				t.cx += 1
			}
		case C.CTRL_H:
			if t.cx > 0 {
				t.input = append(t.input[:t.cx-1], t.input[t.cx:]...)
				t.cx -= 1
			}
		case C.TAB:
			if t.multi && t.merger.Length() > 0 {
				toggle()
				t.vmove(-1)
				req(REQ_LIST)
			}
		case C.BTAB:
			if t.multi && t.merger.Length() > 0 {
				toggle()
				t.vmove(1)
				req(REQ_LIST)
			}
		case C.CTRL_J, C.CTRL_N:
			t.vmove(-1)
			req(REQ_LIST)
		case C.CTRL_K, C.CTRL_P:
			t.vmove(1)
			req(REQ_LIST)
		case C.CTRL_M:
			req(REQ_CLOSE)
		case C.CTRL_L:
			req(REQ_REDRAW)
		case C.CTRL_U:
			if t.cx > 0 {
				t.yanked = copySlice(t.input[:t.cx])
				t.input = t.input[t.cx:]
				t.cx = 0
			}
		case C.CTRL_W:
			if t.cx > 0 {
				t.rubout("\\s\\S")
			}
		case C.ALT_BS:
			if t.cx > 0 {
				t.rubout("[^[:alnum:]][[:alnum:]]")
			}
		case C.CTRL_Y:
			t.input = append(append(t.input[:t.cx], t.yanked...), t.input[t.cx:]...)
			t.cx += len(t.yanked)
		case C.DEL:
			t.delChar()
		case C.PGUP:
			t.vmove(maxItems() - 1)
			req(REQ_LIST)
		case C.PGDN:
			t.vmove(-(maxItems() - 1))
			req(REQ_LIST)
		case C.ALT_B:
			t.cx = findLastMatch("[^[:alnum:]][[:alnum:]]", string(t.input[:t.cx])) + 1
		case C.ALT_F:
			t.cx += findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
		case C.ALT_D:
			ncx := t.cx +
				findFirstMatch("[[:alnum:]][^[:alnum:]]|(.$)", string(t.input[t.cx:])) + 1
			if ncx > t.cx {
				t.yanked = copySlice(t.input[t.cx:ncx])
				t.input = append(t.input[:t.cx], t.input[ncx:]...)
			}
		case C.RUNE:
			prefix := copySlice(t.input[:t.cx])
			t.input = append(append(prefix, event.Char), t.input[t.cx:]...)
			t.cx += 1
		case C.MOUSE:
			me := event.MouseEvent
			mx, my := Min(len(t.input), Max(0, me.X-len(t.prompt))), me.Y
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
					req(REQ_LIST)
				}
			} else if me.Double {
				// Double-click
				if my >= 2 {
					t.cy = my - 2
					if t.listIndex(t.cy) < t.merger.Length() {
						req(REQ_CLOSE)
					}
				}
			} else if me.Down {
				if my == 0 && mx >= 0 {
					// Prompt
					t.cx = mx
				} else if my >= 2 {
					// List
					t.cy = t.offset + my - 2
					if t.multi && me.Mod {
						toggle()
					}
					req(REQ_LIST)
				}
			}
		}
		changed := string(previousInput) != string(t.input)
		t.mutex.Unlock() // Must be unlocked before touching reqBox

		if changed {
			t.eventBox.Set(EVT_SEARCH_NEW, nil)
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

	t.cy = Max(0, Min(t.cy, count-1))

	if t.cy > t.offset+(height-1) {
		// Ceil
		t.offset = t.cy - (height - 1)
	} else if t.offset > t.cy {
		// Floor
		t.offset = t.cy
	}

	// Adjustment
	if count-t.offset < height {
		t.offset = Max(0, count-height)
		t.cy = Max(0, Min(t.offset+diffpos, count-1))
	}
}

func (t *Terminal) vmove(o int) {
	if t.reverse {
		t.cy -= o
	} else {
		t.cy += o
	}
	t.cy = Max(0, Min(t.cy, t.merger.Length()-1))
}

func maxItems() int {
	return C.MaxY() - 2
}
