package util

import (
	"bytes"
	"fmt"
	"math/bits"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

const (
	overflow64 uint64 = 0x8080808080808080
	overflow32 uint32 = 0x80808080
)

const (
	flagInBytes uint8 = 1 << iota
	flagMayFold
)

type Chars struct {
	slice []byte // or []rune
	// Written while the item is built and never after it reaches a matcher,
	// so nothing reads these bits concurrently with a write. Prepend touches
	// them, but only on the transient tokens inside transformItem, before
	// item.text exists. trimLength* is kept out because TrimLength writes it
	// lazily, long after that point.
	flags           uint8
	trimLengthKnown bool
	trimLength      uint16

	// XXX Piggybacking item index here is a horrible idea. But I'm trying to
	// minimize the memory footprint by not wasting padded spaces.
	Index int32
}

// Rune ranges that case folding or normalization can turn into ASCII, derived
// from algo's normalization table and unicode.ToLower, then merged. They are a
// superset of the exact set, which TestMayFoldToAsciiIsSuperset in the algo
// package pins. Grouped tightly on purpose: a wider merge would swallow Greek
// Extended, General Punctuation and the currency and letterlike blocks, and
// every line holding a curly quote or an em dash would then lose the
// prefilter. Cyrillic, Greek, Hebrew, Arabic, Thai, Devanagari, CJK, Hangul,
// kana, emoji, punctuation and box drawing are all outside.
const (
	foldLo = 0x00C0
	foldHi = 0xFF61
)

var foldableRanges = [...][2]rune{
	{0x00C0, 0x01B6}, // Latin-1 Supplement, Latin Extended-A and -B
	{0x01CD, 0x02AE}, // rest of Latin Extended-B and IPA Extensions
	{0x0363, 0x036F}, // combining Latin small letters
	{0x1D00, 0x1D22}, // Phonetic Extensions, small capitals
	{0x1D62, 0x1D65}, // subscript letters
	{0x1E00, 0x1EF9}, // Latin Extended Additional
	{0x2071, 0x2071}, // superscript i
	{0x2095, 0x209C}, // subscript letters
	{0x212A, 0x212B}, // KELVIN SIGN and ANGSTROM SIGN, which fold by case
	{0x2183, 0x2184}, // reversed roman numeral one hundred
	{0x2C62, 0x2C7F}, // Latin Extended-C
	{0xA78D, 0xA78D}, // Latin Extended-D
	{0xA7AA, 0xA7B2}, // more Latin Extended-D
	{0xA7C5, 0xA7C5},
	{0xFF01, 0xFF61}, // fullwidth ASCII forms, and halfwidth ideographic full stop
}

// Walking the ranges costs a serial chain of comparisons per rune, which is
// measurable at ingestion, so precompute a bitmap instead.
var foldableBits = func() (bits [(foldHi-foldLo)/8 + 1]byte) {
	for _, r := range foldableRanges {
		for c := r[0]; c <= r[1]; c++ {
			i := c - foldLo
			bits[i>>3] |= 1 << (i & 7)
		}
	}
	return
}()

// MayFoldToAscii reports whether case folding or normalization could turn r
// into an ASCII character.
func MayFoldToAscii(r rune) bool {
	i := uint32(r - foldLo)
	if i > foldHi-foldLo {
		return false
	}
	return foldableBits[i>>3]&(1<<(i&7)) != 0
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

// countRunes counts the bytes that are not UTF-8 continuation bytes, which is
// the rune count of valid UTF-8. Each invalid byte decodes to its own
// RuneError, so the result can undercount but never overcount, making it safe
// as a capacity hint.
func countRunes(bytes []byte) int {
	n, i := 0, 0
	for ; i <= len(bytes)-8; i += 8 {
		v := *(*uint64)(unsafe.Pointer(&bytes[i]))
		// Continuation byte: bit 7 set, bit 6 clear. In `v << 1` bit 7 of each
		// lane holds bit 6 of that same lane.
		n += 8 - bits.OnesCount64(v&^(v<<1)&overflow64)
	}
	for ; i < len(bytes); i++ {
		if bytes[i]&0xC0 != 0x80 {
			n++
		}
	}
	return n
}

// ToChars converts byte array into rune array
func ToChars(bytes []byte) Chars {
	inBytes, bytesUntil := checkAscii(bytes)
	if inBytes {
		return Chars{slice: bytes, flags: flagInBytes}
	}

	runes := make([]rune, bytesUntil, bytesUntil+countRunes(bytes[bytesUntil:]))
	for i := range bytesUntil {
		runes[i] = rune(bytes[i])
	}
	mayFold := false
	for i := bytesUntil; i < len(bytes); {
		// utf8.DecodeRune has an ASCII path of its own, but it is too complex
		// to inline, so a mostly-ASCII line pays one call per byte for it.
		// An ASCII rune never sets the fold bit either, so skip both calls.
		if b := bytes[i]; b < utf8.RuneSelf {
			runes = append(runes, rune(b))
			i++
			continue
		}
		r, sz := utf8.DecodeRune(bytes[i:])
		i += sz
		mayFold = mayFold || MayFoldToAscii(r)
		runes = append(runes, r)
	}
	return runesToChars(runes, mayFold)
}

func RunesToChars(runes []rune) Chars {
	mayFold := false
	for _, r := range runes {
		if MayFoldToAscii(r) {
			mayFold = true
			break
		}
	}
	return runesToChars(runes, mayFold)
}

func runesToChars(runes []rune, mayFold bool) Chars {
	var flags uint8
	if mayFold {
		flags = flagMayFold
	}
	return Chars{slice: *(*[]byte)(unsafe.Pointer(&runes)), flags: flags}
}

func (chars *Chars) IsBytes() bool {
	return chars.flags&flagInBytes != 0
}

// MayFoldToAscii reports whether the text holds a rune that case folding or
// normalization could turn into an ASCII character. When false, an ASCII
// pattern character can only match the identical ASCII rune, which is what
// lets the prefilter scan the rune array directly.
func (chars *Chars) MayFoldToAscii() bool {
	return chars.flags&flagMayFold != 0
}

// Runes returns the underlying rune slice, or nil if the text is kept as bytes.
func (chars *Chars) Runes() []rune {
	return chars.optionalRunes()
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
	if chars.IsBytes() {
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
	return fmt.Sprintf("Chars{slice: []byte(%q), inBytes: %v, mayFold: %v, trimLengthKnown: %v, trimLength: %d, Index: %d}", chars.slice, chars.IsBytes(), chars.MayFoldToAscii(), chars.trimLengthKnown, chars.trimLength, chars.Index)
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

func (chars *Chars) TrimTrailingWhitespaces(maxIndex int) {
	whitespaces := chars.TrailingWhitespaces()
	end := len(chars.slice) - whitespaces
	chars.slice = chars.slice[0:max(end, maxIndex)]
}

func (chars *Chars) TrimSuffix(runes []rune) {
	lastIdx := len(chars.slice)
	firstIdx := lastIdx - len(runes)
	if firstIdx < 0 {
		return
	}

	for i := firstIdx; i < lastIdx; i++ {
		char := chars.Get(i)
		if char != runes[i-firstIdx] {
			return
		}
	}

	chars.slice = chars.slice[0:firstIdx]
}

func (chars *Chars) SliceRight(last int) {
	chars.slice = chars.slice[:last]
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
	for _, r := range prefix {
		if MayFoldToAscii(r) {
			chars.flags |= flagMayFold
			break
		}
	}
}

func (chars *Chars) Lines(multiLine bool, maxLines int, wrapCols int, wrapSignWidth int, tabstop int, wrapWord bool) ([][]rune, bool) {
	text := make([]rune, chars.Length())
	copy(text, chars.ToRunes())

	lines := [][]rune{}
	overflow := false
	if !multiLine {
		lines = append(lines, text)
	} else {
		from := 0
		for off := range text {
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

		hasWrapSign := false
		for {
			cols := wrapCols
			if hasWrapSign {
				cols -= wrapSignWidth
			}
			_, overflowIdx := RunesWidth(line, 0, tabstop, cols)
			if overflowIdx >= 0 {
				// Might be a wide character
				if overflowIdx == 0 {
					overflowIdx = 1
				}
				if wrapWord {
					// Find last space/tab at or before overflowIdx
					breakIdx := -1
					for k := overflowIdx; k > 0; k-- {
						if line[k-1] == ' ' || line[k-1] == '\t' {
							breakIdx = k
							break
						}
					}
					if breakIdx > 0 {
						overflowIdx = breakIdx
					}
				}
				if len(wrapped) >= maxLines {
					return wrapped, true
				}
				wrapped = append(wrapped, line[:overflowIdx])
				hasWrapSign = true
				line = line[overflowIdx:]
				continue
			}
			hasWrapSign = false

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

	return wrapped, overflow
}
