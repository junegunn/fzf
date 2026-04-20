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
var ansiRegexReference = regexp.MustCompile("(?:\x1b[\\[()][0-9;:]*[a-zA-Z@]|\x1b][0-9][;:][[:print:]]+(?:\x1b\\\\|\x07)|\x1b.|[\x0e\x0f]|.\x08|\n)")

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
			for i := range got {
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
		for i := range codePoints {
			var r rune
			for range 1000 {
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
	for range 100_000 {
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

	state = nil
	var color24 tui.Color = (1 << 24) + (180 << 16) + (190 << 8) + 254
	src = "\x1b[1mhello \x1b[22;1;38:2:180:190:254mworld"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 2 {
			t.Fail()
		}
		if state.fg != color24 || state.attr != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 6, -1, -1, true)
		assert((*offsets)[1], 6, 11, color24, -1, true)
	})

	src = "\x1b]133;A\x1b\\hello \x1b]133;C\x1b\\world"
	check(func(offsets *[]ansiOffset, state *ansiState) {
		if len(*offsets) != 1 {
			t.Fail()
		}
		assert((*offsets)[0], 0, 11, color24, -1, true)
	})
}

func TestAnsiCodeStringConversion(t *testing.T) {
	assert := func(code string, prevState *ansiState, expected string) {
		state := interpretCode(code, prevState)
		if expected != state.ToString() {
			t.Errorf("expected: %s, actual: %s",
				strings.ReplaceAll(expected, "\x1b[", "\\x1b["),
				strings.ReplaceAll(state.ToString(), "\x1b[", "\\x1b["))
		}
	}
	assert("\x1b[m", nil, "")
	assert("\x1b[m", &ansiState{attr: tui.Blink, ul: -1, lbg: -1}, "")
	assert("\x1b[0m", &ansiState{fg: 4, bg: 4, ul: -1, lbg: -1}, "")
	assert("\x1b[;m", &ansiState{fg: 4, bg: 4, ul: -1, lbg: -1}, "")
	assert("\x1b[;;m", &ansiState{fg: 4, bg: 4, ul: -1, lbg: -1}, "")

	assert("\x1b[31m", nil, "\x1b[31;49m")
	assert("\x1b[41m", nil, "\x1b[39;41m")

	assert("\x1b[92m", nil, "\x1b[92;49m")
	assert("\x1b[102m", nil, "\x1b[39;102m")

	assert("\x1b[31m", &ansiState{fg: 4, bg: 4, ul: -1, lbg: -1}, "\x1b[31;44m")
	assert("\x1b[1;2;31m", &ansiState{fg: 2, bg: -1, ul: -1, attr: tui.Reverse, lbg: -1}, "\x1b[1;2;7;31;49m")
	assert("\x1b[38;5;100;48;5;200m", nil, "\x1b[38;5;100;48;5;200m")
	assert("\x1b[38:5:100:48:5:200m", nil, "\x1b[38;5;100;48;5;200m")
	assert("\x1b[48;5;100;38;5;200m", nil, "\x1b[38;5;200;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;1m", nil, "\x1b[1;38;2;10;20;30;48;5;100m")
	assert("\x1b[48;5;100;38;2;10;20;30;7m",
		&ansiState{attr: tui.Dim | tui.Italic, fg: 1, bg: 1, ul: -1},
		"\x1b[2;3;7;38;2;10;20;30;48;5;100m")

	// Underline styles
	assert("\x1b[4:3m", nil, "\x1b[4:3;39;49m")
	assert("\x1b[4:2m", nil, "\x1b[4:2;39;49m")
	assert("\x1b[4:4m", nil, "\x1b[4:4;39;49m")
	assert("\x1b[4:5m", nil, "\x1b[4:5;39;49m")
	assert("\x1b[4:1m", nil, "\x1b[4;39;49m")

	// Underline color (256-color)
	assert("\x1b[4;58;5;100m", nil, "\x1b[4;39;49;58;5;100m")
	// Underline color (24-bit)
	assert("\x1b[4;58;2;255;0;128m", nil, "\x1b[4;39;49;58;2;255;0;128m")
	// Curly underline + underline color
	assert("\x1b[4:3;58;2;255;0;0m", nil, "\x1b[4:3;39;49;58;2;255;0;0m")
	// SGR 59 resets underline color
	assert("\x1b[59m", &ansiState{fg: 1, bg: -1, ul: 100, lbg: -1}, "\x1b[31;49m")
}

