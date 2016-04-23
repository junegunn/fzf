package algo

import (
	"strings"
	"testing"
)

func assertMatch(t *testing.T, fun func(bool, bool, []rune, []rune) Result, caseSensitive, forward bool, input, pattern string, sidx int32, eidx int32, bonus int32) {
	if !caseSensitive {
		pattern = strings.ToLower(pattern)
	}
	res := fun(caseSensitive, forward, []rune(input), []rune(pattern))
	if res.Start != sidx {
		t.Errorf("Invalid start index: %d (expected: %d, %s / %s)", res.Start, sidx, input, pattern)
	}
	if res.End != eidx {
		t.Errorf("Invalid end index: %d (expected: %d, %s / %s)", res.End, eidx, input, pattern)
	}
	if res.Bonus != bonus {
		t.Errorf("Invalid bonus: %d (expected: %d, %s / %s)", res.Bonus, bonus, input, pattern)
	}
}

func TestFuzzyMatch(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, true, "fooBarbaz", "oBZ", 2, 9, 2)
	assertMatch(t, FuzzyMatch, false, true, "foo bar baz", "fbb", 0, 9, 8)
	assertMatch(t, FuzzyMatch, false, true, "/AutomatorDocument.icns", "rdoc", 9, 13, 4)
	assertMatch(t, FuzzyMatch, false, true, "/man1/zshcompctl.1", "zshc", 6, 10, 7)
	assertMatch(t, FuzzyMatch, false, true, "/.oh-my-zsh/cache", "zshc", 8, 13, 8)
	assertMatch(t, FuzzyMatch, false, true, "ab0123 456", "12356", 3, 10, 3)
	assertMatch(t, FuzzyMatch, false, true, "abc123 456", "12356", 3, 10, 5)

	assertMatch(t, FuzzyMatch, false, true, "foo/bar/baz", "fbb", 0, 9, 8)
	assertMatch(t, FuzzyMatch, false, true, "fooBarBaz", "fbb", 0, 7, 6)
	assertMatch(t, FuzzyMatch, false, true, "foo barbaz", "fbb", 0, 8, 6)
	assertMatch(t, FuzzyMatch, false, true, "fooBar Baz", "foob", 0, 4, 8)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "oBZ", -1, -1, 0)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "oBz", 2, 9, 2)
	assertMatch(t, FuzzyMatch, true, true, "Foo Bar Baz", "fbb", -1, -1, 0)
	assertMatch(t, FuzzyMatch, true, true, "Foo/Bar/Baz", "FBB", 0, 9, 8)
	assertMatch(t, FuzzyMatch, true, true, "FooBarBaz", "FBB", 0, 7, 6)
	assertMatch(t, FuzzyMatch, true, true, "foo BarBaz", "fBB", 0, 8, 7)
	assertMatch(t, FuzzyMatch, true, true, "FooBar Baz", "FooB", 0, 4, 8)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "fooBarbazz", -1, -1, 0)
}

func TestFuzzyMatchBackward(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, true, "foobar fb", "fb", 0, 4, 4)
	assertMatch(t, FuzzyMatch, false, false, "foobar fb", "fb", 7, 9, 5)
}

func TestExactMatchNaive(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, ExactMatchNaive, false, dir, "fooBarbaz", "oBA", 2, 5, 3)
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "oBA", -1, -1, 0)
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "fooBarbazz", -1, -1, 0)

		assertMatch(t, ExactMatchNaive, false, dir, "/AutomatorDocument.icns", "rdoc", 9, 13, 4)
		assertMatch(t, ExactMatchNaive, false, dir, "/man1/zshcompctl.1", "zshc", 6, 10, 7)
		assertMatch(t, ExactMatchNaive, false, dir, "/.oh-my-zsh/cache", "zsh/c", 8, 13, 10)
	}
}

func TestExactMatchNaiveBackward(t *testing.T) {
	assertMatch(t, ExactMatchNaive, false, true, "foobar foob", "oo", 1, 3, 1)
	assertMatch(t, ExactMatchNaive, false, false, "foobar foob", "oo", 8, 10, 1)
}

func TestPrefixMatch(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, PrefixMatch, true, dir, "fooBarbaz", "Foo", -1, -1, 0)
		assertMatch(t, PrefixMatch, false, dir, "fooBarBaz", "baz", -1, -1, 0)
		assertMatch(t, PrefixMatch, false, dir, "fooBarbaz", "Foo", 0, 3, 6)
		assertMatch(t, PrefixMatch, false, dir, "foOBarBaZ", "foo", 0, 3, 7)
		assertMatch(t, PrefixMatch, false, dir, "f-oBarbaz", "f-o", 0, 3, 8)
	}
}

func TestSuffixMatch(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "Foo", -1, -1, 0)
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "baz", 6, 9, 2)
		assertMatch(t, SuffixMatch, false, dir, "fooBarBaZ", "baz", 6, 9, 5)
		assertMatch(t, SuffixMatch, true, dir, "fooBarbaz", "Baz", -1, -1, 0)
	}
}

func TestEmptyPattern(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, FuzzyMatch, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, ExactMatchNaive, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, PrefixMatch, true, dir, "foobar", "", 0, 0, 0)
		assertMatch(t, SuffixMatch, true, dir, "foobar", "", 6, 6, 0)
	}
}
