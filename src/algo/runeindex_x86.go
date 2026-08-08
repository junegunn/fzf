//go:build 386 || amd64 || arm64

package algo

import (
	"bytes"
	"unsafe"
)

// On these architectures a []rune is a little-endian array of 4-byte lanes, so
// an ASCII rune is the byte itself followed by three zero bytes at a 4-byte
// aligned offset. That lets the SIMD byte scanners run over the rune array
// directly: find the low byte, then confirm alignment and the three zeroes.
// A byte equal to the needle can also appear as the low byte of a multi-byte
// rune (0x0165 has low byte 'e'), which those two checks reject.

func runeBytes(runes []rune) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(runes))), len(runes)*4)
}

// indexAsciiRune returns the index of the first rune equal to b, or to its
// uppercase form when ignoring case, at or after rune index from.
func indexAsciiRune(runes []rune, caseSensitive bool, b byte, from int) int {
	view := runeBytes(runes)
	both := !caseSensitive && b >= 'a' && b <= 'z'
	for off := from * 4; off < len(view); {
		var idx int
		if both {
			idx = IndexByteTwo(view[off:], b, b-32)
		} else {
			idx = bytes.IndexByte(view[off:], b)
		}
		if idx < 0 {
			return -1
		}
		pos := off + idx
		if pos&3 == 0 && view[pos+1]|view[pos+2]|view[pos+3] == 0 {
			return pos >> 2
		}
		off = pos + 1
	}
	return -1
}

// runeNeedle picks which of the rune's four bytes to scan for, and returns its
// lane index and value. A zero byte is a useless needle because every ASCII
// rune contributes three of them, so U+AE00 scanned by its low byte would hit
// on almost every character of an ASCII-heavy line. Prefer a byte that cannot
// occur in an ASCII rune at all, then any non-zero byte.
func runeNeedle(r rune) (int, byte) {
	var b [4]byte
	b[0], b[1], b[2], b[3] = byte(r), byte(r>>8), byte(r>>16), byte(r>>24)
	for i, v := range b {
		if v >= 0x80 {
			return i, v
		}
	}
	for i, v := range b {
		if v != 0 {
			return i, v
		}
	}
	return 0, 0
}

func runeAt(view []byte, start int) rune {
	return rune(view[start]) | rune(view[start+1])<<8 | rune(view[start+2])<<16 | rune(view[start+3])<<24
}

// indexRune returns the index of the first rune equal to r at or after rune
// index from. Case is not folded, so the caller must have established that no
// other rune can transform into r.
func indexRune(runes []rune, r rune, from int) int {
	view := runeBytes(runes)
	lane, needle := runeNeedle(r)
	for off := from*4 + lane; off < len(view); {
		idx := bytes.IndexByte(view[off:], needle)
		if idx < 0 {
			return -1
		}
		pos := off + idx
		if start := pos - lane; start&3 == 0 && runeAt(view, start) == r {
			return start >> 2
		}
		off = pos + 1
	}
	return -1
}

// lastIndexRune is indexRune scanning backwards from the end.
func lastIndexRune(runes []rune, r rune, from int) int {
	view := runeBytes(runes)
	lane, needle := runeNeedle(r)
	for end := len(view); end > from*4+lane; {
		idx := bytes.LastIndexByte(view[from*4+lane:end], needle)
		if idx < 0 {
			return -1
		}
		pos := from*4 + lane + idx
		if start := pos - lane; start&3 == 0 && runeAt(view, start) == r {
			return start >> 2
		}
		end = pos
	}
	return -1
}

// lastIndexAsciiRune is indexAsciiRune scanning backwards from the end.
func lastIndexAsciiRune(runes []rune, caseSensitive bool, b byte, from int) int {
	view := runeBytes(runes)[from*4:]
	both := !caseSensitive && b >= 'a' && b <= 'z'
	for end := len(view); end > 0; {
		var idx int
		if both {
			idx = lastIndexByteTwo(view[:end], b, b-32)
		} else {
			idx = bytes.LastIndexByte(view[:end], b)
		}
		if idx < 0 {
			return -1
		}
		if idx&3 == 0 && view[idx+1]|view[idx+2]|view[idx+3] == 0 {
			return from + idx>>2
		}
		end = idx
	}
	return -1
}
