package tui

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Types of user action
type EventType int

const (
	Rune EventType = iota

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
	CtrlSpace

	// https://apple.stackexchange.com/questions/24261/how-do-i-send-c-that-is-control-slash-to-the-terminal
	CtrlBackSlash
	CtrlRightBracket
	CtrlCaret
	CtrlSlash

	Invalid
	Resize
	Mouse
	DoubleClick
	LeftClick
	RightClick

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
	Insert

	SUp
	SDown
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

	Change
	BackwardEOF
	Start
	Load
	Focus

	AltBS

	AltUp
	AltDown
	AltLeft
	AltRight

	AltSUp
	AltSDown
	AltSLeft
	AltSRight

	Alt
	CtrlAlt
)

func (t EventType) AsEvent() Event {
	return Event{t, 0, nil}
}

func (t EventType) Int() int {
	return int(t)
}

func (t EventType) Byte() byte {
	return byte(t)
}

func (e Event) Comparable() Event {
	// Ignore MouseEvent pointer
	return Event{e.Type, e.Char, nil}
}

func Key(r rune) Event {
	return Event{Rune, r, nil}
}

func AltKey(r rune) Event {
	return Event{Alt, r, nil}
}

func CtrlAltKey(r rune) Event {
	return Event{CtrlAlt, r, nil}
}

const (
	doubleClickDuration = 500 * time.Millisecond
)

type Color int32

func (c Color) IsDefault() bool {
	return c == colDefault
}

func (c Color) is24() bool {
	return c > 0 && (c&(1<<24)) > 0
}

type ColorAttr struct {
	Color Color
	Attr  Attr
}

func NewColorAttr() ColorAttr {
	return ColorAttr{Color: colUndefined, Attr: AttrUndefined}
}

