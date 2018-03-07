// Copyright 2016 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// NewTerminfoScreen returns a Screen that uses the stock TTY interface
// and POSIX termios, combined with a terminfo description taken from
// the $TERM environment variable.  It returns an error if the terminal
// is not supported for any reason.
//
// For terminals that do not support dynamic resize events, the $LINES
// $COLUMNS environment variables can be set to the actual window size,
// otherwise defaults taken from the terminal database are used.
func NewTerminfoScreen() (Screen, error) {
	ti, e := LookupTerminfo(os.Getenv("TERM"))
	if e != nil {
		return nil, e
	}
	t := &tScreen{ti: ti}

	t.keyexist = make(map[Key]bool)
	t.keycodes = make(map[string]*tKeyCode)
	if len(ti.Mouse) > 0 {
		t.mouse = []byte(ti.Mouse)
	}
	t.prepareKeys()
	t.buildAcsMap()
	t.sigwinch = make(chan os.Signal, 10)
	t.fallback = make(map[rune]string)
	for k, v := range RuneFallbacks {
		t.fallback[k] = v
	}

	return t, nil
}

// tKeyCode represents a combination of a key code and modifiers.
type tKeyCode struct {
	key Key
	mod ModMask
}

// tScreen represents a screen backed by a terminfo implementation.
type tScreen struct {
	ti        *Terminfo
	h         int
	w         int
	fini      bool
	cells     CellBuffer
	in        *os.File
	out       *os.File
	curstyle  Style
	style     Style
	evch      chan Event
	sigwinch  chan os.Signal
	quit      chan struct{}
	indoneq   chan struct{}
	keyexist  map[Key]bool
	keycodes  map[string]*tKeyCode
	cx        int
	cy        int
	mouse     []byte
	clear     bool
	cursorx   int
	cursory   int
	tiosp     *termiosPrivate
	baud      int
	wasbtn    bool
	acs       map[rune]string
	charset   string
	encoder   transform.Transformer
	decoder   transform.Transformer
	fallback  map[rune]string
	colors    map[Color]Color
	palette   []Color
	truecolor bool
	escaped   bool
	buttondn  bool

	sync.Mutex
}

func (t *tScreen) Init() error {
	t.evch = make(chan Event, 10)
	t.indoneq = make(chan struct{})
	t.charset = "UTF-8"

	t.charset = getCharset()
	if enc := GetEncoding(t.charset); enc != nil {
		t.encoder = enc.NewEncoder()
		t.decoder = enc.NewDecoder()
	} else {
		return ErrNoCharset
	}
	ti := t.ti

	// environment overrides
	w := ti.Columns
	h := ti.Lines
	if i, _ := strconv.Atoi(os.Getenv("LINES")); i != 0 {
		h = i
	}
	if i, _ := strconv.Atoi(os.Getenv("COLUMNS")); i != 0 {
		w = i
	}
	if e := t.termioInit(); e != nil {
		return e
	}

	if t.ti.SetFgBgRGB != "" || t.ti.SetFgRGB != "" || t.ti.SetBgRGB != "" {
		t.truecolor = true
	}
	// A user who wants to have his themes honored can
	// set this environment variable.
	if os.Getenv("TCELL_TRUECOLOR") == "disable" {
		t.truecolor = false
	}
	if !t.truecolor {
		t.colors = make(map[Color]Color)
		t.palette = make([]Color, t.Colors())
		for i := 0; i < t.Colors(); i++ {
			t.palette[i] = Color(i)
			// identity map for our builtin colors
			t.colors[Color(i)] = Color(i)
		}
	}

	t.TPuts(ti.EnterCA)
	t.TPuts(ti.HideCursor)
	t.TPuts(ti.EnableAcs)
	t.TPuts(ti.Clear)

	t.quit = make(chan struct{})

	t.Lock()
	t.cx = -1
	t.cy = -1
	t.style = StyleDefault
	t.cells.Resize(w, h)
	t.cursorx = -1
	t.cursory = -1
	t.resize()
	t.Unlock()

	go t.inputLoop()

	return nil
}

func (t *tScreen) prepareKeyMod(key Key, mod ModMask, val string) {
	if val != "" {
		// Do not overrride codes that already exist
		if _, exist := t.keycodes[val]; !exist {
			t.keyexist[key] = true
			t.keycodes[val] = &tKeyCode{key: key, mod: mod}
		}
	}
}

func (t *tScreen) prepareKey(key Key, val string) {
	t.prepareKeyMod(key, ModNone, val)
}

