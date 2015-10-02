package fzf

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/junegunn/fzf/src/util"
)

const rangeEllipsis = 0

// Range represents nth-expression
type Range struct {
	begin int
	end   int
}

// Token contains the tokenized part of the strings and its prefix length
type Token struct {
	text         []rune
	prefixLength int
	trimLength   int
}

// Delimiter for tokenizing the input
type Delimiter struct {
	regex *regexp.Regexp
	str   *string
}

func newRange(begin int, end int) Range {
	if begin == 1 {
		begin = rangeEllipsis
	}
	if end == -1 {
		end = rangeEllipsis
	}
	return Range{begin, end}
}

// ParseRange parses nth-expression and returns the corresponding Range object
func ParseRange(str *string) (Range, bool) {
	if (*str) == ".." {
		return newRange(rangeEllipsis, rangeEllipsis), true
	} else if strings.HasPrefix(*str, "..") {
		end, err := strconv.Atoi((*str)[2:])
		if err != nil || end == 0 {
			return Range{}, false
		}
		return newRange(rangeEllipsis, end), true
	} else if strings.HasSuffix(*str, "..") {
		begin, err := strconv.Atoi((*str)[:len(*str)-2])
		if err != nil || begin == 0 {
			return Range{}, false
		}
		return newRange(begin, rangeEllipsis), true
	} else if strings.Contains(*str, "..") {
		ns := strings.Split(*str, "..")
		if len(ns) != 2 {
			return Range{}, false
		}
		begin, err1 := strconv.Atoi(ns[0])
		end, err2 := strconv.Atoi(ns[1])
		if err1 != nil || err2 != nil || begin == 0 || end == 0 {
			return Range{}, false
		}
		return newRange(begin, end), true
	}

	n, err := strconv.Atoi(*str)
	if err != nil || n == 0 {
		return Range{}, false
	}
	return newRange(n, n), true
}

func withPrefixLengths(tokens [][]rune, begin int) []Token {
	ret := make([]Token, len(tokens))

	prefixLength := begin
	for idx, token := range tokens {
		// Need to define a new local variable instead of the reused token to take
		// the pointer to it
		ret[idx] = Token{token, prefixLength, util.TrimLen(token)}
		prefixLength += len(token)
	}
	return ret
}

const (
	awkNil = iota
	awkBlack
	awkWhite
)

func awkTokenizer(input []rune) ([][]rune, int) {
	// 9, 32
	ret := [][]rune{}
	str := []rune{}
	prefixLength := 0
	state := awkNil
	for _, r := range input {
		white := r == 9 || r == 32
		switch state {
		case awkNil:
			if white {
				prefixLength++
			} else {
				state = awkBlack
				str = append(str, r)
			}
		case awkBlack:
			str = append(str, r)
			if white {
				state = awkWhite
			}
		case awkWhite:
			if white {
				str = append(str, r)
			} else {
				ret = append(ret, str)
				state = awkBlack
				str = []rune{r}
			}
		}
	}
	if len(str) > 0 {
		ret = append(ret, str)
	}
	return ret, prefixLength
}

// Tokenize tokenizes the given string with the delimiter
func Tokenize(runes []rune, delimiter Delimiter) []Token {
	if delimiter.str == nil && delimiter.regex == nil {
		// AWK-style (\S+\s*)
		tokens, prefixLength := awkTokenizer(runes)
		return withPrefixLengths(tokens, prefixLength)
	}

	var tokens []string
	if delimiter.str != nil {
		tokens = strings.Split(string(runes), *delimiter.str)
		for i := 0; i < len(tokens)-1; i++ {
			tokens[i] = tokens[i] + *delimiter.str
		}
	} else if delimiter.regex != nil {
		str := string(runes)
		for len(str) > 0 {
			loc := delimiter.regex.FindStringIndex(str)
			if loc == nil {
				loc = []int{0, len(str)}
			}
			last := util.Max(loc[1], 1)
			tokens = append(tokens, str[:last])
			str = str[last:]
		}
	}
	asRunes := make([][]rune, len(tokens))
	for i, token := range tokens {
		asRunes[i] = []rune(token)
	}
	return withPrefixLengths(asRunes, 0)
}

func joinTokens(tokens []Token) []rune {
	ret := []rune{}
	for _, token := range tokens {
		ret = append(ret, token.text...)
	}
	return ret
}

func joinTokensAsRunes(tokens []Token) []rune {
	ret := []rune{}
	for _, token := range tokens {
		ret = append(ret, token.text...)
	}
	return ret
}

// Transform is used to transform the input when --with-nth option is given
func Transform(tokens []Token, withNth []Range) []Token {
	transTokens := make([]Token, len(withNth))
	numTokens := len(tokens)
	for idx, r := range withNth {
		part := []rune{}
		minIdx := 0
		if r.begin == r.end {
			idx := r.begin
			if idx == rangeEllipsis {
				part = append(part, joinTokensAsRunes(tokens)...)
			} else {
				if idx < 0 {
					idx += numTokens + 1
				}
				if idx >= 1 && idx <= numTokens {
					minIdx = idx - 1
					part = append(part, tokens[idx-1].text...)
				}
			}
		} else {
			var begin, end int
			if r.begin == rangeEllipsis { // ..N
				begin, end = 1, r.end
				if end < 0 {
					end += numTokens + 1
				}
			} else if r.end == rangeEllipsis { // N..
				begin, end = r.begin, numTokens
				if begin < 0 {
					begin += numTokens + 1
				}
			} else {
				begin, end = r.begin, r.end
				if begin < 0 {
					begin += numTokens + 1
				}
				if end < 0 {
					end += numTokens + 1
				}
			}
			minIdx = util.Max(0, begin-1)
			for idx := begin; idx <= end; idx++ {
				if idx >= 1 && idx <= numTokens {
					part = append(part, tokens[idx-1].text...)
				}
			}
		}
		var prefixLength int
		if minIdx < numTokens {
			prefixLength = tokens[minIdx].prefixLength
		} else {
			prefixLength = 0
		}
		transTokens[idx] = Token{part, prefixLength, util.TrimLen(part)}
	}
	return transTokens
}
