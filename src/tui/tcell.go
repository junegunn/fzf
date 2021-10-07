// +build tcell windows

package tui

import (
	"os"
	"time"

	"runtime"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

func HasFullscreenRenderer() bool {
	return true
}

func (p ColorPair) style() tcell.Style {
	style := tcell.StyleDefault
	return style.Foreground(tcell.Color(p.Fg())).Background(tcell.Color(p.Bg()))
}

type Attr tcell.Style

type TcellWindow struct {
	color       bool
	preview     bool
	top         int
	left        int
	width       int
	height      int
	normal      ColorPair
	lastX       int
	lastY       int
	moveCursor  bool
	borderStyle BorderStyle
}

func (w *TcellWindow) Top() int {
	return w.top
}

func (w *TcellWindow) Left() int {
	return w.left
}

func (w *TcellWindow) Width() int {
	return w.width
}

func (w *TcellWindow) Height() int {
	return w.height
}

func (w *TcellWindow) Refresh() {
	if w.moveCursor {
		_screen.ShowCursor(w.left+w.lastX, w.top+w.lastY)
		w.moveCursor = false
	}
	w.lastX = 0
	w.lastY = 0

	w.drawBorder()
}

func (w *TcellWindow) FinishFill() {
	// NO-OP
}

const (
	Bold      Attr = Attr(tcell.AttrBold)
	Dim            = Attr(tcell.AttrDim)
	Blink          = Attr(tcell.AttrBlink)
	Reverse        = Attr(tcell.AttrReverse)
	Underline      = Attr(tcell.AttrUnderline)
	Italic         = Attr(tcell.AttrItalic)
)

const (
	AttrUndefined = Attr(0)
	AttrRegular   = Attr(1 << 7)
	AttrClear     = Attr(1 << 8)
)

func (r *FullscreenRenderer) defaultTheme() *ColorTheme {
	if _screen.Colors() >= 256 {
		return Dark256
	}
	return Default16
}

var (
	_colorToAttribute = []tcell.Color{
		tcell.ColorBlack,
		tcell.ColorRed,
		tcell.ColorGreen,
		tcell.ColorYellow,
		tcell.ColorBlue,
		tcell.ColorDarkMagenta,
		tcell.ColorLightCyan,
		tcell.ColorWhite,
	}
)

func (c Color) Style() tcell.Color {
	if c <= colDefault {
		return tcell.ColorDefault
	} else if c >= colBlack && c <= colWhite {
		return _colorToAttribute[int(c)]
	} else {
		return tcell.Color(c)
	}
}

func (a Attr) Merge(b Attr) Attr {
	return a | b
}

// handle the following as private members of FullscreenRenderer instance
// they are declared here to prevent introducing tcell library in non-windows builds
var (
	_screen          tcell.Screen
	_prevMouseButton tcell.ButtonMask
)

func (r *FullscreenRenderer) initScreen() {
	s, e := tcell.NewScreen()
	if e != nil {
		errorExit(e.Error())
	}
	if e = s.Init(); e != nil {
		errorExit(e.Error())
	}
	if r.mouse {
		s.EnableMouse()
	} else {
		s.DisableMouse()
	}
	_screen = s
}

func (r *FullscreenRenderer) Init() {
	if os.Getenv("TERM") == "cygwin" {
		os.Setenv("TERM", "")
	}
	encoding.Register()

	r.initScreen()
	initTheme(r.theme, r.defaultTheme(), r.forceBlack)
}

func (r *FullscreenRenderer) MaxX() int {
	ncols, _ := _screen.Size()
	return int(ncols)
}

func (r *FullscreenRenderer) MaxY() int {
	_, nlines := _screen.Size()
	return int(nlines)
}

func (w *TcellWindow) X() int {
	return w.lastX
}

func (w *TcellWindow) Y() int {
	return w.lastY
}

func (r *FullscreenRenderer) Clear() {
	_screen.Sync()
	_screen.Clear()
}

func (r *FullscreenRenderer) Refresh() {
	// noop
}

func (r *FullscreenRenderer) GetChar() Event {
	ev := _screen.PollEvent()
	switch ev := ev.(type) {
	case *tcell.EventResize:
		return Event{Resize, 0, nil}

	// process mouse events:
	case *tcell.EventMouse:
		// mouse down events have zeroed buttons, so we can't use them
		// mouse up event consists of two events, 1. (main) event with modifier and other metadata, 2. event with zeroed buttons
		// so mouse click is three consecutive events, but the first and last are indistinguishable from movement events (with released buttons)
		// dragging has same structure, it only repeats the middle (main) event appropriately
		x, y := ev.Position()
		mod := ev.Modifiers() != 0

		// since we dont have mouse down events (unlike LightRenderer), we need to track state in prevButton
		prevButton, button := _prevMouseButton, ev.Buttons()
		_prevMouseButton = button
		drag := prevButton == button

		switch {
		case button&tcell.WheelDown != 0:
			return Event{Mouse, 0, &MouseEvent{y, x, -1, false, false, false, mod}}
		case button&tcell.WheelUp != 0:
			return Event{Mouse, 0, &MouseEvent{y, x, +1, false, false, false, mod}}
		case button&tcell.Button1 != 0 && !drag:
			// all potential double click events put their 'line' coordinate in the clickY array
			// double click event has two conditions, temporal and spatial, the first is checked here
			now := time.Now()
			if now.Sub(r.prevDownTime) < doubleClickDuration {
				r.clickY = append(r.clickY, y)
			} else {
				r.clickY = []int{y}
			}
			r.prevDownTime = now

			// detect double clicks (also check for spatial condition)
			n := len(r.clickY)
			double := n > 1 && r.clickY[n-2] == r.clickY[n-1]
			if double {
				// make sure two consecutive double clicks require four clicks
				r.clickY = []int{}
			}

			// fire single or double click event
			return Event{Mouse, 0, &MouseEvent{y, x, 0, true, !double, double, mod}}
		case button&tcell.Button2 != 0 && !drag:
			return Event{Mouse, 0, &MouseEvent{y, x, 0, false, true, false, mod}}
		case runtime.GOOS != "windows":

			// double and single taps on Windows don't quite work due to
			// the console acting on the events and not allowing us
			// to consume them.

			left := button&tcell.Button1 != 0
			down := left || button&tcell.Button3 != 0
			double := false
			if down {
				now := time.Now()
				if !left {
					r.clickY = []int{}
				} else if now.Sub(r.prevDownTime) < doubleClickDuration {
					r.clickY = append(r.clickY, x)
				} else {
					r.clickY = []int{x}
					r.prevDownTime = now
				}
			} else {
				if len(r.clickY) > 1 && r.clickY[0] == r.clickY[1] &&
					time.Now().Sub(r.prevDownTime) < doubleClickDuration {
					double = true
				}
			}

			return Event{Mouse, 0, &MouseEvent{y, x, 0, left, down, double, mod}}
		}

		// process keyboard:
	case *tcell.EventKey:
		mods := ev.Modifiers()
		none := mods == tcell.ModNone
		alt := (mods & tcell.ModAlt) > 0
		ctrl := (mods & tcell.ModCtrl) > 0
		shift := (mods & tcell.ModShift) > 0
		ctrlAlt := ctrl && alt
		altShift := alt && shift

		keyfn := func(r rune) Event {
			if alt {
				return CtrlAltKey(r)
			}
			return EventType(CtrlA.Int() - 'a' + int(r)).AsEvent()
		}
		switch ev.Key() {
		// section 1: Ctrl+(Alt)+[a-z]
		case tcell.KeyCtrlA:
			return keyfn('a')
		case tcell.KeyCtrlB:
			return keyfn('b')
		case tcell.KeyCtrlC:
			return keyfn('c')
		case tcell.KeyCtrlD:
			return keyfn('d')
		case tcell.KeyCtrlE:
			return keyfn('e')
		case tcell.KeyCtrlF:
			return keyfn('f')
		case tcell.KeyCtrlG:
			return keyfn('g')
		case tcell.KeyCtrlH:
			switch ev.Rune() {
			case 0:
				if ctrl {
					return Event{BSpace, 0, nil}
				}
			case rune(tcell.KeyCtrlH):
				switch {
				case ctrl:
					return keyfn('h')
				case alt:
					return Event{AltBS, 0, nil}
				case none, shift:
					return Event{BSpace, 0, nil}
				}
			}
		case tcell.KeyCtrlI:
			return keyfn('i')
		case tcell.KeyCtrlJ:
			return keyfn('j')
		case tcell.KeyCtrlK:
			return keyfn('k')
		case tcell.KeyCtrlL:
			return keyfn('l')
		case tcell.KeyCtrlM:
			return keyfn('m')
		case tcell.KeyCtrlN:
			return keyfn('n')
		case tcell.KeyCtrlO:
			return keyfn('o')
		case tcell.KeyCtrlP:
			return keyfn('p')
		case tcell.KeyCtrlQ:
			return keyfn('q')
		case tcell.KeyCtrlR:
			return keyfn('r')
		case tcell.KeyCtrlS:
			return keyfn('s')
		case tcell.KeyCtrlT:
			return keyfn('t')
		case tcell.KeyCtrlU:
			return keyfn('u')
		case tcell.KeyCtrlV:
			return keyfn('v')
		case tcell.KeyCtrlW:
			return keyfn('w')
		case tcell.KeyCtrlX:
			return keyfn('x')
		case tcell.KeyCtrlY:
			return keyfn('y')
		case tcell.KeyCtrlZ:
			return keyfn('z')
		// section 2: Ctrl+[ \]_]
		case tcell.KeyCtrlSpace:
			return Event{CtrlSpace, 0, nil}
		case tcell.KeyCtrlBackslash:
			return Event{CtrlBackSlash, 0, nil}
		case tcell.KeyCtrlRightSq:
			return Event{CtrlRightBracket, 0, nil}
		case tcell.KeyCtrlCarat:
			return Event{CtrlCaret, 0, nil}
		case tcell.KeyCtrlUnderscore:
			return Event{CtrlSlash, 0, nil}
		// section 3: (Alt)+Backspace2
		case tcell.KeyBackspace2:
			if alt {
				return Event{AltBS, 0, nil}
			}
			return Event{BSpace, 0, nil}

		// section 4: (Alt+Shift)+Key(Up|Down|Left|Right)
		case tcell.KeyUp:
			if altShift {
				return Event{AltSUp, 0, nil}
			}
			if shift {
				return Event{SUp, 0, nil}
			}
			if alt {
				return Event{AltUp, 0, nil}
			}
			return Event{Up, 0, nil}
		case tcell.KeyDown:
			if altShift {
				return Event{AltSDown, 0, nil}
			}
			if shift {
				return Event{SDown, 0, nil}
			}
			if alt {
				return Event{AltDown, 0, nil}
			}
			return Event{Down, 0, nil}
		case tcell.KeyLeft:
			if altShift {
				return Event{AltSLeft, 0, nil}
			}
			if shift {
				return Event{SLeft, 0, nil}
			}
			if alt {
				return Event{AltLeft, 0, nil}
			}
			return Event{Left, 0, nil}
		case tcell.KeyRight:
			if altShift {
				return Event{AltSRight, 0, nil}
			}
			if shift {
				return Event{SRight, 0, nil}
			}
			if alt {
				return Event{AltRight, 0, nil}
			}
			return Event{Right, 0, nil}

		// section 5: (Insert|Home|Delete|End|PgUp|PgDn|BackTab|F1-F12)
		case tcell.KeyInsert:
			return Event{Insert, 0, nil}
		case tcell.KeyHome:
			return Event{Home, 0, nil}
		case tcell.KeyDelete:
			return Event{Del, 0, nil}
		case tcell.KeyEnd:
			return Event{End, 0, nil}
		case tcell.KeyPgUp:
			return Event{PgUp, 0, nil}
		case tcell.KeyPgDn:
			return Event{PgDn, 0, nil}
		case tcell.KeyBacktab:
			return Event{BTab, 0, nil}
		case tcell.KeyF1:
			return Event{F1, 0, nil}
		case tcell.KeyF2:
			return Event{F2, 0, nil}
		case tcell.KeyF3:
			return Event{F3, 0, nil}
		case tcell.KeyF4:
			return Event{F4, 0, nil}
		case tcell.KeyF5:
			return Event{F5, 0, nil}
		case tcell.KeyF6:
			return Event{F6, 0, nil}
		case tcell.KeyF7:
			return Event{F7, 0, nil}
		case tcell.KeyF8:
			return Event{F8, 0, nil}
		case tcell.KeyF9:
			return Event{F9, 0, nil}
		case tcell.KeyF10:
			return Event{F10, 0, nil}
		case tcell.KeyF11:
			return Event{F11, 0, nil}
		case tcell.KeyF12:
			return Event{F12, 0, nil}

		// section 6: (Ctrl+Alt)+'rune'
		case tcell.KeyRune:
			r := ev.Rune()

			switch {
			// translate native key events to ascii control characters
			case r == ' ' && ctrl:
				return Event{CtrlSpace, 0, nil}
			// handle AltGr characters
			case ctrlAlt:
				return Event{Rune, r, nil} // dropping modifiers
			// simple characters (possibly with modifier)
			case alt:
				return AltKey(r)
			default:
				return Event{Rune, r, nil}
			}

		// section 7: Esc
		case tcell.KeyEsc:
			return Event{ESC, 0, nil}
		}
	}

	// section 8: Invalid
	return Event{Invalid, 0, nil}
}

func (r *FullscreenRenderer) Pause(clear bool) {
	if clear {
		_screen.Fini()
	}
}

func (r *FullscreenRenderer) Resume(clear bool, sigcont bool) {
	if clear {
		r.initScreen()
	}
}

func (r *FullscreenRenderer) Close() {
	_screen.Fini()
}

func (r *FullscreenRenderer) RefreshWindows(windows []Window) {
	// TODO
	for _, w := range windows {
		w.Refresh()
	}
	_screen.Show()
}

func (r *FullscreenRenderer) NewWindow(top int, left int, width int, height int, preview bool, borderStyle BorderStyle) Window {
	normal := ColNormal
	if preview {
		normal = ColPreview
	}
	return &TcellWindow{
		color:       r.theme.Colored,
		preview:     preview,
		top:         top,
		left:        left,
		width:       width,
		height:      height,
		normal:      normal,
		borderStyle: borderStyle}
}

func (w *TcellWindow) Close() {
	// TODO
}

func fill(x, y, w, h int, n ColorPair, r rune) {
	for ly := 0; ly <= h; ly++ {
		for lx := 0; lx <= w; lx++ {
			_screen.SetContent(x+lx, y+ly, r, nil, n.style())
		}
	}
}

func (w *TcellWindow) Erase() {
	fill(w.left-1, w.top, w.width+1, w.height, w.normal, ' ')
}

func (w *TcellWindow) Enclose(y int, x int) bool {
	return x >= w.left && x < (w.left+w.width) &&
		y >= w.top && y < (w.top+w.height)
}

func (w *TcellWindow) Move(y int, x int) {
	w.lastX = x
	w.lastY = y
	w.moveCursor = true
}

func (w *TcellWindow) MoveAndClear(y int, x int) {
	w.Move(y, x)
	for i := w.lastX; i < w.width; i++ {
		_screen.SetContent(i+w.left, w.lastY+w.top, rune(' '), nil, w.normal.style())
	}
	w.lastX = x
}

func (w *TcellWindow) Print(text string) {
	w.printString(text, w.normal)
}

func (w *TcellWindow) printString(text string, pair ColorPair) {
	lx := 0
	a := pair.Attr()

	style := pair.style()
	if a&AttrClear == 0 {
		style = style.
			Reverse(a&Attr(tcell.AttrReverse) != 0).
			Underline(a&Attr(tcell.AttrUnderline) != 0).
			Italic(a&Attr(tcell.AttrItalic) != 0).
			Blink(a&Attr(tcell.AttrBlink) != 0).
			Dim(a&Attr(tcell.AttrDim) != 0)
	}

	gr := uniseg.NewGraphemes(text)
	for gr.Next() {
		rs := gr.Runes()

		if len(rs) == 1 {
			r := rs[0]
			if r < rune(' ') { // ignore control characters
				continue
			} else if r == '\n' {
				w.lastY++
				lx = 0
				continue
			} else if r == '\u000D' { // skip carriage return
				continue
			}
		}
		var xPos = w.left + w.lastX + lx
		var yPos = w.top + w.lastY
		if xPos < (w.left+w.width) && yPos < (w.top+w.height) {
			_screen.SetContent(xPos, yPos, rs[0], rs[1:], style)
		}
		lx += runewidth.StringWidth(string(rs))
	}
	w.lastX += lx
}

func (w *TcellWindow) CPrint(pair ColorPair, text string) {
	w.printString(text, pair)
}

func (w *TcellWindow) fillString(text string, pair ColorPair) FillReturn {
	lx := 0
	a := pair.Attr()

	var style tcell.Style
	if w.color {
		style = pair.style()
	} else {
		style = w.normal.style()
	}
	style = style.
		Blink(a&Attr(tcell.AttrBlink) != 0).
		Bold(a&Attr(tcell.AttrBold) != 0).
		Dim(a&Attr(tcell.AttrDim) != 0).
		Reverse(a&Attr(tcell.AttrReverse) != 0).
		Underline(a&Attr(tcell.AttrUnderline) != 0).
		Italic(a&Attr(tcell.AttrItalic) != 0)

	gr := uniseg.NewGraphemes(text)
	for gr.Next() {
		rs := gr.Runes()
		if len(rs) == 1 && rs[0] == '\n' {
			w.lastY++
			w.lastX = 0
			lx = 0
			continue
		}

		// word wrap:
		xPos := w.left + w.lastX + lx
		if xPos >= (w.left + w.width) {
			w.lastY++
			w.lastX = 0
			lx = 0
			xPos = w.left
		}

		yPos := w.top + w.lastY
		if yPos >= (w.top + w.height) {
			return FillSuspend
		}

		_screen.SetContent(xPos, yPos, rs[0], rs[1:], style)
		lx += runewidth.StringWidth(string(rs))
	}
	w.lastX += lx
	if w.lastX == w.width {
		w.lastY++
		w.lastX = 0
		return FillNextLine
	}

	return FillContinue
}

func (w *TcellWindow) Fill(str string) FillReturn {
	return w.fillString(str, w.normal)
}

func (w *TcellWindow) CFill(fg Color, bg Color, a Attr, str string) FillReturn {
	if fg == colDefault {
		fg = w.normal.Fg()
	}
	if bg == colDefault {
		bg = w.normal.Bg()
	}
	return w.fillString(str, NewColorPair(fg, bg, a))
}

func (w *TcellWindow) drawBorder() {
	shape := w.borderStyle.shape
	if shape == BorderNone {
		return
	}

	left := w.left
	right := left + w.width
	top := w.top
	bot := top + w.height

	var style tcell.Style
	if w.color {
		if w.preview {
			style = ColPreviewBorder.style()
		} else {
			style = ColBorder.style()
		}
	} else {
		style = w.normal.style()
	}

	switch shape {
	case BorderRounded, BorderSharp, BorderHorizontal, BorderTop:
		for x := left; x < right; x++ {
			_screen.SetContent(x, top, w.borderStyle.horizontal, nil, style)
		}
	}
	switch shape {
	case BorderRounded, BorderSharp, BorderHorizontal, BorderBottom:
		for x := left; x < right; x++ {
			_screen.SetContent(x, bot-1, w.borderStyle.horizontal, nil, style)
		}
	}
	switch shape {
	case BorderRounded, BorderSharp, BorderVertical, BorderLeft:
		for y := top; y < bot; y++ {
			_screen.SetContent(left, y, w.borderStyle.vertical, nil, style)
		}
	}
	switch shape {
	case BorderRounded, BorderSharp, BorderVertical, BorderRight:
		for y := top; y < bot; y++ {
			_screen.SetContent(right-1, y, w.borderStyle.vertical, nil, style)
		}
	}
	switch shape {
	case BorderRounded, BorderSharp:
		_screen.SetContent(left, top, w.borderStyle.topLeft, nil, style)
		_screen.SetContent(right-1, top, w.borderStyle.topRight, nil, style)
		_screen.SetContent(left, bot-1, w.borderStyle.bottomLeft, nil, style)
		_screen.SetContent(right-1, bot-1, w.borderStyle.bottomRight, nil, style)
	}
}
