package algo

// Correctness tests for the rune-array prefilter (Step C).
//
// The prefilter may narrow the search scope but must never change a Result or
// its positions, and must never reject an item the general path would match.
// Each result is compared against the same code with the prefilter disabled.

import (
	"math/rand"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/util"
)

// foldForTest mirrors what Phase 2 does to a non-ASCII text rune: lowercase if
// uppercase, then normalize.
func foldForTest(r rune, normalize bool) rune {
	if charClassOfNonAscii(r) == charUpper {
		r = unicode.To(unicode.LowerCase, r)
	}
	if normalize {
		r = normalizeRune(r)
	}
	return r
}

// The prefilter is only safe on items whose runes cannot become ASCII. This
// pins util.MayFoldToAscii as a superset of the runes that actually can, over
// the whole Unicode range and both normalization modes. If normalize.go or the
// Go unicode tables change, this fails.
func TestMayFoldToAsciiIsSuperset(t *testing.T) {
	missed := 0
	for r := rune(utf8.RuneSelf); r <= unicode.MaxRune; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		for _, normalize := range []bool{true, false} {
			if foldForTest(r, normalize) < utf8.RuneSelf && !util.MayFoldToAscii(r) {
				if missed++; missed < 10 {
					t.Errorf("U+%04X folds to ASCII (normalize=%v) but MayFoldToAscii is false", r, normalize)
				}
			}
		}
	}
	if missed > 0 {
		t.Fatalf("%d runes fold to ASCII without being flagged", missed)
	}
}

// Scripts that must stay unflagged, otherwise the prefilter never engages for
// them and Step C buys nothing.
func TestMayFoldToAsciiExcludesMajorScripts(t *testing.T) {
	for _, s := range []struct {
		name   string
		lo, hi rune
	}{
		{"Cyrillic", 0x0400, 0x04FF}, {"Greek", 0x0370, 0x03FF}, {"Hebrew", 0x0590, 0x05FF},
		{"Arabic", 0x0600, 0x06FF}, {"Thai", 0x0E00, 0x0E7F}, {"Devanagari", 0x0900, 0x097F},
		{"CJK", 0x4E00, 0x9FFF}, {"Hangul", 0xAC00, 0xD7A3}, {"kana", 0x3040, 0x30FF},
		{"box drawing", 0x2500, 0x257F}, {"emoji", 0x1F300, 0x1FAFF},
		// These sit between the Latin blocks and were swallowed by an earlier,
		// wider grouping of foldableRanges. General Punctuation is the costly
		// one: curly quotes, en and em dashes and the ellipsis live there.
		{"Greek Extended", 0x1F00, 0x1FFF}, {"General Punctuation", 0x2000, 0x206F},
		{"Currency Symbols", 0x20A0, 0x20CF}, {"CJK Symbols", 0x3000, 0x303F},
	} {
		for r := s.lo; r <= s.hi; r++ {
			if util.MayFoldToAscii(r) {
				t.Errorf("%s U+%04X should not be flagged foldable", s.name, r)
				break
			}
		}
	}
}

// The byte-view scan must agree with the shipped reference scanners. The interesting inputs
// are runes whose low byte collides with the needle (U+0165 has low byte 'e')
// and runes sharing a lane offset, which the alignment and zero checks reject.
func TestIndexAsciiRuneMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	alphabet := []rune{'a', 'A', 'e', 'E', '/', '1', 0x0165, 0x00E9, 0x4E00, 0xD55C,
		0x1F389, 0x0065 + 0x100, 0x0041 + 0x100, 0x2F65}
	for trial := range 20000 {
		n := rng.Intn(24)
		runes := make([]rune, n)
		for i := range runes {
			runes[i] = alphabet[rng.Intn(len(alphabet))]
		}
		b := []byte{'a', 'e', 'A', 'E', '/', '1'}[rng.Intn(6)]
		cs := rng.Intn(2) == 0
		from := 0
		if n > 0 {
			from = rng.Intn(n)
		}
		if got, exp := indexAsciiRune(runes, cs, b, from), indexAsciiRuneRef(runes, cs, b, from); got != exp {
			t.Fatalf("trial %d: indexAsciiRune(%U, cs=%v, %q, %d) = %d, expected %d", trial, runes, cs, b, from, got, exp)
		}
		if got, exp := lastIndexAsciiRune(runes, cs, b, from), lastIndexAsciiRuneRef(runes, cs, b, from); got != exp {
			t.Fatalf("trial %d: lastIndexAsciiRune(%U, cs=%v, %q, %d) = %d, expected %d", trial, runes, cs, b, from, got, exp)
		}
	}
}

