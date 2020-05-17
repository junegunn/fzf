package tui

import (
	"fmt"
	"os"
	"strconv"
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

	AltSpace
	AltSlash
	AltBS

	AltUp
	AltDown
	AltLeft
	AltRight

	Alt0
)

const ( // Reset iota
	AltA = Alt0 + 'a' - '0' + iota
	AltB
	AltC
	AltD
	AltE
	AltF
	AltZ     = AltA + 'z' - 'a'
	CtrlAltA = AltZ + 1
	CtrlAltM = CtrlAltA + 'm' - 'a'
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
	fg Color
	bg Color
	id int
}

func HexToColor(rrggbb string) Color {
	r, _ := strconv.ParseInt(rrggbb[1:3], 16, 0)
	g, _ := strconv.ParseInt(rrggbb[3:5], 16, 0)
	b, _ := strconv.ParseInt(rrggbb[5:7], 16, 0)
	return Color((1 << 24) + (r << 16) + (g << 8) + b)
}

func NewColorPair(fg Color, bg Color) ColorPair {
	return ColorPair{fg, bg, -1}
}

func (p ColorPair) Fg() Color {
	return p.fg
}

func (p ColorPair) Bg() Color {
	return p.bg
}

type ColorTheme struct {
	Fg           Color
	Bg           Color
	PreviewFg    Color
	PreviewBg    Color
	DarkBg       Color
	Gutter       Color
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
	BorderHorizontal
)

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
	if unicode {
		if shape == BorderRounded {
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
		return BorderStyle{
			shape:       shape,
			horizontal:  '─',
			vertical:    '│',
			topLeft:     '┌',
			topRight:    '┐',
			bottomLeft:  '└',
			bottomRight: '┘',
		}
	}
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
	Pause(clear bool)
	Resume(clear bool, sigcont bool)
	Clear()
	RefreshWindows(windows []Window)
	Refresh()
	Close()

	GetChar() Event

	MaxX() int
	MaxY() int
	DoesAutoWrap() bool

	NewWindow(top int, left int, width int, height int, preview bool, borderStyle BorderStyle) Window
}

type Window interface {
	Top() int
	Left() int
	Width() int
	Height() int

	Refresh()
	FinishFill()
	Close()

	X() int
	Y() int
	Enclose(y int, x int) bool

	Move(y int, x int)
	MoveAndClear(y int, x int)
	Print(text string)
	CPrint(color ColorPair, attr Attr, text string)
	Fill(text string) FillReturn
	CFill(fg Color, bg Color, attr Attr, text string) FillReturn
	Erase()
}

type FullscreenRenderer struct {
	theme        *ColorTheme
	mouse        bool
	forceBlack   bool
	prevDownTime time.Time
	clickY       []int
}

func NewFullscreenRenderer(theme *ColorTheme, forceBlack bool, mouse bool) Renderer {
	r := &FullscreenRenderer{
		theme:        theme,
		mouse:        mouse,
		forceBlack:   forceBlack,
		prevDownTime: time.Unix(0, 0),
		clickY:       []int{}}
	return r
}

var (
	Default16 *ColorTheme
	Dark256   *ColorTheme
	Light256  *ColorTheme

	ColPrompt          ColorPair
	ColNormal          ColorPair
	ColMatch           ColorPair
	ColCursor          ColorPair
	ColSelected        ColorPair
	ColCurrent         ColorPair
	ColCurrentMatch    ColorPair
	ColCurrentCursor   ColorPair
	ColCurrentSelected ColorPair
	ColSpinner         ColorPair
	ColInfo            ColorPair
	ColHeader          ColorPair
	ColBorder          ColorPair
	ColPreview         ColorPair
	ColPreviewBorder   ColorPair
)

func EmptyTheme() *ColorTheme {
	return &ColorTheme{
		Fg:           colUndefined,
		Bg:           colUndefined,
		PreviewFg:    colUndefined,
		PreviewBg:    colUndefined,
		DarkBg:       colUndefined,
		Gutter:       colUndefined,
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

func errorExit(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}

func init() {
	Default16 = &ColorTheme{
		Fg:           colDefault,
		Bg:           colDefault,
		PreviewFg:    colUndefined,
		PreviewBg:    colUndefined,
		DarkBg:       colBlack,
		Gutter:       colUndefined,
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
		PreviewFg:    colUndefined,
		PreviewBg:    colUndefined,
		DarkBg:       236,
		Gutter:       colUndefined,
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
		PreviewFg:    colUndefined,
		PreviewBg:    colUndefined,
		DarkBg:       251,
		Gutter:       colUndefined,
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

func initTheme(theme *ColorTheme, baseTheme *ColorTheme, forceBlack bool) {
	if theme == nil {
		initPalette(theme)
		return
	}

	if forceBlack {
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
	theme.PreviewFg = o(theme.Fg, o(baseTheme.PreviewFg, theme.PreviewFg))
	theme.PreviewBg = o(theme.Bg, o(baseTheme.PreviewBg, theme.PreviewBg))
	theme.DarkBg = o(baseTheme.DarkBg, theme.DarkBg)
	theme.Gutter = o(theme.DarkBg, o(baseTheme.Gutter, theme.Gutter))
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

	initPalette(theme)
}

func initPalette(theme *ColorTheme) {
	idx := 0
	pair := func(fg, bg Color) ColorPair {
		idx++
		return ColorPair{fg, bg, idx}
	}
	if theme != nil {
		ColPrompt = pair(theme.Prompt, theme.Bg)
		ColNormal = pair(theme.Fg, theme.Bg)
		ColMatch = pair(theme.Match, theme.Bg)
		ColCursor = pair(theme.Cursor, theme.Gutter)
		ColSelected = pair(theme.Selected, theme.Gutter)
		ColCurrent = pair(theme.Current, theme.DarkBg)
		ColCurrentMatch = pair(theme.CurrentMatch, theme.DarkBg)
		ColCurrentCursor = pair(theme.Cursor, theme.DarkBg)
		ColCurrentSelected = pair(theme.Selected, theme.DarkBg)
		ColSpinner = pair(theme.Spinner, theme.Bg)
		ColInfo = pair(theme.Info, theme.Bg)
		ColHeader = pair(theme.Header, theme.Bg)
		ColBorder = pair(theme.Border, theme.Bg)
		ColPreview = pair(theme.PreviewFg, theme.PreviewBg)
		ColPreviewBorder = pair(theme.Border, theme.PreviewBg)
	} else {
		ColPrompt = pair(colDefault, colDefault)
		ColNormal = pair(colDefault, colDefault)
		ColMatch = pair(colDefault, colDefault)
		ColCursor = pair(colDefault, colDefault)
		ColSelected = pair(colDefault, colDefault)
		ColCurrent = pair(colDefault, colDefault)
		ColCurrentMatch = pair(colDefault, colDefault)
		ColCurrentCursor = pair(colDefault, colDefault)
		ColCurrentSelected = pair(colDefault, colDefault)
		ColSpinner = pair(colDefault, colDefault)
		ColInfo = pair(colDefault, colDefault)
		ColHeader = pair(colDefault, colDefault)
		ColBorder = pair(colDefault, colDefault)
		ColPreview = pair(colDefault, colDefault)
		ColPreviewBorder = pair(colDefault, colDefault)
	}
}

func attrFor(color ColorPair, attr Attr) Attr {
	switch color {
	case ColCurrent:
		return attr | Reverse
	case ColMatch:
		return attr | Underline
	case ColCurrentMatch:
		return attr | Underline | Reverse
	}
	return attr
}