func (t *tScreen) prepareKeys() {
	ti := t.ti
	t.prepareKey(KeyBackspace, ti.KeyBackspace)
	t.prepareKey(KeyF1, ti.KeyF1)
	t.prepareKey(KeyF2, ti.KeyF2)
	t.prepareKey(KeyF3, ti.KeyF3)
	t.prepareKey(KeyF4, ti.KeyF4)
	t.prepareKey(KeyF5, ti.KeyF5)
	t.prepareKey(KeyF6, ti.KeyF6)
	t.prepareKey(KeyF7, ti.KeyF7)
	t.prepareKey(KeyF8, ti.KeyF8)
	t.prepareKey(KeyF9, ti.KeyF9)
	t.prepareKey(KeyF10, ti.KeyF10)
	t.prepareKey(KeyF11, ti.KeyF11)
	t.prepareKey(KeyF12, ti.KeyF12)
	t.prepareKey(KeyF13, ti.KeyF13)
	t.prepareKey(KeyF14, ti.KeyF14)
	t.prepareKey(KeyF15, ti.KeyF15)
	t.prepareKey(KeyF16, ti.KeyF16)
	t.prepareKey(KeyF17, ti.KeyF17)
	t.prepareKey(KeyF18, ti.KeyF18)
	t.prepareKey(KeyF19, ti.KeyF19)
	t.prepareKey(KeyF20, ti.KeyF20)
	t.prepareKey(KeyF21, ti.KeyF21)
	t.prepareKey(KeyF22, ti.KeyF22)
	t.prepareKey(KeyF23, ti.KeyF23)
	t.prepareKey(KeyF24, ti.KeyF24)
	t.prepareKey(KeyF25, ti.KeyF25)
	t.prepareKey(KeyF26, ti.KeyF26)
	t.prepareKey(KeyF27, ti.KeyF27)
	t.prepareKey(KeyF28, ti.KeyF28)
	t.prepareKey(KeyF29, ti.KeyF29)
	t.prepareKey(KeyF30, ti.KeyF30)
	t.prepareKey(KeyF31, ti.KeyF31)
	t.prepareKey(KeyF32, ti.KeyF32)
	t.prepareKey(KeyF33, ti.KeyF33)
	t.prepareKey(KeyF34, ti.KeyF34)
	t.prepareKey(KeyF35, ti.KeyF35)
	t.prepareKey(KeyF36, ti.KeyF36)
	t.prepareKey(KeyF37, ti.KeyF37)
	t.prepareKey(KeyF38, ti.KeyF38)
	t.prepareKey(KeyF39, ti.KeyF39)
	t.prepareKey(KeyF40, ti.KeyF40)
	t.prepareKey(KeyF41, ti.KeyF41)
	t.prepareKey(KeyF42, ti.KeyF42)
	t.prepareKey(KeyF43, ti.KeyF43)
	t.prepareKey(KeyF44, ti.KeyF44)
	t.prepareKey(KeyF45, ti.KeyF45)
	t.prepareKey(KeyF46, ti.KeyF46)
	t.prepareKey(KeyF47, ti.KeyF47)
	t.prepareKey(KeyF48, ti.KeyF48)
	t.prepareKey(KeyF49, ti.KeyF49)
	t.prepareKey(KeyF50, ti.KeyF50)
	t.prepareKey(KeyF51, ti.KeyF51)
	t.prepareKey(KeyF52, ti.KeyF52)
	t.prepareKey(KeyF53, ti.KeyF53)
	t.prepareKey(KeyF54, ti.KeyF54)
	t.prepareKey(KeyF55, ti.KeyF55)
	t.prepareKey(KeyF56, ti.KeyF56)
	t.prepareKey(KeyF57, ti.KeyF57)
	t.prepareKey(KeyF58, ti.KeyF58)
	t.prepareKey(KeyF59, ti.KeyF59)
	t.prepareKey(KeyF60, ti.KeyF60)
	t.prepareKey(KeyF61, ti.KeyF61)
	t.prepareKey(KeyF62, ti.KeyF62)
	t.prepareKey(KeyF63, ti.KeyF63)
	t.prepareKey(KeyF64, ti.KeyF64)
	t.prepareKey(KeyInsert, ti.KeyInsert)
	t.prepareKey(KeyDelete, ti.KeyDelete)
	t.prepareKey(KeyHome, ti.KeyHome)
	t.prepareKey(KeyEnd, ti.KeyEnd)
	t.prepareKey(KeyUp, ti.KeyUp)
	t.prepareKey(KeyDown, ti.KeyDown)
	t.prepareKey(KeyLeft, ti.KeyLeft)
	t.prepareKey(KeyRight, ti.KeyRight)
	t.prepareKey(KeyPgUp, ti.KeyPgUp)
	t.prepareKey(KeyPgDn, ti.KeyPgDn)
	t.prepareKey(KeyHelp, ti.KeyHelp)
	t.prepareKey(KeyPrint, ti.KeyPrint)
	t.prepareKey(KeyCancel, ti.KeyCancel)
	t.prepareKey(KeyExit, ti.KeyExit)
	t.prepareKey(KeyBacktab, ti.KeyBacktab)

	t.prepareKeyMod(KeyRight, ModShift, ti.KeyShfRight)
	t.prepareKeyMod(KeyLeft, ModShift, ti.KeyShfLeft)
	t.prepareKeyMod(KeyUp, ModShift, ti.KeyShfUp)
	t.prepareKeyMod(KeyDown, ModShift, ti.KeyShfDown)
	t.prepareKeyMod(KeyHome, ModShift, ti.KeyShfHome)
	t.prepareKeyMod(KeyEnd, ModShift, ti.KeyShfEnd)

	t.prepareKeyMod(KeyRight, ModCtrl, ti.KeyCtrlRight)
	t.prepareKeyMod(KeyLeft, ModCtrl, ti.KeyCtrlLeft)
	t.prepareKeyMod(KeyUp, ModCtrl, ti.KeyCtrlUp)
	t.prepareKeyMod(KeyDown, ModCtrl, ti.KeyCtrlDown)
	t.prepareKeyMod(KeyHome, ModCtrl, ti.KeyCtrlHome)
	t.prepareKeyMod(KeyEnd, ModCtrl, ti.KeyCtrlEnd)

	t.prepareKeyMod(KeyRight, ModAlt, ti.KeyAltRight)
	t.prepareKeyMod(KeyLeft, ModAlt, ti.KeyAltLeft)
	t.prepareKeyMod(KeyUp, ModAlt, ti.KeyAltUp)
	t.prepareKeyMod(KeyDown, ModAlt, ti.KeyAltDown)
	t.prepareKeyMod(KeyHome, ModAlt, ti.KeyAltHome)
	t.prepareKeyMod(KeyEnd, ModAlt, ti.KeyAltEnd)

	t.prepareKeyMod(KeyRight, ModAlt, ti.KeyMetaRight)
	t.prepareKeyMod(KeyLeft, ModAlt, ti.KeyMetaLeft)
	t.prepareKeyMod(KeyUp, ModAlt, ti.KeyMetaUp)
	t.prepareKeyMod(KeyDown, ModAlt, ti.KeyMetaDown)
	t.prepareKeyMod(KeyHome, ModAlt, ti.KeyMetaHome)
	t.prepareKeyMod(KeyEnd, ModAlt, ti.KeyMetaEnd)

	t.prepareKeyMod(KeyRight, ModAlt|ModShift, ti.KeyAltShfRight)
	t.prepareKeyMod(KeyLeft, ModAlt|ModShift, ti.KeyAltShfLeft)
	t.prepareKeyMod(KeyUp, ModAlt|ModShift, ti.KeyAltShfUp)
	t.prepareKeyMod(KeyDown, ModAlt|ModShift, ti.KeyAltShfDown)
	t.prepareKeyMod(KeyHome, ModAlt|ModShift, ti.KeyAltShfHome)
	t.prepareKeyMod(KeyEnd, ModAlt|ModShift, ti.KeyAltShfEnd)

	t.prepareKeyMod(KeyRight, ModAlt|ModShift, ti.KeyMetaShfRight)
	t.prepareKeyMod(KeyLeft, ModAlt|ModShift, ti.KeyMetaShfLeft)
	t.prepareKeyMod(KeyUp, ModAlt|ModShift, ti.KeyMetaShfUp)
	t.prepareKeyMod(KeyDown, ModAlt|ModShift, ti.KeyMetaShfDown)
	t.prepareKeyMod(KeyHome, ModAlt|ModShift, ti.KeyMetaShfHome)
	t.prepareKeyMod(KeyEnd, ModAlt|ModShift, ti.KeyMetaShfEnd)

	t.prepareKeyMod(KeyRight, ModCtrl|ModShift, ti.KeyCtrlShfRight)
	t.prepareKeyMod(KeyLeft, ModCtrl|ModShift, ti.KeyCtrlShfLeft)
	t.prepareKeyMod(KeyUp, ModCtrl|ModShift, ti.KeyCtrlShfUp)
	t.prepareKeyMod(KeyDown, ModCtrl|ModShift, ti.KeyCtrlShfDown)
	t.prepareKeyMod(KeyHome, ModCtrl|ModShift, ti.KeyCtrlShfHome)
	t.prepareKeyMod(KeyEnd, ModCtrl|ModShift, ti.KeyCtrlShfEnd)

	// Sadly, xterm handling of keycodes is somewhat erratic.  In
	// particular, different codes are sent depending on application
	// mode is in use or not, and the entries for many of these are
	// simply absent from terminfo on many systems.  So we insert
	// a number of escape sequences if they are not already used, in
	// order to have the widest correct usage.  Note that prepareKey
	// will not inject codes if the escape sequence is already known.
	// We also only do this for terminals that have the application
	// mode present.

	// Cursor mode
	if ti.EnterKeypad != "" {
		t.prepareKey(KeyUp, "\x1b[A")
		t.prepareKey(KeyDown, "\x1b[B")
		t.prepareKey(KeyRight, "\x1b[C")
		t.prepareKey(KeyLeft, "\x1b[D")
		t.prepareKey(KeyEnd, "\x1b[F")
		t.prepareKey(KeyHome, "\x1b[H")
		t.prepareKey(KeyDelete, "\x1b[3~")
		t.prepareKey(KeyHome, "\x1b[1~")
		t.prepareKey(KeyEnd, "\x1b[4~")
		t.prepareKey(KeyPgUp, "\x1b[5~")
		t.prepareKey(KeyPgDn, "\x1b[6~")

		// Application mode
		t.prepareKey(KeyUp, "\x1bOA")
		t.prepareKey(KeyDown, "\x1bOB")
		t.prepareKey(KeyRight, "\x1bOC")
		t.prepareKey(KeyLeft, "\x1bOD")
		t.prepareKey(KeyHome, "\x1bOH")
	}

outer:
	// Add key mappings for control keys.
	for i := 0; i < ' '; i++ {
		// Do not insert direct key codes for ambiguous keys.
		// For example, ESC is used for lots of other keys, so
		// when parsing this we don't want to fast path handling
		// of it, but instead wait a bit before parsing it as in
		// isolation.
		for esc := range t.keycodes {
			if []byte(esc)[0] == byte(i) {
				continue outer
			}
		}

		t.keyexist[Key(i)] = true

		mod := ModCtrl
		switch Key(i) {
		case KeyBS, KeyTAB, KeyESC, KeyCR:
			// directly typeable- no control sequence
			mod = ModNone
		}
		t.keycodes[string(rune(i))] = &tKeyCode{key: Key(i), mod: mod}
	}
}

