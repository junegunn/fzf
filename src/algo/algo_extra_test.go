package algo

import (
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func TestIndexAt(t *testing.T) {
	tests := []struct {
		index    int
		max      int
		forward  bool
		expected int
	}{
		{0, 10, true, 0},
		{0, 10, false, 9},
		{5, 10, true, 5},
		{5, 10, false, 4},
		{9, 10, true, 9},
		{9, 10, false, 0},
	}

	for _, tt := range tests {
		result := indexAt(tt.index, tt.max, tt.forward)
		if result != tt.expected {
			t.Errorf("indexAt(%d, %d, %v) = %d, expected %d",
				tt.index, tt.max, tt.forward, result, tt.expected)
		}
	}
}

func TestFuzzyMatchV2EmptyPattern(t *testing.T) {
	// Test with empty pattern - returns (0, 0, 0) and pos array when withPos=true
	chars := util.ToChars([]byte("hello world"))
	res, pos := FuzzyMatchV2(false, false, true, &chars, []rune{}, true, nil)
	if res.Start != 0 || res.End != 0 {
		t.Errorf("Empty pattern should return (0, 0), got (%d, %d)", res.Start, res.End)
	}
	// When withPos=true and M=0, posArray returns a non-nil empty slice
	if pos == nil {
		t.Error("Empty pattern with withPos=true should return non-nil pos")
	}
}

func TestFuzzyMatchV2NoMatch(t *testing.T) {
	// Test with pattern that doesn't match
	chars := util.ToChars([]byte("hello world"))
	res, pos := FuzzyMatchV2(false, false, true, &chars, []rune("xyz"), true, nil)
	if res.Start != -1 || res.End != -1 {
		t.Errorf("No match should return (-1, -1), got (%d, %d)", res.Start, res.End)
	}
	if pos != nil {
		t.Error("No match should return nil pos")
	}
}

func TestExactMatchNaiveEmptyPattern(t *testing.T) {
	chars := util.ToChars([]byte("hello world"))
	res, pos := ExactMatchNaive(false, false, true, &chars, []rune{}, true, nil)
	if res.Start != 0 || res.End != 0 {
		t.Errorf("Empty pattern should return (0, 0), got (%d, %d)", res.Start, res.End)
	}
	if pos != nil {
		t.Error("Empty pattern should return nil pos")
	}
}

func TestPrefixMatchEmptyPattern(t *testing.T) {
	chars := util.ToChars([]byte("hello world"))
	res, pos := PrefixMatch(false, false, true, &chars, []rune{}, true, nil)
	if res.Start != 0 || res.End != 0 {
		t.Errorf("Empty pattern should return (0, 0), got (%d, %d)", res.Start, res.End)
	}
	if pos != nil {
		t.Error("Empty pattern should return nil pos")
	}
}

func TestSuffixMatchEmptyPattern(t *testing.T) {
	chars := util.ToChars([]byte("hello world"))
	res, pos := SuffixMatch(false, false, true, &chars, []rune{}, true, nil)
	if res.Start != 11 || res.End != 11 {
		t.Errorf("Empty pattern should return (11, 11), got (%d, %d)", res.Start, res.End)
	}
	if pos != nil {
		t.Error("Empty pattern should return nil pos")
	}
}

func TestEqualMatchBasic(t *testing.T) {
	tests := []struct {
		input    string
		pattern  string
		caseSens bool
		expStart int
		expEnd   int
	}{
		// EqualMatch requires exact match of entire string (minus leading/trailing whitespace)
		{"hello", "hello", false, 0, 5},
		{"hello", "HELLO", true, -1, -1}, // case sensitive match fails
		{"hello", "xyz", false, -1, -1},
		// Pattern longer than input
		{"hello", "hello world", false, -1, -1},
	}

	for _, tt := range tests {
		chars := util.ToChars([]byte(tt.input))
		pattern := []rune(tt.pattern)
		res, pos := EqualMatch(tt.caseSens, false, true, &chars, pattern, true, nil)
		if res.Start != tt.expStart || res.End != tt.expEnd {
			t.Errorf("EqualMatch(%q, %q, caseSens=%v) = (%d, %d), expected (%d, %d)",
				tt.input, tt.pattern, tt.caseSens, res.Start, res.End, tt.expStart, tt.expEnd)
		}
		if pos != nil {
			t.Error("EqualMatch should return nil pos when withPos=false")
		}
	}
}

func TestFuzzyMatchV2WithPos(t *testing.T) {
	// Test that position array is correctly populated
	chars := util.ToChars([]byte("hello world"))
	res, pos := FuzzyMatchV2(false, false, true, &chars, []rune("hw"), true, nil)
	if res.Start == -1 {
		t.Fatal("Expected match")
	}
	if pos == nil || len(*pos) != 2 {
		t.Fatalf("Expected pos array of length 2, got %v", pos)
	}
	// Positions are stored in the order they appear in the pattern
	// 'h' at index 0, 'w' at index 6
	// Note: The actual order may depend on implementation details
	if !((*pos)[0] == 0 && (*pos)[1] == 6) && !((*pos)[0] == 6 && (*pos)[1] == 0) {
		t.Errorf("Expected positions containing [0, 6] or [6, 0], got %v", *pos)
	}
}

func TestFuzzyMatchV2Backward(t *testing.T) {
	// Test backward matching
	assertMatch(t, FuzzyMatchV2, false, false, "foobar fb", "fb", 7, 9,
		scoreMatch*2+int(bonusBoundaryWhite)*bonusFirstCharMultiplier+int(bonusBoundaryWhite))
}

func TestInit(t *testing.T) {
	tests := []struct {
		scheme   string
		expected bool
	}{
		{"default", true},
		{"path", true},
		{"history", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := Init(tt.scheme)
		if result != tt.expected {
			t.Errorf("Init(%q) = %v, expected %v", tt.scheme, result, tt.expected)
		}
	}

	// Reset to default after tests
	Init("default")
}
