// +build tcell windows

package tui

import (
	"time"
	"unicode/utf8"

	"fmt"
	"os"

	"runtime"

	// https://github.com/gdamore/tcell/pull/135
	"github.com/junegunn/tcell"
	"github.com/junegunn/tcell/encoding"

	"github.com/junegunn/go-runewidth"
)

type ColorPair [2]Color

func (p ColorPair) fg() Color {
	return p[0]
}

func (p ColorPair) bg() Color {
	return p[1]
}

func (p ColorPair) style() tcell.Style {
	style := tcell.StyleDefault
	return style.Foreground(tcell.Color(p.fg())).Background(tcell.Color(p.bg()))
}

type Attr tcell.Style

type WindowTcell struct {
	LastX      int
	LastY      int
	MoveCursor bool
	Border     bool
}
type WindowImpl WindowTcell

const (
	Bold      Attr = Attr(tcell.AttrBold)
	Dim            = Attr(tcell.AttrDim)
	Blink          = Attr(tcell.AttrBlink)
	Reverse        = Attr(tcell.AttrReverse)
	Underline      = Attr(tcell.AttrUnderline)
	Italic         = Attr(tcell.AttrNone) // Not supported
)

const (
	AttrRegular Attr = 0
)

var (
	ColDefault      = ColorPair{colDefault, colDefault}
	ColNormal       ColorPair
	ColPrompt       ColorPair
	ColMatch        ColorPair
	ColCurrent      ColorPair
	ColCurrentMatch ColorPair
	ColSpinner      ColorPair
	ColInfo         ColorPair
	ColCursor       ColorPair
	ColSelected     ColorPair
	ColHeader       ColorPair
	ColBorder       ColorPair
	ColUser         ColorPair
)

func DefaultTheme() *ColorTheme {
	if _screen.Colors() >= 256 {
		return Dark256
	}
	return Default16
}

func PairFor(fg Color, bg Color) ColorPair {
	return [2]Color{fg, bg}
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

var (
	_screen tcell.Screen
	_mouse  bool
)

func initScreen() {
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(2)
	}
	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(2)
	}
	if _mouse {
		s.EnableMouse()
	} else {
		s.DisableMouse()
	}
	_screen = s
}

func Init(theme *ColorTheme, black bool, mouse bool) {
	encoding.Register()

	_mouse = mouse
	initScreen()

	_color = theme != nil
	if _color {
		InitTheme(theme, black)
	} else {
		theme = DefaultTheme()
	}
	ColNormal = ColorPair{theme.Fg, theme.Bg}
	ColPrompt = ColorPair{theme.Prompt, theme.Bg}
	ColMatch = ColorPair{theme.Match, theme.Bg}
	ColCurrent = ColorPair{theme.Current, theme.DarkBg}
	ColCurrentMatch = ColorPair{theme.CurrentMatch, theme.DarkBg}
	ColSpinner = ColorPair{theme.Spinner, theme.Bg}
	ColInfo = ColorPair{theme.Info, theme.Bg}
	ColCursor = ColorPair{theme.Cursor, theme.DarkBg}
	ColSelected = ColorPair{theme.Selected, theme.DarkBg}
	ColHeader = ColorPair{theme.Header, theme.Bg}
	ColBorder = ColorPair{theme.Border, theme.Bg}
}

func MaxX() int {
	ncols, _ := _screen.Size()
	return int(ncols)
}

func MaxY() int {
	_, nlines := _screen.Size()
	return int(nlines)
}

func (w *Window) win() *WindowTcell {
	return (*WindowTcell)(w.impl)
}

func (w *Window) X() int {
	return w.impl.LastX
}

func DoesAutoWrap() bool {
	return false
}

func Clear() {
	_screen.Sync()
	_screen.Clear()
}

func Refresh() {
	// noop
}

