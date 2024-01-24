//go:build !tcell && !windows

package tui

type Attr int32

func HasFullscreenRenderer() bool {
	return false
}

var DefaultBorderShape BorderShape = BorderRounded

func (a Attr) Merge(b Attr) Attr {
	return a | b
}

const (
	AttrUndefined = Attr(0)
	AttrRegular   = Attr(1 << 8)
	AttrClear     = Attr(1 << 9)

	Bold          = Attr(1)
	Dim           = Attr(1 << 1)
	Italic        = Attr(1 << 2)
	Underline     = Attr(1 << 3)
	Blink         = Attr(1 << 4)
	Blink2        = Attr(1 << 5)
	Reverse       = Attr(1 << 6)
	StrikeThrough = Attr(1 << 7)
)

func (r *FullscreenRenderer) Init()                              {}
func (r *FullscreenRenderer) Resize(maxHeightFunc func(int) int) {}
func (r *FullscreenRenderer) Pause(bool)                         {}
func (r *FullscreenRenderer) Resume(bool, bool)                  {}
func (r *FullscreenRenderer) PassThrough(string)                 {}
func (r *FullscreenRenderer) Clear()                             {}
func (r *FullscreenRenderer) NeedScrollbarRedraw() bool          { return false }
func (r *FullscreenRenderer) ShouldEmitResizeEvent() bool        { return false }
func (r *FullscreenRenderer) Refresh()                           {}
func (r *FullscreenRenderer) Close()                             {}
func (r *FullscreenRenderer) Size() TermSize                     { return TermSize{} }

func (r *FullscreenRenderer) GetChar() Event { return Event{} }
func (r *FullscreenRenderer) Top() int       { return 0 }
func (r *FullscreenRenderer) MaxX() int      { return 0 }
func (r *FullscreenRenderer) MaxY() int      { return 0 }

func (r *FullscreenRenderer) RefreshWindows(windows []Window) {}

func (r *FullscreenRenderer) NewWindow(top int, left int, width int, height int, preview bool, borderStyle BorderStyle) Window {
	return nil
}
