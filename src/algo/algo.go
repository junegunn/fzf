package algo

import (
	"strings"
	"unicode"

	"github.com/junegunn/fzf/src/util"
)

/*
 * String matching algorithms here do not use strings.ToLower to avoid
 * performance penalty. And they assume pattern runes are given in lowercase
 * letters when caseSensitive is false.
 *
 * In short: They try to do as little work as possible.
 */

func runeAt(runes []rune, index int, max int, forward bool) rune {
	if forward {
		return runes[index]
	}
	return runes[max-index-1]
}

// FuzzyMatch performs fuzzy-match
func FuzzyMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) (int, int) {
	if len(pattern) == 0 {
		return 0, 0
	}

	// 0. (FIXME) How to find the shortest match?
	//    a_____b__c__abc
	//    ^^^^^^^^^^  ^^^
	// 1. forward scan (abc)
	//   *-----*-----*>
	//   a_____b___abc__
	// 2. reverse scan (cba)
	//   a_____b___abc__
	//            <***
	pidx := 0
	sidx := -1
	eidx := -1

	lenRunes := len(runes)
	lenPattern := len(pattern)

	for index := range runes {
		char := runeAt(runes, index, lenRunes, forward)
		// This is considerably faster than blindly applying strings.ToLower to the
		// whole string
		if !caseSensitive {
			// Partially inlining `unicode.ToLower`. Ugly, but makes a noticeable
			// difference in CPU cost. (Measured on Go 1.4.1. Also note that the Go
			// compiler as of now does not inline non-leaf functions.)
			if char >= 'A' && char <= 'Z' {
				char += 32
			} else if char > unicode.MaxASCII {
				char = unicode.To(unicode.LowerCase, char)
			}
		}
		pchar := runeAt(pattern, pidx, lenPattern, forward)
		if char == pchar {
			if sidx < 0 {
				sidx = index
			}
			if pidx++; pidx == lenPattern {
				eidx = index + 1
				break
			}
		}
	}

	if sidx >= 0 && eidx >= 0 {
		pidx--
		for index := eidx - 1; index >= sidx; index-- {
			char := runeAt(runes, index, lenRunes, forward)
			if !caseSensitive {
				if char >= 'A' && char <= 'Z' {
					char += 32
				} else if char > unicode.MaxASCII {
					char = unicode.To(unicode.LowerCase, char)
				}
			}

			pchar := runeAt(pattern, pidx, lenPattern, forward)
			if char == pchar {
				if pidx--; pidx < 0 {
					sidx = index
					break
				}
			}
		}
		if forward {
			return sidx, eidx
		}
		return lenRunes - eidx, lenRunes - sidx
	}
	return -1, -1
}

// ExactMatchNaive is a basic string searching algorithm that handles case
// sensitivity. Although naive, it still performs better than the combination
// of strings.ToLower + strings.Index for typical fzf use cases where input
// strings and patterns are not very long.
//
// We might try to implement better algorithms in the future:
// http://en.wikipedia.org/wiki/String_searching_algorithm
func ExactMatchNaive(caseSensitive bool, forward bool, runes []rune, pattern []rune) (int, int) {
	if len(pattern) == 0 {
		return 0, 0
	}

	lenRunes := len(runes)
	lenPattern := len(pattern)

	if lenRunes < lenPattern {
		return -1, -1
	}

	pidx := 0
	for index := 0; index < lenRunes; index++ {
		char := runeAt(runes, index, lenRunes, forward)
		if !caseSensitive {
			if char >= 'A' && char <= 'Z' {
				char += 32
			} else if char > unicode.MaxASCII {
				char = unicode.To(unicode.LowerCase, char)
			}
		}
		pchar := runeAt(pattern, pidx, lenPattern, forward)
		if pchar == char {
			pidx++
			if pidx == lenPattern {
				if forward {
					return index - lenPattern + 1, index + 1
				}
				return lenRunes - (index + 1), lenRunes - (index - lenPattern + 1)
			}
		} else {
			index -= pidx
			pidx = 0
		}
	}
	return -1, -1
}

// PrefixMatch performs prefix-match
func PrefixMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) (int, int) {
	if len(runes) < len(pattern) {
		return -1, -1
	}

	for index, r := range pattern {
		char := runes[index]
		if !caseSensitive {
			char = unicode.ToLower(char)
		}
		if char != r {
			return -1, -1
		}
	}
	return 0, len(pattern)
}

// SuffixMatch performs suffix-match
func SuffixMatch(caseSensitive bool, forward bool, input []rune, pattern []rune) (int, int) {
	runes := util.TrimRight(input)
	trimmedLen := len(runes)
	diff := trimmedLen - len(pattern)
	if diff < 0 {
		return -1, -1
	}

	for index, r := range pattern {
		char := runes[index+diff]
		if !caseSensitive {
			char = unicode.ToLower(char)
		}
		if char != r {
			return -1, -1
		}
	}
	return trimmedLen - len(pattern), trimmedLen
}

// EqualMatch performs equal-match
func EqualMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) (int, int) {
	if len(runes) != len(pattern) {
		return -1, -1
	}
	runesStr := string(runes)
	if !caseSensitive {
		runesStr = strings.ToLower(runesStr)
	}
	if runesStr == string(pattern) {
		return 0, len(pattern)
	}
	return -1, -1
}
