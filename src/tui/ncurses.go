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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type ColorPair int16
type Attr C.uint
type WindowImpl C.WINDOW

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

// Pallete
const (
	ColDefault ColorPair = iota
	ColNormal
	ColPrompt
	ColMatch
	ColCurrent
	ColCurrentMatch
	ColSpinner
	ColInfo
	ColCursor
	ColSelected
	ColHeader
	ColBorder
	ColUser // Should be the last entry
)

var (
	_screen   *C.SCREEN
	_colorMap map[int]ColorPair
	_colorFn  func(ColorPair, Attr) (C.short, C.int)
)

func init() {
	_colorMap = make(map[int]ColorPair)
	if strings.HasPrefix(C.GoString(C.curses_version()), "ncurses 5") {
		Italic = C.A_NORMAL
	}
}

func (a Attr) Merge(b Attr) Attr {
	return a | b
}

func DefaultTheme() *ColorTheme {
	if C.tigetnum(C.CString("colors")) >= 256 {
		return Dark256
	}
	return Default16
}

func Init(theme *ColorTheme, black bool, mouse bool) {
	C.setlocale(C.LC_ALL, C.CString(""))
	tty := C.c_tty()
	if tty == nil {
		fmt.Println("Failed to open /dev/tty")
		os.Exit(2)
	}
	_screen = C.c_newterm(tty)
	if _screen == nil {
		fmt.Println("Invalid $TERM: " + os.Getenv("TERM"))
		os.Exit(2)
	}
	C.set_term(_screen)
	if mouse {
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

	_color = theme != nil
	if _color {
		C.start_color()
		InitTheme(theme, black)
		initPairs(theme)
		C.bkgd(C.chtype(C.COLOR_PAIR(C.int(ColNormal))))
		_colorFn = attrColored
	} else {
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
	initPair := func(group ColorPair, fg Color, bg Color) {
		C.init_pair(C.short(group), C.short(fg), C.short(bg))
	}
	initPair(ColNormal, theme.Fg, theme.Bg)
	initPair(ColPrompt, theme.Prompt, theme.Bg)
	initPair(ColMatch, theme.Match, theme.Bg)
	initPair(ColCurrent, theme.Current, theme.DarkBg)
	initPair(ColCurrentMatch, theme.CurrentMatch, theme.DarkBg)
	initPair(ColSpinner, theme.Spinner, theme.Bg)
	initPair(ColInfo, theme.Info, theme.Bg)
	initPair(ColCursor, theme.Cursor, theme.DarkBg)
	initPair(ColSelected, theme.Selected, theme.DarkBg)
	initPair(ColHeader, theme.Header, theme.Bg)
	initPair(ColBorder, theme.Border, theme.Bg)
}

func Pause() {
	C.endwin()
}

func Resume() bool {
	return false
}

func Close() {
	C.endwin()
	C.delscreen(_screen)
}

func NewWindow(top int, left int, width int, height int, border bool) *Window {
	win := C.newwin(C.int(height), C.int(width), C.int(top), C.int(left))
	if _color {
		C.wbkgd(win, C.chtype(C.COLOR_PAIR(C.int(ColNormal))))
	}
	if border {
		pair, attr := _colorFn(ColBorder, 0)
		C.wcolor_set(win, pair, nil)
		C.wattron(win, attr)
		C.box(win, 0, 0)
		C.wattroff(win, attr)
		C.wcolor_set(win, 0, nil)
	}

	return &Window{
		impl:   (*WindowImpl)(win),
		Top:    top,
		Left:   left,
		Width:  width,
		Height: height,
	}
}

func attrColored(pair ColorPair, a Attr) (C.short, C.int) {
	return C.short(pair), C.int(a)
}

func attrMono(pair ColorPair, a Attr) (C.short, C.int) {
	var attr C.int
	switch pair {
	case ColCurrent:
		attr = C.A_REVERSE
	case ColMatch:
		attr = C.A_UNDERLINE
	case ColCurrentMatch:
		attr = C.A_UNDERLINE | C.A_REVERSE
	}
	if C.int(a)&C.A_BOLD == C.A_BOLD {
		attr = attr | C.A_BOLD
	}
	return 0, attr
}

func MaxX() int {
	return int(C.COLS)
}

func MaxY() int {
	return int(C.LINES)
}

func (w *Window) win() *C.WINDOW {
	return (*C.WINDOW)(w.impl)
}

func (w *Window) Close() {
	C.delwin(w.win())
}

func (w *Window) Enclose(y int, x int) bool {
	return bool(C.wenclose(w.win(), C.int(y), C.int(x)))
}

func (w *Window) Move(y int, x int) {
	C.wmove(w.win(), C.int(y), C.int(x))
}

func (w *Window) MoveAndClear(y int, x int) {
	w.Move(y, x)
	C.wclrtoeol(w.win())
}

func (w *Window) Print(text string) {
	C.waddstr(w.win(), C.CString(strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, text)))
}

func (w *Window) CPrint(pair ColorPair, attr Attr, text string) {
	p, a := _colorFn(pair, attr)
	C.wcolor_set(w.win(), p, nil)
	C.wattron(w.win(), a)
	w.Print(text)
	C.wattroff(w.win(), a)
	C.wcolor_set(w.win(), 0, nil)
}

func Clear() {
	C.clear()
	C.endwin()
}

func Refresh() {
	C.refresh()
}

func (w *Window) Erase() {
	C.werase(w.win())
}

func (w *Window) X() int {
	return int(C.c_getcurx(w.win()))
}

func DoesAutoWrap() bool {
	return true
}

func (w *Window) Fill(str string) bool {
	return C.waddstr(w.win(), C.CString(str)) == C.OK
}

func (w *Window) CFill(str string, fg Color, bg Color, attr Attr) bool {
	pair := PairFor(fg, bg)
	C.wcolor_set(w.win(), C.short(pair), nil)
	C.wattron(w.win(), C.int(attr))
	ret := w.Fill(str)
	C.wattroff(w.win(), C.int(attr))
	C.wcolor_set(w.win(), 0, nil)
	return ret
}

func RefreshWindows(windows []*Window) {
	for _, w := range windows {
		C.wnoutrefresh(w.win())
	}
	C.doupdate()
}

func PairFor(fg Color, bg Color) ColorPair {
	// ncurses does not support 24-bit colors
	if fg.is24() || bg.is24() {
		return ColDefault
	}
	key := (int(fg) << 8) + int(bg)
	if found, prs := _colorMap[key]; prs {
		return found
	}

	id := ColorPair(len(_colorMap) + int(ColUser))
	C.init_pair(C.short(id), C.short(fg), C.short(bg))
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
		return Event{AltEnter, 0, nil}
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

func GetChar() Event {
	c := C.getch()
	switch c {
	case C.ERR:
		return Event{Invalid, 0, nil}
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
				if now.Sub(_prevDownTime) < doubleClickDuration {
					_clickY = append(_clickY, y)
				} else {
					_clickY = []int{y}
					_prevDownTime = now
				}
				return Event{Mouse, 0, &MouseEvent{y, x, 0, true, false, mod}}
			} else if (me.bstate & C.BUTTON1_RELEASED) > 0 {
				double := false
				if len(_clickY) > 1 && _clickY[0] == _clickY[1] &&
					time.Now().Sub(_prevDownTime) < doubleClickDuration {
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