const (
	colUndefined Color = -2
	colDefault   Color = -1
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

type FillReturn int

const (
	FillContinue FillReturn = iota
	FillNextLine
	FillSuspend
)

type ColorPair struct {
	fg   Color
	bg   Color
	attr Attr
}

func HexToColor(rrggbb string) Color {
	r, _ := strconv.ParseInt(rrggbb[1:3], 16, 0)
	g, _ := strconv.ParseInt(rrggbb[3:5], 16, 0)
	b, _ := strconv.ParseInt(rrggbb[5:7], 16, 0)
	return Color((1 << 24) + (r << 16) + (g << 8) + b)
}

func NewColorPair(fg Color, bg Color, attr Attr) ColorPair {
	return ColorPair{fg, bg, attr}
}

func (p ColorPair) Fg() Color {
	return p.fg
}

func (p ColorPair) Bg() Color {
	return p.bg
}

func (p ColorPair) Attr() Attr {
	return p.attr
}

func (p ColorPair) HasBg() bool {
	return p.attr&Reverse == 0 && p.bg != colDefault ||
		p.attr&Reverse > 0 && p.fg != colDefault
}

func (p ColorPair) merge(other ColorPair, except Color) ColorPair {
	dup := p
	dup.attr = dup.attr.Merge(other.attr)
	if other.fg != except {
		dup.fg = other.fg
	}
	if other.bg != except {
		dup.bg = other.bg
	}
	return dup
}

func (p ColorPair) WithAttr(attr Attr) ColorPair {
	dup := p
	dup.attr = dup.attr.Merge(attr)
	return dup
}

func (p ColorPair) MergeAttr(other ColorPair) ColorPair {
	return p.WithAttr(other.attr)
}

func (p ColorPair) Merge(other ColorPair) ColorPair {
	return p.merge(other, colUndefined)
}

func (p ColorPair) MergeNonDefault(other ColorPair) ColorPair {
	return p.merge(other, colDefault)
}

type ColorTheme struct {
	Colored      bool
	Input        ColorAttr
	Disabled     ColorAttr
	Fg           ColorAttr
	Bg           ColorAttr
	PreviewFg    ColorAttr
	PreviewBg    ColorAttr
	DarkBg       ColorAttr
	Gutter       ColorAttr
	Prompt       ColorAttr
	Match        ColorAttr
	Current      ColorAttr
	CurrentMatch ColorAttr
	Spinner      ColorAttr
	Info         ColorAttr
	Cursor       ColorAttr
	Selected     ColorAttr
	Header       ColorAttr
	Separator    ColorAttr
	Scrollbar    ColorAttr
	Border       ColorAttr
	BorderLabel  ColorAttr
	PreviewLabel ColorAttr
}

type Event struct {
	Type       EventType
	Char       rune
	MouseEvent *MouseEvent
}

type MouseEvent struct {
	Y      int
	X      int
	S      int
	Left   bool
	Down   bool
	Double bool
	Mod    bool
}

type BorderShape int

const (
	BorderNone BorderShape = iota
	BorderRounded
	BorderSharp
	BorderBold
	BorderDouble
	BorderHorizontal
	BorderVertical
	BorderTop
	BorderBottom
	BorderLeft
	BorderRight
)

func (s BorderShape) HasRight() bool {
	switch s {
	case BorderNone, BorderLeft, BorderTop, BorderBottom, BorderHorizontal: // No right
		return false
	}
	return true
}

func (s BorderShape) HasTop() bool {
	switch s {
	case BorderNone, BorderLeft, BorderRight, BorderBottom, BorderVertical: // No top
		return false
	}
	return true
}

type BorderStyle struct {
	shape       BorderShape
	horizontal  rune
	vertical    rune
	topLeft     rune
	topRight    rune
	bottomLeft  rune
	bottomRight rune
}

type BorderCharacter int

func MakeBorderStyle(shape BorderShape, unicode bool) BorderStyle {
	if !unicode {
		return BorderStyle{
			shape:       shape,
			horizontal:  '-',
			vertical:    '|',
			topLeft:     '+',
			topRight:    '+',
			bottomLeft:  '+',
			bottomRight: '+',
		}
	}
	switch shape {
	case BorderSharp:
		return BorderStyle{
			shape:       shape,
			horizontal:  '─',
			vertical:    '│',
			topLeft:     '┌',
			topRight:    '┐',
			bottomLeft:  '└',
			bottomRight: '┘',
		}
	case BorderBold:
		return BorderStyle{
			shape:       shape,
			horizontal:  '━',
			vertical:    '┃',
			topLeft:     '┏',
			topRight:    '┓',
			bottomLeft:  '┗',
			bottomRight: '┛',
		}
	case BorderDouble:
		return BorderStyle{
			shape:       shape,
			horizontal:  '═',
			vertical:    '║',
			topLeft:     '╔',
			topRight:    '╗',
			bottomLeft:  '╚',
			bottomRight: '╝',
		}
	}
	return BorderStyle{
		shape:       shape,
		horizontal:  '─',
		vertical:    '│',
		topLeft:     '╭',
		topRight:    '╮',
		bottomLeft:  '╰',
		bottomRight: '╯',
	}
}

func MakeTransparentBorder() BorderStyle {
	return BorderStyle{
		shape:       BorderRounded,
		horizontal:  ' ',
		vertical:    ' ',
		topLeft:     ' ',
		topRight:    ' ',
		bottomLeft:  ' ',
		bottomRight: ' '}
}

type Renderer interface {
	Init()
	Resize(maxHeightFunc func(int) int)
	Pause(clear bool)
	Resume(clear bool, sigcont bool)
	Clear()
	RefreshWindows(windows []Window)
	Refresh()
	Close()
	NeedScrollbarRedraw() bool

	GetChar() Event

	MaxX() int
	MaxY() int

	NewWindow(top int, left int, width int, height int, preview bool, borderStyle BorderStyle) Window
}

type Window interface {
	Top() int
	Left() int
	Width() int
	Height() int

	DrawHBorder()
	Refresh()
	FinishFill()
	Close()

	X() int
	Y() int
	Enclose(y int, x int) bool

	Move(y int, x int)
	MoveAndClear(y int, x int)
	Print(text string)
	CPrint(color ColorPair, text string)
	Fill(text string) FillReturn
	CFill(fg Color, bg Color, attr Attr, text string) FillReturn
	Erase()
}

type FullscreenRenderer struct {
	theme        *ColorTheme
	mouse        bool
	forceBlack   bool
	prevDownTime time.Time
	clicks       [][2]int
}

func NewFullscreenRenderer(theme *ColorTheme, forceBlack bool, mouse bool) Renderer {
	r := &FullscreenRenderer{
		theme:        theme,
		mouse:        mouse,
		forceBlack:   forceBlack,
		prevDownTime: time.Unix(0, 0),
		clicks:       [][2]int{}}
	return r
}

var (
	Default16 *ColorTheme
	Dark256   *ColorTheme
	Light256  *ColorTheme

	ColPrompt               ColorPair
	ColNormal               ColorPair
	ColInput                ColorPair
	ColDisabled             ColorPair
	ColMatch                ColorPair
	ColCursor               ColorPair
	ColCursorEmpty          ColorPair
	ColSelected             ColorPair
	ColCurrent              ColorPair
	ColCurrentMatch         ColorPair
	ColCurrentCursor        ColorPair
	ColCurrentCursorEmpty   ColorPair
	ColCurrentSelected      ColorPair
	ColCurrentSelectedEmpty ColorPair
	ColSpinner              ColorPair
	ColInfo                 ColorPair
	ColHeader               ColorPair
	ColSeparator            ColorPair
	ColScrollbar            ColorPair
	ColBorder               ColorPair
	ColPreview              ColorPair
	ColPreviewBorder        ColorPair
	ColBorderLabel          ColorPair
	ColPreviewLabel         ColorPair
)

func EmptyTheme() *ColorTheme {
	return &ColorTheme{
		Colored:      true,
		Input:        ColorAttr{colUndefined, AttrUndefined},
		Fg:           ColorAttr{colUndefined, AttrUndefined},
		Bg:           ColorAttr{colUndefined, AttrUndefined},
		DarkBg:       ColorAttr{colUndefined, AttrUndefined},
		Prompt:       ColorAttr{colUndefined, AttrUndefined},
		Match:        ColorAttr{colUndefined, AttrUndefined},
		Current:      ColorAttr{colUndefined, AttrUndefined},
		CurrentMatch: ColorAttr{colUndefined, AttrUndefined},
		Spinner:      ColorAttr{colUndefined, AttrUndefined},
		Info:         ColorAttr{colUndefined, AttrUndefined},
		Cursor:       ColorAttr{colUndefined, AttrUndefined},
		Selected:     ColorAttr{colUndefined, AttrUndefined},
		Header:       ColorAttr{colUndefined, AttrUndefined},
		Border:       ColorAttr{colUndefined, AttrUndefined},
		BorderLabel:  ColorAttr{colUndefined, AttrUndefined},
		Disabled:     ColorAttr{colUndefined, AttrUndefined},
		PreviewFg:    ColorAttr{colUndefined, AttrUndefined},
		PreviewBg:    ColorAttr{colUndefined, AttrUndefined},
		Gutter:       ColorAttr{colUndefined, AttrUndefined},
		PreviewLabel: ColorAttr{colUndefined, AttrUndefined},
		Separator:    ColorAttr{colUndefined, AttrUndefined},
		Scrollbar:    ColorAttr{colUndefined, AttrUndefined},
	}
}

func NoColorTheme() *ColorTheme {
	return &ColorTheme{
		Colored:      false,
		Input:        ColorAttr{colDefault, AttrRegular},
		Fg:           ColorAttr{colDefault, AttrRegular},
		Bg:           ColorAttr{colDefault, AttrRegular},
		DarkBg:       ColorAttr{colDefault, AttrRegular},
		Prompt:       ColorAttr{colDefault, AttrRegular},
		Match:        ColorAttr{colDefault, Underline},
		Current:      ColorAttr{colDefault, Reverse},
		CurrentMatch: ColorAttr{colDefault, Reverse | Underline},
		Spinner:      ColorAttr{colDefault, AttrRegular},
		Info:         ColorAttr{colDefault, AttrRegular},
		Cursor:       ColorAttr{colDefault, AttrRegular},
		Selected:     ColorAttr{colDefault, AttrRegular},
		Header:       ColorAttr{colDefault, AttrRegular},
		Border:       ColorAttr{colDefault, AttrRegular},
		BorderLabel:  ColorAttr{colDefault, AttrRegular},
		Disabled:     ColorAttr{colDefault, AttrRegular},
		PreviewFg:    ColorAttr{colDefault, AttrRegular},
		PreviewBg:    ColorAttr{colDefault, AttrRegular},
		Gutter:       ColorAttr{colDefault, AttrRegular},
		PreviewLabel: ColorAttr{colDefault, AttrRegular},
		Separator:    ColorAttr{colDefault, AttrRegular},
		Scrollbar:    ColorAttr{colDefault, AttrRegular},
	}
}

func errorExit(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}

func init() {
	Default16 = &ColorTheme{
		Colored:      true,
		Input:        ColorAttr{colDefault, AttrUndefined},
		Fg:           ColorAttr{colDefault, AttrUndefined},
		Bg:           ColorAttr{colDefault, AttrUndefined},
		DarkBg:       ColorAttr{colBlack, AttrUndefined},
		Prompt:       ColorAttr{colBlue, AttrUndefined},
		Match:        ColorAttr{colGreen, AttrUndefined},
		Current:      ColorAttr{colYellow, AttrUndefined},
		CurrentMatch: ColorAttr{colGreen, AttrUndefined},
		Spinner:      ColorAttr{colGreen, AttrUndefined},
		Info:         ColorAttr{colWhite, AttrUndefined},
		Cursor:       ColorAttr{colRed, AttrUndefined},
		Selected:     ColorAttr{colMagenta, AttrUndefined},
		Header:       ColorAttr{colCyan, AttrUndefined},
		Border:       ColorAttr{colBlack, AttrUndefined},
		BorderLabel:  ColorAttr{colWhite, AttrUndefined},
		Disabled:     ColorAttr{colUndefined, AttrUndefined},
		PreviewFg:    ColorAttr{colUndefined, AttrUndefined},
		PreviewBg:    ColorAttr{colUndefined, AttrUndefined},
		Gutter:       ColorAttr{colUndefined, AttrUndefined},
		PreviewLabel: ColorAttr{colUndefined, AttrUndefined},
		Separator:    ColorAttr{colUndefined, AttrUndefined},
		Scrollbar:    ColorAttr{colUndefined, AttrUndefined},
	}
	Dark256 = &ColorTheme{
		Colored:      true,
		Input:        ColorAttr{colDefault, AttrUndefined},
		Fg:           ColorAttr{colDefault, AttrUndefined},
		Bg:           ColorAttr{colDefault, AttrUndefined},
		DarkBg:       ColorAttr{236, AttrUndefined},
		Prompt:       ColorAttr{110, AttrUndefined},
		Match:        ColorAttr{108, AttrUndefined},
		Current:      ColorAttr{254, AttrUndefined},
		CurrentMatch: ColorAttr{151, AttrUndefined},
		Spinner:      ColorAttr{148, AttrUndefined},
		Info:         ColorAttr{144, AttrUndefined},
		Cursor:       ColorAttr{161, AttrUndefined},
		Selected:     ColorAttr{168, AttrUndefined},
		Header:       ColorAttr{109, AttrUndefined},
		Border:       ColorAttr{59, AttrUndefined},
		BorderLabel:  ColorAttr{145, AttrUndefined},
		Disabled:     ColorAttr{colUndefined, AttrUndefined},
		PreviewFg:    ColorAttr{colUndefined, AttrUndefined},
		PreviewBg:    ColorAttr{colUndefined, AttrUndefined},
		Gutter:       ColorAttr{colUndefined, AttrUndefined},
		PreviewLabel: ColorAttr{colUndefined, AttrUndefined},
		Separator:    ColorAttr{colUndefined, AttrUndefined},
		Scrollbar:    ColorAttr{colUndefined, AttrUndefined},
	}
	Light256 = &ColorTheme{
		Colored:      true,
		Input:        ColorAttr{colDefault, AttrUndefined},
		Fg:           ColorAttr{colDefault, AttrUndefined},
		Bg:           ColorAttr{colDefault, AttrUndefined},
		DarkBg:       ColorAttr{251, AttrUndefined},
		Prompt:       ColorAttr{25, AttrUndefined},
		Match:        ColorAttr{66, AttrUndefined},
		Current:      ColorAttr{237, AttrUndefined},
		CurrentMatch: ColorAttr{23, AttrUndefined},
		Spinner:      ColorAttr{65, AttrUndefined},
		Info:         ColorAttr{101, AttrUndefined},
		Cursor:       ColorAttr{161, AttrUndefined},
		Selected:     ColorAttr{168, AttrUndefined},
		Header:       ColorAttr{31, AttrUndefined},
		Border:       ColorAttr{145, AttrUndefined},
		BorderLabel:  ColorAttr{59, AttrUndefined},
		Disabled:     ColorAttr{colUndefined, AttrUndefined},
		PreviewFg:    ColorAttr{colUndefined, AttrUndefined},
		PreviewBg:    ColorAttr{colUndefined, AttrUndefined},
		Gutter:       ColorAttr{colUndefined, AttrUndefined},
		PreviewLabel: ColorAttr{colUndefined, AttrUndefined},
		Separator:    ColorAttr{colUndefined, AttrUndefined},
		Scrollbar:    ColorAttr{colUndefined, AttrUndefined},
	}
}

func initTheme(theme *ColorTheme, baseTheme *ColorTheme, forceBlack bool) {
	if forceBlack {
		theme.Bg = ColorAttr{colBlack, AttrUndefined}
	}

	o := func(a ColorAttr, b ColorAttr) ColorAttr {
		c := a
		if b.Color != colUndefined {
			c.Color = b.Color
		}
		if b.Attr != AttrUndefined {
			c.Attr = b.Attr
		}
		return c
	}
	theme.Input = o(baseTheme.Input, theme.Input)
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
	theme.BorderLabel = o(baseTheme.BorderLabel, theme.BorderLabel)

	// These colors are not defined in the base themes
	theme.Disabled = o(theme.Input, theme.Disabled)
	theme.Gutter = o(theme.DarkBg, theme.Gutter)
	theme.PreviewFg = o(theme.Fg, theme.PreviewFg)
	theme.PreviewBg = o(theme.Bg, theme.PreviewBg)
	theme.PreviewLabel = o(theme.BorderLabel, theme.PreviewLabel)
	theme.Separator = o(theme.Border, theme.Separator)
	theme.Scrollbar = o(theme.Border, theme.Scrollbar)

	initPalette(theme)
}

func initPalette(theme *ColorTheme) {
	pair := func(fg, bg ColorAttr) ColorPair {
		if fg.Color == colDefault && (fg.Attr&Reverse) > 0 {
			bg.Color = colDefault
		}
		return ColorPair{fg.Color, bg.Color, fg.Attr}
	}
	blank := theme.Fg
	blank.Attr = AttrRegular

	ColPrompt = pair(theme.Prompt, theme.Bg)
	ColNormal = pair(theme.Fg, theme.Bg)
	ColInput = pair(theme.Input, theme.Bg)
	ColDisabled = pair(theme.Disabled, theme.Bg)
	ColMatch = pair(theme.Match, theme.Bg)
	ColCursor = pair(theme.Cursor, theme.Gutter)
	ColCursorEmpty = pair(blank, theme.Gutter)
	ColSelected = pair(theme.Selected, theme.Gutter)
	ColCurrent = pair(theme.Current, theme.DarkBg)
	ColCurrentMatch = pair(theme.CurrentMatch, theme.DarkBg)
	ColCurrentCursor = pair(theme.Cursor, theme.DarkBg)
	ColCurrentCursorEmpty = pair(blank, theme.DarkBg)
	ColCurrentSelected = pair(theme.Selected, theme.DarkBg)
	ColCurrentSelectedEmpty = pair(blank, theme.DarkBg)
	ColSpinner = pair(theme.Spinner, theme.Bg)
	ColInfo = pair(theme.Info, theme.Bg)
	ColHeader = pair(theme.Header, theme.Bg)
	ColSeparator = pair(theme.Separator, theme.Bg)
	ColScrollbar = pair(theme.Scrollbar, theme.Bg)
	ColBorder = pair(theme.Border, theme.Bg)
	ColBorderLabel = pair(theme.BorderLabel, theme.Bg)
	ColPreviewLabel = pair(theme.PreviewLabel, theme.PreviewBg)
	ColPreview = pair(theme.PreviewFg, theme.PreviewBg)
	ColPreviewBorder = pair(theme.Border, theme.PreviewBg)
}
