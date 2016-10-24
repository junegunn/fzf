// +build termbox windows

package tui

import (
	"github.com/nsf/termbox-go"
)

type ColorPair [2]Color
type Attr uint16
type WindowImpl int // FIXME

const (
	// TODO
	_ = iota
	Bold
	Dim
	Blink
	Reverse
	Underline
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
}

func MaxX() int {
	// TODO
	return 80
}

func MaxY() int {
	// TODO
	return 24
}

func Clear() {
	// TODO
}

func Refresh() {
	// TODO
}

func GetChar() Event {
	// TODO
	return Event{}
}

func Pause() {
	// TODO
}

func Close() {
	// TODO
}

func RefreshWindows(windows []*Window) {
	// TODO
}

func NewWindow(top int, left int, width int, height int, border bool) *Window {
	// TODO
	return &Window{}
}

func (w *Window) Close() {
	// TODO
}

func (w *Window) Erase() {
	// TODO
}

func (w *Window) Enclose(y int, x int) bool {
	// TODO
	return false
}

func (w *Window) Move(y int, x int) {
	// TODO
}

func (w *Window) MoveAndClear(y int, x int) {
	// TODO
}

func (w *Window) Print(text string) {
	// TODO
}

func (w *Window) CPrint(pair ColorPair, a Attr, text string) {
	// TODO
}

func (w *Window) Fill(str string) bool {
	// TODO
	return false
}

func (w *Window) CFill(str string, fg Color, bg Color, a Attr) bool {
	// TODO
	return false
}
