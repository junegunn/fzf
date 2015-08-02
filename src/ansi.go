package fzf

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ansiOffset struct {
	offset [2]int32
	color  ansiState
}

type ansiState struct {
	fg   int
	bg   int
	bold bool
}

func (s *ansiState) colored() bool {
	return s.fg != -1 || s.bg != -1 || s.bold
}

func (s *ansiState) equals(t *ansiState) bool {
	if t == nil {
		return !s.colored()
	}
	return s.fg == t.fg && s.bg == t.bg && s.bold == t.bold
}

var ansiRegex *regexp.Regexp

func init() {
	ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*[mK]")
}

func extractColor(str string, state *ansiState) (string, []ansiOffset, *ansiState) {
	var offsets []ansiOffset
	var output bytes.Buffer

	if state != nil {
		offsets = append(offsets, ansiOffset{[2]int32{0, 0}, *state})
	}

	idx := 0
	for _, offset := range ansiRegex.FindAllStringIndex(str, -1) {
		output.WriteString(str[idx:offset[0]])
		newState := interpretCode(str[offset[0]:offset[1]], state)

		if !newState.equals(state) {
			if state != nil {
				// Update last offset
				(&offsets[len(offsets)-1]).offset[1] = int32(utf8.RuneCount(output.Bytes()))
			}

			if newState.colored() {
				// Append new offset
				state = newState
				newLen := int32(utf8.RuneCount(output.Bytes()))
				offsets = append(offsets, ansiOffset{[2]int32{newLen, newLen}, *state})
			} else {
				// Discard state
				state = nil
			}
		}

		idx = offset[1]
	}

	rest := str[idx:]
	if len(rest) > 0 {
		output.WriteString(rest)
		if state != nil {
			// Update last offset
			(&offsets[len(offsets)-1]).offset[1] = int32(utf8.RuneCount(output.Bytes()))
		}
	}
	return output.String(), offsets, state
}

func interpretCode(ansiCode string, prevState *ansiState) *ansiState {
	// State
	var state *ansiState
	if prevState == nil {
		state = &ansiState{-1, -1, false}
	} else {
		state = &ansiState{prevState.fg, prevState.bg, prevState.bold}
	}
	if ansiCode[len(ansiCode)-1] == 'K' {
		return state
	}

	ptr := &state.fg
	state256 := 0

	init := func() {
		state.fg = -1
		state.bg = -1
		state.bold = false
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
					state.bold = true
				case 0:
					init()
				default:
					if num >= 30 && num <= 37 {
						state.fg = num - 30
					} else if num >= 40 && num <= 47 {
						state.bg = num - 40
					}
				}
			case 1:
				switch num {
				case 5:
					state256++
				default:
					state256 = 0
				}
			case 2:
				*ptr = num
				state256 = 0
			}
		}
	}
	return state
}
