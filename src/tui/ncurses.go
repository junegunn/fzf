// +build ncurses
// +build !windows
// +build !tcell

package tui

/*
#include <ncurses.h>
#include <locale.h>
#cgo !static LDFLAGS: -lncurses
#cgo static LDFLAGS: -l:libncursesw.a -l:libtinfo.a -l:libgpm.a -ldl
#cgo android static LDFLAGS: -l:libncurses.a -fPIE -march=armv7-a -mfpu=neon -mhard-float -Wl,--no-warn-mismatch

FILE* c_tty() {
	return fopen("/dev/tty", "r");
}

SCREEN* c_newterm(FILE* tty) {
	return newterm(NULL, stderr, tty);
}

int c_getcurx(WINDOW* win) {
	return getcurx(win);
}
*/
import "C"

import (
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func HasFullscreenRenderer() bool {
	return true
}

type Attr C.uint

type CursesWindow struct {
	impl   *C.WINDOW
	top    int
	left   int
	width  int
	height int
}

func (w *CursesWindow) Top() int {
	return w.top
}

func (w *CursesWindow) Left() int {
	return w.left
}

func (w *CursesWindow) Width() int {
	return w.width
}

func (w *CursesWindow) Height() int {
	return w.height
}

func (w *CursesWindow) Refresh() {
	C.wnoutrefresh(w.impl)
}

func (w *CursesWindow) FinishFill() {
	// NO-OP
}

const (
	Bold      Attr = C.A_BOLD
	Dim            = C.A_DIM
	Blink          = C.A_BLINK
	Reverse        = C.A_REVERSE
	Underline      = C.A_UNDERLINE
)

var Italic Attr = C.A_VERTICAL << 1 // FIXME

const (
	AttrRegular Attr = 0
)

var (
	_screen   *C.SCREEN
	_colorMap map[int]int16
	_colorFn  func(ColorPair, Attr) (C.short, C.int)
)

func init() {
	_colorMap = make(map[int]int16)
	if strings.HasPrefix(C.GoString(C.curses_version()), "ncurses 5") {
		Italic = C.A_NORMAL
	}
}

func (a Attr) Merge(b Attr) Attr {
	return a | b
}

func (r *FullscreenRenderer) defaultTheme() *ColorTheme {
	if C.tigetnum(C.CString("colors")) >= 256 {
		return Dark256
	}
	return Default16
}

func (r *FullscreenRenderer) Init() {
	C.setlocale(C.LC_ALL, C.CString(""))
	tty := C.c_tty()
	if tty == nil {
		errorExit("Failed to open /dev/tty")
	}
	_screen = C.c_newterm(tty)
	if _screen == nil {
		errorExit("Invalid $TERM: " + os.Getenv("TERM"))
	}
	C.set_term(_screen)
	if r.mouse {
		C.mousemask(C.ALL_MOUSE_EVENTS, nil)
		C.mouseinterval(0)
	}
	C.noecho()
	C.raw() // stty dsusp undef
	C.nonl()
	C.keypad(C.stdscr, true)

	delay := 50
	delayEnv := os.Getenv("ESCDELAY")
	if len(delayEnv) > 0 {
		num, err := strconv.Atoi(delayEnv)
		if err == nil && num >= 0 {
			delay = num
		}
	}
	C.set_escdelay(C.int(delay))

	if r.theme != nil {
		C.start_color()
		initTheme(r.theme, r.defaultTheme(), r.forceBlack)
		initPairs(r.theme)
		C.bkgd(C.chtype(C.COLOR_PAIR(C.int(ColNormal.index()))))
		_colorFn = attrColored
	} else {
		initTheme(r.theme, nil, r.forceBlack)
		_colorFn = attrMono
	}

	C.nodelay(C.stdscr, true)
	ch := C.getch()
	if ch != C.ERR {
		C.ungetch(ch)
	}
	C.nodelay(C.stdscr, false)
}

func initPairs(theme *ColorTheme) {
	C.assume_default_colors(C.int(theme.Fg), C.int(theme.Bg))
	for _, pair := range []ColorPair{
		ColNormal,
		ColPrompt,
		ColMatch,
		ColCurrent,
		ColCurrentMatch,
		ColSpinner,
		ColInfo,
		ColCursor,
		ColSelected,
		ColHeader,
		ColBorder} {
		C.init_pair(C.short(pair.index()), C.short(pair.Fg()), C.short(pair.Bg()))
	}
}

func (r *FullscreenRenderer) Pause(bool) {
	C.endwin()
}

func (r *FullscreenRenderer) Resume(bool) {
}

func (r *FullscreenRenderer) Close() {
	C.endwin()
	C.delscreen(_screen)
}

func (r *FullscreenRenderer) NewWindow(top int, left int, width int, height int, borderStyle BorderStyle) Window {
	win := C.newwin(C.int(height), C.int(width), C.int(top), C.int(left))
	if r.theme != nil {
		C.wbkgd(win, C.chtype(C.COLOR_PAIR(C.int(ColNormal.index()))))
	}
	// FIXME Does not implement BorderHorizontal
	if borderStyle != BorderNone {
		pair, attr := _colorFn(ColBorder, 0)
		C.wcolor_set(win, pair, nil)
		C.wattron(win, attr)
		C.box(win, 0, 0)
		C.wattroff(win, attr)
		C.wcolor_set(win, 0, nil)
	}

	return &CursesWindow{
		impl:   win,
		top:    top,
		left:   left,
		width:  width,
		height: height,
	}
}

func attrColored(color ColorPair, a Attr) (C.short, C.int) {
	return C.short(color.index()), C.int(a)
}

func attrMono(color ColorPair, a Attr) (C.short, C.int) {
	return 0, C.int(attrFor(color, a))
}

func (r *FullscreenRenderer) MaxX() int {
	return int(C.COLS)
}

func (r *FullscreenRenderer) MaxY() int {
	return int(C.LINES)
}

func (w *CursesWindow) Close() {
	C.delwin(w.impl)
}

func (w *CursesWindow) Enclose(y int, x int) bool {
	return bool(C.wenclose(w.impl, C.int(y), C.int(x)))
}

func (w *CursesWindow) Move(y int, x int) {
	C.wmove(w.impl, C.int(y), C.int(x))
}

func (w *CursesWindow) MoveAndClear(y int, x int) {
	w.Move(y, x)
	C.wclrtoeol(w.impl)
}

func (w *CursesWindow) Print(text string) {
	C.waddstr(w.impl, C.CString(strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, text)))
}

