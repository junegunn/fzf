package util

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"
)

func TestCountRunes(t *testing.T) {
	for _, str := range []string{
		"", "a", "abc", "한글", "🎉🎉", "\tabc한글  ",
		strings.Repeat("漢字", 50), strings.Repeat("a", 33) + "é",
	} {
		if got, exp := countRunes([]byte(str)), utf8.RuneCountInString(str); got != exp {
			t.Errorf("countRunes(%q) = %d, expected %d", str, got, exp)
		}
	}
}

func TestCountRunesRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	// Exact on valid UTF-8
	for trial := range 20000 {
		var sb strings.Builder
		for range rng.Intn(20) {
			r := rune(rng.Intn(utf8.MaxRune + 1))
			for r >= 0xD800 && r <= 0xDFFF {
				r = rune(rng.Intn(utf8.MaxRune + 1))
			}
			sb.WriteRune(r)
		}
		str := sb.String()
		if got, exp := countRunes([]byte(str)), utf8.RuneCountInString(str); got != exp {
			t.Fatalf("trial %d: countRunes(%q) = %d, expected %d", trial, str, got, exp)
		}
	}

	// Never an overcount on arbitrary bytes, so the capacity hint never truncates
	for trial := range 20000 {
		buf := make([]byte, rng.Intn(40))
		rng.Read(buf)
		if got, exp := countRunes(buf), utf8.RuneCount(buf); got > exp {
			t.Fatalf("trial %d: countRunes(%x) = %d, overcounts %d", trial, buf, got, exp)
		}
	}
}

// ToChars must produce exactly what a []rune conversion produces, including
// one RuneError per invalid byte, and must size the rune slice exactly when
// the input is valid UTF-8.
func TestToCharsIntegrity(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	check := func(buf []byte, exactCap bool) {
		chars := ToChars(buf)
		exp := []rune(string(buf))
		if chars.Length() != len(exp) {
			t.Fatalf("ToChars(%x).Length() = %d, expected %d", buf, chars.Length(), len(exp))
		}
		for i, r := range exp {
			if chars.Get(i) != r {
				t.Fatalf("ToChars(%x).Get(%d) = %q, expected %q", buf, i, chars.Get(i), r)
			}
		}
		if runes := chars.optionalRunes(); runes != nil && exactCap && cap(runes) != len(exp) {
			t.Fatalf("ToChars(%x) cap = %d, expected %d", buf, cap(runes), len(exp))
		}
	}

	for range 5000 {
		var sb strings.Builder
		sb.WriteString("ascii")
		for range 1 + rng.Intn(10) {
			r := rune(0x80 + rng.Intn(utf8.MaxRune-0x80))
			for r >= 0xD800 && r <= 0xDFFF {
				r = rune(0x80 + rng.Intn(utf8.MaxRune-0x80))
			}
			sb.WriteRune(r)
		}
		check([]byte(sb.String()), true)
	}

	// Invalid UTF-8: still correct, capacity may grow
	for range 5000 {
		buf := make([]byte, 1+rng.Intn(40))
		rng.Read(buf)
		buf[rng.Intn(len(buf))] |= 0x80 // force the rune path
		check(buf, false)
	}
}

func TestToCharsAscii(t *testing.T) {
	chars := ToChars([]byte("foobar"))
	if !chars.IsBytes() || chars.ToString() != "foobar" {
		t.Error()
	}
}

func TestCharsLength(t *testing.T) {
	chars := ToChars([]byte("\tabc한글  "))
	if chars.IsBytes() || chars.Length() != 8 || chars.TrimLength() != 5 {
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

	// wrapWord=false still character-wraps
	chars3 := ToChars([]byte("hello world"))
	lines3, _ := chars3.Lines(false, 100, 8, 0, 8, false)
	if len(lines3) != 2 {
		t.Errorf("Expected 2 lines with wrapWord=false, got %d: %v", len(lines3), lines3)
	}
	if string(lines3[0]) != "hello wo" {
		t.Errorf("Expected first line 'hello wo', got %q", string(lines3[0]))
	}
}

// Chars is one per input line, so its size matters. It has no spare padding,
// which is why new state goes in the flags byte rather than a field.
// Derive the expectation from the slice header so the invariant holds on
// 32-bit builds too, where the header is 12 bytes and Chars is 20.
func TestCharsSize(t *testing.T) {
	var slice []byte
	// flags 1 + trimLengthKnown 1 + trimLength 2 + Index 4, no padding
	want := unsafe.Sizeof(slice) + 8
	if size := unsafe.Sizeof(Chars{}); size != want {
		t.Errorf("unsafe.Sizeof(Chars{}) = %d, expected %d", size, want)
	}
}

func TestMayFoldFlag(t *testing.T) {
	for _, c := range []struct {
		text string
		fold bool
	}{
		{"한글/src", false}, {"漢字", false}, {"мир", false}, {"🎉", false},
		{"café", true}, {"Müller", true}, {"Å", true}, {"ｆｕｌｌ", true},
	} {
		chars := ToChars([]byte(c.text))
		if chars.MayFoldToAscii() != c.fold {
			t.Errorf("ToChars(%q).MayFoldToAscii() = %v, expected %v", c.text, chars.MayFoldToAscii(), c.fold)
		}
		if runes := RunesToChars([]rune(c.text)); runes.MayFoldToAscii() != c.fold {
			t.Errorf("RunesToChars(%q).MayFoldToAscii() = %v, expected %v", c.text, runes.MayFoldToAscii(), c.fold)
		}
	}

	// Prepend can introduce foldable runes
	chars := ToChars([]byte("한글"))
	if chars.MayFoldToAscii() {
		t.Fatal("baseline should not be foldable")
	}
	chars.Prepend("é")
	if !chars.MayFoldToAscii() {
		t.Error("Prepend of a foldable prefix must set the flag")
	}
}

// Runes and ToRunes alias the text in rune mode, so a consumer that mutates
// what they return changes the text without updating the cached fold bit. This
// verifies the aliasing so the read-only contract on those methods is not
// silently dropped later.
func TestRuneSlicesAliasTheText(t *testing.T) {
	chars := ToChars([]byte("한글abc"))
	runes := chars.Runes()
	if runes == nil {
		t.Fatal("expected rune mode")
	}
	if &runes[0] != &chars.ToRunes()[0] {
		t.Error("Runes and ToRunes should return the same backing array")
	}
	if chars.MayFoldToAscii() {
		t.Fatal("baseline should not be foldable")
	}
	// Demonstrates why callers must copy: the flag does not follow the text.
	runes[0] = 'e'
	if chars.MayFoldToAscii() {
		t.Error("flag unexpectedly updated")
	}
	if got := chars.ToString(); got != "e글abc" {
		t.Errorf("expected the write to reach the text, got %q", got)
	}
}
