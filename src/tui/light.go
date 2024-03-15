package tui

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
)

const consoleDevice string = "/dev/tty"

var offsetRegexp *regexp.Regexp = regexp.MustCompile("(.*)\x1b\\[([0-9]+);([0-9]+)R")
var offsetRegexpBegin *regexp.Regexp = regexp.MustCompile("^\x1b\\[[0-9]+;[0-9]+R")

func (r *LightRenderer) PassThrough(str string) {
	r.queued.WriteString("\x1b7" + str + "\x1b8")
}

func (r *LightRenderer) stderr(str string) {
	r.stderrInternal(str, true, "")
}

const CR string = "\x1b[2m␍"
const LF string = "\x1b[2m␊"

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
		fmt.Fprint(os.Stderr, "\x1b[?25l"+r.queued.String()+"\x1b[?25h")
		r.queued.Reset()
	}
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

	// Windows only
	ttyinChannel    chan byte
	inHandle        uintptr
	outHandle       uintptr
	origStateInput  uint32
	origStateOutput uint32
}

type LightWindow struct {
	renderer *LightRenderer
	colored  bool
	preview  bool
	border   BorderStyle
	top      int
	left     int
	width    int
	height   int
	posx     int
	posy     int
	tabstop  int
	fg       Color
	bg       Color
}

