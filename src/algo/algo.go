package algo

import (
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

// FuzzyMatch performs fuzzy-match
func FuzzyMatch(caseSensitive bool, runes *[]rune, pattern []rune) (int, int) {
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

	for index, char := range *runes {
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
		if char == pattern[pidx] {
			if sidx < 0 {
				sidx = index
			}
			if pidx++; pidx == len(pattern) {
				eidx = index + 1
				break
			}
		}
	}

	if sidx >= 0 && eidx >= 0 {
		pidx--
		for index := eidx - 1; index >= sidx; index-- {
			char := (*runes)[index]
			if !caseSensitive {
				if char >= 'A' && char <= 'Z' {
					char += 32
				} else if char > unicode.MaxASCII {
					char = unicode.To(unicode.LowerCase, char)
				}
			}
			if char == pattern[pidx] {
				if pidx--; pidx < 0 {
					sidx = index
					break
				}
			}
		}
		return sidx, eidx
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
func ExactMatchNaive(caseSensitive bool, runes *[]rune, pattern []rune) (int, int) {
	if len(pattern) == 0 {
		return 0, 0
	}

	numRunes := len(*runes)
	plen := len(pattern)
	if numRunes < plen {
		return -1, -1
	}

	pidx := 0
	for index := 0; index < numRunes; index++ {
		char := (*runes)[index]
		if !caseSensitive {
			if char >= 'A' && char <= 'Z' {
				char += 32
			} else if char > unicode.MaxASCII {
				char = unicode.To(unicode.LowerCase, char)
			}
		}
		if pattern[pidx] == char {
			pidx++
			if pidx == plen {
				return index - plen + 1, index + 1
			}
		} else {
			index -= pidx
			pidx = 0
		}
	}
	return -1, -1
}

// PrefixMatch performs prefix-match
func PrefixMatch(caseSensitive bool, runes *[]rune, pattern []rune) (int, int) {
	if len(*runes) < len(pattern) {
		return -1, -1
	}

	for index, r := range pattern {
		char := (*runes)[index]
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
func SuffixMatch(caseSensitive bool, input *[]rune, pattern []rune) (int, int) {
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
