// Copyright 2015 The TCell Authors
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

// The names of these constants are chosen to match Terminfo names,
// modulo case, and changing the prefix from ACS_ to Rune.  These are
// the runes we provide extra special handling for, with ASCII fallbacks
// for terminals that lack them.
const (
	RuneSterling = '£'
	RuneDArrow   = '↓'
	RuneLArrow   = '←'
	RuneRArrow   = '→'
	RuneUArrow   = '↑'
	RuneBullet   = '·'
	RuneBoard    = '░'
	RuneCkBoard  = '▒'
	RuneDegree   = '°'
	RuneDiamond  = '◆'
	RuneGEqual   = '≥'
	RunePi       = 'π'
	RuneHLine    = '─'
	RuneLantern  = '§'
	RunePlus     = '┼'
	RuneLEqual   = '≤'
	RuneLLCorner = '└'
	RuneLRCorner = '┘'
	RuneNEqual   = '≠'
	RunePlMinus  = '±'
	RuneS1       = '⎺'
	RuneS3       = '⎻'
	RuneS7       = '⎼'
	RuneS9       = '⎽'
	RuneBlock    = '█'
	RuneTTee     = '┬'
	RuneRTee     = '┤'
	RuneLTee     = '├'
	RuneBTee     = '┴'
	RuneULCorner = '┌'
	RuneURCorner = '┐'
	RuneVLine    = '│'
)

// RuneFallbacks is the default map of fallback strings that will be
// used to replace a rune when no other more appropriate transformation
// is available, and the rune cannot be displayed directly.
//
// New entries may be added to this map over time, as it becomes clear
// that such is desirable.  Characters that represent either letters or
// numbers should not be added to this list unless it is certain that
// the meaning will still convey unambiguously.
//
// As an example, it would be appropriate to add an ASCII mapping for
// the full width form of the letter 'A', but it would not be appropriate
// to do so a glyph representing the country China.
//
// Programs that desire richer fallbacks may register additional ones,
// or change or even remove these mappings with Screen.RegisterRuneFallback
// Screen.UnregisterRuneFallback methods.
//
// Note that Unicode is presumed to be able to display all glyphs.
// This is a pretty poor assumption, but there is no easy way to
// figure out which glyphs are supported in a given font.  Hence,
// some care in selecting the characters you support in your application
// is still appropriate.
var RuneFallbacks = map[rune]string{
	RuneSterling: "f",
	RuneDArrow:   "v",
	RuneLArrow:   "<",
	RuneRArrow:   ">",
	RuneUArrow:   "^",
	RuneBullet:   "o",
	RuneBoard:    "#",
	RuneCkBoard:  ":",
	RuneDegree:   "\\",
	RuneDiamond:  "+",
	RuneGEqual:   ">",
	RunePi:       "*",
	RuneHLine:    "-",
	RuneLantern:  "#",
	RunePlus:     "+",
	RuneLEqual:   "<",
	RuneLLCorner: "+",
	RuneLRCorner: "+",
	RuneNEqual:   "!",
	RunePlMinus:  "#",
	RuneS1:       "~",
	RuneS3:       "-",
	RuneS7:       "-",
	RuneS9:       "_",
	RuneBlock:    "#",
	RuneTTee:     "+",
	RuneRTee:     "+",
	RuneLTee:     "+",
	RuneBTee:     "+",
	RuneULCorner: "+",
	RuneURCorner: "+",
	RuneVLine:    "|",
}
