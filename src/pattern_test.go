package fzf

import (
	"reflect"
	"testing"

	"github.com/junegunn/fzf/src/algo"
)

func TestParseTermsExtended(t *testing.T) {
	terms := parseTerms(true, CaseSmart,
		"| aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$ | ^iii$ ^xxx | 'yyy | | zzz$ | !ZZZ |")
	if len(terms) != 9 ||
		terms[0][0].typ != termFuzzy || terms[0][0].inv ||
		terms[1][0].typ != termExact || terms[1][0].inv ||
		terms[2][0].typ != termPrefix || terms[2][0].inv ||
		terms[3][0].typ != termSuffix || terms[3][0].inv ||
		terms[4][0].typ != termFuzzy || !terms[4][0].inv ||
		terms[5][0].typ != termExact || !terms[5][0].inv ||
		terms[6][0].typ != termPrefix || !terms[6][0].inv ||
		terms[7][0].typ != termSuffix || !terms[7][0].inv ||
		terms[7][1].typ != termEqual || terms[7][1].inv ||
		terms[8][0].typ != termPrefix || terms[8][0].inv ||
		terms[8][1].typ != termExact || terms[8][1].inv ||
		terms[8][2].typ != termSuffix || terms[8][2].inv ||
		terms[8][3].typ != termFuzzy || !terms[8][3].inv {
		t.Errorf("%s", terms)
	}
	for idx, termSet := range terms[:8] {
		term := termSet[0]
		if len(term.text) != 3 {
			t.Errorf("%s", term)
		}
		if idx > 0 && len(term.origText) != 4+idx/5 {
			t.Errorf("%s", term)
		}
	}
	for _, term := range terms[8] {
		if len(term.origText) != 4 {
			t.Errorf("%s", term)
		}
	}
}

func TestParseTermsExtendedExact(t *testing.T) {
	terms := parseTerms(false, CaseSmart,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0][0].typ != termExact || terms[0][0].inv || len(terms[0][0].text) != 3 ||
		terms[1][0].typ != termFuzzy || terms[1][0].inv || len(terms[1][0].text) != 3 ||
		terms[2][0].typ != termPrefix || terms[2][0].inv || len(terms[2][0].text) != 3 ||
		terms[3][0].typ != termSuffix || terms[3][0].inv || len(terms[3][0].text) != 3 ||
		terms[4][0].typ != termExact || !terms[4][0].inv || len(terms[4][0].text) != 3 ||
		terms[5][0].typ != termFuzzy || !terms[5][0].inv || len(terms[5][0].text) != 3 ||
		terms[6][0].typ != termPrefix || !terms[6][0].inv || len(terms[6][0].text) != 3 ||
		terms[7][0].typ != termSuffix || !terms[7][0].inv || len(terms[7][0].text) != 3 {
		t.Errorf("%s", terms)
	}
}

func TestParseTermsEmpty(t *testing.T) {
	terms := parseTerms(true, CaseSmart, "' $ ^ !' !^ !$")
	if len(terms) != 0 {
		t.Errorf("%s", terms)
	}
}

func TestExact(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pattern := BuildPattern(true, true, CaseSmart, true,
		[]Range{}, Delimiter{}, []rune("'abc"))
	res := algo.ExactMatchNaive(
		pattern.caseSensitive, pattern.forward, []rune("aabbcc abc"), pattern.termSets[0][0].text)
	if res.Start != 7 || res.End != 10 {
		t.Errorf("%s / %d / %d", pattern.termSets, res.Start, res.End)
	}
}

func TestEqual(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pattern := BuildPattern(true, true, CaseSmart, true, []Range{}, Delimiter{}, []rune("^AbC$"))

	match := func(str string, sidxExpected int32, eidxExpected int32) {
		res := algo.EqualMatch(
			pattern.caseSensitive, pattern.forward, []rune(str), pattern.termSets[0][0].text)
		if res.Start != sidxExpected || res.End != eidxExpected {
			t.Errorf("%s / %d / %d", pattern.termSets, res.Start, res.End)
		}
	}
	match("ABC", -1, -1)
	match("AbC", 0, 3)
}

func TestCaseSensitivity(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pat1 := BuildPattern(true, false, CaseSmart, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat2 := BuildPattern(true, false, CaseSmart, true, []Range{}, Delimiter{}, []rune("Abc"))
	clearPatternCache()
	pat3 := BuildPattern(true, false, CaseIgnore, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat4 := BuildPattern(true, false, CaseIgnore, true, []Range{}, Delimiter{}, []rune("Abc"))
	clearPatternCache()
	pat5 := BuildPattern(true, false, CaseRespect, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat6 := BuildPattern(true, false, CaseRespect, true, []Range{}, Delimiter{}, []rune("Abc"))

	if string(pat1.text) != "abc" || pat1.caseSensitive != false ||
		string(pat2.text) != "Abc" || pat2.caseSensitive != true ||
		string(pat3.text) != "abc" || pat3.caseSensitive != false ||
		string(pat4.text) != "abc" || pat4.caseSensitive != false ||
		string(pat5.text) != "abc" || pat5.caseSensitive != true ||
		string(pat6.text) != "Abc" || pat6.caseSensitive != true {
		t.Error("Invalid case conversion")
	}
}

func TestOrigTextAndTransformed(t *testing.T) {
	pattern := BuildPattern(true, true, CaseSmart, true, []Range{}, Delimiter{}, []rune("jg"))
	tokens := Tokenize([]rune("junegunn"), Delimiter{})
	trans := Transform(tokens, []Range{Range{1, 1}})

	origRunes := []rune("junegunn.choi")
	for _, extended := range []bool{false, true} {
		chunk := Chunk{
			&Item{
				text:        []rune("junegunn"),
				origText:    &origRunes,
				transformed: trans},
		}
		pattern.extended = extended
		matches := pattern.matchChunk(&chunk)
		if string(matches[0].text) != "junegunn" || string(*matches[0].origText) != "junegunn.choi" ||
			matches[0].offsets[0][0] != 0 || matches[0].offsets[0][1] != 5 ||
			!reflect.DeepEqual(matches[0].transformed, trans) {
			t.Error("Invalid match result", matches)
		}
	}
}

func TestCacheKey(t *testing.T) {
	test := func(extended bool, patStr string, expected string, cacheable bool) {
		pat := BuildPattern(true, extended, CaseSmart, true, []Range{}, Delimiter{}, []rune(patStr))
		if pat.CacheKey() != expected {
			t.Errorf("Expected: %s, actual: %s", expected, pat.CacheKey())
		}
		if pat.cacheable != cacheable {
			t.Errorf("Expected: %s, actual: %s (%s)", cacheable, pat.cacheable, patStr)
		}
		clearPatternCache()
	}
	test(false, "foo !bar", "foo !bar", true)
	test(false, "foo | bar !baz", "foo | bar !baz", true)
	test(true, "foo  bar  baz", "foo bar baz", true)
	test(true, "foo !bar", "foo", false)
	test(true, "foo !bar   baz", "foo baz", false)
	test(true, "foo | bar baz", "baz", false)
	test(true, "foo | bar | baz", "", false)
	test(true, "foo | bar !baz", "", false)
	test(true, "| | | foo", "foo", true)
}