func GetChar() Event {
	ev := _screen.PollEvent()
	switch ev := ev.(type) {
	case *tcell.EventResize:
		return Event{Resize, 0, nil}

	// process mouse events:
	case *tcell.EventMouse:
		x, y := ev.Position()
		button := ev.Buttons()
		mod := ev.Modifiers() != 0
		if button&tcell.WheelDown != 0 {
			return Event{Mouse, 0, &MouseEvent{y, x, -1, false, false, mod}}
		} else if button&tcell.WheelUp != 0 {
			return Event{Mouse, 0, &MouseEvent{y, x, +1, false, false, mod}}
		} else if runtime.GOOS != "windows" {
			// double and single taps on Windows don't quite work due to
			// the console acting on the events and not allowing us
			// to consume them.

			down := button&tcell.Button1 != 0 // left
			double := false
			if down {
				now := time.Now()
				if now.Sub(_prevDownTime) < doubleClickDuration {
					_clickY = append(_clickY, x)
				} else {
					_clickY = []int{x}
					_prevDownTime = now
				}
			} else {
				if len(_clickY) > 1 && _clickY[0] == _clickY[1] &&
					time.Now().Sub(_prevDownTime) < doubleClickDuration {
					double = true
				}
			}

			return Event{Mouse, 0, &MouseEvent{y, x, 0, down, double, mod}}
		}

		// process keyboard:
	case *tcell.EventKey:
		alt := (ev.Modifiers() & tcell.ModAlt) > 0
		switch ev.Key() {
		case tcell.KeyCtrlA:
			return Event{CtrlA, 0, nil}
		case tcell.KeyCtrlB:
			return Event{CtrlB, 0, nil}
		case tcell.KeyCtrlC:
			return Event{CtrlC, 0, nil}
		case tcell.KeyCtrlD:
			return Event{CtrlD, 0, nil}
		case tcell.KeyCtrlE:
			return Event{CtrlE, 0, nil}
		case tcell.KeyCtrlF:
			return Event{CtrlF, 0, nil}
		case tcell.KeyCtrlG:
			return Event{CtrlG, 0, nil}
		case tcell.KeyCtrlJ:
			return Event{CtrlJ, 0, nil}
		case tcell.KeyCtrlK:
			return Event{CtrlK, 0, nil}
		case tcell.KeyCtrlL:
			return Event{CtrlL, 0, nil}
		case tcell.KeyCtrlM:
			if alt {
				return Event{AltEnter, 0, nil}
			}
			return Event{CtrlM, 0, nil}
		case tcell.KeyCtrlN:
			return Event{CtrlN, 0, nil}
		case tcell.KeyCtrlO:
			return Event{CtrlO, 0, nil}
		case tcell.KeyCtrlP:
			return Event{CtrlP, 0, nil}
		case tcell.KeyCtrlQ:
			return Event{CtrlQ, 0, nil}
		case tcell.KeyCtrlR:
			return Event{CtrlR, 0, nil}
		case tcell.KeyCtrlS:
			return Event{CtrlS, 0, nil}
		case tcell.KeyCtrlT:
			return Event{CtrlT, 0, nil}
		case tcell.KeyCtrlU:
			return Event{CtrlU, 0, nil}
		case tcell.KeyCtrlV:
			return Event{CtrlV, 0, nil}
		case tcell.KeyCtrlW:
			return Event{CtrlW, 0, nil}
		case tcell.KeyCtrlX:
			return Event{CtrlX, 0, nil}
		case tcell.KeyCtrlY:
			return Event{CtrlY, 0, nil}
		case tcell.KeyCtrlZ:
			return Event{CtrlZ, 0, nil}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if alt {
				return Event{AltBS, 0, nil}
			}
			return Event{BSpace, 0, nil}

		case tcell.KeyUp:
			return Event{Up, 0, nil}
		case tcell.KeyDown:
			return Event{Down, 0, nil}
		case tcell.KeyLeft:
			return Event{Left, 0, nil}
		case tcell.KeyRight:
			return Event{Right, 0, nil}

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

		case tcell.KeyTab:
			return Event{Tab, 0, nil}
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

		// ev.Ch doesn't work for some reason for space:
		case tcell.KeyRune:
			r := ev.Rune()
			if alt {
				switch r {
				case ' ':
					return Event{AltSpace, 0, nil}
				case '/':
					return Event{AltSlash, 0, nil}
				}
				if r >= 'a' && r <= 'z' {
					return Event{AltA + int(r) - 'a', 0, nil}
				}
				if r >= '0' && r <= '9' {
					return Event{Alt0 + int(r) - '0', 0, nil}
				}
			}
			return Event{Rune, r, nil}

		case tcell.KeyEsc:
			return Event{ESC, 0, nil}

		}
	}

	return Event{Invalid, 0, nil}
}

func Pause() {
	_screen.Fini()
}

func Resume() bool {
	initScreen()
	return true
}

func Close() {
	_screen.Fini()
}

func RefreshWindows(windows []*Window) {
	// TODO
	for _, w := range windows {
		if w.win().MoveCursor {
			_screen.ShowCursor(w.Left+w.win().LastX, w.Top+w.win().LastY)
			w.win().MoveCursor = false
		}
		w.win().LastX = 0
		w.win().LastY = 0
		if w.win().Border {
			w.DrawBorder()
		}
	}
	_screen.Show()
}

func NewWindow(top int, left int, width int, height int, border bool) *Window {
	// TODO
	win := new(WindowTcell)
	win.Border = border
	return &Window{
		impl:   (*WindowImpl)(win),
		Top:    top,
		Left:   left,
		Width:  width,
		Height: height,
	}
}

func (w *Window) Close() {
	// TODO
}

func fill(x, y, w, h int, r rune) {
	for ly := 0; ly <= h; ly++ {
		for lx := 0; lx <= w; lx++ {
			_screen.SetContent(x+lx, y+ly, r, nil, ColDefault.style())
		}
	}
}

func (w *Window) Erase() {
	// TODO
	fill(w.Left, w.Top, w.Width, w.Height, ' ')
}

