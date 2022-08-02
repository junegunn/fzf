package algo

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func assertMatch(t *testing.T, fun Algo, caseSensitive, forward bool, input, pattern string, sidx int, eidx int, score int) {
	assertMatch2(t, fun, caseSensitive, false, forward, input, pattern, sidx, eidx, score)
}

func assertMatch2(t *testing.T, fun Algo, caseSensitive, normalize, forward bool, input, pattern string, sidx int, eidx int, score int) {
	if !caseSensitive {
		pattern = strings.ToLower(pattern)
	}
	chars := util.ToChars([]byte(input))
	res, pos := fun(caseSensitive, normalize, forward, &chars, []rune(pattern), true, nil)
	var start, end int
	if pos == nil || len(*pos) == 0 {
		start = res.Start
		end = res.End
	} else {
		sort.Ints(*pos)
		start = (*pos)[0]
		end = (*pos)[len(*pos)-1] + 1
	}
	if start != sidx {
		t.Errorf("Invalid start index: %d (expected: %d, %s / %s)", start, sidx, input, pattern)
	}
	if end != eidx {
		t.Errorf("Invalid end index: %d (expected: %d, %s / %s)", end, eidx, input, pattern)
	}
	if res.Score != score {
		t.Errorf("Invalid score: %d (expected: %d, %s / %s)", res.Score, score, input, pattern)
	}
}

func TestFuzzyMatch(t *testing.T) {
	for _, fn := range []Algo{FuzzyMatchV1, FuzzyMatchV2} {
		for _, forward := range []bool{true, false} {
			assertMatch(t, fn, false, forward, "fooBarbaz1", "oBZ", 2, 9,
				scoreMatch*3+bonusCamel123+scoreGapStart+scoreGapExtension*3)
			assertMatch(t, fn, false, forward, "foo bar baz", "fbb", 0, 9,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+
					bonusBoundaryWhite*2+2*scoreGapStart+4*scoreGapExtension)
			assertMatch(t, fn, false, forward, "/AutomatorDocument.icns", "rdoc", 9, 13,
				scoreMatch*4+bonusCamel123+bonusConsecutive*2)
			assertMatch(t, fn, false, forward, "/man1/zshcompctl.1", "zshc", 6, 10,
				scoreMatch*4+bonusBoundaryDelimiter*bonusFirstCharMultiplier+bonusBoundaryDelimiter*3)
			assertMatch(t, fn, false, forward, "/.oh-my-zsh/cache", "zshc", 8, 13,
				scoreMatch*4+bonusBoundary*bonusFirstCharMultiplier+bonusBoundary*2+scoreGapStart+bonusBoundaryDelimiter)
			assertMatch(t, fn, false, forward, "ab0123 456", "12356", 3, 10,
				scoreMatch*5+bonusConsecutive*3+scoreGapStart+scoreGapExtension)
			assertMatch(t, fn, false, forward, "abc123 456", "12356", 3, 10,
				scoreMatch*5+bonusCamel123*bonusFirstCharMultiplier+bonusCamel123*2+bonusConsecutive+scoreGapStart+scoreGapExtension)
			assertMatch(t, fn, false, forward, "foo/bar/baz", "fbb", 0, 9,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+
					bonusBoundaryDelimiter*2+2*scoreGapStart+4*scoreGapExtension)
			assertMatch(t, fn, false, forward, "fooBarBaz", "fbb", 0, 7,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+
					bonusCamel123*2+2*scoreGapStart+2*scoreGapExtension)
			assertMatch(t, fn, false, forward, "foo barbaz", "fbb", 0, 8,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusBoundaryWhite+
					scoreGapStart*2+scoreGapExtension*3)
			assertMatch(t, fn, false, forward, "fooBar Baz", "foob", 0, 4,
				scoreMatch*4+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusBoundaryWhite*3)
			assertMatch(t, fn, false, forward, "xFoo-Bar Baz", "foo-b", 1, 6,
				scoreMatch*5+bonusCamel123*bonusFirstCharMultiplier+bonusCamel123*2+
					bonusNonWord+bonusBoundary)

			assertMatch(t, fn, true, forward, "fooBarbaz", "oBz", 2, 9,
				scoreMatch*3+bonusCamel123+scoreGapStart+scoreGapExtension*3)
			assertMatch(t, fn, true, forward, "Foo/Bar/Baz", "FBB", 0, 9,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusBoundaryDelimiter*2+
					scoreGapStart*2+scoreGapExtension*4)
			assertMatch(t, fn, true, forward, "FooBarBaz", "FBB", 0, 7,
				scoreMatch*3+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusCamel123*2+
					scoreGapStart*2+scoreGapExtension*2)
			assertMatch(t, fn, true, forward, "FooBar Baz", "FooB", 0, 4,
				scoreMatch*4+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusBoundaryWhite*2+
					util.Max(bonusCamel123, bonusBoundaryWhite))

			// Consecutive bonus updated
			assertMatch(t, fn, true, forward, "foo-bar", "o-ba", 2, 6,
				scoreMatch*4+bonusBoundary*3)

			// Non-match
			assertMatch(t, fn, true, forward, "fooBarbaz", "oBZ", -1, -1, 0)
			assertMatch(t, fn, true, forward, "Foo Bar Baz", "fbb", -1, -1, 0)
			assertMatch(t, fn, true, forward, "fooBarbaz", "fooBarbazz", -1, -1, 0)
		}
	}
}

