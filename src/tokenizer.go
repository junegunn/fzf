package fzf

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/junegunn/fzf/src/util"
)

const rangeEllipsis = 0

// Range represents nth-expression
type Range struct {
	Begin int
	End   int
}

// Token contains the tokenized part of the strings and its prefix length
type Token struct {
	text         *util.Chars
	prefixLength int32
}

// Delimiter for tokenizing the input
type Delimiter struct {
	Regex *regexp.Regexp
	Str   *string
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

func withPrefixLengths(tokens []string, begin int) []Token {
	ret := make([]Token, len(tokens))

	prefixLength := begin
	for idx := range tokens {
		chars := util.ToChars([]byte(tokens[idx]))
		ret[idx] = Token{&chars, int32(prefixLength)}
		prefixLength += chars.Length()
	}
	return ret
}

const (
	awkNil = iota
	awkBlack
	awkWhite
)

func awkTokenizer(input string) ([]string, int) {
	// 9, 32
	ret := []string{}
	prefixLength := 0
	state := awkNil
	begin := 0
	end := 0
	for idx := 0; idx < len(input); idx++ {
		r := input[idx]
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
				ret = append(ret, input[begin:end])
				state, begin, end = awkBlack, idx, idx+1
			}
		}
	}
	if begin < end {
		ret = append(ret, input[begin:end])
	}
	return ret, prefixLength
}

// Tokenize tokenizes the given string with the delimiter
func Tokenize(text string, delimiter Delimiter) []Token {
	if delimiter.Str == nil && delimiter.Regex == nil {
		// AWK-style (\S+\s*)
		tokens, prefixLength := awkTokenizer(text)
		return withPrefixLengths(tokens, prefixLength)
	}

	if delimiter.Str != nil {
		return withPrefixLengths(strings.SplitAfter(text, *delimiter.Str), 0)
	}

	// FIXME performance
	var tokens []string
	if delimiter.Regex != nil {
		for len(text) > 0 {
			loc := delimiter.Regex.FindStringIndex(text)
			if len(loc) < 2 {
				loc = []int{0, len(text)}
			}
			last := util.Max(loc[1], 1)
			tokens = append(tokens, text[:last])
			text = text[last:]
		}
	}
	return withPrefixLengths(tokens, 0)
}

func joinTokens(tokens []Token) string {
	var output bytes.Buffer
	for _, token := range tokens {
		output.WriteString(token.text.ToString())
	}
	return output.String()
}

// Transform is used to transform the input when --with-nth option is given
func Transform(tokens []Token, withNth []Range) []Token {
	transTokens := make([]Token, len(withNth))
	numTokens := len(tokens)
	for idx, r := range withNth {
		parts := []*util.Chars{}
		minIdx := 0
		if r.Begin == r.End {
			idx := r.Begin
			if idx == rangeEllipsis {
				chars := util.ToChars([]byte(joinTokens(tokens)))
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
			if r.Begin == rangeEllipsis { // ..N
				begin, end = 1, r.End
				if end < 0 {
					end += numTokens + 1
				}
			} else if r.End == rangeEllipsis { // N..
				begin, end = r.Begin, numTokens
				if begin < 0 {
					begin += numTokens + 1
				}
			} else {
				begin, end = r.Begin, r.End
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
			merged = util.ToChars([]byte{})
		case 1:
			merged = *parts[0]
		default:
			var output bytes.Buffer
			for _, part := range parts {
				output.WriteString(part.ToString())
			}
			merged = util.ToChars([]byte(output.String()))
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
