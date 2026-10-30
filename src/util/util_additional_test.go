package util

import (
	"testing"
)

func TestStringsWidthWithTabs(t *testing.T) {
	tests := []struct {
		str         string
		prefixWidth int
		tabstop     int
		limit       int
		expWidth    int
		expIdx      int
	}{
		// Tab expansion tests
		{"hello\tworld", 0, 8, 100, 13, -1},
		{"\t", 0, 8, 100, 8, -1},
		{"\t", 0, 4, 100, 4, -1},
		// Tab at different positions
		{"ab\tc", 0, 4, 100, 5, -1}, // tab at pos 2, expands to 2 (4-2)
	}

	for _, tt := range tests {
		width, idx := StringsWidth(tt.str, tt.prefixWidth, tt.tabstop, tt.limit)
		if width != tt.expWidth || idx != tt.expIdx {
			t.Errorf("StringsWidth(%q, %d, %d, %d) = (%d, %d), expected (%d, %d)",
				tt.str, tt.prefixWidth, tt.tabstop, tt.limit, width, idx, tt.expWidth, tt.expIdx)
		}
	}
}

func TestStringsWidthOverflow(t *testing.T) {
	tests := []struct {
		str         string
		prefixWidth int
		tabstop     int
		limit       int
		expWidth    int
		expIdx      int
	}{
		// Overflow cases - actual behavior: returns width at overflow point
		{"hello world", 0, 8, 5, 6, 5},
		{"abcdefghij", 0, 8, 5, 6, 5},
		// Unicode overflow
		{"你好世界", 0, 8, 3, 4, 1},
	}

	for _, tt := range tests {
		width, idx := StringsWidth(tt.str, tt.prefixWidth, tt.tabstop, tt.limit)
		if width != tt.expWidth || idx != tt.expIdx {
			t.Errorf("StringsWidth(%q, %d, %d, %d) = (%d, %d), expected (%d, %d)",
				tt.str, tt.prefixWidth, tt.tabstop, tt.limit, width, idx, tt.expWidth, tt.expIdx)
		}
	}
}

func TestTruncateEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		limit    int
		expRunes string
		expWidth int
	}{
		// Edge cases
		{"", 5, "", 0},
		{"a", 0, "", 0},
		{"hello", 0, "", 0},
		// Unicode truncation
		{"你好世界", 4, "你好", 4},
		{"你好世界", 3, "你", 2},
		{"你好世界", 2, "你", 2},
		{"你好世界", 1, "", 0},
		// Mixed content
		{"a你b好", 3, "a你", 3},
	}

	for _, tt := range tests {
		runes, width := Truncate(tt.input, tt.limit)
		if string(runes) != tt.expRunes || width != tt.expWidth {
			t.Errorf("Truncate(%q, %d) = (%q, %d), expected (%q, %d)",
				tt.input, tt.limit, string(runes), width, tt.expRunes, tt.expWidth)
		}
	}
}

func TestConstrainEdgeCases(t *testing.T) {
	tests := []struct {
		val, min, max, expected int
	}{
		// Edge cases
		{3, 3, 3, 3},
		{0, 0, 0, 0},
		{-5, -10, -1, -5},
		{-5, -10, -5, -5},
		{-5, -5, -1, -5},
	}

	for _, tt := range tests {
		result := Constrain(tt.val, tt.min, tt.max)
		if result != tt.expected {
			t.Errorf("Constrain(%d, %d, %d) = %d, expected %d",
				tt.val, tt.min, tt.max, result, tt.expected)
		}
	}
}

func TestAsUint16EdgeCases(t *testing.T) {
	tests := []struct {
		input    int
		expected uint16
	}{
		{65535, 65535},
		{65536, 65535},
		{-1, 0},
		{-65535, 0},
		{-65536, 0},
	}

	for _, tt := range tests {
		result := AsUint16(tt.input)
		if result != tt.expected {
			t.Errorf("AsUint16(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestRunOnceMultiple(t *testing.T) {
	counter := 0
	f := RunOnce(func() { counter++ })

	// First call should execute
	f()
	if counter != 1 {
		t.Errorf("Expected counter=1 after first call, got %d", counter)
	}

	// Subsequent calls should not execute
	for i := 0; i < 10; i++ {
		f()
	}
	if counter != 1 {
		t.Errorf("Expected counter=1 after multiple calls, got %d", counter)
	}
}

func TestRepeatToFillEdgeCases(t *testing.T) {
	tests := []struct {
		str      string
		length   int
		limit    int
		expected string
	}{
		// Edge cases - skip empty string with length 0 to avoid divide by zero
		{"a", 1, 0, ""},
		{"abc", 3, 1, "a"},
		{"abc", 3, 2, "ab"},
		{"abc", 3, 3, "abc"},
		// Unicode
		{"你", 2, 4, "你你"},
		{"你", 2, 3, "你"},
	}

	for _, tt := range tests {
		result := RepeatToFill(tt.str, tt.length, tt.limit)
		if result != tt.expected {
			t.Errorf("RepeatToFill(%q, %d, %d) = %q, expected %q",
				tt.str, tt.length, tt.limit, result, tt.expected)
		}
	}
}

func TestToKebabCaseEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"A", "a"},
		{"AB", "a-b"},
		{"ABC", "a-b-c"},
		{"aBC", "a-b-c"},
		{"TestHTTPServer", "test-h-t-t-p-server"},
		{"XMLParser", "x-m-l-parser"},
	}

	for _, tt := range tests {
		result := ToKebabCase(tt.input)
		if result != tt.expected {
			t.Errorf("ToKebabCase(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestCompareVersionsEdgeCases(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		// Edge cases
		{"", "", 0},
		{"0", "0", 0},
		{"0.0.0", "0", 0},
		{"1", "1.0.0.0.0", 0},
		{"1.0.0.1", "1.0.0.1.0", 0},
		{"1.0.0.1", "1.0.0.2", -1},
		{"01", "1", 0}, // Leading zeros
		{"1.01", "1.1", 0},
	}

	for _, tt := range tests {
		result := CompareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("CompareVersions(%q, %q) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}
