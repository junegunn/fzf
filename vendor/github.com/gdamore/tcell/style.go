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

// Style represents a complete text style, including both foreground
// and background color.  We encode it in a 64-bit int for efficiency.
// The coding is (MSB): <7b flags><1b><24b fgcolor><7b attr><1b><24b bgcolor>.
// The <1b> is set true to indicate that the color is an RGB color, rather
// than a named index.
//
// This gives 24bit color options, if it ever becomes truly necessary.
// However, applications must not rely on this encoding.
//
// Note that not all terminals can display all colors or attributes, and
// many might have specific incompatibilities between specific attributes
// and color combinations.
//
// The intention is to extend styles to support paletting, in which case
// some flag bit(s) would be set, and the foreground and background colors
// would be replaced with a palette number and palette index.
//
// To use Style, just declare a variable of its type.
type Style int64

// StyleDefault represents a default style, based upon the context.
// It is the zero value.
const StyleDefault Style = 0

// styleFlags -- used internally for now.
const (
	styleBgSet = 1 << (iota + 57)
	styleFgSet
	stylePalette
)

// Foreground returns a new style based on s, with the foreground color set
// as requested.  ColorDefault can be used to select the global default.
func (s Style) Foreground(c Color) Style {
	if c == ColorDefault {
		return (s &^ (0x1ffffff00000000 | styleFgSet))
	}
	return (s &^ Style(0x1ffffff00000000)) |
		((Style(c) & 0x1ffffff) << 32) | styleFgSet
}

// Background returns a new style based on s, with the background color set
// as requested.  ColorDefault can be used to select the global default.
func (s Style) Background(c Color) Style {
	if c == ColorDefault {
		return (s &^ (0x1ffffff | styleBgSet))
	}
	return (s &^ (0x1ffffff)) | (Style(c) & 0x1ffffff) | styleBgSet
}

// Decompose breaks a style up, returning the foreground, background,
// and other attributes.
func (s Style) Decompose() (fg Color, bg Color, attr AttrMask) {
	if s&styleFgSet != 0 {
		fg = Color(s>>32) & 0x1ffffff
	} else {
		fg = ColorDefault
	}
	if s&styleBgSet != 0 {
		bg = Color(s & 0x1ffffff)
	} else {
		bg = ColorDefault
	}
	attr = AttrMask(s) & attrAll

	return fg, bg, attr
}

func (s Style) setAttrs(attrs Style, on bool) Style {
	if on {
		return s | attrs
	}
	return s &^ attrs
}

// Normal returns the style with all attributes disabled.
func (s Style) Normal() Style {
	return s &^ Style(attrAll)
}

// Bold returns a new style based on s, with the bold attribute set
// as requested.
func (s Style) Bold(on bool) Style {
	return s.setAttrs(Style(AttrBold), on)
}

// Blink returns a new style based on s, with the blink attribute set
// as requested.
func (s Style) Blink(on bool) Style {
	return s.setAttrs(Style(AttrBlink), on)
}

// Dim returns a new style based on s, with the dim attribute set
// as requested.
func (s Style) Dim(on bool) Style {
	return s.setAttrs(Style(AttrDim), on)
}

// Reverse returns a new style based on s, with the reverse attribute set
// as requested.  (Reverse usually changes the foreground and background
// colors.)
func (s Style) Reverse(on bool) Style {
	return s.setAttrs(Style(AttrReverse), on)
}

// Underline returns a new style based on s, with the underline attribute set
// as requested.
func (s Style) Underline(on bool) Style {
	return s.setAttrs(Style(AttrUnderline), on)
}
