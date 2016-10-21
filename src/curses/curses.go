package curses

/*
#include <ncurses.h>
#include <locale.h>
#cgo !static LDFLAGS: -lncurses
#cgo static LDFLAGS: -l:libncursesw.a -l:libtinfo.a -l:libgpm.a -ldl
#cgo android static LDFLAGS: -l:libncurses.a -fPIE -march=armv7-a -mfpu=neon -mhard-float -Wl,--no-warn-mismatch

SCREEN *c_newterm () {
	return newterm(NULL, stderr, stdin);
}

*/
import "C"

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	Bold      = C.A_BOLD
	Dim       = C.A_DIM
	Blink     = C.A_BLINK
	Reverse   = C.A_REVERSE
	Underline = C.A_UNDERLINE
)

type Attr C.int

// Types of user action
const (
	Rune = iota

	CtrlA
	CtrlB
	CtrlC
	CtrlD
	CtrlE
	CtrlF
	CtrlG
	CtrlH
	Tab
	CtrlJ
	CtrlK
	CtrlL
	CtrlM
	CtrlN
	CtrlO
	CtrlP
	CtrlQ
	CtrlR
	CtrlS
	CtrlT
	CtrlU
	CtrlV
	CtrlW
	CtrlX
	CtrlY
	CtrlZ
	ESC

	Invalid
	Mouse
	DoubleClick

	BTab
	BSpace

	Del
	PgUp
	PgDn

	Up
	Down
	Left
	Right
	Home
	End

	SLeft
	SRight

	F1
	F2
	F3
	F4
	F5
	F6
	F7
	F8
	F9
	F10

	AltEnter
	AltSpace
	AltSlash
	AltBS
	AltA
	AltB
	AltC
	AltD
	AltE
	AltF

	AltZ = AltA + 'z' - 'a'
)