func (w *Window) Enclose(y int, x int) bool {
	return x >= w.Left && x <= (w.Left+w.Width) &&
		y >= w.Top && y <= (w.Top+w.Height)
}

func (w *Window) Move(y int, x int) {
	w.win().LastX = x
	w.win().LastY = y
	w.win().MoveCursor = true
}

func (w *Window) MoveAndClear(y int, x int) {
	w.Move(y, x)
	for i := w.win().LastX; i < w.Width; i++ {
		_screen.SetContent(i+w.Left, w.win().LastY+w.Top, rune(' '), nil, ColDefault.style())
	}
	w.win().LastX = x
}

func (w *Window) Print(text string) {
	w.PrintString(text, ColDefault, 0)
}

func (w *Window) PrintString(text string, pair ColorPair, a Attr) {
	t := text
	lx := 0

	var style tcell.Style
	if _color {
		style = pair.style().
			Reverse(a&Attr(tcell.AttrReverse) != 0).
			Underline(a&Attr(tcell.AttrUnderline) != 0)
	} else {
		style = ColDefault.style().
			Reverse(a&Attr(tcell.AttrReverse) != 0 || pair == ColCurrent || pair == ColCurrentMatch).
			Underline(a&Attr(tcell.AttrUnderline) != 0 || pair == ColMatch || pair == ColCurrentMatch)
	}
	style = style.
		Blink(a&Attr(tcell.AttrBlink) != 0).
		Bold(a&Attr(tcell.AttrBold) != 0).
		Dim(a&Attr(tcell.AttrDim) != 0)

	for {
		if len(t) == 0 {
			break
		}
		r, size := utf8.DecodeRuneInString(t)
		t = t[size:]

		if r < rune(' ') { // ignore control characters
			continue
		}

		if r == '\n' {
			w.win().LastY++
			lx = 0
		} else {

			if r == '\u000D' { // skip carriage return
				continue
			}

			var xPos = w.Left + w.win().LastX + lx
			var yPos = w.Top + w.win().LastY
			if xPos < (w.Left+w.Width) && yPos < (w.Top+w.Height) {
				_screen.SetContent(xPos, yPos, r, nil, style)
			}
			lx += runewidth.RuneWidth(r)
		}
	}
	w.win().LastX += lx
}

func (w *Window) CPrint(pair ColorPair, a Attr, text string) {
	w.PrintString(text, pair, a)
}

func (w *Window) FillString(text string, pair ColorPair, a Attr) bool {
	lx := 0

	var style tcell.Style
	if _color {
		style = pair.style()
	} else {
		style = ColDefault.style()
	}
	style = style.
		Blink(a&Attr(tcell.AttrBlink) != 0).
		Bold(a&Attr(tcell.AttrBold) != 0).
		Dim(a&Attr(tcell.AttrDim) != 0).
		Reverse(a&Attr(tcell.AttrReverse) != 0).
		Underline(a&Attr(tcell.AttrUnderline) != 0)

	for _, r := range text {
		if r == '\n' {
			w.win().LastY++
			w.win().LastX = 0
			lx = 0
		} else {
			var xPos = w.Left + w.win().LastX + lx

			// word wrap:
			if xPos >= (w.Left + w.Width) {
				w.win().LastY++
				w.win().LastX = 0
				lx = 0
				xPos = w.Left
			}
			var yPos = w.Top + w.win().LastY

			if yPos >= (w.Top + w.Height) {
				return false
			}

			_screen.SetContent(xPos, yPos, r, nil, style)
			lx += runewidth.RuneWidth(r)
		}
	}
	w.win().LastX += lx

	return true
}

func (w *Window) Fill(str string) bool {
	return w.FillString(str, ColDefault, 0)
}

func (w *Window) CFill(str string, fg Color, bg Color, a Attr) bool {
	return w.FillString(str, ColorPair{fg, bg}, a)
}

func (w *Window) DrawBorder() {
	left := w.Left
	right := left + w.Width
	top := w.Top
	bot := top + w.Height

	var style tcell.Style
	if _color {
		style = ColBorder.style()
	} else {
		style = ColDefault.style()
	}

	for x := left; x < right; x++ {
		_screen.SetContent(x, top, tcell.RuneHLine, nil, style)
		_screen.SetContent(x, bot-1, tcell.RuneHLine, nil, style)
	}

	for y := top; y < bot; y++ {
		_screen.SetContent(left, y, tcell.RuneVLine, nil, style)
		_screen.SetContent(right-1, y, tcell.RuneVLine, nil, style)
	}

	_screen.SetContent(left, top, tcell.RuneULCorner, nil, style)
	_screen.SetContent(right-1, top, tcell.RuneURCorner, nil, style)
	_screen.SetContent(left, bot-1, tcell.RuneLLCorner, nil, style)
	_screen.SetContent(right-1, bot-1, tcell.RuneLRCorner, nil, style)
}