func (t *tScreen) Fini() {
	ti := t.ti
	t.Lock()
	t.cells.Resize(0, 0)
	t.TPuts(ti.ShowCursor)
	t.TPuts(ti.AttrOff)
	t.TPuts(ti.Clear)
	t.TPuts(ti.ExitCA)
	t.TPuts(ti.ExitKeypad)
	t.TPuts(ti.TParm(ti.MouseMode, 0))
	t.curstyle = Style(-1)
	t.clear = false
	t.fini = true
	t.Unlock()

	if t.quit != nil {
		close(t.quit)
	}
	t.termioFini()
}

func (t *tScreen) SetStyle(style Style) {
	t.Lock()
	if !t.fini {
		t.style = style
	}
	t.Unlock()
}

func (t *tScreen) Clear() {
	t.Fill(' ', t.style)
}

func (t *tScreen) Fill(r rune, style Style) {
	t.Lock()
	if !t.fini {
		t.cells.Fill(r, style)
	}
	t.Unlock()
}

func (t *tScreen) SetContent(x, y int, mainc rune, combc []rune, style Style) {
	t.Lock()
	if !t.fini {
		t.cells.SetContent(x, y, mainc, combc, style)
	}
	t.Unlock()
}

func (t *tScreen) GetContent(x, y int) (rune, []rune, Style, int) {
	t.Lock()
	mainc, combc, style, width := t.cells.GetContent(x, y)
	t.Unlock()
	return mainc, combc, style, width
}