// Pallete
const (
	_ = iota
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

const (
	doubleClickDuration = 500 * time.Millisecond
	colDefault          = -1
	colUndefined        = -2
)

type ColorTheme struct {
	Fg           int16
	Bg           int16
	DarkBg       int16
	Prompt       int16
	Match        int16
	Current      int16
	CurrentMatch int16
	Spinner      int16
	Info         int16
	Cursor       int16
	Selected     int16
	Header       int16
	Border       int16
}

type Event struct {
	Type       int
	Char       rune
	MouseEvent *MouseEvent
}

type MouseEvent struct {
	Y      int
	X      int
	S      int
	Down   bool
	Double bool
	Mod    bool
}

var (
	_buf          []byte
	_in           *os.File
	_color        bool
	_colorFn      func(int, Attr) C.int
	_colorMap     map[int]int
	_prevDownTime time.Time
	_clickY       []int
	_screen       *C.SCREEN
	Default16     *ColorTheme
	Dark256       *ColorTheme
	Light256      *ColorTheme
)

type Window struct {
	win    *C.WINDOW
	Top    int
	Left   int
	Width  int
	Height int
}

func NewWindow(top int, left int, width int, height int, border bool) *Window {
	win := C.newwin(C.int(height), C.int(width), C.int(top), C.int(left))
	if _color {
		C.wbkgd(win, C.chtype(C.COLOR_PAIR(ColNormal)))
	}
	if border {
		attr := _colorFn(ColBorder, 0)
		C.wattron(win, attr)
		C.box(win, 0, 0)
		C.wattroff(win, attr)
	}

	return &Window{
		win:    win,
		Top:    top,
		Left:   left,
		Width:  width,
		Height: height,
	}
}

func EmptyTheme() *ColorTheme {
	return &ColorTheme{
		Fg:           colUndefined,
		Bg:           colUndefined,
		DarkBg:       colUndefined,
		Prompt:       colUndefined,
		Match:        colUndefined,
		Current:      colUndefined,
		CurrentMatch: colUndefined,
		Spinner:      colUndefined,
		Info:         colUndefined,
		Cursor:       colUndefined,
		Selected:     colUndefined,
		Header:       colUndefined,
		Border:       colUndefined}
}

func init() {
	_prevDownTime = time.Unix(0, 0)
	_clickY = []int{}
	_colorMap = make(map[int]int)
	Default16 = &ColorTheme{
		Fg:           colDefault,
		Bg:           colDefault,
		DarkBg:       C.COLOR_BLACK,
		Prompt:       C.COLOR_BLUE,
		Match:        C.COLOR_GREEN,
		Current:      C.COLOR_YELLOW,
		CurrentMatch: C.COLOR_GREEN,
		Spinner:      C.COLOR_GREEN,
		Info:         C.COLOR_WHITE,
		Cursor:       C.COLOR_RED,
		Selected:     C.COLOR_MAGENTA,
		Header:       C.COLOR_CYAN,
		Border:       C.COLOR_BLACK}
	Dark256 = &ColorTheme{
		Fg:           colDefault,
		Bg:           colDefault,
		DarkBg:       236,
		Prompt:       110,
		Match:        108,
		Current:      254,
		CurrentMatch: 151,
		Spinner:      148,
		Info:         144,
		Cursor:       161,
		Selected:     168,
		Header:       109,
		Border:       59}
	Light256 = &ColorTheme{
		Fg:           colDefault,
		Bg:           colDefault,
		DarkBg:       251,
		Prompt:       25,
		Match:        66,
		Current:      237,
		CurrentMatch: 23,
		Spinner:      65,
		Info:         101,
		Cursor:       161,
		Selected:     168,
		Header:       31,
		Border:       145}
}

func attrColored(pair int, a Attr) C.int {
	var attr C.int
	if pair > 0 {
		attr = C.COLOR_PAIR(C.int(pair))
	}
	return attr | C.int(a)
}

func attrMono(pair int, a Attr) C.int {
	var attr C.int
	switch pair {
	case ColCurrent:
		if a&C.A_BOLD == C.A_BOLD {
			attr = C.A_REVERSE
		}
	case ColMatch:
		attr = C.A_UNDERLINE
	case ColCurrentMatch:
		attr = C.A_UNDERLINE | C.A_REVERSE
	}
	if a&C.A_BOLD == C.A_BOLD {
		attr = attr | C.A_BOLD
	}
	return attr
}

func MaxX() int {
	return int(C.COLS)
}

func MaxY() int {
	return int(C.LINES)
}

func getch(nonblock bool) int {
	b := make([]byte, 1)
	syscall.SetNonblock(int(_in.Fd()), nonblock)
	_, err := _in.Read(b)
	if err != nil {
		return -1
	}
	return int(b[0])
}

func Init(theme *ColorTheme, black bool, mouse bool) {
	{
		in, err := os.OpenFile("/dev/tty", syscall.O_RDONLY, 0)
		if err != nil {
			panic("Failed to open /dev/tty")
		}
		_in = in
		// Break STDIN
		// syscall.Dup2(int(in.Fd()), int(os.Stdin.Fd()))
	}

	C.setlocale(C.LC_ALL, C.CString(""))
	_screen = C.c_newterm()
	if _screen == nil {
		fmt.Println("Invalid $TERM: " + os.Getenv("TERM"))
		os.Exit(2)
	}
	C.set_term(_screen)
	if mouse {
		C.mousemask(C.ALL_MOUSE_EVENTS, nil)
	}
	C.noecho()
	C.raw() // stty dsusp undef

	_color = theme != nil
	if _color {
		C.start_color()
		var baseTheme *ColorTheme
		if C.tigetnum(C.CString("colors")) >= 256 {
			baseTheme = Dark256
		} else {
			baseTheme = Default16
		}
		initPairs(baseTheme, theme, black)
		C.bkgd(C.chtype(C.COLOR_PAIR(ColNormal)))
		_colorFn = attrColored
	} else {
		_colorFn = attrMono
	}
}

func override(baseTheme *ColorTheme, theme *ColorTheme) {
	o := func(a int16, b int16) int16 {
		if b == colUndefined {
			return a
		}
		return b
	}
	theme.Fg = o(baseTheme.Fg, theme.Fg)
	theme.Bg = o(baseTheme.Bg, theme.Bg)
	theme.DarkBg = o(baseTheme.DarkBg, theme.DarkBg)
	theme.Prompt = o(baseTheme.Prompt, theme.Prompt)
	theme.Match = o(baseTheme.Match, theme.Match)
	theme.Current = o(baseTheme.Current, theme.Current)
	theme.CurrentMatch = o(baseTheme.CurrentMatch, theme.CurrentMatch)
	theme.Spinner = o(baseTheme.Spinner, theme.Spinner)
	theme.Info = o(baseTheme.Info, theme.Info)
	theme.Cursor = o(baseTheme.Cursor, theme.Cursor)
	theme.Selected = o(baseTheme.Selected, theme.Selected)
	theme.Header = o(baseTheme.Header, theme.Header)
	theme.Border = o(baseTheme.Border, theme.Border)
}

func initPairs(baseTheme *ColorTheme, theme *ColorTheme, black bool) {
	if black {
		theme.Bg = C.COLOR_BLACK
	}
	// Updates theme
	override(baseTheme, theme)

	C.assume_default_colors(C.int(theme.Fg), C.int(theme.Bg))
	initPair := func(group C.short, fg int16, bg int16) {
		C.init_pair(group, C.short(fg), C.short(bg))
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

func Close() {
	C.endwin()
	C.delscreen(_screen)
}

func GetBytes() []byte {
	c := getch(false)
	_buf = append(_buf, byte(c))

	for {
		c = getch(true)
		if c == -1 {
			break
		}
		_buf = append(_buf, byte(c))
	}

	return _buf
}

// 27 (91 79) 77 type x y
func mouseSequence(sz *int) Event {
	if len(_buf) < 6 {
		return Event{Invalid, 0, nil}
	}
	*sz = 6
	switch _buf[3] {
	case 32, 36, 40, 48, // mouse-down / shift / cmd / ctrl
		35, 39, 43, 51: // mouse-up / shift / cmd / ctrl
		mod := _buf[3] >= 36
		down := _buf[3]%2 == 0
		x := int(_buf[4] - 33)
		y := int(_buf[5] - 33)
		double := false
		if down {
			now := time.Now()
			if now.Sub(_prevDownTime) < doubleClickDuration {
				_clickY = append(_clickY, y)
			} else {
				_clickY = []int{y}
			}
			_prevDownTime = now
		} else {
			if len(_clickY) > 1 && _clickY[0] == _clickY[1] &&
				time.Now().Sub(_prevDownTime) < doubleClickDuration {
				double = true
			}
		}
		return Event{Mouse, 0, &MouseEvent{y, x, 0, down, double, mod}}
	case 96, 100, 104, 112, // scroll-up / shift / cmd / ctrl
		97, 101, 105, 113: // scroll-down / shift / cmd / ctrl
		mod := _buf[3] >= 100
		s := 1 - int(_buf[3]%2)*2
		x := int(_buf[4] - 33)
		y := int(_buf[5] - 33)
		return Event{Mouse, 0, &MouseEvent{y, x, s, false, false, mod}}
	}
	return Event{Invalid, 0, nil}
}

func escSequence(sz *int) Event {
	if len(_buf) < 2 {
		return Event{ESC, 0, nil}
	}
	*sz = 2
	switch _buf[1] {
	case 13:
		return Event{AltEnter, 0, nil}
	case 32:
		return Event{AltSpace, 0, nil}
	case 47:
		return Event{AltSlash, 0, nil}
	case 98:
		return Event{AltB, 0, nil}
	case 100:
		return Event{AltD, 0, nil}
	case 102:
		return Event{AltF, 0, nil}
	case 127:
		return Event{AltBS, 0, nil}
	case 91, 79:
		if len(_buf) < 3 {
			return Event{Invalid, 0, nil}
		}
		*sz = 3
		switch _buf[2] {
		case 68:
			return Event{Left, 0, nil}
		case 67:
			return Event{Right, 0, nil}
		case 66:
			return Event{Down, 0, nil}
		case 65:
			return Event{Up, 0, nil}
		case 90:
			return Event{BTab, 0, nil}
		case 72:
			return Event{Home, 0, nil}
		case 70:
			return Event{End, 0, nil}
		case 77:
			return mouseSequence(sz)
		case 80:
			return Event{F1, 0, nil}
		case 81:
			return Event{F2, 0, nil}
		case 82:
			return Event{F3, 0, nil}
		case 83:
			return Event{F4, 0, nil}
		case 49, 50, 51, 52, 53, 54:
			if len(_buf) < 4 {
				return Event{Invalid, 0, nil}
			}
			*sz = 4
			switch _buf[2] {
			case 50:
				if len(_buf) == 5 && _buf[4] == 126 {
					*sz = 5
					switch _buf[3] {
					case 48:
						return Event{F9, 0, nil}
					case 49:
						return Event{F10, 0, nil}
					}
				}
				// Bracketed paste mode \e[200~ / \e[201
				if _buf[3] == 48 && (_buf[4] == 48 || _buf[4] == 49) && _buf[5] == 126 {
					*sz = 6
					return Event{Invalid, 0, nil}
				}
				return Event{Invalid, 0, nil} // INS
			case 51:
				return Event{Del, 0, nil}
			case 52:
				return Event{End, 0, nil}
			case 53:
				return Event{PgUp, 0, nil}
			case 54:
				return Event{PgDn, 0, nil}
			case 49:
				switch _buf[3] {
				case 126:
					return Event{Home, 0, nil}
				case 53, 55, 56, 57:
					if len(_buf) == 5 && _buf[4] == 126 {
						*sz = 5
						switch _buf[3] {
						case 53:
							return Event{F5, 0, nil}
						case 55:
							return Event{F6, 0, nil}
						case 56:
							return Event{F7, 0, nil}
						case 57:
							return Event{F8, 0, nil}
						}
					}
					return Event{Invalid, 0, nil}
				case 59:
					if len(_buf) != 6 {
						return Event{Invalid, 0, nil}
					}
					*sz = 6
					switch _buf[4] {
					case 50:
						switch _buf[5] {
						case 68:
							return Event{Home, 0, nil}
						case 67:
							return Event{End, 0, nil}
						}
					case 53:
						switch _buf[5] {
						case 68:
							return Event{SLeft, 0, nil}
						case 67:
							return Event{SRight, 0, nil}
						}
					} // _buf[4]
				} // _buf[3]
			} // _buf[2]
		} // _buf[2]
	} // _buf[1]
	if _buf[1] >= 'a' && _buf[1] <= 'z' {
		return Event{AltA + int(_buf[1]) - 'a', 0, nil}
	}
	return Event{Invalid, 0, nil}
}

func GetChar() Event {
	if len(_buf) == 0 {
		_buf = GetBytes()
	}
	if len(_buf) == 0 {
		panic("Empty _buffer")
	}

	sz := 1
	defer func() {
		_buf = _buf[sz:]
	}()

	switch _buf[0] {
	case CtrlC:
		return Event{CtrlC, 0, nil}
	case CtrlG:
		return Event{CtrlG, 0, nil}
	case CtrlQ:
		return Event{CtrlQ, 0, nil}
	case 127:
		return Event{BSpace, 0, nil}
	case ESC:
		return escSequence(&sz)
	}

	// CTRL-A ~ CTRL-Z
	if _buf[0] <= CtrlZ {
		return Event{int(_buf[0]), 0, nil}
	}
	r, rsz := utf8.DecodeRune(_buf)
	if r == utf8.RuneError {
		return Event{ESC, 0, nil}
	}
	sz = rsz
	return Event{Rune, r, nil}
}

func (w *Window) Close() {
	C.delwin(w.win)
}

func (w *Window) Enclose(y int, x int) bool {
	return bool(C.wenclose(w.win, C.int(y), C.int(x)))
}

func (w *Window) Move(y int, x int) {
	C.wmove(w.win, C.int(y), C.int(x))
}

func (w *Window) MoveAndClear(y int, x int) {
	w.Move(y, x)
	C.wclrtoeol(w.win)
}

func (w *Window) Print(text string) {
	C.waddstr(w.win, C.CString(strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, text)))
}

func (w *Window) CPrint(pair int, a Attr, text string) {
	attr := _colorFn(pair, a)
	C.wattron(w.win, attr)
	w.Print(text)
	C.wattroff(w.win, attr)
}

func Clear() {
	C.clear()
}

func Endwin() {
	C.endwin()
}

func Refresh() {
	C.refresh()
}

func (w *Window) Erase() {
	C.werase(w.win)
}

func (w *Window) Fill(str string) bool {
	return C.waddstr(w.win, C.CString(str)) == C.OK
}

func (w *Window) CFill(str string, fg int, bg int, a Attr) bool {
	attr := _colorFn(PairFor(fg, bg), a)
	C.wattron(w.win, attr)
	ret := w.Fill(str)
	C.wattroff(w.win, attr)
	return ret
}

func (w *Window) Refresh() {
	C.wnoutrefresh(w.win)
}

func DoUpdate() {
	C.doupdate()
}

func PairFor(fg int, bg int) int {
	key := (fg << 8) + bg
	if found, prs := _colorMap[key]; prs {
		return found
	}

	id := len(_colorMap) + ColUser
	C.init_pair(C.short(id), C.short(fg), C.short(bg))
	_colorMap[key] = id
	return id
}
