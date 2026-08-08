//go:build !386 && !amd64 && !arm64

package algo

// The byte-view scanners in runeindex_x86.go reinterpret a []rune as
// little-endian 4-byte lanes, which is not valid everywhere. Elsewhere the
// reference scanners are the implementation.

func indexAsciiRune(runes []rune, caseSensitive bool, b byte, from int) int {
	return indexAsciiRuneRef(runes, caseSensitive, b, from)
}

func lastIndexAsciiRune(runes []rune, caseSensitive bool, b byte, from int) int {
	return lastIndexAsciiRuneRef(runes, caseSensitive, b, from)
}

func indexRune(runes []rune, r rune, from int) int {
	return indexRuneRef(runes, r, from)
}

func lastIndexRune(runes []rune, r rune, from int) int {
	return lastIndexRuneRef(runes, r, from)
}