func (t *tScreen) SetCell(x, y int, style Style, ch ...rune) {
	if len(ch) > 0 {
		t.SetContent(x, y, ch[0], ch[1:], style)
	} else {
		t.SetContent(x, y, ' ', nil, style)
	}
}

func (t *tScreen) encodeRune(r rune, buf []byte) []byte {

	nb := make([]byte, 6)
	ob := make([]byte, 6)
	num := utf8.EncodeRune(ob, r)
	ob = ob[:num]
	dst := 0
	var err error
	if enc := t.encoder; enc != nil {
		enc.Reset()
		dst, _, err = enc.Transform(nb, ob, true)
	}
	if err != nil || dst == 0 || nb[0] == '\x1a' {
		// Combining characters are elided
		if len(buf) == 0 {
			if acs, ok := t.acs[r]; ok {
				buf = append(buf, []byte(acs)...)
			} else if fb, ok := t.fallback[r]; ok {
				buf = append(buf, []byte(fb)...)
			} else {
				buf = append(buf, '?')
			}
		}
	} else {
		buf = append(buf, nb[:dst]...)
	}

	return buf
}

func (t *tScreen) sendFgBg(fg Color, bg Color) {
	ti := t.ti
	if ti.Colors == 0 {
		return
	}
	if t.truecolor {
		if ti.SetFgBgRGB != "" &&
			fg != ColorDefault && bg != ColorDefault {
			r1, g1, b1 := fg.RGB()
			r2, g2, b2 := bg.RGB()
			t.TPuts(ti.TParm(ti.SetFgBgRGB,
				int(r1), int(g1), int(b1),
				int(r2), int(g2), int(b2)))
		} else {
			if fg != ColorDefault && ti.SetFgRGB != "" {
				r, g, b := fg.RGB()
				t.TPuts(ti.TParm(ti.SetFgRGB,
					int(r), int(g), int(b)))
			}
			if bg != ColorDefault && ti.SetBgRGB != "" {
				r, g, b := bg.RGB()
				t.TPuts(ti.TParm(ti.SetBgRGB,
					int(r), int(g), int(b)))
			}
		}
		return
	}

	if fg != ColorDefault {
		if v, ok := t.colors[fg]; ok {
			fg = v
		} else {
			v = FindColor(fg, t.palette)
			t.colors[fg] = v
			fg = v
		}
	}

	if bg != ColorDefault {
		if v, ok := t.colors[bg]; ok {
			bg = v
		} else {
			v = FindColor(bg, t.palette)
			t.colors[bg] = v
			bg = v
		}
	}

	if ti.SetFgBg != "" && fg != ColorDefault && bg != ColorDefault {
		t.TPuts(ti.TParm(ti.SetFgBg, int(fg), int(bg)))
	} else {
		if fg != ColorDefault && ti.SetFg != "" {
			t.TPuts(ti.TParm(ti.SetFg, int(fg)))
		}
		if bg != ColorDefault && ti.SetBg != "" {
			t.TPuts(ti.TParm(ti.SetBg, int(bg)))
		}
	}
}

