package tui

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/util"
	"github.com/rivo/uniseg"

	"golang.org/x/term"
)

const (
	defaultWidth  = 80
	defaultHeight = 24

	defaultEscDelay = 100
	escPollInterval = 5
	offsetPollTries = 10
	maxInputBuffer  = 1024 * 1024
	maxSelectTries  = 100
)

const DefaultTtyDevice string = "/dev/tty"

var offsetRegexp = regexp.MustCompile("(.*?)\x00?\x1b\\[([0-9]+);([0-9]+)R")
var offsetRegexpBegin = regexp.MustCompile("^\x1b\\[[0-9]+;[0-9]+R")

func (r *LightRenderer) Bell() {
	r.flushRaw("\a")
}

func (r *LightRenderer) PassThrough(str string) {
	r.queued.WriteString("\x1b7" + str + "\x1b8")
}

func (r *LightRenderer) stderr(str string) {
	r.stderrInternal(str, true, "")
}

const DIM string = "\x1b[2m"
const CR string = DIM + "␍"
const LF string = DIM + "␊"

type getCharResult int

const (
	getCharSuccess getCharResult = iota
	getCharError
	getCharCancelled
)

func (r getCharResult) ok() bool {
	return r == getCharSuccess
}

func (r *LightRenderer) stderrInternal(str string, allowNLCR bool, resetCode string) {
	bytes := []byte(str)
	runes := []rune{}
	for len(bytes) > 0 {
		r, sz := utf8.DecodeRune(bytes)
		nlcr := r == '\n' || r == '\r'
		if r >= 32 || r == '\x1b' || nlcr {
			if nlcr && !allowNLCR {
				if r == '\r' {
					runes = append(runes, []rune(CR+resetCode)...)
				} else {
					runes = append(runes, []rune(LF+resetCode)...)
				}
			} else if r != utf8.RuneError {
				runes = append(runes, r)
			}
		}
		bytes = bytes[sz:]
	}
	r.queued.WriteString(string(runes))
}

func (r *LightRenderer) csi(code string) string {
	fullcode := "\x1b[" + code
	r.stderr(fullcode)
	return fullcode
}

func (r *LightRenderer) flush() {
	if r.queued.Len() > 0 {
		raw := "\x1b[?7l\x1b[?25l" + r.queued.String()
		if r.showCursor {
			raw += "\x1b[?25h\x1b[?7h"
		} else {
			raw += "\x1b[?7h"
		}
		r.flushRaw(raw)
		r.queued.Reset()
	}
}

func (r *LightRenderer) flushRaw(sequence string) {
	fmt.Fprint(r.ttyout, sequence)
}

// Light renderer
type LightRenderer struct {
	theme         *ColorTheme
	mouse         bool
	forceBlack    bool
	clearOnExit   bool
	prevDownTime  time.Time
	clicks        [][2]int
	ttyin         *os.File
	ttyout        *os.File
	cancel        func()
	buffer        []byte
	origState     *term.State
	width         int
	height        int
	yoffset       int
	tabstop       int
	escDelay      int
	fullscreen    bool
	upOneLine     bool
	queued        strings.Builder
	y             int
	x             int
	maxHeightFunc func(int) int
	showCursor    bool
	mutex         sync.Mutex

	// Windows only
	ttyinChannel    chan byte
	inHandle        uintptr
	outHandle       uintptr
	origStateInput  uint32
	origStateOutput uint32
}

type LightWindow struct {
	renderer      *LightRenderer
	colored       bool
	windowType    WindowType
	border        BorderStyle
	top           int
	left          int
	width         int
	height        int
	posx          int
	posy          int
	tabstop       int
	fg            Color
	bg            Color
	wrapSign      string
	wrapSignWidth int
}

func NewLightRenderer(ttyDefault string, ttyin *os.File, theme *ColorTheme, forceBlack bool, mouse bool, tabstop int, clearOnExit bool, fullscreen bool, maxHeightFunc func(int) int) (Renderer, error) {
	out, err := openTtyOut(ttyDefault)
	if err != nil {
		out = os.Stderr
	}
	r := LightRenderer{
		theme:         theme,
		forceBlack:    forceBlack,
		mouse:         mouse,
		clearOnExit:   clearOnExit,
		ttyin:         ttyin,
		ttyout:        out,
		yoffset:       0,
		tabstop:       tabstop,
		fullscreen:    fullscreen,
		upOneLine:     false,
		maxHeightFunc: maxHeightFunc,
		showCursor:    true}
	return &r, nil
}

func repeat(r rune, times int) string {
	if times > 0 {
		return strings.Repeat(string(r), times)
	}
	return ""
}

func atoi(s string, defaultValue int) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return value
}

func (r *LightRenderer) Init() error {
	r.escDelay = atoi(os.Getenv("ESCDELAY"), defaultEscDelay)

	if err := r.initPlatform(); err != nil {
		return err
	}
	r.updateTerminalSize()

	if r.fullscreen {
		r.smcup()
	} else {
		y, x := r.findOffset()
		r.mouse = r.mouse && y >= 0
		// When --no-clear is used for repetitive relaunching, there is a small
		// time frame between fzf processes where the user keystrokes are not
		// captured by either of fzf process which can cause x offset to be
		// increased and we're left with unwanted extra new line.
		if x > 0 && r.clearOnExit {
			r.upOneLine = true
			r.makeSpace()
		}
		// We assume that --no-clear is used for repetitive relaunching of fzf.
		// So we do not clear the lower bottom of the screen.
		if r.clearOnExit {
			r.csi("J")
		}
		for i := 1; i < r.MaxY(); i++ {
			r.makeSpace()
		}
	}

	r.enableModes()
	r.csi(fmt.Sprintf("%dA", r.MaxY()-1))
	r.csi("G")
	r.csi("K")
	if !r.clearOnExit && !r.fullscreen {
		r.csi("s")
	}
	if !r.fullscreen && r.mouse {
		r.yoffset, _ = r.findOffset()
	}
	return nil
}

