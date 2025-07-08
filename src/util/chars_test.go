package util

import (
	"fmt"
	"testing"
)

func TestToCharsAscii(t *testing.T) {
	chars := ToChars([]byte("foobar"))
	if !chars.inBytes || chars.ToString() != "foobar" || !chars.inBytes {
		t.Error()
	}
}

func TestCharsLength(t *testing.T) {
	chars := ToChars([]byte("\tabc한글  "))
	if chars.inBytes || chars.Length() != 8 || chars.TrimLength() != 5 {
		t.Error()
	}
}

func TestCharsToString(t *testing.T) {
	text := "\tabc한글  "
	chars := ToChars([]byte(text))
	if chars.ToString() != text {
		t.Error()
	}
}

func TestTrimLength(t *testing.T) {
	check := func(str string, exp uint16) {
		chars := ToChars([]byte(str))
		trimmed := chars.TrimLength()
		if trimmed != exp {
			t.Errorf("Invalid TrimLength result for '%s': %d (expected %d)",
				str, trimmed, exp)
		}
	}
	check("hello", 5)
	check("hello ", 5)
	check("hello  ", 5)
	check(" hello", 5)
	check("  hello", 5)
	check(" hello ", 5)
	check("  hello  ", 5)
	check("h   o", 5)
	check("  h   o  ", 5)
	check("         ", 0)
}

func TestCharsLines(t *testing.T) {
	chars := ToChars([]byte("abcdef\n가나다\n\tdef"))
	check := func(multiLine bool, maxLines int, wrapCols int, wrapSignWidth int, tabstop int, expectedNumLines int, expectedOverflow bool) {
		lines, overflow := chars.Lines(multiLine, maxLines, wrapCols, wrapSignWidth, tabstop)
		fmt.Println(lines, overflow)
		if len(lines) != expectedNumLines || overflow != expectedOverflow {
			t.Errorf("Invalid result: %d %v (expected %d %v)", len(lines), overflow, expectedNumLines, expectedOverflow)
		}
	}

	// No wrap
	check(true, 1, 0, 0, 8, 1, true)
	check(true, 2, 0, 0, 8, 2, true)
	check(true, 3, 0, 0, 8, 3, false)

	// Wrap (2)
	check(true, 4, 2, 0, 8, 4, true)
	check(true, 5, 2, 0, 8, 5, true)
	check(true, 6, 2, 0, 8, 6, true)
	check(true, 7, 2, 0, 8, 7, true)
	check(true, 8, 2, 0, 8, 8, true)
	check(true, 9, 2, 0, 8, 9, false)
	check(true, 9, 2, 0, 1, 8, false) // Smaller tab size

	// With wrap sign (3 + 1)
	check(true, 100, 3, 1, 1, 8, false)

	// With wrap sign (3 + 2)
	check(true, 100, 3, 2, 1, 10, false)

	// With wrap sign (3 + 2) and no multi-line
	check(false, 100, 3, 2, 1, 13, false)
}
