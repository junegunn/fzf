package util

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

const (
	overflow64 uint64 = 0x8080808080808080
	overflow32 uint32 = 0x80808080
)

type Chars struct {
	slice           []byte // or []rune
	inBytes         bool
	trimLengthKnown bool
	trimLength      uint16

	// XXX Piggybacking item index here is a horrible idea. But I'm trying to
	// minimize the memory footprint by not wasting padded spaces.
	Index int32
}

func checkAscii(bytes []byte) (bool, int) {
	i := 0
	for ; i <= len(bytes)-8; i += 8 {
		if (overflow64 & *(*uint64)(unsafe.Pointer(&bytes[i]))) > 0 {
			return false, i
		}
	}
	for ; i <= len(bytes)-4; i += 4 {
		if (overflow32 & *(*uint32)(unsafe.Pointer(&bytes[i]))) > 0 {
			return false, i
		}
	}
	for ; i < len(bytes); i++ {
		if bytes[i] >= utf8.RuneSelf {
			return false, i
		}
	}
	return true, 0
}

// ToChars converts byte array into rune array
func ToChars(bytes []byte) Chars {
	inBytes, bytesUntil := checkAscii(bytes)
	if inBytes {
		return Chars{slice: bytes, inBytes: inBytes}
	}

	runes := make([]rune, bytesUntil, len(bytes))
	for i := 0; i < bytesUntil; i++ {
		runes[i] = rune(bytes[i])
	}
	for i := bytesUntil; i < len(bytes); {
		r, sz := utf8.DecodeRune(bytes[i:])
		i += sz
		runes = append(runes, r)
	}
	return RunesToChars(runes)
}

func RunesToChars(runes []rune) Chars {
	return Chars{slice: *(*[]byte)(unsafe.Pointer(&runes)), inBytes: false}
}

func (chars *Chars) IsBytes() bool {
	return chars.inBytes
}

func (chars *Chars) Bytes() []byte {
	return chars.slice
}

func (chars *Chars) NumLines(atMost int) (int, bool) {
	lines := 1
	if runes := chars.optionalRunes(); runes != nil {
		for _, r := range runes {
			if r == '\n' {
				lines++
			}
			if lines > atMost {
				return atMost, true
			}
		}
		return lines, false
	}

	for idx := 0; idx < len(chars.slice); idx++ {
		found := bytes.IndexByte(chars.slice[idx:], '\n')
		if found < 0 {
			break
		}

		idx += found
		lines++
		if lines > atMost {
			return atMost, true
		}
	}
	return lines, false
}

func (chars *Chars) optionalRunes() []rune {
	if chars.inBytes {
		return nil
	}
	return *(*[]rune)(unsafe.Pointer(&chars.slice))
}

func (chars *Chars) Get(i int) rune {
	if runes := chars.optionalRunes(); runes != nil {
		return runes[i]
	}
	return rune(chars.slice[i])
}

func (chars *Chars) Length() int {
	if runes := chars.optionalRunes(); runes != nil {
		return len(runes)
	}
	return len(chars.slice)
}

// String returns the string representation of a Chars object.
func (chars *Chars) String() string {
	return fmt.Sprintf("Chars{slice: []byte(%q), inBytes: %v, trimLengthKnown: %v, trimLength: %d, Index: %d}", chars.slice, chars.inBytes, chars.trimLengthKnown, chars.trimLength, chars.Index)
}

// TrimLength returns the length after trimming leading and trailing whitespaces
func (chars *Chars) TrimLength() uint16 {
	if chars.trimLengthKnown {
		return chars.trimLength
	}
	chars.trimLengthKnown = true
	var i int
	len := chars.Length()
	for i = len - 1; i >= 0; i-- {
		char := chars.Get(i)
		if !unicode.IsSpace(char) {
			break
		}
	}
	// Completely empty
	if i < 0 {
		return 0
	}

	var j int
	for j = 0; j < len; j++ {
		char := chars.Get(j)
		if !unicode.IsSpace(char) {
			break
		}
	}
	chars.trimLength = AsUint16(i - j + 1)
	return chars.trimLength
}