func (t *tScreen) drawCell(x, y int) int {

	ti := t.ti

	mainc, combc, style, width := t.cells.GetContent(x, y)
	if !t.cells.Dirty(x, y) {
		return width
	}

	if t.cy != y || t.cx != x {
		t.TPuts(ti.TGoto(x, y))
		t.cx = x
		t.cy = y
	}

	if style == StyleDefault {
		style = t.style
	}
	if style != t.curstyle {
		fg, bg, attrs := style.Decompose()

		t.TPuts(ti.AttrOff)

		t.sendFgBg(fg, bg)
		if attrs&AttrBold != 0 {
			t.TPuts(ti.Bold)
		}
		if attrs&AttrUnderline != 0 {
			t.TPuts(ti.Underline)
		}
		if attrs&AttrReverse != 0 {
			t.TPuts(ti.Reverse)
		}
		if attrs&AttrBlink != 0 {
			t.TPuts(ti.Blink)
		}
		if attrs&AttrDim != 0 {
			t.TPuts(ti.Dim)
		}
		t.curstyle = style
	}
	// now emit runes - taking care to not overrun width with a
	// wide character, and to ensure that we emit exactly one regular
	// character followed up by any residual combing characters

	if width < 1 {
		width = 1
	}

	var str string

	buf := make([]byte, 0, 6)

	buf = t.encodeRune(mainc, buf)
	for _, r := range combc {
		buf = t.encodeRune(r, buf)
	}

	str = string(buf)
	if width > 1 && str == "?" {
		// No FullWidth character support
		str = "? "
		t.cx = -1
	}

	// XXX: check for hazeltine not being able to display ~

	if x > t.w-width {
		// too wide to fit; emit a single space instead
		width = 1
		str = " "
	}
	io.WriteString(t.out, str)
	t.cx += width
	t.cells.SetDirty(x, y, false)
	if width > 1 {
		t.cx = -1
	}

	return width
}

func (t *tScreen) ShowCursor(x, y int) {
	t.Lock()
	t.cursorx = x
	t.cursory = y
	t.Unlock()
}

func (t *tScreen) HideCursor() {
	t.ShowCursor(-1, -1)
}

func (t *tScreen) showCursor() {

	x, y := t.cursorx, t.cursory
	w, h := t.cells.Size()
	if x < 0 || y < 0 || x >= w || y >= h {
		t.hideCursor()
		return
	}
	t.TPuts(t.ti.TGoto(x, y))
	t.TPuts(t.ti.ShowCursor)
	t.cx = x
	t.cy = y
}

func (t *tScreen) TPuts(s string) {
	t.ti.TPuts(t.out, s, t.baud)
}

func (t *tScreen) Show() {
	t.Lock()
	if !t.fini {
		t.resize()
		t.draw()
	}
	t.Unlock()
}

func (t *tScreen) clearScreen() {
	fg, bg, _ := t.style.Decompose()
	t.sendFgBg(fg, bg)
	t.TPuts(t.ti.Clear)
	t.clear = false
}

func (t *tScreen) hideCursor() {
	// does not update cursor position
	if t.ti.HideCursor != "" {
		t.TPuts(t.ti.HideCursor)
	} else {
		// No way to hide cursor, stick it
		// at bottom right of screen
		t.cx, t.cy = t.cells.Size()
		t.TPuts(t.ti.TGoto(t.cx, t.cy))
	}
}

func (t *tScreen) draw() {
	// clobber cursor position, because we're gonna change it all
	t.cx = -1
	t.cy = -1

	// hide the cursor while we move stuff around
	t.hideCursor()

	if t.clear {
		t.clearScreen()
	}

	for y := 0; y < t.h; y++ {
		for x := 0; x < t.w; x++ {
			width := t.drawCell(x, y)
			if width > 1 {
				if x+1 < t.w {
					// this is necessary so that if we ever
					// go back to drawing that cell, we
					// actually will *draw* it.
					t.cells.SetDirty(x+1, y, true)
				}
			}
			x += width - 1
		}
	}

	// restore the cursor
	t.showCursor()
}

func (t *tScreen) EnableMouse() {
	if len(t.mouse) != 0 {
		t.TPuts(t.ti.TParm(t.ti.MouseMode, 1))
	}
}

func (t *tScreen) DisableMouse() {
	if len(t.mouse) != 0 {
		t.TPuts(t.ti.TParm(t.ti.MouseMode, 0))
	}
}