func (r *LightRenderer) Resize(maxHeightFunc func(int) int) {
	r.maxHeightFunc = maxHeightFunc
}

func (r *LightRenderer) makeSpace() {
	r.stderr("\n")
	r.csi("G")
}

func (r *LightRenderer) move(y int, x int) {
	// w.csi("u")
	if r.y < y {
		r.csi(fmt.Sprintf("%dB", y-r.y))
	} else if r.y > y {
		r.csi(fmt.Sprintf("%dA", r.y-y))
	}
	r.stderr("\r")
	if x > 0 {
		r.csi(fmt.Sprintf("%dC", x))
	}
	r.y = y
	r.x = x
}

func (r *LightRenderer) origin() {
	r.move(0, 0)
}

func getEnv(name string, defaultValue int) int {
	env := os.Getenv(name)
	if len(env) == 0 {
		return defaultValue
	}
	return atoi(env, defaultValue)
}

func (r *LightRenderer) getBytes(cancellable bool) ([]byte, getCharResult, error) {
	return r.getBytesInternal(cancellable, r.buffer, false)
}

func (r *LightRenderer) getBytesInternal(cancellable bool, buffer []byte, nonblock bool) ([]byte, getCharResult, error) {
	c, result := r.getch(cancellable, nonblock)
	if result == getCharCancelled {
		return buffer, getCharCancelled, nil
	}
	if !nonblock && !result.ok() {
		r.Close()
		return nil, getCharError, errors.New("failed to read " + DefaultTtyDevice)
	}

	retries := 0
	if c == Esc.Int() || nonblock {
		retries = r.escDelay / escPollInterval
	}
	buffer = append(buffer, byte(c))

	pc := c
	for {
		c, result = r.getch(false, true)
		if !result.ok() {
			if retries > 0 {
				retries--
				time.Sleep(escPollInterval * time.Millisecond)
				continue
			}
			break
		} else if c == Esc.Int() && pc != c {
			retries = r.escDelay / escPollInterval
		} else {
			retries = 0
		}
		buffer = append(buffer, byte(c))
		pc = c

		// This should never happen under normal conditions,
		// so terminate fzf immediately.
		if len(buffer) > maxInputBuffer {
			r.Close()
			return nil, getCharError, fmt.Errorf("input buffer overflow (%d): %v", len(buffer), buffer)
		}
	}

	return buffer, getCharSuccess, nil
}

func (r *LightRenderer) GetChar(cancellable bool) Event {
	var err error
	var result getCharResult
	if len(r.buffer) == 0 {
		r.buffer, result, err = r.getBytes(cancellable)
		if err != nil {
			return Event{Fatal, 0, nil}
		}
		if result == getCharCancelled {
			return Event{Invalid, 0, nil}
		}
	}
	if len(r.buffer) == 0 {
		return Event{Fatal, 0, nil}
	}

	sz := 1
	defer func() {
		r.buffer = r.buffer[sz:]
	}()

	switch r.buffer[0] {
	case CtrlC.Byte():
		return Event{CtrlC, 0, nil}
	case CtrlG.Byte():
		return Event{CtrlG, 0, nil}
	case CtrlQ.Byte():
		return Event{CtrlQ, 0, nil}
	case 127:
		return Event{Backspace, 0, nil}
	case 8:
		return Event{CtrlBackspace, 0, nil}
	case 0:
		return Event{CtrlSpace, 0, nil}
	case 28:
		return Event{CtrlBackSlash, 0, nil}
	case 29:
		return Event{CtrlRightBracket, 0, nil}
	case 30:
		return Event{CtrlCaret, 0, nil}
	case 31:
		return Event{CtrlSlash, 0, nil}
	case Esc.Byte():
		ev := r.escSequence(&sz)
		// Second chance
		if ev.Type == Invalid {
			r.buffer, result, err = r.getBytes(true)
			if err != nil {
				return Event{Fatal, 0, nil}
			}
			if result == getCharCancelled {
				return Event{Invalid, 0, nil}
			}

			ev = r.escSequence(&sz)
		}
		return ev
	}

	// CTRL-A ~ CTRL-Z
	if r.buffer[0] <= CtrlZ.Byte() {
		return Event{EventType(r.buffer[0]), 0, nil}
	}
	char, rsz := utf8.DecodeRune(r.buffer)
	if char == utf8.RuneError {
		return Event{Esc, 0, nil}
	}
	sz = rsz
	return Event{Rune, char, nil}
}

func (r *LightRenderer) CancelGetChar() {
	r.mutex.Lock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.mutex.Unlock()
}

func (r *LightRenderer) setCancel(f func()) {
	r.mutex.Lock()
	r.cancel = f
	r.mutex.Unlock()
}