func TestFuzzyMatchBackward(t *testing.T) {
	assertMatch(t, FuzzyMatchV1, false, true, "foobar fb", "fb", 0, 4,
		scoreMatch*2+bonusBoundaryWhite*bonusFirstCharMultiplier+
			scoreGapStart+scoreGapExtension)
	assertMatch(t, FuzzyMatchV1, false, false, "foobar fb", "fb", 7, 9,
		scoreMatch*2+bonusBoundaryWhite*bonusFirstCharMultiplier+bonusBoundaryWhite)
}

func TestExactMatchNaive(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "oBA", -1, -1, 0)
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "fooBarbazz", -1, -1, 0)

		assertMatch(t, ExactMatchNaive, false, dir, "fooBarbaz", "oBA", 2, 5,
			scoreMatch*3+bonusCamel123+bonusConsecutive)
		assertMatch(t, ExactMatchNaive, false, dir, "/AutomatorDocument.icns", "rdoc", 9, 13,
			scoreMatch*4+bonusCamel123+bonusConsecutive*2)
		assertMatch(t, ExactMatchNaive, false, dir, "/man1/zshcompctl.1", "zshc", 6, 10,
			scoreMatch*4+bonusBoundaryDelimiter*(bonusFirstCharMultiplier+3))
		assertMatch(t, ExactMatchNaive, false, dir, "/.oh-my-zsh/cache", "zsh/c", 8, 13,
			scoreMatch*5+bonusBoundary*(bonusFirstCharMultiplier+3)+bonusBoundaryDelimiter)
	}
}

func TestExactMatchNaiveBackward(t *testing.T) {
	assertMatch(t, ExactMatchNaive, false, true, "foobar foob", "oo", 1, 3,
		scoreMatch*2+bonusConsecutive)
	assertMatch(t, ExactMatchNaive, false, false, "foobar foob", "oo", 8, 10,
		scoreMatch*2+bonusConsecutive)
}

func TestPrefixMatch(t *testing.T) {
	score := scoreMatch*3 + bonusBoundaryWhite*bonusFirstCharMultiplier + bonusBoundaryWhite*2

	for _, dir := range []bool{true, false} {
		assertMatch(t, PrefixMatch, true, dir, "fooBarbaz", "Foo", -1, -1, 0)
		assertMatch(t, PrefixMatch, false, dir, "fooBarBaz", "baz", -1, -1, 0)
		assertMatch(t, PrefixMatch, false, dir, "fooBarbaz", "Foo", 0, 3, score)
		assertMatch(t, PrefixMatch, false, dir, "foOBarBaZ", "foo", 0, 3, score)
		assertMatch(t, PrefixMatch, false, dir, "f-oBarbaz", "f-o", 0, 3, score)

		assertMatch(t, PrefixMatch, false, dir, " fooBar", "foo", 1, 4, score)
		assertMatch(t, PrefixMatch, false, dir, " fooBar", " fo", 0, 3, score)
		assertMatch(t, PrefixMatch, false, dir, "     fo", "foo", -1, -1, 0)
	}
}

func TestSuffixMatch(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, SuffixMatch, true, dir, "fooBarbaz", "Baz", -1, -1, 0)
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "Foo", -1, -1, 0)

		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "baz", 6, 9,
			scoreMatch*3+bonusConsecutive*2)
		assertMatch(t, SuffixMatch, false, dir, "fooBarBaZ", "baz", 6, 9,
			(scoreMatch+bonusCamel123)*3+bonusCamel123*(bonusFirstCharMultiplier-1))

		// Strip trailing white space from the string
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz ", "baz", 6, 9,
			scoreMatch*3+bonusConsecutive*2)

		// Only when the pattern doesn't end with a space
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz ", "baz ", 6, 10,
			scoreMatch*4+bonusConsecutive*2+bonusBoundaryWhite)
	}
}

func TestEmptyPattern(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, FuzzyMatchV1, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, FuzzyMatchV2, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, ExactMatchNaive, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, PrefixMatch, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, SuffixMatch, true, dir, "foobar", "", 6, 6, 0)
	}
}

func TestNormalize(t *testing.T) {
	caseSensitive := false
	normalize := true
	forward := true
	test := func(input, pattern string, sidx, eidx, score int, funs ...Algo) {
		for _, fun := range funs {
			assertMatch2(t, fun, caseSensitive, normalize, forward,
				input, pattern, sidx, eidx, score)
		}
	}
	test("Só Danço Samba", "So", 0, 2, 62, FuzzyMatchV1, FuzzyMatchV2, PrefixMatch, ExactMatchNaive)
	test("Só Danço Samba", "sodc", 0, 7, 97, FuzzyMatchV1, FuzzyMatchV2)
	test("Danço", "danco", 0, 5, 140, FuzzyMatchV1, FuzzyMatchV2, PrefixMatch, SuffixMatch, ExactMatchNaive, EqualMatch)
}

func TestLongString(t *testing.T) {
	bytes := make([]byte, math.MaxUint16*2)
	for i := range bytes {
		bytes[i] = 'x'
	}
	bytes[math.MaxUint16] = 'z'
	assertMatch(t, FuzzyMatchV2, true, true, string(bytes), "zx", math.MaxUint16, math.MaxUint16+2, scoreMatch*2+bonusConsecutive)
}
