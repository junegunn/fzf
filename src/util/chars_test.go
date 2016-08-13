package util

import "testing"

func TestToCharsNil(t *testing.T) {
	bs := Chars{bytes: []byte{}}
	if bs.bytes == nil || bs.runes != nil {
		t.Error()
	}
	rs := RunesToChars([]rune{})
	if rs.bytes != nil || rs.runes == nil {
		t.Error()
	}
}

func TestToCharsAscii(t *testing.T) {
	chars := ToChars([]byte("foobar"))
	if chars.ToString() != "foobar" || chars.runes != nil {
		t.Error()
	}
}

func TestCharsLength(t *testing.T) {
	chars := ToChars([]byte("\tabc한글  "))
	if chars.Length() != 8 || chars.TrimLength() != 5 {
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
	check := func(str string, exp int) {
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
