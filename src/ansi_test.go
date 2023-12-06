package fzf

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/tui"
)

// The following regular expression will include not all but most of the
// frequently used ANSI sequences. This regex is used as a reference for
// testing nextAnsiEscapeSequence().
//
// References:
//   - https://github.com/gnachman/iTerm2
//   - https://web.archive.org/web/20090204053813/http://ascii-table.com/ansi-escape-sequences.php
//     (archived from http://ascii-table.com/ansi-escape-sequences.php)
//   - https://web.archive.org/web/20090227051140/http://ascii-table.com/ansi-escape-sequences-vt-100.php
//     (archived from http://ascii-table.com/ansi-escape-sequences-vt-100.php)
//   - http://tldp.org/HOWTO/Bash-Prompt-HOWTO/x405.html
//   - https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
var ansiRegexReference = regexp.MustCompile("(?:\x1b[\\[()][0-9;:]*[a-zA-Z@]|\x1b][0-9][;:][[:print:]]+(?:\x1b\\\\|\x07)|\x1b.|[\x0e\x0f]|.\x08)")

func testParserReference(t testing.TB, str string) {
	t.Helper()

	toSlice := func(start, end int) []int {
		if start == -1 {
			return nil
		}
		return []int{start, end}
	}

	s := str
	for i := 0; ; i++ {
		got := toSlice(nextAnsiEscapeSequence(s))
		exp := ansiRegexReference.FindStringIndex(s)

		equal := len(got) == len(exp)
		if equal {
			for i := 0; i < len(got); i++ {
				if got[i] != exp[i] {
					equal = false
					break
				}
			}
		}
		if !equal {
			var exps, gots []rune
			if len(got) == 2 {
				gots = []rune(s[got[0]:got[1]])
			}
			if len(exp) == 2 {
				exps = []rune(s[exp[0]:exp[1]])
			}
			t.Errorf("%d: %q: got: %v (%q) want: %v (%q)", i, s, got, gots, exp, exps)
			return
		}
		if len(exp) == 0 {
			return
		}
		s = s[exp[1]:]
	}
}

func TestNextAnsiEscapeSequence(t *testing.T) {
	testStrs := []string{
		"\x1b[0mhello world",
		"\x1b[1mhello world",
		"椙\x1b[1m椙",
		"椙\x1b[1椙m椙",
		"\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d",
		"\x1b[1mhello \x1b[Kworld",
		"hello \x1b[34;45;1mworld",
		"hello \x1b[34;45;1mwor\x1b[34;45;1mld",
		"hello \x1b[34;45;1mwor\x1b[0mld",
		"hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md",
		"hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md",
		"hello \x1b[32;1mworld",
		"hello world",
		"hello \x1b[0;38;5;200;48;5;100mworld",
		"\x1b椙",
		"椙\x08",
		"\n\x08",
		"X\x08",
		"",
		"\x1b]4;3;rgb:aa/bb/cc\x07 ",
		"\x1b]4;3;rgb:aa/bb/cc\x1b\\ ",
		ansiBenchmarkString,
	}

	for _, s := range testStrs {
		testParserReference(t, s)
	}
}

func TestNextAnsiEscapeSequence_Fuzz_Modified(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("short test")
	}

	testStrs := []string{
		"\x1b[0mhello world",
		"\x1b[1mhello world",
		"椙\x1b[1m椙",
		"椙\x1b[1椙m椙",
		"\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d",
		"\x1b[1mhello \x1b[Kworld",
		"hello \x1b[34;45;1mworld",
		"hello \x1b[34;45;1mwor\x1b[34;45;1mld",
		"hello \x1b[34;45;1mwor\x1b[0mld",
		"hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md",
		"hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md",
		"hello \x1b[32;1mworld",
		"hello world",
		"hello \x1b[0;38;5;200;48;5;100mworld",
		ansiBenchmarkString,
	}

	replacementBytes := [...]rune{'\x0e', '\x0f', '\x1b', '\x08'}

	modifyString := func(s string, rr *rand.Rand) string {
		n := rr.Intn(len(s))
		b := []rune(s)
		for ; n >= 0 && len(b) != 0; n-- {
			i := rr.Intn(len(b))
			switch x := rr.Intn(4); x {
			case 0:
				b = append(b[:i], b[i+1:]...)
			case 1:
				j := rr.Intn(len(replacementBytes) - 1)
				b[i] = replacementBytes[j]
			case 2:
				x := rune(rr.Intn(utf8.MaxRune))
				for !utf8.ValidRune(x) {
					x = rune(rr.Intn(utf8.MaxRune))
				}
				b[i] = x
			case 3:
				b[i] = rune(rr.Intn(utf8.MaxRune)) // potentially invalid
			default:
				t.Fatalf("unsupported value: %d", x)
			}
		}
		return string(b)
	}

	rr := rand.New(rand.NewSource(1))
	for _, s := range testStrs {
		for i := 1_000; i >= 0; i-- {
			testParserReference(t, modifyString(s, rr))
		}
	}
}

