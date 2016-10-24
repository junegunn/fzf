// +build termbox windows

package tui

import (
	"time"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

type ColorPair [2]Color
type Attr termbox.Attribute
type WindowImpl int // FIXME

const (
	// TODO
	_         = iota
	Bold      = Attr(termbox.AttrBold)
	Dim       = Attr(0) // termbox lacks this
	Blink     = Attr(0) // termbox lacks this
	Reverse   = Attr(termbox.AttrReverse)
	Underline = Attr(termbox.AttrUnderline)
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
	if termbox.SetOutputMode(termbox.OutputCurrent) == termbox.Output256 {
		return Dark256
	}
	return Default16
}

func PairFor(fg Color, bg Color) ColorPair {
	return [2]Color{fg, bg}
}

func (a Attr) Merge(b Attr) Attr {
	return a | b
}

func Init(theme *ColorTheme, black bool, mouse bool) {
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

	// TODO
	termbox.Init()
	if mouse {
		termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	}
}

func MaxX() int {
	ncols, _ := termbox.Size()
	return int(ncols)
}

func MaxY() int {
	_, nlines := termbox.Size()
	return int(nlines)
}

func Clear() {
	//termbox.Clear(ColNormal[0], ColNormal[1])
	termbox.Clear(termbox.ColorWhite, termbox.ColorBlack)
}

func Refresh() {
	termbox.SetCursor(_lastX, _lastY) //not sure if this is needed?
}

func GetChar() Event {
	ev := termbox.PollEvent()
	switch ev.Type {
	case termbox.EventResize:
		return Event{Invalid, 0, nil}
	// process mouse events:
	case termbox.EventMouse:
		down := ev.Key == termbox.MouseLeft
		double := false
		if down {
			now := time.Now()
			if now.Sub(_prevDownTime) < doubleClickDuration {
				_clickY = append(_clickY, ev.MouseY)
			} else {
				_clickY = []int{ev.MouseY}
			}
			_prevDownTime = now
		} else {
			if len(_clickY) > 1 && _clickY[0] == _clickY[1] &&
				time.Now().Sub(_prevDownTime) < doubleClickDuration {
				double = true
			}
		}

		return Event{Mouse, 0, &MouseEvent{ev.MouseY, ev.MouseX, 0, down, double, ev.Mod != 0}}
	}

	// process keyboard:
	switch ev.Key {
	case termbox.KeyCtrlA:
		return Event{CtrlA, 0, nil}
	case termbox.KeyCtrlB:
		return Event{CtrlB, 0, nil}
	case termbox.KeyCtrlC:
		return Event{CtrlC, 0, nil}
	case termbox.KeyCtrlD:
		return Event{CtrlD, 0, nil}
	case termbox.KeyCtrlE:
		return Event{CtrlE, 0, nil}
	case termbox.KeyCtrlF:
		return Event{CtrlF, 0, nil}
	case termbox.KeyCtrlG:
		return Event{CtrlG, 0, nil}
	case termbox.KeyCtrlJ:
		return Event{CtrlJ, 0, nil}
	case termbox.KeyCtrlK:
		return Event{CtrlK, 0, nil}
	case termbox.KeyCtrlL:
		return Event{CtrlL, 0, nil}
	case termbox.KeyCtrlM:
		return Event{CtrlM, 0, nil}
	case termbox.KeyCtrlN:
		return Event{CtrlN, 0, nil}
	case termbox.KeyCtrlO:
		return Event{CtrlO, 0, nil}
	case termbox.KeyCtrlP:
		return Event{CtrlP, 0, nil}
	case termbox.KeyCtrlQ:
		return Event{CtrlQ, 0, nil}
	case termbox.KeyCtrlR:
		return Event{CtrlR, 0, nil}
	case termbox.KeyCtrlS:
		return Event{CtrlS, 0, nil}
	case termbox.KeyCtrlT:
		return Event{CtrlT, 0, nil}
	case termbox.KeyCtrlU:
		return Event{CtrlU, 0, nil}
	case termbox.KeyCtrlV:
		return Event{CtrlV, 0, nil}
	case termbox.KeyCtrlW:
		return Event{CtrlW, 0, nil}
	case termbox.KeyCtrlX:
		return Event{CtrlX, 0, nil}
	case termbox.KeyCtrlY:
		return Event{CtrlY, 0, nil}
	case termbox.KeyCtrlZ:
		return Event{CtrlZ, 0, nil}
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		return Event{BSpace, 0, nil}

	case termbox.KeyArrowUp:
		return Event{Up, 0, nil}
	case termbox.KeyArrowDown:
		return Event{Down, 0, nil}
	case termbox.KeyArrowLeft:
		return Event{Left, 0, nil}
	case termbox.KeyArrowRight:
		return Event{Right, 0, nil}

	case termbox.KeyHome:
		return Event{Home, 0, nil}
	case termbox.KeyDelete:
		return Event{Del, 0, nil}
	case termbox.KeyEnd:
		return Event{End, 0, nil}
	case termbox.KeyPgup:
		return Event{PgUp, 0, nil}
	case termbox.KeyPgdn:
		return Event{PgDn, 0, nil}

	case termbox.KeyTab:
		return Event{Tab, 0, nil}

	case termbox.KeyF1:
		return Event{F1, 0, nil}
	case termbox.KeyF2:
		return Event{F2, 0, nil}
	case termbox.KeyF3:
		return Event{F3, 0, nil}
	case termbox.KeyF4:
		return Event{F4, 0, nil}
	case termbox.KeyF5:
		return Event{F5, 0, nil}
	case termbox.KeyF6:
		return Event{F6, 0, nil}
	case termbox.KeyF7:
		return Event{F7, 0, nil}
	case termbox.KeyF8:
		return Event{F8, 0, nil}
	case termbox.KeyF9:
		return Event{F9, 0, nil}
	case termbox.KeyF10:
		return Event{F10, 0, nil}
	case termbox.KeyF11:
		return Event{Invalid, 0, nil}
	case termbox.KeyF12:
		return Event{Invalid, 0, nil}

	// ev.Ch doesn't work for some reason for space:
	case termbox.KeySpace:
		return Event{Rune, ' ', nil}

	case termbox.KeyEsc:
		return Event{ESC, 0, nil}
	}

	return Event{Rune, ev.Ch, nil}
}

func Pause() {
	// TODO
}

func Close() {
	termbox.Close()
}

func RefreshWindows(windows []*Window) {
	// TODO
	termbox.Flush()
	termbox.SetCursor(_lastX, _lastY)
}

func NewWindow(top int, left int, width int, height int, border bool) *Window {
	// TODO
	return &Window{
		Top:    top,
		Left:   left,
		Width:  width,
		Height: height,
	}
}

func (w *Window) Close() {
	// TODO
}

func (w *Window) Erase() {
	// TODO
}

var (
	_lastX      int
	_lastY      int
	_moveCursor = false
)

func (w *Window) Enclose(y int, x int) bool {
	return y >= w.Left && y <= (w.Left+w.Width) &&
		x >= w.Top && y <= (w.Top+w.Height)
}

func (w *Window) Move(y int, x int) {
	_lastX = x
	_lastY = y
	_moveCursor = true
}

func (w *Window) MoveAndClear(y int, x int) {
	w.Move(y, x)
	r, _ := utf8.DecodeRuneInString(" ")
	for i := _lastX; i < MaxX(); i++ {
		//TODO: get colors right
		termbox.SetCell(i, _lastY, r, termbox.ColorWhite, termbox.ColorBlack)
	}
}

func (w *Window) Print(text string) {
	//TODO: get colors right
	w.PrintPalette(text, termbox.ColorWhite, termbox.ColorBlack)
}

func (w *Window) PrintPalette(text string, fg, bg termbox.Attribute) {
	t := text
	lx := 0

	for {
		if len(t) == 0 {
			break
		}
		r, size := utf8.DecodeRuneInString(t)
		t = t[size:]

		termbox.SetCell(_lastX+lx, _lastY, r, fg, bg)
		lx++

	}
	_lastX += lx
}

func (w *Window) CPrint(pair ColorPair, a Attr, text string) {
	//w.PrintPalette(text, Palette[pair].Fg|Color(a), Palette[pair].Bg)
	//TODO: get colors right
	w.PrintPalette(text, termbox.ColorWhite|termbox.Attribute(a), termbox.ColorBlack)
}

func (w *Window) Fill(str string) bool {
	// TODO
	return false
}

func (w *Window) CFill(str string, fg Color, bg Color, a Attr) bool {
	// TODO
	return false
}