func (r *LightRenderer) escSequence(sz *int) Event {
	if len(r.buffer) < 2 {
		return Event{Esc, 0, nil}
	}

	loc := offsetRegexpBegin.FindIndex(r.buffer)
	if loc != nil && loc[0] == 0 {
		*sz = loc[1]
		return Event{Invalid, 0, nil}
	}

	*sz = 2
	if r.buffer[1] == 8 {
		return Event{CtrlAltBackspace, 0, nil}
	}
	if r.buffer[1] >= 1 && r.buffer[1] <= 'z'-'a'+1 {
		return CtrlAltKey(rune(r.buffer[1] + 'a' - 1))
	}
	alt := false
	if len(r.buffer) > 2 && r.buffer[1] == Esc.Byte() {
		r.buffer = r.buffer[1:]
		alt = true
	}
	switch r.buffer[1] {
	case Esc.Byte():
		return Event{Esc, 0, nil}
	case 127:
		return Event{AltBackspace, 0, nil}
	case '[', 'O':
		if len(r.buffer) < 3 {
			return Event{Invalid, 0, nil}
		}
		*sz = 3
		switch r.buffer[2] {
		case 'D':
			if alt {
				return Event{AltLeft, 0, nil}
			}
			return Event{Left, 0, nil}
		case 'C':
			if alt {
				// Ugh..
				return Event{AltRight, 0, nil}
			}
			return Event{Right, 0, nil}
		case 'B':
			if alt {
				return Event{AltDown, 0, nil}
			}
			return Event{Down, 0, nil}
		case 'A':
			if alt {
				return Event{AltUp, 0, nil}
			}
			return Event{Up, 0, nil}
		case 'Z':
			return Event{ShiftTab, 0, nil}
		case 'H':
			return Event{Home, 0, nil}
		case 'F':
			return Event{End, 0, nil}
		case '<':
			return r.mouseSequence(sz)
		case 'P':
			return Event{F1, 0, nil}
		case 'Q':
			return Event{F2, 0, nil}
		case 'R':
			return Event{F3, 0, nil}
		case 'S':
			return Event{F4, 0, nil}
		case '1', '2', '3', '4', '5', '6', '7', '8':
			if len(r.buffer) < 4 {
				return Event{Invalid, 0, nil}
			}
			*sz = 4
			switch r.buffer[2] {
			case '2':
				if r.buffer[3] == '~' {
					return Event{Insert, 0, nil}
				}
				if len(r.buffer) > 4 && r.buffer[4] == '~' {
					*sz = 5
					switch r.buffer[3] {
					case '0':
						return Event{F9, 0, nil}
					case '1':
						return Event{F10, 0, nil}
					case '3':
						return Event{F11, 0, nil}
					case '4':
						return Event{F12, 0, nil}
					}
				}
				// Bracketed paste mode: \e[200~ ... \e[201~
				if len(r.buffer) > 5 && r.buffer[3] == '0' && (r.buffer[4] == '0' || r.buffer[4] == '1') && r.buffer[5] == '~' {
					*sz = 6
					if r.buffer[4] == '0' {
						return Event{BracketedPasteBegin, 0, nil}
					}
					return Event{BracketedPasteEnd, 0, nil}
				}
				return Event{Invalid, 0, nil} // INS
			case '3':
				if r.buffer[3] == '~' {
					return Event{Delete, 0, nil}
				}
				if len(r.buffer) == 7 && r.buffer[6] == '~' && r.buffer[4] == '1' {
					*sz = 7
					switch r.buffer[5] {
					case '0':
						return Event{AltShiftDelete, 0, nil}
					case '1':
						return Event{AltDelete, 0, nil}
					case '2':
						return Event{AltShiftDelete, 0, nil}
					case '3':
						return Event{CtrlAltDelete, 0, nil}
					case '4':
						return Event{CtrlAltShiftDelete, 0, nil}
					case '5':
						return Event{CtrlAltDelete, 0, nil}
					case '6':
						return Event{CtrlAltShiftDelete, 0, nil}
					}
				}
				if len(r.buffer) == 6 && r.buffer[5] == '~' {
					*sz = 6
					switch r.buffer[4] {
					case '2':
						return Event{ShiftDelete, 0, nil}
					case '3':
						return Event{AltDelete, 0, nil}
					case '4':
						return Event{AltShiftDelete, 0, nil}
					case '5':
						return Event{CtrlDelete, 0, nil}
					case '6':
						return Event{CtrlShiftDelete, 0, nil}
					case '7':
						return Event{CtrlAltDelete, 0, nil}
					case '8':
						return Event{CtrlAltShiftDelete, 0, nil}
					case '9':
						return Event{AltDelete, 0, nil}
					}
				}
				return Event{Invalid, 0, nil}
			case '4':
				return Event{End, 0, nil}
			case '5':
				if r.buffer[3] == '~' {
					return Event{PageUp, 0, nil}
				}
				if len(r.buffer) == 7 && r.buffer[6] == '~' && r.buffer[4] == '1' {
					*sz = 7
					switch r.buffer[5] {
					case '0':
						return Event{AltShiftPageUp, 0, nil}
					case '1':
						return Event{AltPageUp, 0, nil}
					case '2':
						return Event{AltShiftPageUp, 0, nil}
					case '3':
						return Event{CtrlAltPageUp, 0, nil}
					case '4':
						return Event{CtrlAltShiftPageUp, 0, nil}
					case '5':
						return Event{CtrlAltPageUp, 0, nil}
					case '6':
						return Event{CtrlAltShiftPageUp, 0, nil}
					}
				}
				if len(r.buffer) == 6 && r.buffer[5] == '~' {
					*sz = 6
					switch r.buffer[4] {
					case '2':
						return Event{ShiftPageUp, 0, nil}
					case '3':
						return Event{AltPageUp, 0, nil}
					case '4':
						return Event{AltShiftPageUp, 0, nil}
					case '5':
						return Event{CtrlPageUp, 0, nil}
					case '6':
						return Event{CtrlShiftPageUp, 0, nil}
					case '7':
						return Event{CtrlAltPageUp, 0, nil}
					case '8':
						return Event{CtrlAltShiftPageUp, 0, nil}
					case '9':
						return Event{AltPageUp, 0, nil}
					}
				}
				return Event{Invalid, 0, nil}
			case '6':
				if r.buffer[3] == '~' {
					return Event{PageDown, 0, nil}
				}
				if len(r.buffer) == 7 && r.buffer[6] == '~' && r.buffer[4] == '1' {
					*sz = 7
					switch r.buffer[5] {
					case '0':
						return Event{AltShiftPageDown, 0, nil}
					case '1':
						return Event{AltPageDown, 0, nil}
					case '2':
						return Event{AltShiftPageDown, 0, nil}
					case '3':
						return Event{CtrlAltPageDown, 0, nil}
					case '4':
						return Event{CtrlAltShiftPageDown, 0, nil}
					case '5':
						return Event{CtrlAltPageDown, 0, nil}
					case '6':
						return Event{CtrlAltShiftPageDown, 0, nil}
					}
				}
				if len(r.buffer) == 6 && r.buffer[5] == '~' {
					*sz = 6
					switch r.buffer[4] {
					case '2':
						return Event{ShiftPageDown, 0, nil}
					case '3':
						return Event{AltPageDown, 0, nil}
					case '4':
						return Event{AltShiftPageDown, 0, nil}
					case '5':
						return Event{CtrlPageDown, 0, nil}
					case '6':
						return Event{CtrlShiftPageDown, 0, nil}
					case '7':
						return Event{CtrlAltPageDown, 0, nil}
					case '8':
						return Event{CtrlAltShiftPageDown, 0, nil}
					case '9':
						return Event{AltPageDown, 0, nil}
					}
				}
				return Event{Invalid, 0, nil}
			case '7':
				return Event{Home, 0, nil}
			case '8':
				return Event{End, 0, nil}
			case '1':
				switch r.buffer[3] {
				case '~':
					return Event{Home, 0, nil}
				case '1', '2', '3', '4', '5', '7', '8', '9':
					if len(r.buffer) == 5 && r.buffer[4] == '~' {
						*sz = 5
						switch r.buffer[3] {
						case '1':
							return Event{F1, 0, nil}
						case '2':
							return Event{F2, 0, nil}
						case '3':
							return Event{F3, 0, nil}
						case '4':
							return Event{F4, 0, nil}
						case '5':
							return Event{F5, 0, nil}
						case '7':
							return Event{F6, 0, nil}
						case '8':
							return Event{F7, 0, nil}
						case '9':
							return Event{F8, 0, nil}
						}
					}
					return Event{Invalid, 0, nil}
				case ';':
					if len(r.buffer) < 6 {
						return Event{Invalid, 0, nil}
					}
					*sz = 6
					switch r.buffer[4] {
					case '1', '2', '3', '4', '5', '6', '7', '8', '9':
						//                   Kitty      iTerm2     WezTerm
						// SHIFT-ARROW       "\e[1;2D"
						// ALT-SHIFT-ARROW   "\e[1;4D"  "\e[1;10D" "\e[1;4D"
						// CTRL-SHIFT-ARROW  "\e[1;6D"             N/A
						// CMD-SHIFT-ARROW   "\e[1;10D" N/A        N/A ("\e[1;2D")
						ctrl := bytes.IndexByte([]byte{'5', '6', '7', '8'}, r.buffer[4]) >= 0
						alt := bytes.IndexByte([]byte{'3', '4', '7', '8'}, r.buffer[4]) >= 0
						shift := bytes.IndexByte([]byte{'2', '4', '6', '8'}, r.buffer[4]) >= 0
						char := r.buffer[5]
						if r.buffer[4] == '9' {
							ctrl = false
							alt = true
							shift = false
							if len(r.buffer) < 6 {
								return Event{Invalid, 0, nil}
							}
							*sz = 6
							char = r.buffer[5]
						} else if r.buffer[4] == '1' && bytes.IndexByte([]byte{'0', '1', '2', '3', '4', '5', '6'}, r.buffer[5]) >= 0 {
							ctrl = bytes.IndexByte([]byte{'3', '4', '5', '6'}, r.buffer[5]) >= 0
							alt = true
							shift = bytes.IndexByte([]byte{'0', '2', '4', '6'}, r.buffer[5]) >= 0
							if len(r.buffer) < 7 {
								return Event{Invalid, 0, nil}
							}
							*sz = 7
							char = r.buffer[6]
						}
						ctrlShift := ctrl && shift
						ctrlAlt := ctrl && alt
						altShift := alt && shift
						ctrlAltShift := ctrl && alt && shift
						switch char {
						case 'A':
							if ctrlAltShift {
								return Event{CtrlAltShiftUp, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltUp, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftUp, 0, nil}
							}
							if altShift {
								return Event{AltShiftUp, 0, nil}
							}
							if ctrl {
								return Event{CtrlUp, 0, nil}
							}
							if alt {
								return Event{AltUp, 0, nil}
							}
							if shift {
								return Event{ShiftUp, 0, nil}
							}
						case 'B':
							if ctrlAltShift {
								return Event{CtrlAltShiftDown, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltDown, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftDown, 0, nil}
							}
							if altShift {
								return Event{AltShiftDown, 0, nil}
							}
							if ctrl {
								return Event{CtrlDown, 0, nil}
							}
							if alt {
								return Event{AltDown, 0, nil}
							}
							if shift {
								return Event{ShiftDown, 0, nil}
							}
						case 'C':
							if ctrlAltShift {
								return Event{CtrlAltShiftRight, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltRight, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftRight, 0, nil}
							}
							if altShift {
								return Event{AltShiftRight, 0, nil}
							}
							if ctrl {
								return Event{CtrlRight, 0, nil}
							}
							if shift {
								return Event{ShiftRight, 0, nil}
							}
							if alt {
								return Event{AltRight, 0, nil}
							}
						case 'D':
							if ctrlAltShift {
								return Event{CtrlAltShiftLeft, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltLeft, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftLeft, 0, nil}
							}
							if altShift {
								return Event{AltShiftLeft, 0, nil}
							}
							if ctrl {
								return Event{CtrlLeft, 0, nil}
							}
							if alt {
								return Event{AltLeft, 0, nil}
							}
							if shift {
								return Event{ShiftLeft, 0, nil}
							}
						case 'H':
							if ctrlAltShift {
								return Event{CtrlAltShiftHome, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltHome, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftHome, 0, nil}
							}
							if altShift {
								return Event{AltShiftHome, 0, nil}
							}
							if ctrl {
								return Event{CtrlHome, 0, nil}
							}
							if alt {
								return Event{AltHome, 0, nil}
							}
							if shift {
								return Event{ShiftHome, 0, nil}
							}
						case 'F':
							if ctrlAltShift {
								return Event{CtrlAltShiftEnd, 0, nil}
							}
							if ctrlAlt {
								return Event{CtrlAltEnd, 0, nil}
							}
							if ctrlShift {
								return Event{CtrlShiftEnd, 0, nil}
							}
							if altShift {
								return Event{AltShiftEnd, 0, nil}
							}
							if ctrl {
								return Event{CtrlEnd, 0, nil}
							}
							if alt {
								return Event{AltEnd, 0, nil}
							}
							if shift {
								return Event{ShiftEnd, 0, nil}
							}
						}
					} // r.buffer[4]
				} // r.buffer[3]
			} // r.buffer[2]
		} // r.buffer[2]
	} // r.buffer[1]
	rest := bytes.NewBuffer(r.buffer[1:])
	c, size, err := rest.ReadRune()
	if err == nil {
		*sz = 1 + size
		return AltKey(c)
	}
	return Event{Invalid, 0, nil}
}

// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Mouse-Tracking
func (r *LightRenderer) mouseSequence(sz *int) Event {
	// "\e[<0;0;0M"
	if len(r.buffer) < 9 || !r.mouse {
		return Event{Invalid, 0, nil}
	}

	rest := r.buffer[*sz:]
	end := bytes.IndexAny(rest, "mM")
	if end == -1 {
		return Event{Invalid, 0, nil}
	}

	elems := strings.SplitN(string(rest[:end]), ";", 3)
	if len(elems) != 3 {
		return Event{Invalid, 0, nil}
	}

	t := atoi(elems[0], -1)
	x := atoi(elems[1], -1) - 1
	y := atoi(elems[2], -1) - 1 - r.yoffset
	if t < 0 || x < 0 {
		return Event{Invalid, 0, nil}
	}
	*sz += end + 1

	down := rest[end] == 'M'

	scroll := 0
	if t >= 64 {
		t -= 64
		if t&0b1 == 1 {
			scroll = -1
		} else {
			scroll = 1
		}
	}

	// middle := t & 0b1
	left := t&0b11 == 0
	ctrl := t&0b10000 > 0
	alt := t&0b01000 > 0
	shift := t&0b00100 > 0
	drag := t&0b100000 > 0 // 32

	if scroll != 0 {
		return Event{Mouse, 0, &MouseEvent{y, x, scroll, false, false, false, ctrl, alt, shift}}
	}

	double := false
	if down && !drag {
		now := time.Now()
		if !left { // Right double click is not allowed
			r.clicks = [][2]int{}
		} else if now.Sub(r.prevDownTime) < doubleClickDuration {
			r.clicks = append(r.clicks, [2]int{x, y})
		} else {
			r.clicks = [][2]int{{x, y}}
		}
		r.prevDownTime = now
	} else {
		n := len(r.clicks)
		if len(r.clicks) > 1 && r.clicks[n-2][0] == r.clicks[n-1][0] && r.clicks[n-2][1] == r.clicks[n-1][1] &&
			time.Since(r.prevDownTime) < doubleClickDuration {
			double = true
			if double {
				r.clicks = [][2]int{}
			}
		}
	}
	return Event{Mouse, 0, &MouseEvent{y, x, 0, left, down, double, ctrl, alt, shift}}
}

