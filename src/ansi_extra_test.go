package fzf

import (
	"testing"

	"github.com/junegunn/fzf/src/tui"
)

func TestAnsiStateEquals(t *testing.T) {
	// Test nil comparison
	s1 := &ansiState{fg: -1, bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}
	if !s1.equals(nil) {
		t.Error("Empty state should equal nil")
	}

	// Test with colors
	s2 := &ansiState{fg: tui.Color(31), bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}
	if s2.equals(nil) {
		t.Error("Colored state should not equal nil")
	}

	// Test same state
	s3 := &ansiState{fg: tui.Color(31), bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}
	if !s2.equals(s3) {
		t.Error("Identical states should be equal")
	}

	// Test different states
	s4 := &ansiState{fg: tui.Color(32), bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}
	if s2.equals(s4) {
		t.Error("Different fg colors should not be equal")
	}
}

func TestAnsiStateColored(t *testing.T) {
	tests := []struct {
		state    ansiState
		expected bool
	}{
		{ansiState{fg: -1, bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}, false},
		{ansiState{fg: tui.Color(31), bg: -1, ul: -1, attr: 0, lbg: -1, url: nil}, true},
		{ansiState{fg: -1, bg: tui.Color(40), ul: -1, attr: 0, lbg: -1, url: nil}, true},
		{ansiState{fg: -1, bg: -1, ul: tui.Color(58), attr: 0, lbg: -1, url: nil}, true},
		{ansiState{fg: -1, bg: -1, ul: -1, attr: tui.Bold, lbg: -1, url: nil}, true},
		{ansiState{fg: -1, bg: -1, ul: -1, attr: 0, lbg: tui.Color(41), url: nil}, true},
		{ansiState{fg: -1, bg: -1, ul: -1, attr: 0, lbg: -1, url: &url{uri: "http://example.com"}}, true},
	}

	for _, tt := range tests {
		result := tt.state.colored()
		if result != tt.expected {
			t.Errorf("colored() for state %+v = %v, expected %v", tt.state, result, tt.expected)
		}
	}
}

func TestExtractColorEdgeCases(t *testing.T) {
	// Test empty string
	trimmed, offsets, state := extractColor("", nil, nil)
	if trimmed != "" || offsets != nil || state != nil {
		t.Errorf("Empty string: got %q, %v, %v", trimmed, offsets, state)
	}

	// Test string with no ANSI codes
	trimmed, offsets, state = extractColor("hello world", nil, nil)
	if trimmed != "hello world" || offsets != nil || state != nil {
		t.Errorf("No ANSI: got %q, %v, %v", trimmed, offsets, state)
	}

	// Test with initial state
	initialState := &ansiState{fg: tui.Color(31)}
	trimmed, offsets, state = extractColor("hello", initialState, nil)
	if trimmed != "hello" || offsets == nil || len(*offsets) != 1 {
		t.Errorf("With initial state: got %q, %v, %v", trimmed, offsets, state)
	}
}
