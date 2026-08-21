package fzf

import (
	"testing"

	"github.com/junegunn/fzf/src/tui"
)

func TestParseAlgo(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
	}{
		{"v1", false},
		{"v2", false},
		{"V1", true},
		{"V2", true},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		result, err := parseAlgo(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("parseAlgo(%q) expected error, got %v", tt.input, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAlgo(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result == nil {
			t.Errorf("parseAlgo(%q) returned nil algo", tt.input)
		}
	}
}

func TestIsAlphabet(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'a', true},
		{'z', true},
		{'m', true},
		{'A', false}, // Only lowercase
		{'Z', false},
		{'0', false},
		{'9', false},
		{' ', false},
		{'@', false},
	}

	for _, tt := range tests {
		result := isAlphabet(tt.input)
		if result != tt.expected {
			t.Errorf("isAlphabet(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input    byte
		expected bool
	}{
		{'0', true},
		{'5', true},
		{'9', true},
		{'a', false},
		{'z', false},
		{' ', false},
		{'/', false},
		{':', false},
	}

	for _, tt := range tests {
		result := isNumeric(tt.input)
		if result != tt.expected {
			t.Errorf("isNumeric(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseBorder(t *testing.T) {
	tests := []struct {
		input       string
		optional    bool
		expected    tui.BorderShape
		expectError bool
	}{
		{"rounded", false, tui.BorderRounded, false},
		{"sharp", false, tui.BorderSharp, false},
		{"bold", false, tui.BorderBold, false},
		{"block", false, tui.BorderBlock, false},
		{"thinblock", false, tui.BorderThinBlock, false},
		{"double", false, tui.BorderDouble, false},
		{"horizontal", false, tui.BorderHorizontal, false},
		{"vertical", false, tui.BorderVertical, false},
		{"top", false, tui.BorderTop, false},
		{"bottom", false, tui.BorderBottom, false},
		{"left", false, tui.BorderLeft, false},
		{"right", false, tui.BorderRight, false},
		{"line", false, tui.BorderLine, false},
		{"none", false, tui.BorderNone, false},
		// Optional with empty string
		{"", true, tui.DefaultBorderShape, false},
		// Invalid
		{"invalid", false, tui.BorderNone, true},
		{"", false, tui.BorderNone, true},
	}

	for _, tt := range tests {
		result, err := parseBorder(tt.input, tt.optional)
		if tt.expectError {
			if err == nil {
				t.Errorf("parseBorder(%q, %v) expected error, got %v", tt.input, tt.optional, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBorder(%q, %v) unexpected error: %v", tt.input, tt.optional, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("parseBorder(%q, %v) = %v, expected %v", tt.input, tt.optional, result, tt.expected)
		}
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"100", 100, false},
		{"-1", -1, false},
		{"-100", -100, false},
		// Invalid
		{"abc", 0, true},
		{"12.34", 0, true},
		{"", 0, true},
		{" 1", 0, true},
		{"1 ", 0, true},
	}

	for _, tt := range tests {
		result, err := atoi(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("atoi(%q) expected error, got %d", tt.input, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("atoi(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("atoi(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestAtof(t *testing.T) {
	tests := []struct {
		input       string
		expected    float64
		expectError bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"1.5", 1.5, false},
		{"-1.5", -1.5, false},
		{"100.123", 100.123, false},
		// Invalid
		{"abc", 0, true},
		{"", 0, true},
		{" 1", 0, true},
		{"1 ", 0, true},
	}

	for _, tt := range tests {
		result, err := atof(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("atof(%q) expected error, got %f", tt.input, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("atof(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("atof(%q) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestFilterNonEmpty(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{[]string{}, []string{}},
		{[]string{""}, []string{}},
		{[]string{"", ""}, []string{}},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "b"}, []string{"a", "b"}},
		{[]string{"", "a", ""}, []string{"a"}},
		{[]string{"a", "", "b"}, []string{"a", "b"}},
		{[]string{"", "a", "", "b", ""}, []string{"a", "b"}},
	}

	for _, tt := range tests {
		result := filterNonEmpty(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("filterNonEmpty(%v) = %v, expected %v", tt.input, result, tt.expected)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("filterNonEmpty(%v)[%d] = %q, expected %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestIsDir(t *testing.T) {
	// Testing isDir is difficult without creating actual directories
	// We can at least test that it returns false for non-existent paths
	if isDir("/nonexistent/path/that/should/not/exist") {
		t.Error("isDir should return false for non-existent paths")
	}

	// Test with a file (this file itself)
	if isDir("options_extra_test.go") {
		t.Error("isDir should return false for files")
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello\nworld", "hello"},
		{"hello\r\nworld", "hello\r"}, // firstLine splits on \n only
		{"hello\nworld\nfoo", "hello"},
		{"\n", ""},
		{"\nhello", ""},
	}

	for _, tt := range tests {
		result := firstLine(tt.input)
		if result != tt.expected {
			t.Errorf("firstLine(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	// Check some default values
	if opts.Fuzzy != true {
		t.Error("Default Fuzzy should be true")
	}
	if opts.Extended != true {
		t.Error("Default Extended should be true")
	}
	if opts.Case != CaseSmart {
		t.Error("Default Case should be CaseSmart")
	}
	if opts.Normalize != true {
		t.Error("Default Normalize should be true")
	}
	if opts.Sort != 1000 {
		t.Errorf("Default Sort should be 1000, got %d", opts.Sort)
	}
	if opts.Tabstop != 8 {
		t.Errorf("Default Tabstop should be 8, got %d", opts.Tabstop)
	}
	if opts.HscrollOff != 10 {
		t.Errorf("Default HscrollOff should be 10, got %d", opts.HscrollOff)
	}
	if opts.ScrollOff != 3 {
		t.Errorf("Default ScrollOff should be 3, got %d", opts.ScrollOff)
	}
	if opts.Bold != true {
		t.Error("Default Bold should be true")
	}
	if opts.Mouse != true {
		t.Error("Default Mouse should be true")
	}
	if opts.ClearOnExit != true {
		t.Error("Default ClearOnExit should be true")
	}
	if opts.Unicode != true {
		t.Error("Default Unicode should be true")
	}
	if opts.MultiLine != true {
		t.Error("Default MultiLine should be true")
	}
	if opts.Hscroll != true {
		t.Error("Default Hscroll should be true")
	}
}