func (r *LightRenderer) smcup() {
	r.flush()
	r.flushRaw("\x1b[?1049h")
}

func (r *LightRenderer) rmcup() {
	r.flush()
	r.flushRaw("\x1b[?1049l")
}

func (r *LightRenderer) Pause(clear bool) {
	r.disableModes()
	r.restoreTerminal()
	if clear {
		if r.fullscreen {
			r.rmcup()
		} else {
			r.smcup()
			r.csi("H")
		}
		r.flush()
	}
}

func (r *LightRenderer) enableModes() {
	if r.mouse {
		r.csi("?1000h")
		r.csi("?1002h")
		r.csi("?1006h")
	}
	r.csi("?2004h") // Enable bracketed paste mode
}

func (r *LightRenderer) disableMouse() {
	if r.mouse {
		r.csi("?1000l")
		r.csi("?1002l")
		r.csi("?1006l")
	}
}

func (r *LightRenderer) disableModes() {
	r.disableMouse()
	r.csi("?2004l")
}

func (r *LightRenderer) Resume(clear bool, sigcont bool) {
	r.setupTerminal()
	if clear {
		if r.fullscreen {
			r.smcup()
		} else {
			r.rmcup()
		}
		r.enableModes()
		r.flush()
	} else if sigcont && !r.fullscreen && r.mouse {
		// NOTE: SIGCONT (Coming back from CTRL-Z):
		// It's highly likely that the offset we obtained at the beginning is
		// no longer correct, so we simply disable mouse input.
		r.disableMouse()
		r.mouse = false
	}
}