func (w *CursesWindow) CPrint(color ColorPair, attr Attr, text string) {
	p, a := _colorFn(color, attr)
	C.wcolor_set(w.impl, p, nil)
	C.wattron(w.impl, a)
	w.Print(text)
	C.wattroff(w.impl, a)
	C.wcolor_set(w.impl, 0, nil)
}

func (r *FullscreenRenderer) Clear() {
	C.clear()
	C.endwin()
}

func (r *FullscreenRenderer) Refresh() {
	C.refresh()
}

func (w *CursesWindow) Erase() {
	C.werase(w.impl)
}

func (w *CursesWindow) X() int {
	return int(C.c_getcurx(w.impl))
}

func (r *FullscreenRenderer) DoesAutoWrap() bool {
	return true
}

func (r *FullscreenRenderer) IsOptimized() bool {
	return true
}

func (w *CursesWindow) Fill(str string) FillReturn {
	if C.waddstr(w.impl, C.CString(str)) == C.OK {
		return FillContinue
	}
	return FillSuspend
}

func (w *CursesWindow) CFill(fg Color, bg Color, attr Attr, str string) FillReturn {
	index := ColorPair{fg, bg, -1}.index()
	C.wcolor_set(w.impl, C.short(index), nil)
	C.wattron(w.impl, C.int(attr))
	ret := w.Fill(str)
	C.wattroff(w.impl, C.int(attr))
	C.wcolor_set(w.impl, 0, nil)
	return ret
}

func (r *FullscreenRenderer) RefreshWindows(windows []Window) {
	for _, w := range windows {
		w.Refresh()
	}
	C.doupdate()
}

func (p ColorPair) index() int16 {
	if p.id >= 0 {
		return p.id
	}

	// ncurses does not support 24-bit colors
	if p.is24() {
		return ColDefault.index()
	}

	key := p.key()
	if found, prs := _colorMap[key]; prs {
		return found
	}

	id := int16(len(_colorMap)) + ColUser.id
	C.init_pair(C.short(id), C.short(p.Fg()), C.short(p.Bg()))
	_colorMap[key] = id
	return id
}

func consume(expects ...rune) bool {
	for _, r := range expects {
		if int(C.getch()) != int(r) {
			return false
		}
	}
	return true
}

func escSequence() Event {
	C.nodelay(C.stdscr, true)
	defer func() {
		C.nodelay(C.stdscr, false)
	}()
	c := C.getch()
	switch c {
	case C.ERR:
		return Event{ESC, 0, nil}
	case CtrlM:
		return Event{CtrlAltM, 0, nil}
	case '/':
		return Event{AltSlash, 0, nil}
	case ' ':
		return Event{AltSpace, 0, nil}
	case 127, C.KEY_BACKSPACE:
		return Event{AltBS, 0, nil}
	case '[':
		// Bracketed paste mode (printf "\e[?2004h")
		// \e[200~ TEXT \e[201~
		if consume('2', '0', '0', '~') {
			return Event{Invalid, 0, nil}
		}
	}
	if c >= 'a' && c <= 'z' {
		return Event{AltA + int(c) - 'a', 0, nil}
	}

	if c >= '0' && c <= '9' {
		return Event{Alt0 + int(c) - '0', 0, nil}
	}

	// Don't care. Ignore the rest.
	for ; c != C.ERR; c = C.getch() {
	}
	return Event{Invalid, 0, nil}
}

