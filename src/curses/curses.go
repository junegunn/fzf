package curses

/*
#include <ncurses.h>
#include <locale.h>
#cgo LDFLAGS: -lncurses
void swapOutput() {
  FILE* temp = stdout;
  stdout = stderr;
  stderr = temp;
}
*/
import "C"

import (
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicode/utf8"
)

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

	BTab

	Del
	PgUp
	PgDn

	AltB
	AltF
	AltD
	AltBS
)

// Pallete
const (
	ColNormal = iota
	ColPrompt
	ColMatch
	ColCurrent
	ColCurrentMatch
	ColSpinner
	ColInfo
	ColCursor
	ColSelected
	ColUser
)

const (
	doubleClickDuration = 500 * time.Millisecond
)

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
	_color        func(int, bool) C.int
	_colorMap     map[int]int
	_prevDownTime time.Time
	_prevDownY    int
	_clickY       []int
	DarkBG        C.short
)

func init() {
	_prevDownTime = time.Unix(0, 0)
	_clickY = []int{}
	_colorMap = make(map[int]int)
}

func attrColored(pair int, bold bool) C.int {
	var attr C.int
	if pair > ColNormal {
		attr = C.COLOR_PAIR(C.int(pair))
	}
	if bold {
		attr = attr | C.A_BOLD
	}
	return attr
}

func attrMono(pair int, bold bool) C.int {
	var attr C.int
	switch pair {
	case ColCurrent:
		if bold {
			attr = C.A_REVERSE
		}
	case ColMatch:
		attr = C.A_UNDERLINE
	case ColCurrentMatch:
		attr = C.A_UNDERLINE | C.A_REVERSE
	}
	if bold {
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

func Init(color bool, color256 bool, black bool, mouse bool) {
	{
		in, err := os.OpenFile("/dev/tty", syscall.O_RDONLY, 0)
		if err != nil {
			panic("Failed to open /dev/tty")
		}
		_in = in
		// Break STDIN
		// syscall.Dup2(int(in.Fd()), int(os.Stdin.Fd()))
	}

	C.swapOutput()

	C.setlocale(C.LC_ALL, C.CString(""))
	C.initscr()
	if mouse {
		C.mousemask(C.ALL_MOUSE_EVENTS, nil)
	}
	C.cbreak()
	C.noecho()
	C.raw() // stty dsusp undef

	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, os.Kill)
	go func() {
		<-intChan
		Close()
		os.Exit(1)
	}()

	if color {
		C.start_color()
		var bg C.short
		if black {
			bg = C.COLOR_BLACK
		} else {
			C.use_default_colors()
			bg = -1
		}
		if color256 {
			DarkBG = 236
			C.init_pair(ColPrompt, 110, bg)
			C.init_pair(ColMatch, 108, bg)
			C.init_pair(ColCurrent, 254, DarkBG)
			C.init_pair(ColCurrentMatch, 151, DarkBG)
			C.init_pair(ColSpinner, 148, bg)
			C.init_pair(ColInfo, 144, bg)
			C.init_pair(ColCursor, 161, DarkBG)
			C.init_pair(ColSelected, 168, DarkBG)
		} else {
			DarkBG = C.COLOR_BLACK
			C.init_pair(ColPrompt, C.COLOR_BLUE, bg)
			C.init_pair(ColMatch, C.COLOR_GREEN, bg)
			C.init_pair(ColCurrent, C.COLOR_YELLOW, DarkBG)
			C.init_pair(ColCurrentMatch, C.COLOR_GREEN, DarkBG)
			C.init_pair(ColSpinner, C.COLOR_GREEN, bg)
			C.init_pair(ColInfo, C.COLOR_WHITE, bg)
			C.init_pair(ColCursor, C.COLOR_RED, DarkBG)
			C.init_pair(ColSelected, C.COLOR_MAGENTA, DarkBG)
		}
		_color = attrColored
	} else {
		_color = attrMono
	}
}

func Close() {
	C.endwin()
	C.swapOutput()
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
		return Event{Mouse, 0, &MouseEvent{0, 0, s, false, false, mod}}
	}
	return Event{Invalid, 0, nil}
}

func escSequence(sz *int) Event {
	if len(_buf) < 2 {
		return Event{ESC, 0, nil}
	}
	*sz = 2
	switch _buf[1] {
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
			return Event{CtrlB, 0, nil}
		case 67:
			return Event{CtrlF, 0, nil}
		case 66:
			return Event{CtrlJ, 0, nil}
		case 65:
			return Event{CtrlK, 0, nil}
		case 90:
			return Event{BTab, 0, nil}
		case 72:
			return Event{CtrlA, 0, nil}
		case 70:
			return Event{CtrlE, 0, nil}
		case 77:
			return mouseSequence(sz)
		case 49, 50, 51, 52, 53, 54:
			if len(_buf) < 4 {
				return Event{Invalid, 0, nil}
			}
			*sz = 4
			switch _buf[2] {
			case 50:
				return Event{Invalid, 0, nil} // INS
			case 51:
				return Event{Del, 0, nil}
			case 52:
				return Event{CtrlE, 0, nil}
			case 53:
				return Event{PgUp, 0, nil}
			case 54:
				return Event{PgDn, 0, nil}
			case 49:
				switch _buf[3] {
				case 126:
					return Event{CtrlA, 0, nil}
				case 59:
					if len(_buf) != 6 {
						return Event{Invalid, 0, nil}
					}
					*sz = 6
					switch _buf[4] {
					case 50:
						switch _buf[5] {
						case 68:
							return Event{CtrlA, 0, nil}
						case 67:
							return Event{CtrlE, 0, nil}
						}
					case 53:
						switch _buf[5] {
						case 68:
							return Event{AltB, 0, nil}
						case 67:
							return Event{AltF, 0, nil}
						}
					} // _buf[4]
				} // _buf[3]
			} // _buf[2]
		} // _buf[2]
	} // _buf[1]
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
	case CtrlC, CtrlG, CtrlQ:
		return Event{CtrlC, 0, nil}
	case 127:
		return Event{CtrlH, 0, nil}
	case ESC:
		return escSequence(&sz)
	}

	// CTRL-A ~ CTRL-Z
	if _buf[0] <= CtrlZ {
		return Event{int(_buf[0]), 0, nil}
	}
	r, rsz := utf8.DecodeRune(_buf)
	sz = rsz
	return Event{Rune, r, nil}
}

func Move(y int, x int) {
	C.move(C.int(y), C.int(x))
}

func MoveAndClear(y int, x int) {
	Move(y, x)
	C.clrtoeol()
}

func Print(text string) {
	C.addstr(C.CString(text))
}

func CPrint(pair int, bold bool, text string) {
	attr := _color(pair, bold)
	C.attron(attr)
	Print(text)
	C.attroff(attr)
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