func (r *LightRenderer) Clear() {
	if r.fullscreen {
		r.csi("H")
	}
	// r.csi("u")
	r.origin()
	r.csi("J")
	r.flush()
}

func (r *LightRenderer) NeedScrollbarRedraw() bool {
	return false
}

func (r *LightRenderer) ShouldEmitResizeEvent() bool {
	return false
}

func (r *LightRenderer) RefreshWindows(windows []Window) {
	r.flush()
}

func (r *LightRenderer) Refresh() {
	r.updateTerminalSize()
}

func (r *LightRenderer) Close() {
	// r.csi("u")
	if r.clearOnExit {
		if r.fullscreen {
			r.rmcup()
		} else {
			r.origin()
			if r.upOneLine {
				r.csi("A")
			}
			r.csi("J")
		}
	} else if !r.fullscreen {
		r.csi("u")
	}
	if !r.showCursor {
		r.csi("?25h")
	}
	r.disableModes()
	r.flush()
	r.restoreTerminal()
	r.closePlatform()
}

func (r *LightRenderer) Top() int {
	return r.yoffset
}

func (r *LightRenderer) MaxX() int {
	return r.width
}

func (r *LightRenderer) MaxY() int {
	if r.height == 0 {
		r.updateTerminalSize()
	}
	return r.height
}

func (r *LightRenderer) NewWindow(top int, left int, width int, height int, windowType WindowType, borderStyle BorderStyle, erase bool) Window {
	width = max(0, width)
	height = max(0, height)
	w := &LightWindow{
		renderer:   r,
		colored:    r.theme.Colored,
		windowType: windowType,
		border:     borderStyle,
		top:        top,
		left:       left,
		width:      width,
		height:     height,
		tabstop:    r.tabstop,
		fg:         colDefault,
		bg:         colDefault}
	switch windowType {
	case WindowBase:
		w.fg = r.theme.Fg.Color
		w.bg = r.theme.Bg.Color
	case WindowList:
		w.fg = r.theme.ListFg.Color
		w.bg = r.theme.ListBg.Color
	case WindowInput:
		w.fg = r.theme.Input.Color
		w.bg = r.theme.InputBg.Color
	case WindowHeader:
		w.fg = r.theme.Header.Color
		w.bg = r.theme.HeaderBg.Color
	case WindowFooter:
		w.fg = r.theme.Footer.Color
		w.bg = r.theme.FooterBg.Color
	case WindowPreview:
		w.fg = r.theme.PreviewFg.Color
		w.bg = r.theme.PreviewBg.Color
	}
	if erase && !w.bg.IsDefault() && w.border.shape != BorderNone && w.height > 0 {
		// fzf --color bg:blue --border --padding 1,2
		w.Erase()
	}
	w.drawBorder(false)
	return w
}

func (w *LightWindow) DrawBorder() {
	w.drawBorder(false)
}

func (w *LightWindow) DrawHBorder() {
	w.drawBorder(true)
}

func (w *LightWindow) drawBorder(onlyHorizontal bool) {
	if w.height == 0 {
		return
	}
	switch w.border.shape {
	case BorderRounded, BorderSharp, BorderBold, BorderBlock, BorderThinBlock, BorderDouble:
		w.drawBorderAround(onlyHorizontal)
	case BorderHorizontal:
		w.drawBorderHorizontal(true, true)
	case BorderVertical:
		if onlyHorizontal {
			return
		}
		w.drawBorderVertical(true, true)
	case BorderTop:
		w.drawBorderHorizontal(true, false)
	case BorderBottom:
		w.drawBorderHorizontal(false, true)
	case BorderLeft:
		if onlyHorizontal {
			return
		}
		w.drawBorderVertical(true, false)
	case BorderRight:
		if onlyHorizontal {
			return
		}
		w.drawBorderVertical(false, true)
	}
}

func (w *LightWindow) drawBorderHorizontal(top, bottom bool) {
	color := ColBorder
	switch w.windowType {
	case WindowList:
		color = ColListBorder
	case WindowInput:
		color = ColInputBorder
	case WindowHeader:
		color = ColHeaderBorder
	case WindowFooter:
		color = ColFooterBorder
	case WindowPreview:
		color = ColPreviewBorder
	}
	hw := runeWidth(w.border.top)
	if top {
		w.Move(0, 0)
		w.CPrint(color, repeat(w.border.top, w.width/hw))
	}

	if bottom {
		w.Move(w.height-1, 0)
		w.CPrint(color, repeat(w.border.bottom, w.width/hw))
	}
}

