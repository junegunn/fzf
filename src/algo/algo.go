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

// Result conatins the results of running a match function.
type Result struct {
	Start int32
	End   int32

	// Items are basically sorted by the lengths of matched substrings.
	// But we slightly adjust the score with bonus for better results.
	Bonus int32
}

type charClass int

const (
	charNonWord charClass = iota
	charLower
	charUpper
	charLetter
	charNumber
)

func evaluateBonus(caseSensitive bool, runes []rune, pattern []rune, sidx int, eidx int) int32 {
	var bonus int32
	pidx := 0
	lenPattern := len(pattern)
	consecutive := false
	prevClass := charNonWord
	for index := 0; index < eidx; index++ {
		char := runes[index]
		var class charClass
		if unicode.IsLower(char) {
			class = charLower
		} else if unicode.IsUpper(char) {
			class = charUpper
		} else if unicode.IsLetter(char) {
			class = charLetter
		} else if unicode.IsNumber(char) {
			class = charNumber
		} else {
			class = charNonWord
		}

		var point int32
		if prevClass == charNonWord && class != charNonWord {
			// Word boundary
			point = 2
		} else if prevClass == charLower && class == charUpper ||
			prevClass != charNumber && class == charNumber {
			// camelCase letter123
			point = 1
		}
		prevClass = class

		if index >= sidx {
			if !caseSensitive {
				if char >= 'A' && char <= 'Z' {
					char += 32
				} else if char > unicode.MaxASCII {
					char = unicode.To(unicode.LowerCase, char)
				}
			}
			pchar := pattern[pidx]
			if pchar == char {
				// Boost bonus for the first character in the pattern
				if pidx == 0 {
					point *= 2
				}
				// Bonus to consecutive matching chars
				if consecutive {
					point++
				}
				bonus += point

				if pidx++; pidx == lenPattern {
					break
				}
				consecutive = true
			} else {
				consecutive = false
			}
		}
	}
	return bonus
}

// FuzzyMatch performs fuzzy-match
func FuzzyMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) Result {
	if len(pattern) == 0 {
		return Result{0, 0, 0}
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

		// Calculate the bonus. This can't be done at the same time as the
		// pattern scan above because 'forward' may be false.
		if !forward {
			sidx, eidx = lenRunes-eidx, lenRunes-sidx
		}

		return Result{int32(sidx), int32(eidx),
			evaluateBonus(caseSensitive, runes, pattern, sidx, eidx)}
	}
	return Result{-1, -1, 0}
}

// ExactMatchNaive is a basic string searching algorithm that handles case
// sensitivity. Although naive, it still performs better than the combination
// of strings.ToLower + strings.Index for typical fzf use cases where input
// strings and patterns are not very long.
//
// We might try to implement better algorithms in the future:
// http://en.wikipedia.org/wiki/String_searching_algorithm
func ExactMatchNaive(caseSensitive bool, forward bool, runes []rune, pattern []rune) Result {
	if len(pattern) == 0 {
		return Result{0, 0, 0}
	}

	lenRunes := len(runes)
	lenPattern := len(pattern)

	if lenRunes < lenPattern {
		return Result{-1, -1, 0}
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
				var sidx, eidx int
				if forward {
					sidx = index - lenPattern + 1
					eidx = index + 1
				} else {
					sidx = lenRunes - (index + 1)
					eidx = lenRunes - (index - lenPattern + 1)
				}
				return Result{int32(sidx), int32(eidx),
					evaluateBonus(caseSensitive, runes, pattern, sidx, eidx)}
			}
		} else {
			index -= pidx
			pidx = 0
		}
	}
	return Result{-1, -1, 0}
}

// PrefixMatch performs prefix-match
func PrefixMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) Result {
	if len(runes) < len(pattern) {
		return Result{-1, -1, 0}
	}

	for index, r := range pattern {
		char := runes[index]
		if !caseSensitive {
			char = unicode.ToLower(char)
		}
		if char != r {
			return Result{-1, -1, 0}
		}
	}
	lenPattern := len(pattern)
	return Result{0, int32(lenPattern),
		evaluateBonus(caseSensitive, runes, pattern, 0, lenPattern)}
}

// SuffixMatch performs suffix-match
func SuffixMatch(caseSensitive bool, forward bool, input []rune, pattern []rune) Result {
	runes := util.TrimRight(input)
	trimmedLen := len(runes)
	diff := trimmedLen - len(pattern)
	if diff < 0 {
		return Result{-1, -1, 0}
	}

	for index, r := range pattern {
		char := runes[index+diff]
		if !caseSensitive {
			char = unicode.ToLower(char)
		}
		if char != r {
			return Result{-1, -1, 0}
		}
	}
	lenPattern := len(pattern)
	sidx := trimmedLen - lenPattern
	eidx := trimmedLen
	return Result{int32(sidx), int32(eidx),
		evaluateBonus(caseSensitive, runes, pattern, sidx, eidx)}
}

// EqualMatch performs equal-match
func EqualMatch(caseSensitive bool, forward bool, runes []rune, pattern []rune) Result {
	// Note: EqualMatch always return a zero bonus.
	if len(runes) != len(pattern) {
		return Result{-1, -1, 0}
	}
	runesStr := string(runes)
	if !caseSensitive {
		runesStr = strings.ToLower(runesStr)
	}
	if runesStr == string(pattern) {
		return Result{0, int32(len(pattern)), 0}
	}
	return Result{-1, -1, 0}
}
