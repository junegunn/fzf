package fzf

import (
	"testing"
)

func TestParseRangeEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expectOk bool
		expBegin int
		expEnd   int
	}{
		// Valid cases
		{"..", true, rangeEllipsis, rangeEllipsis},
		{"1..", true, rangeEllipsis, rangeEllipsis}, // newRange converts begin=1 to rangeEllipsis
		{"..5", true, rangeEllipsis, 5},
		{"1..5", true, rangeEllipsis, 5}, // newRange converts begin=1 to rangeEllipsis
		{"-1..-5", true, -1, -5},
		{"-5..-1", true, -5, rangeEllipsis}, // newRange converts end=-1 to rangeEllipsis
		{"1", true, 1, 1},                   // single 1 returns {1, 1} then newRange converts to {0, 0}
		{"-1", true, -1, rangeEllipsis},     // newRange converts end=-1 to rangeEllipsis
		{"0", false, 0, 0},                  // 0 is invalid
		// Invalid cases
		{"1..3..5", false, 0, 0},
		{"-3..3", false, 0, 0}, // Mixed signs
		{"abc", false, 0, 0},
		{"", false, 0, 0},
	}

	for _, tt := range tests {
		r, ok := ParseRange(&tt.input)
		if ok != tt.expectOk {
			t.Errorf("ParseRange(%q) ok=%v, expected %v", tt.input, ok, tt.expectOk)
			continue
		}
		if !ok {
			continue
		}
		if r.begin != tt.expBegin || r.end != tt.expEnd {
			t.Errorf("ParseRange(%q) = {%d, %d}, expected {%d, %d}",
				tt.input, r.begin, r.end, tt.expBegin, tt.expEnd)
		}
	}
}

func TestRangeIsFull(t *testing.T) {
	tests := []struct {
		r        Range
		expected bool
	}{
		{Range{rangeEllipsis, rangeEllipsis}, true},
		{Range{1, rangeEllipsis}, false},
		{Range{rangeEllipsis, 1}, false},
		{Range{1, 1}, false},
	}

	for _, tt := range tests {
		result := tt.r.IsFull()
		if result != tt.expected {
			t.Errorf("Range{%d, %d}.IsFull() = %v, expected %v",
				tt.r.begin, tt.r.end, result, tt.expected)
		}
	}
}

func TestCompareRanges(t *testing.T) {
	tests := []struct {
		r1       []Range
		r2       []Range
		expected bool
	}{
		{[]Range{{1, 1}}, []Range{{1, 1}}, true},
		{[]Range{{1, 1}}, []Range{{1, 2}}, false},
		{[]Range{{1, 1}, {2, 2}}, []Range{{1, 1}, {2, 2}}, true},
		{[]Range{{1, 1}}, []Range{{1, 1}, {2, 2}}, false},
		{[]Range{}, []Range{}, true},
		{[]Range{{1, 1}}, []Range{}, false},
	}

	for _, tt := range tests {
		result := compareRanges(tt.r1, tt.r2)
		if result != tt.expected {
			t.Errorf("compareRanges(%v, %v) = %v, expected %v",
				tt.r1, tt.r2, result, tt.expected)
		}
	}
}

func TestRangesToString(t *testing.T) {
	tests := []struct {
		ranges   []Range
		expected string
	}{
		{[]Range{{rangeEllipsis, rangeEllipsis}}, ".."},
		{[]Range{{1, 1}}, "1"},
		{[]Range{{1, 5}}, "1..5"},
		{[]Range{{rangeEllipsis, 5}}, "..5"},
		{[]Range{{1, rangeEllipsis}}, "1.."},
		{[]Range{{1, 1}, {2, 2}}, "1,2"},
		{[]Range{{1, 5}, {rangeEllipsis, rangeEllipsis}}, "1..5,.."},
		{[]Range{}, ""},
	}

	for _, tt := range tests {
		result := RangesToString(tt.ranges)
		if result != tt.expected {
			t.Errorf("RangesToString(%v) = %q, expected %q",
				tt.ranges, result, tt.expected)
		}
	}
}

func TestAwkTokenizer(t *testing.T) {
	tests := []struct {
		input        string
		expTokens    []string
		expPrefixLen int
	}{
		{"hello world", []string{"hello ", "world"}, 0},
		{"  hello   world  ", []string{"hello   ", "world  "}, 2},
		{"hello", []string{"hello"}, 0},
		{"  hello", []string{"hello"}, 2},
		{"hello  ", []string{"hello  "}, 0},
		{"", []string{}, 0},
		{"   ", []string{}, 3},
		{"a b c d", []string{"a ", "b ", "c ", "d"}, 0},
		{"hello\tworld", []string{"hello\t", "world"}, 0},
		{"hello\nworld", []string{"hello\n", "world"}, 0},
	}

	for _, tt := range tests {
		tokens, prefixLen := awkTokenizer(tt.input)
		if prefixLen != tt.expPrefixLen {
			t.Errorf("awkTokenizer(%q) prefixLen=%d, expected %d",
				tt.input, prefixLen, tt.expPrefixLen)
		}
		if len(tokens) != len(tt.expTokens) {
			t.Errorf("awkTokenizer(%q) tokens=%v, expected %v",
				tt.input, tokens, tt.expTokens)
			continue
		}
		for i := range tokens {
			if tokens[i] != tt.expTokens[i] {
				t.Errorf("awkTokenizer(%q)[%d] = %q, expected %q",
					tt.input, i, tokens[i], tt.expTokens[i])
			}
		}
	}
}

func TestDelimiterIsAwk(t *testing.T) {
	str := ","
	tests := []struct {
		d        Delimiter
		expected bool
	}{
		{Delimiter{}, true},
		{Delimiter{str: &str}, false},
	}

	for _, tt := range tests {
		result := tt.d.IsAwk()
		if result != tt.expected {
			t.Errorf("Delimiter.IsAwk() = %v, expected %v", result, tt.expected)
		}
	}
}