// Differential test: the prefilter must not change any Result or position.
// Corpora deliberately mix scripts that clear the foldable bit (CJK, Hangul,
// Cyrillic, emoji) with scripts that set it (accented Latin, fullwidth), so
// both the engaged and the bypassed path are exercised.
func TestRunePrefilterEquivalence(t *testing.T) {
	t.Cleanup(func() { disableRunePrefilter = false })
	rng := rand.New(rand.NewSource(4))
	parts := []string{
		"src", "util", "conf", "a", "e", "E", "A", "/", "_", "1", " ",
		"漢字", "한글", "мир", "ελλ", "🎉", "café", "Müller", "naïve", "ｆｕｌｌ",
		"Å", "İ", "K", "ǰ", "ﬀ",
	}
	patterns := []string{"a", "e", "conf", "src/util", "ae", "A", "E", "K", "k", "i", "//", "zz", "s l",
		// non-ASCII patterns, the Step G path
		"漢", "漢字", "한", "한글", "мир", "м", "ελλ", "🎉", "é", "ß", "Å", "İ", "ｆ",
		"漢a", "a漢", "한글/src", "🎉e"}

	slab := util.MakeSlab(100*1024, 2048)
	engaged, bypassed := 0, 0

	for trial := range 30000 {
		var sb strings.Builder
		for range 1 + rng.Intn(8) {
			sb.WriteString(parts[rng.Intn(len(parts))])
		}
		chars := util.ToChars([]byte(sb.String()))
		if chars.IsBytes() {
			continue
		}
		if chars.MayFoldToAscii() {
			bypassed++
		} else {
			engaged++
		}
		pat := patterns[rng.Intn(len(patterns))]
		cs := rng.Intn(2) == 0
		if !cs {
			pat = strings.ToLower(pat)
		}
		pattern := []rune(pat)
		norm := rng.Intn(2) == 0
		fwd := rng.Intn(2) == 0
		wp := rng.Intn(2) == 0

		disableRunePrefilter = true
		expR, expP := FuzzyMatchV2(cs, norm, fwd, &chars, pattern, wp, slab)
		disableRunePrefilter = false
		gotR, gotP := FuzzyMatchV2(cs, norm, fwd, &chars, pattern, wp, slab)

		if gotR != expR || !samePos(gotP, expP) {
			t.Fatalf("trial %d: %q pattern=%q cs=%v norm=%v fwd=%v wp=%v\n  prefilter on:  %v %v\n  prefilter off: %v %v",
				trial, sb.String(), pat, cs, norm, fwd, wp, gotR, gotP, expR, expP)
		}
	}
	disableRunePrefilter = false
	t.Logf("prefilter engaged on %d items, bypassed on %d", engaged, bypassed)
	if engaged == 0 || bypassed == 0 {
		t.Fatalf("corpus did not exercise both paths (engaged=%d bypassed=%d)", engaged, bypassed)
	}

	// Equivalence alone would still hold if the prefilter never filtered
	// anything, so confirm it both rejects and narrows.
	rejected, narrowed := 0, 0
	for range 5000 {
		var sb strings.Builder
		for range 1 + rng.Intn(8) {
			sb.WriteString(parts[rng.Intn(len(parts))])
		}
		chars := util.ToChars([]byte(sb.String()))
		if chars.IsBytes() || chars.MayFoldToAscii() {
			continue
		}
		pattern := []rune(patterns[rng.Intn(len(patterns))])
		lo, hi := asciiFuzzyIndex(&chars, pattern, false)
		switch {
		case lo < 0:
			rejected++
		case hi-lo < chars.Length():
			narrowed++
		}
	}
	t.Logf("prefilter rejected %d items, narrowed scope on %d", rejected, narrowed)
	if rejected == 0 {
		t.Fatal("prefilter never rejected an item, so equivalence proves nothing")
	}
	if narrowed == 0 {
		t.Fatal("prefilter never narrowed the scope")
	}
}

// FuzzyMatchV1 shares asciiFuzzyIndex, so it needs the same guarantee.
func TestRunePrefilterEquivalenceV1(t *testing.T) {
	t.Cleanup(func() { disableRunePrefilter = false })
	rng := rand.New(rand.NewSource(5))
	parts := []string{"src", "conf", "a", "e", "/", "漢字", "한글", "мир", "café", "Å", "🎉"}
	slab := util.MakeSlab(100*1024, 2048)
	for trial := range 20000 {
		var sb strings.Builder
		for range 1 + rng.Intn(6) {
			sb.WriteString(parts[rng.Intn(len(parts))])
		}
		chars := util.ToChars([]byte(sb.String()))
		if chars.IsBytes() {
			continue
		}
		pattern := []rune([]string{"a", "e", "conf", "src", "ae", "zz"}[rng.Intn(6)])
		cs, norm, fwd, wp := rng.Intn(2) == 0, rng.Intn(2) == 0, rng.Intn(2) == 0, rng.Intn(2) == 0

		disableRunePrefilter = true
		expR, expP := FuzzyMatchV1(cs, norm, fwd, &chars, pattern, wp, slab)
		disableRunePrefilter = false
		gotR, gotP := FuzzyMatchV1(cs, norm, fwd, &chars, pattern, wp, slab)

		if gotR != expR || !samePos(gotP, expP) {
			t.Fatalf("trial %d: %q pattern=%q\n  prefilter on:  %v %v\n  prefilter off: %v %v",
				trial, sb.String(), string(pattern), gotR, gotP, expR, expP)
		}
	}
	disableRunePrefilter = false
}

