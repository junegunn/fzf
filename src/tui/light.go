package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/util"
)

const (
	defaultWidth  = 80
	defaultHeight = 24

	escPollInterval = 5
)

func openTtyIn() *os.File {
	in, err := os.OpenFile("/dev/tty", syscall.O_RDONLY, 0)
	if err != nil {
		panic("Failed to open /dev/tty")
	}
	return in
}

// FIXME: Need better handling of non-displayable characters
func (r *LightRenderer) stderr(str string) {
	bytes := []byte(str)
	runes := []rune{}
	for len(bytes) > 0 {
		r, sz := utf8.DecodeRune(bytes)
		if r == utf8.RuneError || r != '\x1b' && r != '\n' && r != '\r' && r < 32 {
			runes = append(runes, '?')
		} else {
			runes = append(runes, r)
		}
		bytes = bytes[sz:]
	}
	r.queued += string(runes)
}

func (r *LightRenderer) csi(code string) {
	r.stderr("\x1b[" + code)
}

func (r *LightRenderer) flush() {
	if len(r.queued) > 0 {
		fmt.Fprint(os.Stderr, r.queued)
		r.queued = ""
	}
}

// Light renderer
type LightRenderer struct {
	theme         *ColorTheme
	mouse         bool
	forceBlack    bool
	prevDownTime  time.Time
	clickY        []int
	ttyin         *os.File
	buffer        []byte
	ostty         string
	width         int
	height        int
	yoffset       int
	tabstop       int
	escDelay      int
	upOneLine     bool
	queued        string
	y             int
	x             int
	maxHeightFunc func(int) int
}

type LightWindow struct {
	renderer *LightRenderer
	colored  bool
	border   bool
	top      int
	left     int
	width    int
	height   int
	posx     int
	posy     int
	tabstop  int
	bg       Color
}

func NewLightRenderer(theme *ColorTheme, forceBlack bool, mouse bool, tabstop int, maxHeightFunc func(int) int) Renderer {
	r := LightRenderer{
		theme:         theme,
		forceBlack:    forceBlack,
		mouse:         mouse,
		ttyin:         openTtyIn(),
		yoffset:       -1,
		tabstop:       tabstop,
		upOneLine:     false,
		maxHeightFunc: maxHeightFunc}
	return &r
}

func (r *LightRenderer) defaultTheme() *ColorTheme {
	colors, err := util.ExecCommand("tput colors").Output()
	if err == nil && atoi(strings.TrimSpace(string(colors)), 16) > 16 {
		return Dark256
	}
	return Default16
}

func stty(cmd string) string {
	out, err := util.ExecCommand("stty " + cmd + " < /dev/tty").Output()
	if err != nil {
		// Not sure how to handle this
		panic("stty " + cmd + ": " + err.Error())
	}
	return strings.TrimSpace(string(out))
}

func (r *LightRenderer) findOffset() (row int, col int) {
	r.csi("6n")
	r.flush()
	bytes := r.getBytesInternal([]byte{})

	// ^[[*;*R
	if len(bytes) > 5 && bytes[0] == 27 && bytes[1] == 91 && bytes[len(bytes)-1] == 'R' {
		nums := strings.Split(string(bytes[2:len(bytes)-1]), ";")
		if len(nums) == 2 {
			return atoi(nums[0], 0) - 1, atoi(nums[1], 0) - 1
		}
		return -1, -1
	}

	// No idea
	return -1, -1
}