func (t *tScreen) Size() (int, int) {
	t.Lock()
	w, h := t.w, t.h
	t.Unlock()
	return w, h
}

func (t *tScreen) resize() {
	if w, h, e := t.getWinSize(); e == nil {
		if w != t.w || h != t.h {
			t.cx = -1
			t.cy = -1

			t.cells.Resize(w, h)
			t.cells.Invalidate()
			t.h = h
			t.w = w
			ev := NewEventResize(w, h)
			t.PostEvent(ev)
		}
	}
}

func (t *tScreen) Colors() int {
	// this doesn't change, no need for lock
	if t.truecolor {
		return 1 << 24
	}
	return t.ti.Colors
}

func (t *tScreen) PollEvent() Event {
	select {
	case <-t.quit:
		return nil
	case ev := <-t.evch:
		return ev
	}
}

// vtACSNames is a map of bytes defined by terminfo that are used in
// the terminals Alternate Character Set to represent other glyphs.
// For example, the upper left corner of the box drawing set can be
// displayed by printing "l" while in the alternate character set.
// Its not quite that simple, since the "l" is the terminfo name,
// and it may be necessary to use a different character based on
// the terminal implementation (or the terminal may lack support for
// this altogether).  See buildAcsMap below for detail.
var vtACSNames = map[byte]rune{
	'+': RuneRArrow,
	',': RuneLArrow,
	'-': RuneUArrow,
	'.': RuneDArrow,
	'0': RuneBlock,
	'`': RuneDiamond,
	'a': RuneCkBoard,
	'b': '␉', // VT100, Not defined by terminfo
	'c': '␌', // VT100, Not defined by terminfo
	'd': '␋', // VT100, Not defined by terminfo
	'e': '␊', // VT100, Not defined by terminfo
	'f': RuneDegree,
	'g': RunePlMinus,
	'h': RuneBoard,
	'i': RuneLantern,
	'j': RuneLRCorner,
	'k': RuneURCorner,
	'l': RuneULCorner,
	'm': RuneLLCorner,
	'n': RunePlus,
	'o': RuneS1,
	'p': RuneS3,
	'q': RuneHLine,
	'r': RuneS7,
	's': RuneS9,
	't': RuneLTee,
	'u': RuneRTee,
	'v': RuneBTee,
	'w': RuneTTee,
	'x': RuneVLine,
	'y': RuneLEqual,
	'z': RuneGEqual,
	'{': RunePi,
	'|': RuneNEqual,
	'}': RuneSterling,
	'~': RuneBullet,
}

// buildAcsMap builds a map of characters that we translate from Unicode to
// alternate character encodings.  To do this, we use the standard VT100 ACS
// maps.  This is only done if the terminal lacks support for Unicode; we
// always prefer to emit Unicode glyphs when we are able.
func (t *tScreen) buildAcsMap() {
	acsstr := t.ti.AltChars
	t.acs = make(map[rune]string)
	for len(acsstr) > 2 {
		srcv := acsstr[0]
		dstv := string(acsstr[1])
		if r, ok := vtACSNames[srcv]; ok {
			t.acs[r] = t.ti.EnterAcs + dstv + t.ti.ExitAcs
		}
		acsstr = acsstr[2:]
	}
}

func (t *tScreen) PostEventWait(ev Event) {
	t.evch <- ev
}

func (t *tScreen) PostEvent(ev Event) error {
	select {
	case t.evch <- ev:
		return nil
	default:
		return ErrEventQFull
	}
}

func (t *tScreen) clip(x, y int) (int, int) {
	w, h := t.cells.Size()
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > w-1 {
		x = w - 1
	}
	if y > h-1 {
		y = h - 1
	}
	return x, y
}

func (t *tScreen) postMouseEvent(x, y, btn int) {

	// XTerm mouse events only report at most one button at a time,
	// which may include a wheel button.  Wheel motion events are
	// reported as single impulses, while other button events are reported
	// as separate press & release events.

	button := ButtonNone
	mod := ModNone

	// Mouse wheel has bit 6 set, no release events.  It should be noted
	// that wheel events are sometimes misdelivered as mouse button events
	// during a click-drag, so we debounce these, considering them to be
	// button press events unless we see an intervening release event.
	switch btn & 0x43 {
	case 0:
		button = Button1
		t.wasbtn = true
	case 1:
		button = Button2
		t.wasbtn = true
	case 2:
		button = Button3
		t.wasbtn = true
	case 3:
		button = ButtonNone
		t.wasbtn = false
	case 0x40:
		if !t.wasbtn {
			button = WheelUp
		} else {
			button = Button1
		}
	case 0x41:
		if !t.wasbtn {
			button = WheelDown
		} else {
			button = Button2
		}
	}

	if btn&0x4 != 0 {
		mod |= ModShift
	}
	if btn&0x8 != 0 {
		mod |= ModAlt
	}
	if btn&0x10 != 0 {
		mod |= ModCtrl
	}

	// Some terminals will report mouse coordinates outside the
	// screen, especially with click-drag events.  Clip the coordinates
	// to the screen in that case.
	x, y = t.clip(x, y)

	ev := NewEventMouse(x, y, button, mod)
	t.PostEvent(ev)
}

