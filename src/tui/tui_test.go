package tui

import "testing"

func TestWrapLine(t *testing.T) {
	// Basic wrapping
	lines := WrapLine("hello world", 0, 7, 8, 2)
	if len(lines) != 2 || lines[0].Text != "hello w" || lines[1].Text != "orld" {
		t.Errorf("Basic wrap: %v", lines)
	}

	// Exact fit — no wrapping needed
	lines = WrapLine("hello", 0, 5, 8, 2)
	if len(lines) != 1 || lines[0].Text != "hello" || lines[0].DisplayWidth != 5 {
		t.Errorf("Exact fit: %v", lines)
	}

	// With prefix length
	lines = WrapLine("hello", 3, 5, 8, 2)
	if len(lines) != 2 || lines[0].Text != "he" || lines[1].Text != "llo" {
		t.Errorf("Prefix length: %v", lines)
	}

	// Empty string
	lines = WrapLine("", 0, 10, 8, 2)
	if len(lines) != 1 || lines[0].Text != "" || lines[0].DisplayWidth != 0 {
		t.Errorf("Empty string: %v", lines)
	}

	// Continuation lines account for wrapSignWidth
	lines = WrapLine("abcdefghij", 0, 5, 8, 2)
	// First line: "abcde" (5 chars fit in width 5)
	// Continuation max: 5-2=3, so "fgh" then "ij"
	if len(lines) != 3 || lines[0].Text != "abcde" || lines[1].Text != "fgh" || lines[2].Text != "ij" {
		t.Errorf("Continuation: %v", lines)
	}

	// Tab expansion
	lines = WrapLine("\there", 0, 10, 4, 2)
	if len(lines) != 1 || lines[0].DisplayWidth != 8 {
		t.Errorf("Tab: %v", lines)
	}
}

func TestHexToColor(t *testing.T) {
	assert := func(expr string, r, g, b int) {
		color := HexToColor(expr)
		if !color.is24() ||
			int((color>>16)&0xff) != r ||
			int((color>>8)&0xff) != g ||
			int((color)&0xff) != b {
			t.Fail()
		}
	}

	assert("#ff0000", 255, 0, 0)
	assert("#010203", 1, 2, 3)
	assert("#102030", 16, 32, 48)
	assert("#ffffff", 255, 255, 255)
}
