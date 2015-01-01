package fzf

import (
	"strings"
	"testing"
)

func assertMatch(t *testing.T, fun func(bool, *string, []rune) (int, int), caseSensitive bool, input string, pattern string, sidx int, eidx int) {
	if !caseSensitive {
		pattern = strings.ToLower(pattern)
	}
	s, e := fun(caseSensitive, &input, []rune(pattern))
	if s != sidx {
		t.Errorf("Invalid start index: %d (expected: %d, %s / %s)", s, sidx, input, pattern)
	}
	if e != eidx {
		t.Errorf("Invalid end index: %d (expected: %d, %s / %s)", e, eidx, input, pattern)
	}
}

func TestFuzzyMatch(t *testing.T) {
	assertMatch(t, FuzzyMatch, false, "fooBarbaz", "oBZ", 2, 9)
	assertMatch(t, FuzzyMatch, true, "fooBarbaz", "oBZ", -1, -1)
	assertMatch(t, FuzzyMatch, true, "fooBarbaz", "oBz", 2, 9)
	assertMatch(t, FuzzyMatch, true, "fooBarbaz", "fooBarbazz", -1, -1)
}

func TestExactMatchNaive(t *testing.T) {
	assertMatch(t, ExactMatchNaive, false, "fooBarbaz", "oBA", 2, 5)
	assertMatch(t, ExactMatchNaive, true, "fooBarbaz", "oBA", -1, -1)
	assertMatch(t, ExactMatchNaive, true, "fooBarbaz", "fooBarbazz", -1, -1)
}

func TestPrefixMatch(t *testing.T) {
	assertMatch(t, PrefixMatch, false, "fooBarbaz", "Foo", 0, 3)
	assertMatch(t, PrefixMatch, true, "fooBarbaz", "Foo", -1, -1)
	assertMatch(t, PrefixMatch, false, "fooBarbaz", "baz", -1, -1)
}

func TestSuffixMatch(t *testing.T) {
	assertMatch(t, SuffixMatch, false, "fooBarbaz", "Foo", -1, -1)
	assertMatch(t, SuffixMatch, false, "fooBarbaz", "baz", 6, 9)
	assertMatch(t, SuffixMatch, true, "fooBarbaz", "Baz", -1, -1)
}