func repeat(s string, times int) string {
	if times > 0 {
		return strings.Repeat(s, times)
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
	delay := 100
	delayEnv := os.Getenv("ESCDELAY")
	if len(delayEnv) > 0 {
		num, err := strconv.Atoi(delayEnv)
		if err == nil && num >= 0 {
			delay = num
		}
	}
	r.escDelay = delay

	r.ostty = stty("-g")
	stty("raw")
	r.updateTerminalSize()
	initTheme(r.theme, r.defaultTheme(), r.forceBlack)

	_, x := r.findOffset()
	if x > 0 {
		r.upOneLine = true
		r.stderr("\n")
	}
	for i := 1; i < r.MaxY(); i++ {
		r.stderr("\n")
		r.csi("G")
	}

	if r.mouse {
		r.csi("?1000h")
	}
	r.csi(fmt.Sprintf("%dA", r.MaxY()-1))
	r.csi("G")
	// r.csi("s")
	r.yoffset, _ = r.findOffset()
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

func (r *LightRenderer) updateTerminalSize() {
	sizes := strings.Split(stty("size"), " ")
	if len(sizes) < 2 {
		r.width = defaultWidth
		r.height = r.maxHeightFunc(defaultHeight)
	} else {
		r.width = atoi(sizes[1], defaultWidth)
		r.height = r.maxHeightFunc(atoi(sizes[0], defaultHeight))
	}
}

func (r *LightRenderer) getch(nonblock bool) int {
	b := make([]byte, 1)
	util.SetNonblock(r.ttyin, nonblock)
	_, err := r.ttyin.Read(b)
	if err != nil {
		return -1
	}
	return int(b[0])
}

func (r *LightRenderer) getBytes() []byte {
	return r.getBytesInternal(r.buffer)
}

func (r *LightRenderer) getBytesInternal(buffer []byte) []byte {
	c := r.getch(false)

	retries := 0
	if c == ESC {
		retries = r.escDelay / escPollInterval
	}
	buffer = append(buffer, byte(c))

	for {
		c = r.getch(true)
		if c == -1 {
			if retries > 0 {
				retries--
				time.Sleep(escPollInterval * time.Millisecond)
				continue
			}
			break
		}
		retries = 0
		buffer = append(buffer, byte(c))
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
	case CtrlC:
		return Event{CtrlC, 0, nil}
	case CtrlG:
		return Event{CtrlG, 0, nil}
	case CtrlQ:
		return Event{CtrlQ, 0, nil}
	case 127:
		return Event{BSpace, 0, nil}
	case ESC:
		ev := r.escSequence(&sz)
		// Second chance
		if ev.Type == Invalid {
			r.buffer = r.getBytes()
			ev = r.escSequence(&sz)
		}
		return ev
	}

	// CTRL-A ~ CTRL-Z
	if r.buffer[0] <= CtrlZ {
		return Event{int(r.buffer[0]), 0, nil}
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
	*sz = 2
	switch r.buffer[1] {
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
		if len(r.buffer) < 3 {
			return Event{Invalid, 0, nil}
		}
		*sz = 3
		switch r.buffer[2] {
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
			return r.mouseSequence(sz)
		case 80:
			return Event{F1, 0, nil}
		case 81:
			return Event{F2, 0, nil}
		case 82:
			return Event{F3, 0, nil}
		case 83:
			return Event{F4, 0, nil}
		case 49, 50, 51, 52, 53, 54:
			if len(r.buffer) < 4 {
				return Event{Invalid, 0, nil}
			}
			*sz = 4
			switch r.buffer[2] {
			case 50:
				if len(r.buffer) == 5 && r.buffer[4] == 126 {
					*sz = 5
					switch r.buffer[3] {
					case 48:
						return Event{F9, 0, nil}
					case 49:
						return Event{F10, 0, nil}
					case 51:
						return Event{F11, 0, nil}
					case 52:
						return Event{F12, 0, nil}
					}
				}
				// Bracketed paste mode \e[200~ / \e[201
				if r.buffer[3] == 48 && (r.buffer[4] == 48 || r.buffer[4] == 49) && r.buffer[5] == 126 {
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
				switch r.buffer[3] {
				case 126:
					return Event{Home, 0, nil}
				case 53, 55, 56, 57:
					if len(r.buffer) == 5 && r.buffer[4] == 126 {
						*sz = 5
						switch r.buffer[3] {
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
					if len(r.buffer) != 6 {
						return Event{Invalid, 0, nil}
					}
					*sz = 6
					switch r.buffer[4] {
					case 50:
						switch r.buffer[5] {
						case 68:
							return Event{Home, 0, nil}
						case 67:
							return Event{End, 0, nil}
						}
					case 53:
						switch r.buffer[5] {
						case 68:
							return Event{SLeft, 0, nil}
						case 67:
							return Event{SRight, 0, nil}
						}
					} // r.buffer[4]
				} // r.buffer[3]
			} // r.buffer[2]
		} // r.buffer[2]
	} // r.buffer[1]
	if r.buffer[1] >= 'a' && r.buffer[1] <= 'z' {
		return Event{AltA + int(r.buffer[1]) - 'a', 0, nil}
	}
	return Event{Invalid, 0, nil}
}

func (r *LightRenderer) mouseSequence(sz *int) Event {
	if len(r.buffer) < 6 || r.yoffset < 0 {
		return Event{Invalid, 0, nil}
	}
	*sz = 6
	switch r.buffer[3] {
	case 32, 36, 40, 48, // mouse-down / shift / cmd / ctrl
		35, 39, 43, 51: // mouse-up / shift / cmd / ctrl
		mod := r.buffer[3] >= 36
		down := r.buffer[3]%2 == 0
		x := int(r.buffer[4] - 33)
		y := int(r.buffer[5]-33) - r.yoffset
		double := false
		if down {
			now := time.Now()
			if now.Sub(r.prevDownTime) < doubleClickDuration {
				r.clickY = append(r.clickY, y)
			} else {
				r.clickY = []int{y}
			}
			r.prevDownTime = now
		} else {
			if len(r.clickY) > 1 && r.clickY[0] == r.clickY[1] &&
				time.Now().Sub(r.prevDownTime) < doubleClickDuration {
				double = true
			}
		}

		return Event{Mouse, 0, &MouseEvent{y, x, 0, down, double, mod}}
	case 96, 100, 104, 112, // scroll-up / shift / cmd / ctrl
		97, 101, 105, 113: // scroll-down / shift / cmd / ctrl
		mod := r.buffer[3] >= 100
		s := 1 - int(r.buffer[3]%2)*2
		x := int(r.buffer[4] - 33)
		y := int(r.buffer[5]-33) - r.yoffset
		return Event{Mouse, 0, &MouseEvent{y, x, s, false, false, mod}}
	}
	return Event{Invalid, 0, nil}
}

func (r *LightRenderer) Pause() {
	stty(fmt.Sprintf("%q", r.ostty))
	r.csi("?1049h")
	r.flush()
}

func (r *LightRenderer) Resume() bool {
	stty("raw")
	r.csi("?1049l")
	r.flush()
	// Should redraw
	return true
}

func (r *LightRenderer) Clear() {
	// r.csi("u")
	r.origin()
	r.csi("J")
	r.flush()
}

func (r *LightRenderer) RefreshWindows(windows []Window) {
	r.flush()
}

func (r *LightRenderer) Refresh() {
	r.updateTerminalSize()
}

func (r *LightRenderer) Close() {
	// r.csi("u")
	r.origin()
	r.csi("J")
	if r.mouse {
		r.csi("?1000l")
	}
	if r.upOneLine {
		r.csi("A")
	}
	r.flush()
	stty(fmt.Sprintf("%q", r.ostty))
}

func (r *LightRenderer) MaxX() int {
	return r.width
}

func (r *LightRenderer) MaxY() int {
	return r.height
}

func (r *LightRenderer) DoesAutoWrap() bool {
	return true
}

func (r *LightRenderer) NewWindow(top int, left int, width int, height int, border bool) Window {
	w := &LightWindow{
		renderer: r,
		colored:  r.theme != nil,
		border:   border,
		top:      top,
		left:     left,
		width:    width,
		height:   height,
		tabstop:  r.tabstop,
		bg:       colDefault}
	if r.theme != nil {
		w.bg = r.theme.Bg
	}
	if w.border {
		w.drawBorder()
	}
	return w
}

func (w *LightWindow) drawBorder() {
	w.Move(0, 0)
	w.CPrint(ColBorder, AttrRegular, "┌"+repeat("─", w.width-2)+"┐")
	for y := 1; y < w.height-1; y++ {
		w.Move(y, 0)
		w.CPrint(ColBorder, AttrRegular, "│")
		w.cprint2(colDefault, w.bg, AttrRegular, repeat(" ", w.width-2))
		w.CPrint(ColBorder, AttrRegular, "│")
	}
	w.Move(w.height-1, 0)
	w.CPrint(ColBorder, AttrRegular, "└"+repeat("─", w.width-2)+"┘")
}

func (w *LightWindow) csi(code string) {
	w.renderer.csi(code)
}

func (w *LightWindow) stderr(str string) {
	w.renderer.stderr(str)
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
	w.Print(repeat(" ", w.width-x))
	w.Move(y, x)
}

func attrCodes(attr Attr) []string {
	codes := []string{}
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

func (w *LightWindow) csiColor(fg Color, bg Color, attr Attr) bool {
	codes := append(attrCodes(attr), colorCodes(fg, bg)...)
	w.csi(";" + strings.Join(codes, ";") + "m")
	return len(codes) > 0
}

func (w *LightWindow) Print(text string) {
	w.cprint2(colDefault, w.bg, AttrRegular, text)
}

func (w *LightWindow) CPrint(pair ColorPair, attr Attr, text string) {
	if !w.colored {
		w.csiColor(colDefault, colDefault, attrFor(pair, attr))
	} else {
		w.csiColor(pair.Fg(), pair.Bg(), attr)
	}
	w.stderr(text)
	w.csi("m")
}

func (w *LightWindow) cprint2(fg Color, bg Color, attr Attr, text string) {
	if w.csiColor(fg, bg, attr) {
		defer w.csi("m")
	}
	w.stderr(text)
}

type wrappedLine struct {
	text         string
	displayWidth int
}

func wrapLine(input string, prefixLength int, max int, tabstop int) []wrappedLine {
	lines := []wrappedLine{}
	width := 0
	line := ""
	for _, r := range input {
		w := util.Max(util.RuneWidth(r, prefixLength+width, 8), 1)
		width += w
		str := string(r)
		if r == '\t' {
			str = repeat(" ", w)
		}
		if prefixLength+width <= max {
			line += str
		} else {
			lines = append(lines, wrappedLine{string(line), width - w})
			line = str
			prefixLength = 0
			width = util.RuneWidth(r, prefixLength, 8)
		}
	}
	lines = append(lines, wrappedLine{string(line), width})
	return lines
}

func (w *LightWindow) fill(str string, onMove func()) bool {
	allLines := strings.Split(str, "\n")
	for i, line := range allLines {
		lines := wrapLine(line, w.posx, w.width, w.tabstop)
		for j, wl := range lines {
			w.stderr(wl.text)
			w.posx += wl.displayWidth
			if j < len(lines)-1 || i < len(allLines)-1 {
				if w.posy+1 >= w.height {
					return false
				}
				w.MoveAndClear(w.posy+1, 0)
				onMove()
			}
		}
	}
	return true
}

func (w *LightWindow) setBg() {
	if w.bg != colDefault {
		w.csiColor(colDefault, w.bg, AttrRegular)
	}
}

func (w *LightWindow) Fill(text string) bool {
	w.MoveAndClear(w.posy, w.posx)
	w.setBg()
	return w.fill(text, w.setBg)
}

func (w *LightWindow) CFill(fg Color, bg Color, attr Attr, text string) bool {
	w.MoveAndClear(w.posy, w.posx)
	if bg == colDefault {
		bg = w.bg
	}
	if w.csiColor(fg, bg, attr) {
		return w.fill(text, func() { w.csiColor(fg, bg, attr) })
		defer w.csi("m")
	}
	return w.fill(text, w.setBg)
}

func (w *LightWindow) FinishFill() {
	for y := w.posy + 1; y < w.height; y++ {
		w.MoveAndClear(y, 0)
	}
}

func (w *LightWindow) Erase() {
	if w.border {
		w.drawBorder()
	}
	// We don't erase the window here to avoid flickering during scroll
	w.Move(0, 0)
}