func (w *LightWindow) drawBorderVertical(left, right bool) {
	vw := runeWidth(w.border.left)
	color := ColBorder
	switch w.windowType {
	case WindowList:
		color = ColListBorder
	case WindowInput:
		color = ColInputBorder
	case WindowHeader:
		color = ColHeaderBorder
	case WindowFooter:
		color = ColFooterBorder
	case WindowPreview:
		color = ColPreviewBorder
	}
	for y := 0; y < w.height; y++ {
		if left {
			w.Move(y, 0)
			w.CPrint(color, string(w.border.left))
			w.CPrint(color, " ") // Margin
		}
		if right {
			w.Move(y, w.width-vw-1)
			w.CPrint(color, " ") // Margin
			w.CPrint(color, string(w.border.right))
		}
	}
}

func (w *LightWindow) drawBorderAround(onlyHorizontal bool) {
	w.Move(0, 0)
	color := ColBorder
	switch w.windowType {
	case WindowList:
		color = ColListBorder
	case WindowInput:
		color = ColInputBorder
	case WindowHeader:
		color = ColHeaderBorder
	case WindowFooter:
		color = ColFooterBorder
	case WindowPreview:
		color = ColPreviewBorder
	}
	hw := runeWidth(w.border.top)
	tcw := runeWidth(w.border.topLeft) + runeWidth(w.border.topRight)
	bcw := runeWidth(w.border.bottomLeft) + runeWidth(w.border.bottomRight)
	rem := (w.width - tcw) % hw
	w.CPrint(color, string(w.border.topLeft)+repeat(w.border.top, (w.width-tcw)/hw)+repeat(' ', rem)+string(w.border.topRight))
	if !onlyHorizontal {
		vw := runeWidth(w.border.left)
		for y := 1; y < w.height-1; y++ {
			w.Move(y, 0)
			w.CPrint(color, string(w.border.left))
			w.CPrint(color, " ") // Margin

			w.Move(y, w.width-vw-1)
			w.CPrint(color, " ") // Margin
			w.CPrint(color, string(w.border.right))
		}
	}
	w.Move(w.height-1, 0)
	rem = (w.width - bcw) % hw
	w.CPrint(color, string(w.border.bottomLeft)+repeat(w.border.bottom, (w.width-bcw)/hw)+repeat(' ', rem)+string(w.border.bottomRight))
}

func (w *LightWindow) csi(code string) string {
	return w.renderer.csi(code)
}

func (w *LightWindow) stderrInternal(str string, allowNLCR bool, resetCode string) {
	w.renderer.stderrInternal(str, allowNLCR, resetCode)
}

func (w *LightWindow) Top() int {
	return w.top
}

func (w *LightWindow) Left() int {
	return w.left
}

func (w *LightWindow) Width() int {
	return w.width
}

func (w *LightWindow) Height() int {
	return w.height
}

func (w *LightWindow) Refresh() {
}

func (w *LightWindow) X() int {
	return w.posx
}

func (w *LightWindow) Y() int {
	return w.posy
}

func (w *LightWindow) EncloseX(x int) bool {
	return x >= w.left && x < (w.left+w.width)
}

func (w *LightWindow) EncloseY(y int) bool {
	return y >= w.top && y < (w.top+w.height)
}

func (w *LightWindow) Enclose(y int, x int) bool {
	return w.EncloseX(x) && w.EncloseY(y)
}

func (w *LightWindow) Move(y int, x int) {
	w.posx = x
	w.posy = y

	w.renderer.move(w.Top()+y, w.Left()+x)
}

func (w *LightWindow) MoveAndClear(y int, x int) {
	w.Move(y, x)
	// We should not delete preview window on the right
	// csi("K")
	w.Print(repeat(' ', w.width-x))
	w.Move(y, x)
}

func attrCodes(attr Attr) []string {
	codes := []string{}
	if (attr & AttrClear) > 0 {
		return codes
	}
	if (attr&Bold) > 0 || (attr&BoldForce) > 0 {
		codes = append(codes, "1")
	}
	if (attr & Dim) > 0 {
		codes = append(codes, "2")
	}
	if (attr & Italic) > 0 {
		codes = append(codes, "3")
	}
	if (attr & Underline) > 0 {
		codes = append(codes, "4")
	}
	if (attr & Blink) > 0 {
		codes = append(codes, "5")
	}
	if (attr & Reverse) > 0 {
		codes = append(codes, "7")
	}
	if (attr & StrikeThrough) > 0 {
		codes = append(codes, "9")
	}
	return codes
}

func colorCodes(fg Color, bg Color) []string {
	codes := []string{}
	appendCode := func(c Color, offset int) {
		if c == colDefault {
			return
		}
		if c.is24() {
			r := (c >> 16) & 0xff
			g := (c >> 8) & 0xff
			b := (c) & 0xff
			codes = append(codes, fmt.Sprintf("%d;2;%d;%d;%d", 38+offset, r, g, b))
		} else if c >= colBlack && c <= colWhite {
			codes = append(codes, fmt.Sprintf("%d", int(c)+30+offset))
		} else if c > colWhite && c < 16 {
			codes = append(codes, fmt.Sprintf("%d", int(c)+90+offset-8))
		} else if c >= 16 && c < 256 {
			codes = append(codes, fmt.Sprintf("%d;5;%d", 38+offset, c))
		}
	}
	appendCode(fg, 0)
	appendCode(bg, 10)
	return codes
}

