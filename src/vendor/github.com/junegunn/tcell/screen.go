// Copyright 2016 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

// Screen represents the physical (or emulated) screen.
// This can be a terminal window or a physical console.  Platforms implement
// this differerently.
type Screen interface {
	// Init initializes the screen for use.
	Init() error

	// Fini finalizes the screen also releasing resources.
	Fini()

	// Clear erases the screen.  The contents of any screen buffers
	// will also be cleared.  This has the logical effect of
	// filling the screen with spaces, using the global default style.
	Clear()

	// Fill fills the screen with the given character and style.
	Fill(rune, Style)

	// SetCell is an older API, and will be removed.  Please use
	// SetContent instead; SetCell is implemented in terms of SetContent.
	SetCell(x int, y int, style Style, ch ...rune)

	// GetContent returns the contents at the given location.  If the
	// coordinates are out of range, then the values will be 0, nil,
	// StyleDefault.  Note that the contents returned are logical contents
	// and may not actually be what is displayed, but rather are what will
	// be displayed if Show() or Sync() is called.  The width is the width
	// in screen cells; most often this will be 1, but some East Asian
	// characters require two cells.
	GetContent(x, y int) (mainc rune, combc []rune, style Style, width int)

	// SetContent sets the contents of the given cell location.  If
	// the coordinates are out of range, then the operation is ignored.
	//
	// The first rune is the primary non-zero width rune.  The array
	// that follows is a possible list of combining characters to append,
	// and will usually be nil (no combining characters.)
	//
	// The results are not displayd until Show() or Sync() is called.
	//
	// Note that wide (East Asian full width) runes occupy two cells,
	// and attempts to place character at next cell to the right will have
	// undefined effects.  Wide runes that are printed in the
	// last column will be replaced with a single width space on output.
	SetContent(x int, y int, mainc rune, combc []rune, style Style)

	// SetStyle sets the default style to use when clearing the screen
	// or when StyleDefault is specified.  If it is also StyleDefault,
	// then whatever system/terminal default is relevant will be used.
	SetStyle(style Style)

	// ShowCursor is used to display the cursor at a given location.
	// If the coordinates -1, -1 are given or are otherwise outside the
	// dimensions of the screen, the cursor will be hidden.
	ShowCursor(x int, y int)

	// HideCursor is used to hide the cursor.  Its an alias for
	// ShowCursor(-1, -1).
	HideCursor()

	// Size returns the screen size as width, height.  This changes in
	// response to a call to Clear or Flush.
	Size() (int, int)

	// PollEvent waits for events to arrive.  Main application loops
	// must spin on this to prevent the application from stalling.
	// Furthermore, this will return nil if the Screen is finalized.
	PollEvent() Event

	// PostEvent tries to post an event into the event stream.  This
	// can fail if the event queue is full.  In that case, the event
	// is dropped, and ErrEventQFull is returned.
	PostEvent(ev Event) error

	// PostEventWait is like PostEvent, but if the queue is full, it
	// blocks until there is space in the queue, making delivery
	// reliable.  However, it is VERY important that this function
	// never be called from within whatever event loop is polling
	// with PollEvent(), otherwise a deadlock may arise.
	//
	// For this reason, when using this function, the use of a
	// Goroutine is recommended to ensure no deadlock can occur.
	PostEventWait(ev Event)

	// EnableMouse enables the mouse.  (If your terminal supports it.)
	EnableMouse()

	// DisableMouse disables the mouse.
	DisableMouse()

	// HasMouse returns true if the terminal (apparently) supports a
	// mouse.  Note that the a return value of true doesn't guarantee that
	// a mouse/pointing device is present; a false return definitely
	// indicates no mouse support is available.
	HasMouse() bool

	// Colors returns the number of colors.  All colors are assumed to
	// use the ANSI color map.  If a terminal is monochrome, it will
	// return 0.
	Colors() int

	// Show makes all the content changes made using SetContent() visible
	// on the display.
	//
	// It does so in the most efficient and least visually disruptive
	// manner possible.
	Show()

	// Sync works like Show(), but it updates every visible cell on the
	// physical display, assuming that it is not synchronized with any
	// internal model.  This may be both expensive and visually jarring,
	// so it should only be used when believed to actually be necessary.
	//
	// Typically this is called as a result of a user-requested redraw
	// (e.g. to clear up on screen corruption caused by some other program),
	// or during a resize event.
	Sync()

	// CharacterSet returns information about the character set.
	// This isn't the full locale, but it does give us the input/output
	// character set.  Note that this is just for diagnostic purposes,
	// we normally translate input/output to/from UTF-8, regardless of
	// what the user's environment is.
	CharacterSet() string

	// RegisterRuneFallback adds a fallback for runes that are not
	// part of the character set -- for example one coudld register
	// o as a fallback for ø.  This should be done cautiously for
	// characters that might be displayed ordinarily in language
	// specific text -- characters that could change the meaning of
	// of written text would be dangerous.  The intention here is to
	// facilitate fallback characters in pseudo-graphical applications.
	//
	// If the terminal has fallbacks already in place via an alternate
	// character set, those are used in preference.  Also, standard
	// fallbacks for graphical characters in the ACSC terminfo string
	// are registered implicitly.

	// The display string should be the same width as original rune.
	// This makes it possible to register two character replacements
	// for full width East Asian characters, for example.
	//
	// It is recommended that replacement strings consist only of
	// 7-bit ASCII, since other characters may not display everywhere.
	RegisterRuneFallback(r rune, subst string)

	// UnregisterRuneFallback unmaps a replacement.  It will unmap
	// the implicit ASCII replacements for alternate characters as well.
	// When an unmapped char needs to be displayed, but no suitable
	// glyph is available, '?' is emitted instead.  It is not possible
	// to "disable" the use of alternate characters that are supported
	// by your terminal except by changing the terminal database.
	UnregisterRuneFallback(r rune)

	// CanDisplay returns true if the given rune can be displayed on
	// this screen.  Note that this is a best guess effort -- whether
	// your fonts support the character or not may be questionable.
	// Mostly this is for folks who work outside of Unicode.
	//
	// If checkFallbacks is true, then if any (possibly imperfect)
	// fallbacks are registered, this will return true.  This will
	// also return true if the terminal can replace the glyph with
	// one that is visually indistinguishable from the one requested.
	CanDisplay(r rune, checkFallbacks bool) bool

	// Resize does nothing, since its generally not possible to
	// ask a screen to resize, but it allows the Screen to implement
	// the View interface.
	Resize(int, int, int, int)

	// HasKey returns true if the keyboard is believed to have the
	// key.  In some cases a keyboard may have keys with this name
	// but no support for them, while in others a key may be reported
	// as supported but not actually be usable (such as some emulators
	// that hijack certain keys).  Its best not to depend to strictly
	// on this function, but it can be used for hinting when building
	// menus, displayed hot-keys, etc.  Note that KeyRune (literal
	// runes) is always true.
	HasKey(Key) bool
}

// NewScreen returns a default Screen suitable for the user's terminal
// environment.
func NewScreen() (Screen, error) {
	// First we attempt to obtain a terminfo screen.  This should work
	// in most places if $TERM is set.
	if s, e := NewTerminfoScreen(); s != nil {
		return s, nil

	} else if s, _ := NewConsoleScreen(); s != nil {
		return s, nil

	} else {
		return nil, e
	}
}
