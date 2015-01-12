package algo

import "strings"

/*
 * String matching algorithms here do not use strings.ToLower to avoid
 * performance penalty. And they assume pattern runes are given in lowercase
 * letters when caseSensitive is false.
 *
 * In short: They try to do as little work as possible.
 */

// FuzzyMatch performs fuzzy-match
func FuzzyMatch(caseSensitive bool, input *string, pattern []rune) (int, int) {
	runes := []rune(*input)

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

	for index, char := range runes {
		// This is considerably faster than blindly applying strings.ToLower to the
		// whole string
		if !caseSensitive && char >= 65 && char <= 90 {
			char += 32
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
			char := runes[index]
			if !caseSensitive && char >= 65 && char <= 90 {
				char += 32
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

// ExactMatchStrings performs exact-match using strings package.
// Currently not used.
func ExactMatchStrings(caseSensitive bool, input *string, pattern []rune) (int, int) {
	var str string
	if caseSensitive {
		str = *input
	} else {
		str = strings.ToLower(*input)
	}

	if idx := strings.Index(str, string(pattern)); idx >= 0 {
		prefixRuneLen := len([]rune((*input)[:idx]))
		return prefixRuneLen, prefixRuneLen + len(pattern)
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
func ExactMatchNaive(caseSensitive bool, input *string, pattern []rune) (int, int) {
	runes := []rune(*input)
	numRunes := len(runes)
	plen := len(pattern)
	if numRunes < plen {
		return -1, -1
	}

	pidx := 0
	for index := 0; index < numRunes; index++ {
		char := runes[index]
		if !caseSensitive && char >= 65 && char <= 90 {
			char += 32
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
func PrefixMatch(caseSensitive bool, input *string, pattern []rune) (int, int) {
	runes := []rune(*input)
	if len(runes) < len(pattern) {
		return -1, -1
	}

	for index, r := range pattern {
		char := runes[index]
		if !caseSensitive && char >= 65 && char <= 90 {
			char += 32
		}
		if char != r {
			return -1, -1
		}
	}
	return 0, len(pattern)
}

// SuffixMatch performs suffix-match
func SuffixMatch(caseSensitive bool, input *string, pattern []rune) (int, int) {
	runes := []rune(strings.TrimRight(*input, " "))
	trimmedLen := len(runes)
	diff := trimmedLen - len(pattern)
	if diff < 0 {
		return -1, -1
	}

	for index, r := range pattern {
		char := runes[index+diff]
		if !caseSensitive && char >= 65 && char <= 90 {
			char += 32
		}
		if char != r {
			return -1, -1
		}
	}
	return trimmedLen - len(pattern), trimmedLen
}
