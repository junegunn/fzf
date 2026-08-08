package algo

// Reference scanners over a []rune, with no representation tricks.
//
// They have two roles. Where reinterpreting a []rune as little-endian bytes is
// not valid, they are the shipped implementation, via runeindex_others.go.
// Everywhere else, the tests feed the same inputs to these and to the byte-view
// scanners in runeindex_x86.go and require identical answers.
//
// They carry no build tag so that both roles hold on every platform. Otherwise
// the portable build would be code that nothing here ever runs.

func indexAsciiRuneRef(runes []rune, caseSensitive bool, b byte, from int) int {
	lower, upper := rune(b), rune(-1)
	if !caseSensitive && b >= 'a' && b <= 'z' {
		upper = rune(b - 32)
	}
	for i := from; i < len(runes); i++ {
		if runes[i] == lower || runes[i] == upper {
			return i
		}
	}
	return -1
}

func lastIndexAsciiRuneRef(runes []rune, caseSensitive bool, b byte, from int) int {
	lower, upper := rune(b), rune(-1)
	if !caseSensitive && b >= 'a' && b <= 'z' {
		upper = rune(b - 32)
	}
	for i := len(runes) - 1; i >= from; i-- {
		if runes[i] == lower || runes[i] == upper {
			return i
		}
	}
	return -1
}

func indexRuneRef(runes []rune, r rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == r {
			return i
		}
	}
	return -1
}

func lastIndexRuneRef(runes []rune, r rune, from int) int {
	for i := len(runes) - 1; i >= from; i-- {
		if runes[i] == r {
			return i
		}
	}
	return -1
}