func TestNextAnsiEscapeSequence_Fuzz_Random(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("short test")
	}

	randomString := func(rr *rand.Rand) string {
		numChars := rand.Intn(50)
		codePoints := make([]rune, numChars)
		for i := 0; i < len(codePoints); i++ {
			var r rune
			for n := 0; n < 1000; n++ {
				r = rune(rr.Intn(utf8.MaxRune))
				// Allow 10% of runes to be invalid
				if utf8.ValidRune(r) || rr.Float64() < 0.10 {
					break
				}
			}
			codePoints[i] = r
		}
		return string(codePoints)
	}

	rr := rand.New(rand.NewSource(1))
	for i := 0; i < 100_000; i++ {
		testParserReference(t, randomString(rr))
	}
}

func TestExtractColor(t *testing.T) {
	assert := func(offset ansiOffset, b int32, e int32, fg tui.Color, bg tui.Color, bold bool) {
		var attr tui.Attr
		if bold {
			attr = tui.Bold
		}
		if offset.offset[0] != b || offset.offset[1] != e ||
			offset.color.fg != fg || offset.color.bg != bg || offset.color.attr != attr {
			t.Error(offset, b, e, fg, bg, attr)
		}
	}

	src := "hello world"
	var state *ansiState
	clean := "\x1b[0m"
	check := func(assertion func(ansiOffsets *[]ansiOffset, state *ansiState)) {
		output, ansiOffsets, newState := extractColor(src, state, nil)
		state = newState
		if output != "hello world" {
			t.Errorf("Invalid output: %s %v", output, []rune(output))
		}
		t.Log(src, ansiOffsets, clean)
		assertion(ansiOffsets, state)
	}

	check(func(offsets *[]ansiOffset, state *ansiState) {
		if offsets != nil {
			t.Fail()
		}
	})

	state = nil
	src = "\x1b[0mhello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if offsets != nil {
			t.Fail()
		}
	})

	state = nil
	src = "\x1b[1mhello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, -1, -1, true)
	})

	state = nil
	src = "\x1b[1mhello \x1b[mw\x1b7o\x1b8r\x1b(Bl\x1b[2@d"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 6, -1, -1, true)
	})

	state = nil
	src = "\x1b[1mhello \x1b[Kworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, -1, -1, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[34;45;1mld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;45;1mwor\x1b[0mld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 9, 4, 5, true)
	})

	state = nil
	src = "hello \x1b[34;48;5;233;1mwo\x1b[38;5;161mr\x1b[0ml\x1b[38;5;161md"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 3 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 8, 4, 233, true)
		assert((*offsets)[1], 8, 9, 161, 233, true)
		assert((*offsets)[2], 10, 11, 161, -1, false)
	})

	// {38,48};5;{38,48}
	state = nil
	src = "hello \x1b[38;5;38;48;5;48;1mwor\x1b[38;5;48;48;5;38ml\x1b[0md"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 2 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 9, 38, 48, true)
		assert((*offsets)[1], 9, 10, 48, 38, true)
	})

	src = "hello \x1b[32;1mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		if state.fg != 2 || state.bg != -1 || state.attr == 0 {
			t.Fail()
		}
		assert((*offsets)[0], 6, 11, 2, -1, true)
	})

	src = "hello world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		if state.fg != 2 || state.bg != -1 || state.attr == 0 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, 2, -1, true)
	})

	src = "hello \x1b[0;38;5;200;48;5;100mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 2 {
			t.Fail()
		}
		if state.fg != 200 || state.bg != 100 || state.attr > 0 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 6, 2, -1, true)
		assert((*offsets)[1], 6, 11, 200, 100, false)
	})
}