func (r *FullscreenRenderer) GetChar() Event {
	c := C.getch()
	switch c {
	case C.ERR:
		// Unexpected error from blocking read
		r.Close()
		errorExit("Failed to read /dev/tty")
	case C.KEY_UP:
		return Event{Up, 0, nil}
	case C.KEY_DOWN:
		return Event{Down, 0, nil}
	case C.KEY_LEFT:
		return Event{Left, 0, nil}
	case C.KEY_RIGHT:
		return Event{Right, 0, nil}
	case C.KEY_HOME:
		return Event{Home, 0, nil}
	case C.KEY_END:
		return Event{End, 0, nil}
	case C.KEY_BACKSPACE:
		return Event{BSpace, 0, nil}
	case C.KEY_F0 + 1:
		return Event{F1, 0, nil}
	case C.KEY_F0 + 2:
		return Event{F2, 0, nil}
	case C.KEY_F0 + 3:
		return Event{F3, 0, nil}
	case C.KEY_F0 + 4:
		return Event{F4, 0, nil}
	case C.KEY_F0 + 5:
		return Event{F5, 0, nil}
	case C.KEY_F0 + 6:
		return Event{F6, 0, nil}
	case C.KEY_F0 + 7:
		return Event{F7, 0, nil}
	case C.KEY_F0 + 8:
		return Event{F8, 0, nil}
	case C.KEY_F0 + 9:
		return Event{F9, 0, nil}
	case C.KEY_F0 + 10:
		return Event{F10, 0, nil}
	case C.KEY_F0 + 11:
		return Event{F11, 0, nil}
	case C.KEY_F0 + 12:
		return Event{F12, 0, nil}
	case C.KEY_DC:
		return Event{Del, 0, nil}
	case C.KEY_PPAGE:
		return Event{PgUp, 0, nil}
	case C.KEY_NPAGE:
		return Event{PgDn, 0, nil}
	case C.KEY_BTAB:
		return Event{BTab, 0, nil}
	case C.KEY_ENTER:
		return Event{CtrlM, 0, nil}
	case C.KEY_SLEFT:
		return Event{SLeft, 0, nil}
	case C.KEY_SRIGHT:
		return Event{SRight, 0, nil}
	case C.KEY_MOUSE:
		var me C.MEVENT
		if C.getmouse(&me) != C.ERR {
			mod := ((me.bstate & C.BUTTON_SHIFT) | (me.bstate & C.BUTTON_CTRL) | (me.bstate & C.BUTTON_ALT)) > 0
			x := int(me.x)
			y := int(me.y)
			/* Cannot use BUTTON1_DOUBLE_CLICKED due to mouseinterval(0) */
			if (me.bstate & C.BUTTON1_PRESSED) > 0 {
				now := time.Now()
				if now.Sub(r.prevDownTime) < doubleClickDuration {
					r.clickY = append(r.clickY, y)
				} else {
					r.clickY = []int{y}
					r.prevDownTime = now
				}
				return Event{Mouse, 0, &MouseEvent{y, x, 0, true, false, mod}}
			} else if (me.bstate & C.BUTTON1_RELEASED) > 0 {
				double := false
				if len(r.clickY) > 1 && r.clickY[0] == r.clickY[1] &&
					time.Now().Sub(r.prevDownTime) < doubleClickDuration {
					double = true
				}
				return Event{Mouse, 0, &MouseEvent{y, x, 0, false, double, mod}}
			} else if (me.bstate&0x8000000) > 0 || (me.bstate&0x80) > 0 {
				return Event{Mouse, 0, &MouseEvent{y, x, -1, false, false, mod}}
			} else if (me.bstate & C.BUTTON4_PRESSED) > 0 {
				return Event{Mouse, 0, &MouseEvent{y, x, 1, false, false, mod}}
			}
		}
		return Event{Invalid, 0, nil}
	case C.KEY_RESIZE:
		return Event{Resize, 0, nil}
	case ESC:
		return escSequence()
	case 127:
		return Event{BSpace, 0, nil}
	case 0:
		return Event{CtrlSpace, 0, nil}
	}
	// CTRL-A ~ CTRL-Z
	if c >= CtrlA && c <= CtrlZ {
		return Event{int(c), 0, nil}
	}

	// Multi-byte character
	buffer := []byte{byte(c)}
	for {
		r, _ := utf8.DecodeRune(buffer)
		if r != utf8.RuneError {
			return Event{Rune, r, nil}
		}

		c := C.getch()
		if c == C.ERR {
			break
		}
		if c >= C.KEY_CODE_YES {
			C.ungetch(c)
			break
		}
		buffer = append(buffer, byte(c))
	}
	return Event{Invalid, 0, nil}
}