func TestParseAnsiCode(t *testing.T) {
	tests := []struct {
		In  string
		Exp string
		N   int
		Sep byte
	}{
		{"123", "", 123, 0},
		{"1a", "", -1, 0},
		{"1a;12", "12", -1, ';'},
		{"12;a", "a", 12, ';'},
		{"-2", "", -1, 0},
		// Colon sub-parameters: earliest separator wins (@shtse8)
		{"4:3", "3", 4, ':'},
		{"4:3;31", "3;31", 4, ':'},
		{"38:2:255:0:0", "2:255:0:0", 38, ':'},
		{"58:5:200", "5:200", 58, ':'},
		// Semicolon before colon
		{"4;38:2:0:0:0", "38:2:0:0:0", 4, ';'},
	}
	for _, x := range tests {
		n, sep, s := parseAnsiCode(x.In)
		if n != x.N || s != x.Exp || sep != x.Sep {
			t.Fatalf("%q: got: (%d %q %q) want: (%d %q %q)", x.In, n, s, string(sep), x.N, x.Exp, string(x.Sep))
		}
	}
}

// Test cases adapted from @shtse8 (PR #4678)
func TestInterpretCodeUnderlineStyles(t *testing.T) {
	// 4:0 = no underline
	state := interpretCode("\x1b[4:0m", nil)
	if state.attr&tui.Underline != 0 {
		t.Error("4:0 should not set underline")
	}

	// 4:1 = single underline
	state = interpretCode("\x1b[4:1m", nil)
	if state.attr&tui.Underline == 0 {
		t.Error("4:1 should set underline")
	}

	// 4:3 = curly underline
	state = interpretCode("\x1b[4:3m", nil)
	if state.attr&tui.Underline == 0 {
		t.Error("4:3 should set underline")
	}
	if state.attr.UnderlineStyle() != tui.UlStyleCurly {
		t.Error("4:3 should set curly underline style")
	}

	// 4:3 should NOT set italic (3 is a sub-param, not SGR 3)
	if state.attr&tui.Italic != 0 {
		t.Error("4:3 should not set italic")
	}

	// 4:2;31 = double underline + red fg
	state = interpretCode("\x1b[4:2;31m", nil)
	if state.attr&tui.Underline == 0 {
		t.Error("4:2;31 should set underline")
	}
	if state.fg != 1 {
		t.Errorf("4:2;31 should set fg to red (1), got %d", state.fg)
	}
	if state.attr&tui.Dim != 0 {
		t.Error("4:2;31 should not set dim")
	}

	// Plain 4 still works
	state = interpretCode("\x1b[4m", nil)
	if state.attr&tui.Underline == 0 {
		t.Error("4 should set underline")
	}

	// 4;2 (semicolon) = underline + dim
	state = interpretCode("\x1b[4;2m", nil)
	if state.attr&tui.Underline == 0 {
		t.Error("4;2 should set underline")
	}
	if state.attr&tui.Dim == 0 {
		t.Error("4;2 should set dim")
	}
}

// Test cases adapted from @shtse8 (PR #4678)
func TestInterpretCodeUnderlineColor(t *testing.T) {
	// 58:2:R:G:B should not affect fg or bg
	state := interpretCode("\x1b[58:2:255:0:0m", nil)
	if state.fg != -1 || state.bg != -1 {
		t.Errorf("58:2:R:G:B should not affect fg/bg, got fg=%d bg=%d", state.fg, state.bg)
	}

	// 58:5:200 should not affect fg or bg
	state = interpretCode("\x1b[58:5:200m", nil)
	if state.fg != -1 || state.bg != -1 {
		t.Errorf("58:5:N should not affect fg/bg, got fg=%d bg=%d", state.fg, state.bg)
	}

	// 58:2:R:G:B combined with 38:2:R:G:B should only set fg
	state = interpretCode("\x1b[58:2:255:0:0;38:2:0:255:0m", nil)
	expectedFg := tui.Color(1<<24 | 0<<16 | 255<<8 | 0)
	if state.fg != expectedFg {
		t.Errorf("expected fg=%d, got %d", expectedFg, state.fg)
	}
	if state.bg != -1 {
		t.Errorf("bg should be -1, got %d", state.bg)
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
