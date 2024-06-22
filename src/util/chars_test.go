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
	chars := ToChars([]byte("abc\n한글\ndef"))
	for _, ml := range []bool{true, false} {
		// No wrap
		lines, overflow := chars.Lines(ml, 1, 0, 8)
		fmt.Println(lines, overflow)
		lines, overflow = chars.Lines(ml, 2, 0, 8)
		fmt.Println(lines, overflow)
		lines, overflow = chars.Lines(ml, 3, 0, 8)
		fmt.Println(lines, overflow)

		// Wrap
		lines, overflow = chars.Lines(ml, 4, 2, 8)
		fmt.Println(lines, overflow)
		lines, overflow = chars.Lines(ml, 100, 1, 8)
		fmt.Println(lines, overflow)

		chars = ToChars([]byte("abc\n한글\ndef\n\n\n"))
		lines, overflow = chars.Lines(ml, 100, 100, 8)
		fmt.Println(lines, overflow)
		numLines, overflow := chars.NumLines(8)
		fmt.Println(numLines, overflow)
	}
}
