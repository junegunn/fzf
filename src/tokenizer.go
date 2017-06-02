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
	text         *util.Chars
	prefixLength int32
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

func withPrefixLengths(tokens []util.Chars, begin int) []Token {
	ret := make([]Token, len(tokens))

	prefixLength := begin
	for idx, token := range tokens {
		// NOTE: &tokens[idx] instead of &tokens
		ret[idx] = Token{&tokens[idx], int32(prefixLength)}
		prefixLength += token.Length()
	}
	return ret
}

const (
	awkNil = iota
	awkBlack
	awkWhite
)

func awkTokenizer(input util.Chars) ([]util.Chars, int) {
	// 9, 32
	ret := []util.Chars{}
	prefixLength := 0
	state := awkNil
	numChars := input.Length()
	begin := 0
	end := 0
	for idx := 0; idx < numChars; idx++ {
		r := input.Get(idx)
		white := r == 9 || r == 32
		switch state {
		case awkNil:
			if white {
				prefixLength++
			} else {
				state, begin, end = awkBlack, idx, idx+1
			}
		case awkBlack:
			end = idx + 1
			if white {
				state = awkWhite
			}
		case awkWhite:
			if white {
				end = idx + 1
			} else {
				ret = append(ret, input.Slice(begin, end))
				state, begin, end = awkBlack, idx, idx+1
			}
		}
	}
	if begin < end {
		ret = append(ret, input.Slice(begin, end))
	}
	return ret, prefixLength
}

// Tokenize tokenizes the given string with the delimiter
func Tokenize(text util.Chars, delimiter Delimiter) []Token {
	if delimiter.str == nil && delimiter.regex == nil {
		// AWK-style (\S+\s*)
		tokens, prefixLength := awkTokenizer(text)
		return withPrefixLengths(tokens, prefixLength)
	}

	if delimiter.str != nil {
		return withPrefixLengths(text.Split(*delimiter.str), 0)
	}

	// FIXME performance
	var tokens []string
	if delimiter.regex != nil {
		str := text.ToString()
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
	asRunes := make([]util.Chars, len(tokens))
	for i, token := range tokens {
		asRunes[i] = util.RunesToChars([]rune(token))
	}
	return withPrefixLengths(asRunes, 0)
}

func joinTokens(tokens []Token) []rune {
	ret := []rune{}
	for _, token := range tokens {
		ret = append(ret, token.text.ToRunes()...)
	}
	return ret
}

// Transform is used to transform the input when --with-nth option is given
func Transform(tokens []Token, withNth []Range) []Token {
	transTokens := make([]Token, len(withNth))
	numTokens := len(tokens)
	for idx, r := range withNth {
		parts := []*util.Chars{}
		minIdx := 0
		if r.begin == r.end {
			idx := r.begin
			if idx == rangeEllipsis {
				chars := util.RunesToChars(joinTokens(tokens))
				parts = append(parts, &chars)
			} else {
				if idx < 0 {
					idx += numTokens + 1
				}
				if idx >= 1 && idx <= numTokens {
					minIdx = idx - 1
					parts = append(parts, tokens[idx-1].text)
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
					parts = append(parts, tokens[idx-1].text)
				}
			}
		}
		// Merge multiple parts
		var merged util.Chars
		switch len(parts) {
		case 0:
			merged = util.RunesToChars([]rune{})
		case 1:
			merged = *parts[0]
		default:
			runes := []rune{}
			for _, part := range parts {
				runes = append(runes, part.ToRunes()...)
			}
			merged = util.RunesToChars(runes)
		}

		var prefixLength int32
		if minIdx < numTokens {
			prefixLength = tokens[minIdx].prefixLength
		} else {
			prefixLength = 0
		}
		transTokens[idx] = Token{&merged, prefixLength}
	}
	return transTokens
}
