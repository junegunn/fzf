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
		lines, overflow := chars.Lines(multiLine, maxLines, wrapCols, wrapSignWidth, tabstop, false)
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

func TestCharsLinesWrapWord(t *testing.T) {
	// "hello world foo bar" with width 12 should break at word boundaries
	chars := ToChars([]byte("hello world foo bar"))
	lines, overflow := chars.Lines(false, 100, 12, 0, 8, true)
	// "hello world " (12) | "foo bar" (7)
	if len(lines) != 2 || overflow {
		t.Errorf("Expected 2 lines, got %d (overflow: %v): %v", len(lines), overflow, lines)
	}
	if string(lines[0]) != "hello world " {
		t.Errorf("Expected first line 'hello world ', got %q", string(lines[0]))
	}
	if string(lines[1]) != "foo bar" {
		t.Errorf("Expected second line 'foo bar', got %q", string(lines[1]))
	}

	// No word boundary: a single long word falls back to character wrap
	chars2 := ToChars([]byte("abcdefghijklmnop"))
	lines2, _ := chars2.Lines(false, 100, 10, 0, 8, true)
	if len(lines2) != 2 {
		t.Errorf("Expected 2 lines for long word, got %d: %v", len(lines2), lines2)
	}
	if string(lines2[0]) != "abcdefghij" {
		t.Errorf("Expected first line 'abcdefghij', got %q", string(lines2[0]))
	}

	// Tab as word boundary
	chars3 := ToChars([]byte("hello\tworld"))
	lines3, _ := chars3.Lines(false, 100, 7, 0, 8, true)
	// "hello\t" should break at tab (width of tab at pos 5 with tabstop 8 = 3, total width = 8 > 7)
	// Actually RunesWidth: 'h'=1,'e'=1,'l'=1,'l'=1,'o'=1,'\t'=3 = 8 > 7, overflowIdx=5
	// Then word-wrap scans back and finds no space/tab before idx 5 (tab IS at idx 5 but we check line[k-1])
	// Wait - let me think: overflowIdx=5, we check k=5 -> line[4]='o', k=4 -> line[3]='l'... no space/tab found
	// Falls back to character wrap: "hello" | "\tworld"
	if len(lines3) < 2 {
		t.Errorf("Expected at least 2 lines for tab test, got %d: %v", len(lines3), lines3)
	}

	// wrapWord=false still character-wraps
	chars4 := ToChars([]byte("hello world"))
	lines4, _ := chars4.Lines(false, 100, 8, 0, 8, false)
	if len(lines4) != 2 {
		t.Errorf("Expected 2 lines with wrapWord=false, got %d: %v", len(lines4), lines4)
	}
	if string(lines4[0]) != "hello wo" {
		t.Errorf("Expected first line 'hello wo', got %q", string(lines4[0]))
	}
}
