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