// preparePattern mirrors what pattern.go guarantees the algo functions:
// lowercased when case-insensitive, normalized when normalize is on.
func preparePattern(pat string, caseSensitive, normalize bool) []rune {
	if !caseSensitive {
		pat = strings.ToLower(pat)
	}
	r := []rune(pat)
	if normalize {
		r = NormalizeRunes(r)
	}
	return r
}

// FuzzRunePrefilter drives arbitrary rune-mode input and arbitrary patterns
// through the prefilter and through the same code with it disabled, and
// requires identical Results and positions. The existing fast-path fuzzers
// only generate byte-mode input, so they never reach this path.
func FuzzRunePrefilter(f *testing.F) {
	for _, in := range []string{
		"한글/src/util.go", "漢字/conf", "café/binutils", "мир/test", "🎉/a",
		"ＡＢＣ.txt", "Ångström", "ǰ/ß/İ", "a漢b한c", "Āā",
	} {
		for _, p := range []string{"a", "conf", "漢", "한글", "мир", "ß", "É", "a漢"} {
			f.Add(in, p)
		}
	}
	slab := util.MakeSlab(200*1024, 4096)
	f.Fuzz(func(t *testing.T, input, pat string) {
		if len(input) > 512 || len(pat) == 0 || len(pat) > 32 {
			return
		}
		chars := util.ToChars([]byte(input))
		if chars.IsBytes() {
			return // byte mode is the existing fuzzers' territory
		}
		for _, cs := range []bool{false, true} {
			for _, norm := range []bool{false, true} {
				p := preparePattern(pat, cs, norm)
				if len(p) == 0 {
					continue
				}
				for _, fwd := range []bool{true, false} {
					for _, wp := range []bool{false, true} {
						for _, fn := range []Algo{FuzzyMatchV2, FuzzyMatchV1, ExactMatchNaive} {
							disableRunePrefilter = true
							expR, expP := fn(cs, norm, fwd, &chars, p, wp, slab)
							disableRunePrefilter = false
							gotR, gotP := fn(cs, norm, fwd, &chars, p, wp, slab)
							if gotR != expR || !samePos(gotP, expP) {
								t.Fatalf("input=%q pattern=%q cs=%v norm=%v fwd=%v wp=%v\n prefilter on:  %v %v\n prefilter off: %v %v",
									input, pat, cs, norm, fwd, wp, gotR, gotP, expR, expP)
							}
						}
					}
				}
			}
		}
	})
}

// RunesToChars can produce rune-mode Chars holding zero runes, which sends a
// nil pointer through unsafe.SliceData in runeBytes. ToChars cannot produce
// this (an empty input is byte mode), so it needs its own test.
func TestEmptyRuneModeChars(t *testing.T) {
	t.Cleanup(func() { disableRunePrefilter = false })
	slab := util.MakeSlab(100*1024, 2048)
	for _, runes := range [][]rune{{}, nil, {'a'}, {0x4E00}} {
		chars := util.RunesToChars(runes)
		if chars.IsBytes() {
			continue
		}
		for _, pat := range []string{"a", "漢", "ab"} {
			p := []rune(pat)
			for _, fn := range []Algo{FuzzyMatchV2, FuzzyMatchV1, ExactMatchNaive,
				PrefixMatch, SuffixMatch, EqualMatch} {
				disableRunePrefilter = true
				expR, expP := fn(false, true, true, &chars, p, true, slab)
				disableRunePrefilter = false
				gotR, gotP := fn(false, true, true, &chars, p, true, slab)
				if gotR != expR || !samePos(gotP, expP) {
					t.Errorf("runes=%U pat=%q: prefilter on %v %v, off %v %v", runes, pat, gotR, gotP, expR, expP)
				}
			}
		}
	}
}

// MayFoldToAscii subtracts before bounds-checking, so a rune below the range
// (including a negative one, which utf8 decoding never produces but callers
// could construct) must not wrap into a false positive.
func TestMayFoldToAsciiOutOfRange(t *testing.T) {
	for _, r := range []rune{-1, -0x10000, 0, 'a', 0x7F, 0xBF, 0xFF62, unicode.MaxRune, unicode.MaxRune + 1} {
		if util.MayFoldToAscii(r) {
			t.Errorf("MayFoldToAscii(%d) = true, expected false", r)
		}
	}
}