func NewLightRenderer(theme *ColorTheme, forceBlack bool, mouse bool, tabstop int, clearOnExit bool, fullscreen bool, maxHeightFunc func(int) int) Renderer {
	r := LightRenderer{
		theme:         theme,
		forceBlack:    forceBlack,
		mouse:         mouse,
		clearOnExit:   clearOnExit,
		ttyin:         openTtyIn(),
		yoffset:       0,
		tabstop:       tabstop,
		fullscreen:    fullscreen,
		upOneLine:     false,
		maxHeightFunc: maxHeightFunc}
	return &r
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

func (r *LightRenderer) Init() {
	r.escDelay = atoi(os.Getenv("ESCDELAY"), defaultEscDelay)

	if err := r.initPlatform(); err != nil {
		errorExit(err.Error())
	}
	r.updateTerminalSize()
	initTheme(r.theme, r.defaultTheme(), r.forceBlack)

	if r.fullscreen {
		r.smcup()
	} else {
		// We assume that --no-clear is used for repetitive relaunching of fzf.
		// So we do not clear the lower bottom of the screen.
		if r.clearOnExit {
			r.csi("J")
		}
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
		for i := 1; i < r.MaxY(); i++ {
			r.makeSpace()
		}
	}

	r.enableMouse()
	r.csi(fmt.Sprintf("%dA", r.MaxY()-1))
	r.csi("G")
	r.csi("K")
	if !r.clearOnExit && !r.fullscreen {
		r.csi("s")
	}
	if !r.fullscreen && r.mouse {
		r.yoffset, _ = r.findOffset()
	}
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

func (r *LightRenderer) getBytes() []byte {
	return r.getBytesInternal(r.buffer, false)
}

func (r *LightRenderer) getBytesInternal(buffer []byte, nonblock bool) []byte {
	c, ok := r.getch(nonblock)
	if !nonblock && !ok {
		r.Close()
		errorExit("Failed to read " + consoleDevice)
	}

	retries := 0
	if c == ESC.Int() || nonblock {
		retries = r.escDelay / escPollInterval
	}
	buffer = append(buffer, byte(c))

	pc := c
	for {
		c, ok = r.getch(true)
		if !ok {
			if retries > 0 {
				retries--
				time.Sleep(escPollInterval * time.Millisecond)
				continue
			}
			break
		} else if c == ESC.Int() && pc != c {
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
			panic(fmt.Sprintf("Input buffer overflow (%d): %v", len(buffer), buffer))
		}
	}

	return buffer
}

func (r *LightRenderer) GetChar() Event {
	if len(r.buffer) == 0 {
		r.buffer = r.getBytes()
	}
	if len(r.buffer) == 0 {
		panic("Empty buffer")
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
		return Event{BSpace, 0, nil}
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
	case ESC.Byte():
		ev := r.escSequence(&sz)
		// Second chance
		if ev.Type == Invalid {
			r.buffer = r.getBytes()
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
		return Event{ESC, 0, nil}
	}
	sz = rsz
	return Event{Rune, char, nil}
}

func (r *LightRenderer) escSequence(sz *int) Event {
	if len(r.buffer) < 2 {
		return Event{ESC, 0, nil}
	}

	loc := offsetRegexpBegin.FindIndex(r.buffer)
	if loc != nil && loc[0] == 0 {
		*sz = loc[1]
		return Event{Invalid, 0, nil}
	}

	*sz = 2
	if r.buffer[1] >= 1 && r.buffer[1] <= 'z'-'a'+1 {
		return CtrlAltKey(rune(r.buffer[1] + 'a' - 1))
	}
	alt := false
	if len(r.buffer) > 2 && r.buffer[1] == ESC.Byte() {
		r.buffer = r.buffer[1:]
		alt = true
	}
	switch r.buffer[1] {
	case ESC.Byte():
		return Event{ESC, 0, nil}
	case 127:
		return Event{AltBS, 0, nil}
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
			return Event{BTab, 0, nil}
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
					// Immediately discard the sequence from the buffer and reread input
					r.buffer = r.buffer[6:]
					*sz = 0
					return r.GetChar()
				}
				return Event{Invalid, 0, nil} // INS
			case '3':
				if r.buffer[3] == '~' {
					return Event{Del, 0, nil}
				}
				if len(r.buffer) == 6 && r.buffer[5] == '~' {
					*sz = 6
					switch r.buffer[4] {
					case '5':
						return Event{CtrlDelete, 0, nil}
					case '2':
						return Event{SDelete, 0, nil}
					}
				}
				return Event{Invalid, 0, nil}
			case '4':
				return Event{End, 0, nil}
			case '5':
				return Event{PgUp, 0, nil}
			case '6':
				return Event{PgDn, 0, nil}
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
					case '1', '2', '3', '5':
						alt := r.buffer[4] == '3'
						altShift := r.buffer[4] == '1' && r.buffer[5] == '0'
						char := r.buffer[5]
						if altShift {
							if len(r.buffer) < 7 {
								return Event{Invalid, 0, nil}
							}
							*sz = 7
							char = r.buffer[6]
						}
						switch char {
						case 'A':
							if alt {
								return Event{AltUp, 0, nil}
							}
							if altShift {
								return Event{AltSUp, 0, nil}
							}
							return Event{SUp, 0, nil}
						case 'B':
							if alt {
								return Event{AltDown, 0, nil}
							}
							if altShift {
								return Event{AltSDown, 0, nil}
							}
							return Event{SDown, 0, nil}
						case 'C':
							if alt {
								return Event{AltRight, 0, nil}
							}
							if altShift {
								return Event{AltSRight, 0, nil}
							}
							return Event{SRight, 0, nil}
						case 'D':
							if alt {
								return Event{AltLeft, 0, nil}
							}
							if altShift {
								return Event{AltSLeft, 0, nil}
							}
							return Event{SLeft, 0, nil}
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

	// shift := t & 0b100
	// ctrl := t & 0b1000
	mod := t&0b1100 > 0

	drag := t&0b100000 > 0

	if scroll != 0 {
		return Event{Mouse, 0, &MouseEvent{y, x, scroll, false, false, false, mod}}
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
	return Event{Mouse, 0, &MouseEvent{y, x, 0, left, down, double, mod}}
}

func (r *LightRenderer) smcup() {
	r.csi("?1049h")
}

func (r *LightRenderer) rmcup() {
	r.csi("?1049l")
}

func (r *LightRenderer) Pause(clear bool) {
	r.disableMouse()
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

func (r *LightRenderer) enableMouse() {
	if r.mouse {
		r.csi("?1000h")
		r.csi("?1002h")
		r.csi("?1006h")
	}
}

func (r *LightRenderer) disableMouse() {
	if r.mouse {
		r.csi("?1000l")
		r.csi("?1002l")
		r.csi("?1006l")
	}
}

func (r *LightRenderer) Resume(clear bool, sigcont bool) {
	r.setupTerminal()
	if clear {
		if r.fullscreen {
			r.smcup()
		} else {
			r.rmcup()
		}
		r.enableMouse()
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
	r.disableMouse()
	r.flush()
	r.closePlatform()
	r.restoreTerminal()
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

func (r *LightRenderer) NewWindow(top int, left int, width int, height int, preview bool, borderStyle BorderStyle) Window {
	w := &LightWindow{
		renderer: r,
		colored:  r.theme.Colored,
		preview:  preview,
		border:   borderStyle,
		top:      top,
		left:     left,
		width:    width,
		height:   height,
		tabstop:  r.tabstop,
		fg:       colDefault,
		bg:       colDefault}
	if preview {
		w.fg = r.theme.PreviewFg.Color
		w.bg = r.theme.PreviewBg.Color
	} else {
		w.fg = r.theme.Fg.Color
		w.bg = r.theme.Bg.Color
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
	if w.preview {
		color = ColPreviewBorder
	}
	hw := runeWidth(w.border.top)
	pad := repeat(' ', w.width/hw)

	w.Move(0, 0)
	if top {
		w.CPrint(color, repeat(w.border.top, w.width/hw))
	} else {
		w.CPrint(color, pad)
	}

	for y := 1; y < w.height-1; y++ {
		w.Move(y, 0)
		w.CPrint(color, pad)
	}

	w.Move(w.height-1, 0)
	if bottom {
		w.CPrint(color, repeat(w.border.bottom, w.width/hw))
	} else {
		w.CPrint(color, pad)
	}
}

func (w *LightWindow) drawBorderVertical(left, right bool) {
	width := w.width - 2
	if !left || !right {
		width++
	}
	color := ColBorder
	if w.preview {
		color = ColPreviewBorder
	}
	for y := 0; y < w.height; y++ {
		w.Move(y, 0)
		if left {
			w.CPrint(color, string(w.border.left))
		}
		w.CPrint(color, repeat(' ', width))
		if right {
			w.CPrint(color, string(w.border.right))
		}
	}
}

func (w *LightWindow) drawBorderAround(onlyHorizontal bool) {
	w.Move(0, 0)
	color := ColBorder
	if w.preview {
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
			w.CPrint(color, repeat(' ', w.width-vw*2))
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

func (w *LightWindow) Close() {
}

func (w *LightWindow) X() int {
	return w.posx
}

func (w *LightWindow) Y() int {
	return w.posy
}

func (w *LightWindow) Enclose(y int, x int) bool {
	return x >= w.left && x < (w.left+w.width) &&
		y >= w.top && y < (w.top+w.height)
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
	if (attr & Bold) > 0 {
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
	return strings.Replace(str, "\x1b", "", -1)
}

func (w *LightWindow) CPrint(pair ColorPair, text string) {
	_, code := w.csiColor(pair.Fg(), pair.Bg(), pair.Attr())
	w.stderrInternal(cleanse(text), false, code)
	w.csi("m")
}

func (w *LightWindow) cprint2(fg Color, bg Color, attr Attr, text string) {
	hasColors, code := w.csiColor(fg, bg, attr)
	if hasColors {
		defer w.csi("m")
	}
	w.stderrInternal(cleanse(text), false, code)
}

type wrappedLine struct {
	text         string
	displayWidth int
}

func wrapLine(input string, prefixLength int, max int, tabstop int) []wrappedLine {
	lines := []wrappedLine{}
	width := 0
	line := ""
	gr := uniseg.NewGraphemes(input)
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
		}
	}
	lines = append(lines, wrappedLine{string(line), width})
	return lines
}

func (w *LightWindow) fill(str string, resetCode string) FillReturn {
	allLines := strings.Split(str, "\n")
	for i, line := range allLines {
		lines := wrapLine(line, w.posx, w.width, w.tabstop)
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
			}
		}
	}
	if w.posx+1 >= w.Width() {
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
		defer w.csi("m")
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