func TestAnsiCodeStringConversion(t *testing.T) {
	assert := func(code string, prevState *ansiState, expected string) {
		state := interpretCode(code, prevState)
		if expected != state.ToString() {
			t.Errorf("expected: %s, actual: %s",
				strings.Replace(expected, "\x1b[", "\\x1b[", -1),
				strings.Replace(state.ToString(), "\x1b[", "\\x1b[", -1))
		}
	}
	assert("\x1b[m", nil, "")
	assert("\x1b[m", &ansiState{attr: tui.Blink, lbg: -1}, "")

	assert("\x1b[31m", nil, "\x1b[31;49m")
	assert("\x1b[41m", nil, "\x1b[39;41m")

	assert("\x1b[92m", nil, "\x1b[92;49m")
	assert("\x1b[102m", nil, "\x1b[39;102m")

	assert("\x1b[31m", &ansiState{fg: 4, bg: 4, lbg: -1}, "\x1b[31;44m")
	assert("\x1b[1;2;31m", &ansiState{fg: 2, bg: -1, attr: tui.Reverse, lbg: -1}, "\x1b[1;2;7;31;49m")
	assert("\x1b[38;5;100;48;5;200m", nil, "\x1b[38;5;100;48;5;200m")
	assert("\x1b[38:5:100:48:5:200m", nil, "\x1b[38;5;100;48;5;200m")
	assert("\x1b[48;5;100;38;5;200m", nil, "\x1b[38;5;200;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;1m", nil, "\x1b[1;38;2;10;20;30;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;7m",
		&ansiState{attr: tui.Dim | tui.Italic, fg: 1, bg: 1},
		"\x1b[2;3;7;38;2;10;20;30;48;5;100m")
}

func TestParseAnsiCode(t *testing.T) {
	tests := []struct {
		In, Exp string
		N       int
	}{
		{"123", "", 123},
		{"1a", "", -1},
		{"1a;12", "12", -1},
		{"12;a", "a", 12},
		{"-2", "", -1},
	}
	for _, x := range tests {
		n, _, s := parseAnsiCode(x.In, 0)
		if n != x.N || s != x.Exp {
			t.Fatalf("%q: got: (%d %q) want: (%d %q)", x.In, n, s, x.N, x.Exp)
		}
	}
}

// kernel/bpf/preload/iterators/README
const ansiBenchmarkString = "\x1b[38;5;81m\x1b[01;31m\x1b[Kkernel/\x1b[0m\x1b[38:5:81mbpf/" +
	"\x1b[0m\x1b[38:5:81mpreload/\x1b[0m\x1b[38;5;81miterators/" +
	"\x1b[0m\x1b[38:5:149mMakefile\x1b[m\x1b[K\x1b[0m"

func BenchmarkNextAnsiEscapeSequence(b *testing.B) {
	b.SetBytes(int64(len(ansiBenchmarkString)))
	for i := 0; i < b.N; i++ {
		s := ansiBenchmarkString
		for {
			_, o := nextAnsiEscapeSequence(s)
			if o == -1 {
				break
			}
			s = s[o:]
		}
	}
}

// Baseline test to compare the speed of nextAnsiEscapeSequence() to the
// previously used regex based implementation.
func BenchmarkNextAnsiEscapeSequence_Regex(b *testing.B) {
	b.SetBytes(int64(len(ansiBenchmarkString)))
	for i := 0; i < b.N; i++ {
		s := ansiBenchmarkString
		for {
			a := ansiRegexReference.FindStringIndex(s)
			if len(a) == 0 {
				break
			}
			s = s[a[1]:]
		}
	}
}

func BenchmarkExtractColor(b *testing.B) {
	b.SetBytes(int64(len(ansiBenchmarkString)))
	for i := 0; i < b.N; i++ {
		extractColor(ansiBenchmarkString, nil, nil)
	}
}