// parseSgrMouse attempts to locate an SGR mouse record at the start of the
// buffer.  It returns true, true if it found one, and the associated bytes
// be removed from the buffer.  It returns true, false if the buffer might
// contain such an event, but more bytes are necessary (partial match), and
// false, false if the content is definitely *not* an SGR mouse record.
func (t *tScreen) parseSgrMouse(buf *bytes.Buffer) (bool, bool) {

	b := buf.Bytes()

	var x, y, btn, state int
	dig := false
	neg := false
	motion := false
	i := 0
	val := 0

	for i = range b {
		switch b[i] {
		case '\x1b':
			if state != 0 {
				return false, false
			}
			state = 1

		case '\x9b':
			if state != 0 {
				return false, false
			}
			state = 2

		case '[':
			if state != 1 {
				return false, false
			}
			state = 2

		case '<':
			if state != 2 {
				return false, false
			}
			val = 0
			dig = false
			neg = false
			state = 3

		case '-':
			if state != 3 && state != 4 && state != 5 {
				return false, false
			}
			if dig || neg {
				return false, false
			}
			neg = true // stay in state

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if state != 3 && state != 4 && state != 5 {
				return false, false
			}
			val *= 10
			val += int(b[i] - '0')
			dig = true // stay in state

		case ';':
			if neg {
				val = -val
			}
			switch state {
			case 3:
				btn, val = val, 0
				neg, dig, state = false, false, 4
			case 4:
				x, val = val-1, 0
				neg, dig, state = false, false, 5
			default:
				return false, false
			}

		case 'm', 'M':
			if state != 5 {
				return false, false
			}
			if neg {
				val = -val
			}
			y = val - 1

			motion = (btn & 32) != 0
			btn &^= 32
			if b[i] == 'm' {
				// mouse release, clear all buttons
				btn |= 3
				btn &^= 0x40
				t.buttondn = false
			} else if motion {
				/*
				 * Some broken terminals appear to send
				 * mouse button one motion events, instead of
				 * encoding 35 (no buttons) into these events.
				 * We resolve these by looking for a non-motion
				 * event first.
				 */
				if !t.buttondn {
					btn |= 3
					btn &^= 0x40
				}
			} else {
				t.buttondn = true
			}
			// consume the event bytes
			for i >= 0 {
				buf.ReadByte()
				i--
			}
			t.postMouseEvent(x, y, btn)
			return true, true
		}
	}

	// incomplete & inconclusve at this point
	return true, false
}

// parseXtermMouse is like parseSgrMouse, but it parses a legacy
// X11 mouse record.
func (t *tScreen) parseXtermMouse(buf *bytes.Buffer) (bool, bool) {

	b := buf.Bytes()

	state := 0
	btn := 0
	x := 0
	y := 0

	for i := range b {
		switch state {
		case 0:
			switch b[i] {
			case '\x1b':
				state = 1
			case '\x9b':
				state = 2
			default:
				return false, false
			}
		case 1:
			if b[i] != '[' {
				return false, false
			}
			state = 2
		case 2:
			if b[i] != 'M' {
				return false, false
			}
			state++
		case 3:
			btn = int(b[i])
			state++
		case 4:
			x = int(b[i]) - 32 - 1
			state++
		case 5:
			y = int(b[i]) - 32 - 1
			for i >= 0 {
				buf.ReadByte()
				i--
			}
			t.postMouseEvent(x, y, btn)
			return true, true
		}
	}
	return true, false
}

func (t *tScreen) parseFunctionKey(buf *bytes.Buffer) (bool, bool) {
	b := buf.Bytes()
	partial := false
	for e, k := range t.keycodes {
		esc := []byte(e)
		if (len(esc) == 1) && (esc[0] == '\x1b') {
			continue
		}
		if bytes.HasPrefix(b, esc) {
			// matched
			var r rune
			if len(esc) == 1 {
				r = rune(b[0])
			}
			mod := k.mod
			if t.escaped {
				mod |= ModAlt
				t.escaped = false
			}
			ev := NewEventKey(k.key, r, mod)
			t.PostEvent(ev)
			for i := 0; i < len(esc); i++ {
				buf.ReadByte()
			}
			return true, true
		}
		if bytes.HasPrefix(esc, b) {
			partial = true
		}
	}
	return partial, false
}

