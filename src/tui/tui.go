package tui

import (
	"time"
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
	Resize
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
	F11
	F12

	AltEnter
	AltSpace
	AltSlash
	AltBS

	Alt0
)

const ( // Reset iota
	AltA = Alt0 + 'a' - '0' + iota
	AltB
	AltC
	AltD
	AltE
	AltF
	AltZ = AltA + 'z' - 'a'
)

const (
	doubleClickDuration = 500 * time.Millisecond
)

type Color int32

func (c Color) is24() bool {
	return c > 0 && (c&(1<<24)) > 0
}

const (
	colUndefined Color = -2
	colDefault         = -1
)

const (
	colBlack Color = iota
	colRed
	colGreen
	colYellow
	colBlue
	colMagenta
	colCyan
	colWhite
)

type ColorTheme struct {
	Fg           Color
	Bg           Color
	DarkBg       Color
	Prompt       Color
	Match        Color
	Current      Color
	CurrentMatch Color
	Spinner      Color
	Info         Color
	Cursor       Color
	Selected     Color
	Header       Color
	Border       Color
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
	_color        bool
	_prevDownTime time.Time
	_clickY       []int
	Default16     *ColorTheme
	Dark256       *ColorTheme
	Light256      *ColorTheme
)

type Window struct {
	impl   *WindowImpl
	Top    int
	Left   int
	Width  int
	Height int
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
	Default16 = &ColorTheme{
		Fg:           colDefault,
		Bg:           colDefault,
		DarkBg:       colBlack,
		Prompt:       colBlue,
		Match:        colGreen,
		Current:      colYellow,
		CurrentMatch: colGreen,
		Spinner:      colGreen,
		Info:         colWhite,
		Cursor:       colRed,
		Selected:     colMagenta,
		Header:       colCyan,
		Border:       colBlack}
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

func InitTheme(theme *ColorTheme, black bool) {
	_color = theme != nil
	if !_color {
		return
	}

	baseTheme := DefaultTheme()
	if black {
		theme.Bg = colBlack
	}

	o := func(a Color, b Color) Color {
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
