package fzf

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/tui"
)

type ansiOffset struct {
	offset [2]int32
	color  ansiState
}

type ansiState struct {
	fg   tui.Color
	bg   tui.Color
	attr tui.Attr
}

func (s *ansiState) colored() bool {
	return s.fg != -1 || s.bg != -1 || s.attr > 0
}

func (s *ansiState) equals(t *ansiState) bool {
	if t == nil {
		return !s.colored()
	}
	return s.fg == t.fg && s.bg == t.bg && s.attr == t.attr
}

func (s *ansiState) ToString() string {
	if !s.colored() {
		return ""
	}

	ret := ""
	if s.attr&tui.Bold > 0 {
		ret += "1;"
	}
	if s.attr&tui.Dim > 0 {
		ret += "2;"
	}
	if s.attr&tui.Italic > 0 {
		ret += "3;"
	}
	if s.attr&tui.Underline > 0 {
		ret += "4;"
	}
	if s.attr&tui.Blink > 0 {
		ret += "5;"
	}
	if s.attr&tui.Reverse > 0 {
		ret += "7;"
	}
	ret += toAnsiString(s.fg, 30) + toAnsiString(s.bg, 40)

	return "\x1b[" + strings.TrimSuffix(ret, ";") + "m"
}

func toAnsiString(color tui.Color, offset int) string {
	col := int(color)
	ret := ""
	if col == -1 {
		ret += strconv.Itoa(offset + 9)
	} else if col < 8 {
		ret += strconv.Itoa(offset + col)
	} else if col < 16 {
		ret += strconv.Itoa(offset - 30 + 90 + col - 8)
	} else if col < 256 {
		ret += strconv.Itoa(offset+8) + ";5;" + strconv.Itoa(col)
	} else if col >= (1 << 24) {
		r := strconv.Itoa((col >> 16) & 0xff)
		g := strconv.Itoa((col >> 8) & 0xff)
		b := strconv.Itoa(col & 0xff)
		ret += strconv.Itoa(offset+8) + ";2;" + r + ";" + g + ";" + b
	}
	return ret + ";"
}

var ansiRegex *regexp.Regexp

func init() {
	/*
		References:
		- https://github.com/gnachman/iTerm2
		- http://ascii-table.com/ansi-escape-sequences.php
		- http://ascii-table.com/ansi-escape-sequences-vt-100.php
		- http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x405.html
	*/
	// The following regular expression will include not all but most of the
	// frequently used ANSI sequences
	ansiRegex = regexp.MustCompile("(?:\x1b[\\[()][0-9;]*[a-zA-Z@]|\x1b.|[\x0e\x0f]|.\x08)")
}

func findAnsiStart(str string) int {
	idx := 0
	for ; idx < len(str); idx++ {
		b := str[idx]
		if b == 0x1b || b == 0x0e || b == 0x0f {
			return idx
		}
		if b == 0x08 && idx > 0 {
			return idx - 1
		}
	}
	return idx
}

func extractColor(str string, state *ansiState, proc func(string, *ansiState) bool) (string, *[]ansiOffset, *ansiState) {
	var offsets []ansiOffset
	var output bytes.Buffer

	if state != nil {
		offsets = append(offsets, ansiOffset{[2]int32{0, 0}, *state})
	}

	prevIdx := 0
	runeCount := 0
	for idx := 0; idx < len(str); {
		idx += findAnsiStart(str[idx:])
		if idx == len(str) {
			break
		}

		// Make sure that we found an ANSI code
		offset := ansiRegex.FindStringIndex(str[idx:])
		if len(offset) < 2 {
			idx++
			continue
		}
		offset[0] += idx
		offset[1] += idx
		idx = offset[1]

		// Check if we should continue
		prev := str[prevIdx:offset[0]]
		if proc != nil && !proc(prev, state) {
			return "", nil, nil
		}

		prevIdx = offset[1]
		runeCount += utf8.RuneCountInString(prev)
		output.WriteString(prev)

		newState := interpretCode(str[offset[0]:offset[1]], state)
		if !newState.equals(state) {
			if state != nil {
				// Update last offset
				(&offsets[len(offsets)-1]).offset[1] = int32(runeCount)
			}

			if newState.colored() {
				// Append new offset
				state = newState
				offsets = append(offsets, ansiOffset{[2]int32{int32(runeCount), int32(runeCount)}, *state})
			} else {
				// Discard state
				state = nil
			}
		}
	}

	var rest string
	var trimmed string

	if prevIdx == 0 {
		// No ANSI code found
		rest = str
		trimmed = str
	} else {
		rest = str[prevIdx:]
		output.WriteString(rest)
		trimmed = output.String()
	}
	if len(rest) > 0 && state != nil {
		// Update last offset
		runeCount += utf8.RuneCountInString(rest)
		(&offsets[len(offsets)-1]).offset[1] = int32(runeCount)
	}
	if proc != nil {
		proc(rest, state)
	}
	if len(offsets) == 0 {
		return trimmed, nil, state
	}
	return trimmed, &offsets, state
}

func interpretCode(ansiCode string, prevState *ansiState) *ansiState {
	// State
	var state *ansiState
	if prevState == nil {
		state = &ansiState{-1, -1, 0}
	} else {
		state = &ansiState{prevState.fg, prevState.bg, prevState.attr}
	}
	if ansiCode[0] != '\x1b' || ansiCode[1] != '[' || ansiCode[len(ansiCode)-1] != 'm' {
		return state
	}

	ptr := &state.fg
	state256 := 0

	init := func() {
		state.fg = -1
		state.bg = -1
		state.attr = 0
		state256 = 0
	}

	ansiCode = ansiCode[2 : len(ansiCode)-1]
	if len(ansiCode) == 0 {
		init()
	}
	for _, code := range strings.Split(ansiCode, ";") {
		if num, err := strconv.Atoi(code); err == nil {
			switch state256 {
			case 0:
				switch num {
				case 38:
					ptr = &state.fg
					state256++
				case 48:
					ptr = &state.bg
					state256++
				case 39:
					state.fg = -1
				case 49:
					state.bg = -1
				case 1:
					state.attr = state.attr | tui.Bold
				case 2:
					state.attr = state.attr | tui.Dim
				case 3:
					state.attr = state.attr | tui.Italic
				case 4:
					state.attr = state.attr | tui.Underline
				case 5:
					state.attr = state.attr | tui.Blink
				case 7:
					state.attr = state.attr | tui.Reverse
				case 0:
					init()
				default:
					if num >= 30 && num <= 37 {
						state.fg = tui.Color(num - 30)
					} else if num >= 40 && num <= 47 {
						state.bg = tui.Color(num - 40)
					} else if num >= 90 && num <= 97 {
						state.fg = tui.Color(num - 90 + 8)
					} else if num >= 100 && num <= 107 {
						state.bg = tui.Color(num - 100 + 8)
					}
				}
			case 1:
				switch num {
				case 2:
					state256 = 10 // MAGIC
				case 5:
					state256++
				default:
					state256 = 0
				}
			case 2:
				*ptr = tui.Color(num)
				state256 = 0
			case 10:
				*ptr = tui.Color(1<<24) | tui.Color(num<<16)
				state256++
			case 11:
				*ptr = *ptr | tui.Color(num<<8)
				state256++
			case 12:
				*ptr = *ptr | tui.Color(num)
				state256 = 0
			}
		}
	}
	if state256 > 0 {
		*ptr = -1
	}
	return state
}