func (t *tScreen) parseRune(buf *bytes.Buffer) (bool, bool) {
	b := buf.Bytes()
	if b[0] >= ' ' && b[0] <= 0x7F {
		// printable ASCII easy to deal with -- no encodings
		mod := ModNone
		if t.escaped {
			mod = ModAlt
			t.escaped = false
		}
		ev := NewEventKey(KeyRune, rune(b[0]), mod)
		t.PostEvent(ev)
		buf.ReadByte()
		return true, true
	}

	if b[0] < 0x80 {
		// Low numbered values are control keys, not runes.
		return false, false
	}

	utfb := make([]byte, 12)
	for l := 1; l <= len(b); l++ {
		t.decoder.Reset()
		nout, nin, e := t.decoder.Transform(utfb, b[:l], true)
		if e == transform.ErrShortSrc {
			continue
		}
		if nout != 0 {
			r, _ := utf8.DecodeRune(utfb[:nout])
			if r != utf8.RuneError {
				mod := ModNone
				if t.escaped {
					mod = ModAlt
					t.escaped = false
				}
				ev := NewEventKey(KeyRune, r, mod)
				t.PostEvent(ev)
			}
			for nin > 0 {
				buf.ReadByte()
				nin--
			}
			return true, true
		}
	}
	// Looks like potential escape
	return true, false
}

func (t *tScreen) scanInput(buf *bytes.Buffer, expire bool) {

	t.Lock()
	defer t.Unlock()

	for {
		b := buf.Bytes()
		if len(b) == 0 {
			buf.Reset()
			return
		}

		partials := 0

		if part, comp := t.parseRune(buf); comp {
			continue
		} else if part {
			partials++
		}

		if part, comp := t.parseFunctionKey(buf); comp {
			continue
		} else if part {
			partials++
		}

		// Only parse mouse records if this term claims to have
		// mouse support

		if t.ti.Mouse != "" {
			if part, comp := t.parseXtermMouse(buf); comp {
				continue
			} else if part {
				partials++
			}

			if part, comp := t.parseSgrMouse(buf); comp {
				continue
			} else if part {
				partials++
			}
		}

		if partials == 0 || expire {
			if b[0] == '\x1b' {
				if len(b) == 1 {
					ev := NewEventKey(KeyEsc, 0, ModNone)
					t.PostEvent(ev)
					t.escaped = false
				} else {
					t.escaped = true
				}
				buf.ReadByte()
				continue
			}
			// Nothing was going to match, or we timed out
			// waiting for more data -- just deliver the characters
			// to the app & let them sort it out.  Possibly we
			// should only do this for control characters like ESC.
			by, _ := buf.ReadByte()
			mod := ModNone
			if t.escaped {
				t.escaped = false
				mod = ModAlt
			}
			ev := NewEventKey(KeyRune, rune(by), mod)
			t.PostEvent(ev)
			continue
		}

		// well we have some partial data, wait until we get
		// some more
		break
	}
}

func (t *tScreen) inputLoop() {
	buf := &bytes.Buffer{}

	chunk := make([]byte, 128)
	for {
		select {
		case <-t.quit:
			close(t.indoneq)
			return
		case <-t.sigwinch:
			t.Lock()
			t.cx = -1
			t.cy = -1
			t.resize()
			t.cells.Invalidate()
			t.draw()
			t.Unlock()
			continue
		default:
		}
		n, e := t.in.Read(chunk)
		switch e {
		case io.EOF:
			// If we timeout waiting for more bytes, then it's
			// time to give up on it.  Even at 300 baud it takes
			// less than 0.5 ms to transmit a whole byte.
			if buf.Len() > 0 {
				t.scanInput(buf, true)
			}
			continue
		case nil:
		default:
			close(t.indoneq)
			return
		}
		buf.Write(chunk[:n])
		// Now we need to parse the input buffer for events
		t.scanInput(buf, false)
	}
}

func (t *tScreen) Sync() {
	t.Lock()
	t.cx = -1
	t.cy = -1
	if !t.fini {
		t.resize()
		t.clear = true
		t.cells.Invalidate()
		t.draw()
	}
	t.Unlock()
}

func (t *tScreen) CharacterSet() string {
	return t.charset
}

func (t *tScreen) RegisterRuneFallback(orig rune, fallback string) {
	t.Lock()
	t.fallback[orig] = fallback
	t.Unlock()
}

func (t *tScreen) UnregisterRuneFallback(orig rune) {
	t.Lock()
	delete(t.fallback, orig)
	t.Unlock()
}

func (t *tScreen) CanDisplay(r rune, checkFallbacks bool) bool {

	if enc := t.encoder; enc != nil {
		nb := make([]byte, 6)
		ob := make([]byte, 6)
		num := utf8.EncodeRune(ob, r)

		enc.Reset()
		dst, _, err := enc.Transform(nb, ob[:num], true)
		if dst != 0 && err == nil && nb[0] != '\x1A' {
			return true
		}
	}
	// Terminal fallbacks always permitted, since we assume they are
	// basically nearly perfect renditions.
	if _, ok := t.acs[r]; ok {
		return true
	}
	if !checkFallbacks {
		return false
	}
	if _, ok := t.fallback[r]; ok {
		return true
	}
	return false
}

func (t *tScreen) HasMouse() bool {
	return len(t.mouse) != 0
}

func (t *tScreen) HasKey(k Key) bool {
	if k == KeyRune {
		return true
	}
	return t.keyexist[k]
}

func (t *tScreen) Resize(int, int, int, int) {}