func (w *LightWindow) csiColor(fg Color, bg Color, attr Attr) (bool, string) {
	codes := append(attrCodes(attr), colorCodes(fg, bg)...)
	code := w.csi(";" + strings.Join(codes, ";") + "m")
	return len(codes) > 0, code
}

func (w *LightWindow) Print(text string) {
	w.cprint2(colDefault, w.bg, AttrRegular, text)
}

func cleanse(str string) string {
	return strings.ReplaceAll(str, "\x1b", "")
}

func (w *LightWindow) CPrint(pair ColorPair, text string) {
	_, code := w.csiColor(pair.Fg(), pair.Bg(), pair.Attr())
	w.stderrInternal(cleanse(text), false, code)
	w.csi("0m")
}

func (w *LightWindow) cprint2(fg Color, bg Color, attr Attr, text string) {
	hasColors, code := w.csiColor(fg, bg, attr)
	if hasColors {
		defer w.csi("0m")
	}
	w.stderrInternal(cleanse(text), false, code)
}

type wrappedLine struct {
	text         string
	displayWidth int
}

func wrapLine(input string, prefixLength int, initialMax int, tabstop int, wrapSignWidth int) []wrappedLine {
	lines := []wrappedLine{}
	width := 0
	line := ""
	gr := uniseg.NewGraphemes(input)
	max := initialMax
	for gr.Next() {
		rs := gr.Runes()
		str := string(rs)
		var w int
		if len(rs) == 1 && rs[0] == '\t' {
			w = tabstop - (prefixLength+width)%tabstop
			str = repeat(' ', w)
		} else if rs[0] == '\r' {
			w++
		} else {
			w = uniseg.StringWidth(str)
		}
		width += w

		if prefixLength+width <= max {
			line += str
		} else {
			lines = append(lines, wrappedLine{string(line), width - w})
			line = str
			prefixLength = 0
			width = w
			max = initialMax - wrapSignWidth
		}
	}
	lines = append(lines, wrappedLine{string(line), width})
	return lines
}

func (w *LightWindow) fill(str string, resetCode string) FillReturn {
	allLines := strings.Split(str, "\n")
	for i, line := range allLines {
		lines := wrapLine(line, w.posx, w.width, w.tabstop, w.wrapSignWidth)
		for j, wl := range lines {
			w.stderrInternal(wl.text, false, resetCode)
			w.posx += wl.displayWidth

			// Wrap line
			if j < len(lines)-1 || i < len(allLines)-1 {
				if w.posy+1 >= w.height {
					return FillSuspend
				}
				w.MoveAndClear(w.posy, w.posx)
				w.Move(w.posy+1, 0)
				w.renderer.stderr(resetCode)
				if len(lines) > 1 {
					sign := w.wrapSign
					width := w.wrapSignWidth
					if width > w.width {
						runes, truncatedWidth := util.Truncate(w.wrapSign, w.width)
						sign = string(runes)
						width = truncatedWidth
					}
					w.stderrInternal(DIM+sign, false, resetCode)
					w.renderer.stderr(resetCode)
					w.Move(w.posy, width)
				}
			}
		}
	}
	if w.posx >= w.Width() {
		if w.posy+1 >= w.height {
			return FillSuspend
		}
		w.Move(w.posy+1, 0)
		w.renderer.stderr(resetCode)
		return FillNextLine
	}
	return FillContinue
}

func (w *LightWindow) setBg() string {
	if w.bg != colDefault {
		_, code := w.csiColor(colDefault, w.bg, AttrRegular)
		return code
	}
	// Should clear dim attribute after ␍ in the preview window
	// e.g. printf "foo\rbar" | fzf --ansi --preview 'printf "foo\rbar"'
	return "\x1b[m"
}

func (w *LightWindow) LinkBegin(uri string, params string) {
	w.renderer.queued.WriteString("\x1b]8;" + params + ";" + uri + "\x1b\\")
}

func (w *LightWindow) LinkEnd() {
	w.renderer.queued.WriteString("\x1b]8;;\x1b\\")
}

func (w *LightWindow) Fill(text string) FillReturn {
	w.Move(w.posy, w.posx)
	code := w.setBg()
	return w.fill(text, code)
}

func (w *LightWindow) CFill(fg Color, bg Color, attr Attr, text string) FillReturn {
	w.Move(w.posy, w.posx)
	if fg == colDefault {
		fg = w.fg
	}
	if bg == colDefault {
		bg = w.bg
	}
	if hasColors, resetCode := w.csiColor(fg, bg, attr); hasColors {
		defer w.csi("0m")
		return w.fill(text, resetCode)
	}
	return w.fill(text, w.setBg())
}

func (w *LightWindow) FinishFill() {
	if w.posy < w.height {
		w.MoveAndClear(w.posy, w.posx)
	}
	for y := w.posy + 1; y < w.height; y++ {
		w.MoveAndClear(y, 0)
	}
}

func (w *LightWindow) Erase() {
	w.DrawBorder()
	w.Move(0, 0)
	w.FinishFill()
	w.Move(0, 0)
}

func (w *LightWindow) EraseMaybe() bool {
	return false
}

func (w *LightWindow) SetWrapSign(sign string, width int) {
	w.wrapSign = sign
	w.wrapSignWidth = width
}

func (r *LightRenderer) HideCursor() {
	r.showCursor = false
	r.csi("?25l")
}

func (r *LightRenderer) ShowCursor() {
	r.showCursor = true
	r.csi("?25h")
}
