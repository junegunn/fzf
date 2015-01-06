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

const (
	RUNE = iota

	CTRL_A
	CTRL_B
	CTRL_C
	CTRL_D
	CTRL_E
	CTRL_F
	CTRL_G
	CTRL_H
	TAB
	CTRL_J
	CTRL_K
	CTRL_L
	CTRL_M
	CTRL_N
	CTRL_O
	CTRL_P
	CTRL_Q
	CTRL_R
	CTRL_S
	CTRL_T
	CTRL_U
	CTRL_V
	CTRL_W
	CTRL_X
	CTRL_Y
	CTRL_Z
	ESC

	INVALID
	MOUSE

	BTAB

	DEL
	PGUP
	PGDN

	ALT_B
	ALT_F
	ALT_D
	ALT_BS
)

const (
	COL_NORMAL = iota
	COL_PROMPT
	COL_MATCH
	COL_CURRENT
	COL_CURRENT_MATCH
	COL_SPINNER
	COL_INFO
	COL_CURSOR
	COL_SELECTED
)

const (
	DOUBLE_CLICK_DURATION = 500 * time.Millisecond
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
	_prevDownTime time.Time
	_prevDownY    int
	_clickY       []int
)

func init() {
	_prevDownTime = time.Unix(0, 0)
	_clickY = []int{}
}

func attrColored(pair int, bold bool) C.int {
	var attr C.int = 0
	if pair > COL_NORMAL {
		attr = C.COLOR_PAIR(C.int(pair))
	}
	if bold {
		attr = attr | C.A_BOLD
	}
	return attr
}

func attrMono(pair int, bold bool) C.int {
	var attr C.int = 0
	switch pair {
	case COL_CURRENT:
		if bold {
			attr = C.A_REVERSE
		}
	case COL_MATCH:
		attr = C.A_UNDERLINE
	case COL_CURRENT_MATCH:
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
	C.raw()          // stty dsusp undef
	C.set_tabsize(4) // FIXME

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
			C.init_pair(COL_PROMPT, 110, bg)
			C.init_pair(COL_MATCH, 108, bg)
			C.init_pair(COL_CURRENT, 254, 236)
			C.init_pair(COL_CURRENT_MATCH, 151, 236)
			C.init_pair(COL_SPINNER, 148, bg)
			C.init_pair(COL_INFO, 144, bg)
			C.init_pair(COL_CURSOR, 161, 236)
			C.init_pair(COL_SELECTED, 168, 236)
		} else {
			C.init_pair(COL_PROMPT, C.COLOR_BLUE, bg)
			C.init_pair(COL_MATCH, C.COLOR_GREEN, bg)
			C.init_pair(COL_CURRENT, C.COLOR_YELLOW, C.COLOR_BLACK)
			C.init_pair(COL_CURRENT_MATCH, C.COLOR_GREEN, C.COLOR_BLACK)
			C.init_pair(COL_SPINNER, C.COLOR_GREEN, bg)
			C.init_pair(COL_INFO, C.COLOR_WHITE, bg)
			C.init_pair(COL_CURSOR, C.COLOR_RED, C.COLOR_BLACK)
			C.init_pair(COL_SELECTED, C.COLOR_MAGENTA, C.COLOR_BLACK)
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
		return Event{INVALID, 0, nil}
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
			if now.Sub(_prevDownTime) < DOUBLE_CLICK_DURATION {
				_clickY = append(_clickY, y)
			} else {
				_clickY = []int{y}
			}
			_prevDownTime = now
		} else {
			if len(_clickY) > 1 && _clickY[0] == _clickY[1] &&
				time.Now().Sub(_prevDownTime) < DOUBLE_CLICK_DURATION {
				double = true
			}
		}
		return Event{MOUSE, 0, &MouseEvent{y, x, 0, down, double, mod}}
	case 96, 100, 104, 112, // scroll-up / shift / cmd / ctrl
		97, 101, 105, 113: // scroll-down / shift / cmd / ctrl
		mod := _buf[3] >= 100
		s := 1 - int(_buf[3]%2)*2
		return Event{MOUSE, 0, &MouseEvent{0, 0, s, false, false, mod}}
	}
	return Event{INVALID, 0, nil}
}

func escSequence(sz *int) Event {
	if len(_buf) < 2 {
		return Event{ESC, 0, nil}
	}
	*sz = 2
	switch _buf[1] {
	case 98:
		return Event{ALT_B, 0, nil}
	case 100:
		return Event{ALT_D, 0, nil}
	case 102:
		return Event{ALT_F, 0, nil}
	case 127:
		return Event{ALT_BS, 0, nil}
	case 91, 79:
		if len(_buf) < 3 {
			return Event{INVALID, 0, nil}
		}
		*sz = 3
		switch _buf[2] {
		case 68:
			return Event{CTRL_B, 0, nil}
		case 67:
			return Event{CTRL_F, 0, nil}
		case 66:
			return Event{CTRL_J, 0, nil}
		case 65:
			return Event{CTRL_K, 0, nil}
		case 90:
			return Event{BTAB, 0, nil}
		case 72:
			return Event{CTRL_A, 0, nil}
		case 70:
			return Event{CTRL_E, 0, nil}
		case 77:
			return mouseSequence(sz)
		case 49, 50, 51, 52, 53, 54:
			if len(_buf) < 4 {
				return Event{INVALID, 0, nil}
			}
			*sz = 4
			switch _buf[2] {
			case 50:
				return Event{INVALID, 0, nil} // INS
			case 51:
				return Event{DEL, 0, nil}
			case 52:
				return Event{CTRL_E, 0, nil}
			case 53:
				return Event{PGUP, 0, nil}
			case 54:
				return Event{PGDN, 0, nil}
			case 49:
				switch _buf[3] {
				case 126:
					return Event{CTRL_A, 0, nil}
				case 59:
					if len(_buf) != 6 {
						return Event{INVALID, 0, nil}
					}
					*sz = 6
					switch _buf[4] {
					case 50:
						switch _buf[5] {
						case 68:
							return Event{CTRL_A, 0, nil}
						case 67:
							return Event{CTRL_E, 0, nil}
						}
					case 53:
						switch _buf[5] {
						case 68:
							return Event{ALT_B, 0, nil}
						case 67:
							return Event{ALT_F, 0, nil}
						}
					} // _buf[4]
				} // _buf[3]
			} // _buf[2]
		} // _buf[2]
	} // _buf[1]
	return Event{INVALID, 0, nil}
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
	case CTRL_C, CTRL_G, CTRL_Q:
		return Event{CTRL_C, 0, nil}
	case 127:
		return Event{CTRL_H, 0, nil}
	case ESC:
		return escSequence(&sz)
	}

	// CTRL-A ~ CTRL-Z
	if _buf[0] <= CTRL_Z {
		return Event{int(_buf[0]), 0, nil}
	}
	r, rsz := utf8.DecodeRune(_buf)
	sz = rsz
	return Event{RUNE, r, nil}
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

func Refresh() {
	C.refresh()
}