func (chars *Chars) LeadingWhitespaces() int {
	whitespaces := 0
	for i := 0; i < chars.Length(); i++ {
		char := chars.Get(i)
		if !unicode.IsSpace(char) {
			break
		}
		whitespaces++
	}
	return whitespaces
}

func (chars *Chars) TrailingWhitespaces() int {
	whitespaces := 0
	for i := chars.Length() - 1; i >= 0; i-- {
		char := chars.Get(i)
		if !unicode.IsSpace(char) {
			break
		}
		whitespaces++
	}
	return whitespaces
}

func (chars *Chars) TrimTrailingWhitespaces() {
	whitespaces := chars.TrailingWhitespaces()
	chars.slice = chars.slice[0 : len(chars.slice)-whitespaces]
}

func (chars *Chars) ToString() string {
	if runes := chars.optionalRunes(); runes != nil {
		return string(runes)
	}
	return unsafe.String(unsafe.SliceData(chars.slice), len(chars.slice))
}

func (chars *Chars) ToRunes() []rune {
	if runes := chars.optionalRunes(); runes != nil {
		return runes
	}
	bytes := chars.slice
	runes := make([]rune, len(bytes))
	for idx, b := range bytes {
		runes[idx] = rune(b)
	}
	return runes
}

func (chars *Chars) CopyRunes(dest []rune, from int) {
	if runes := chars.optionalRunes(); runes != nil {
		copy(dest, runes[from:])
		return
	}
	for idx, b := range chars.slice[from:][:len(dest)] {
		dest[idx] = rune(b)
	}
}

func (chars *Chars) Prepend(prefix string) {
	if runes := chars.optionalRunes(); runes != nil {
		runes = append([]rune(prefix), runes...)
		chars.slice = *(*[]byte)(unsafe.Pointer(&runes))
	} else {
		chars.slice = append([]byte(prefix), chars.slice...)
	}
}

func (chars *Chars) Lines(multiLine bool, maxLines int, wrapCols int, wrapSignWidth int, tabstop int) ([][]rune, bool) {
	text := make([]rune, chars.Length())
	copy(text, chars.ToRunes())

	lines := [][]rune{}
	overflow := false
	if !multiLine {
		lines = append(lines, text)
	} else {
		from := 0
		for off := 0; off < len(text); off++ {
			if text[off] == '\n' {
				lines = append(lines, text[from:off+1]) // Include '\n'
				from = off + 1
				if len(lines) >= maxLines {
					break
				}
			}
		}

		var lastLine []rune
		if from < len(text) {
			lastLine = text[from:]
		}

		overflow = false
		if len(lines) >= maxLines {
			overflow = true
		} else {
			lines = append(lines, lastLine)
		}
	}

	// If wrapping is disabled, we're done
	if wrapCols == 0 {
		return lines, overflow
	}

	wrapped := [][]rune{}
	for _, line := range lines {
		// Remove trailing '\n' and remember if it was there
		newline := len(line) > 0 && line[len(line)-1] == '\n'
		if newline {
			line = line[:len(line)-1]
		}

		for {
			cols := wrapCols
			if len(wrapped) > 0 {
				cols -= wrapSignWidth
			}
			_, overflowIdx := RunesWidth(line, 0, tabstop, cols)
			if overflowIdx >= 0 {
				// Might be a wide character
				if overflowIdx == 0 {
					overflowIdx = 1
				}
				if len(wrapped) >= maxLines {
					return wrapped, true
				}
				wrapped = append(wrapped, line[:overflowIdx])
				line = line[overflowIdx:]
				continue
			}

			// Restore trailing '\n'
			if newline {
				line = append(line, '\n')
			}

			if len(wrapped) >= maxLines {
				return wrapped, true
			}

			wrapped = append(wrapped, line)
			break
		}
	}

	return wrapped, false
}
