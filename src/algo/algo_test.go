package algo

import (
	"strings"
	"testing"
)

func assertMatch(t *testing.T, fun func(bool, bool, []rune, []rune) (int, int), caseSensitive bool, forward bool, input string, pattern string, sidx int, eidx int) {
	if !caseSensitive {
		pattern = strings.ToLower(pattern)
	}
	s, e := fun(caseSensitive, forward, []rune(input), []rune(pattern))
	if s != sidx {
		t.Errorf("Invalid start index: %d (expected: %d, %s / %s)", s, sidx, input, pattern)
	}
	if e != eidx {
		t.Errorf("Invalid end index: %d (expected: %d, %s / %s)", e, eidx, input, pattern)
	}
}

func TestFuzzyMatch(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, true, "fooBarbaz", "oBZ", 2, 9)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "oBZ", -1, -1)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "oBz", 2, 9)
	assertMatch(t, FuzzyMatch, true, true, "fooBarbaz", "fooBarbazz", -1, -1)
}

func TestFuzzyMatchBackward(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, true, "foobar fb", "fb", 0, 4)
	assertMatch(t, FuzzyMatch, false, false, "foobar fb", "fb", 7, 9)
}

func TestExactMatchNaive(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, ExactMatchNaive, false, dir, "fooBarbaz", "oBA", 2, 5)
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "oBA", -1, -1)
		assertMatch(t, ExactMatchNaive, true, dir, "fooBarbaz", "fooBarbazz", -1, -1)
	}
}

func TestExactMatchNaiveBackward(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, true, "foobar foob", "oo", 1, 3)
	assertMatch(t, FuzzyMatch, false, false, "foobar foob", "oo", 8, 10)
}

func TestPrefixMatch(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, PrefixMatch, false, dir, "fooBarbaz", "Foo", 0, 3)
		assertMatch(t, PrefixMatch, true, dir, "fooBarbaz", "Foo", -1, -1)
		assertMatch(t, PrefixMatch, false, dir, "fooBarbaz", "baz", -1, -1)
	}
}

func TestSuffixMatch(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "Foo", -1, -1)
		assertMatch(t, SuffixMatch, false, dir, "fooBarbaz", "baz", 6, 9)
		assertMatch(t, SuffixMatch, true, dir, "fooBarbaz", "Baz", -1, -1)
	}
}

func TestEmptyPattern(t *testing.T) {
	for _, dir := range []bool{true, false} {
		assertMatch(t, FuzzyMatch, true, dir, "foobar", "", 0, 0)
		assertMatch(t, ExactMatchNaive, true, dir, "foobar", "", 0, 0)
		assertMatch(t, PrefixMatch, true, dir, "foobar", "", 0, 0)
		assertMatch(t, SuffixMatch, true, dir, "foobar", "", 6, 6)
	}
}
