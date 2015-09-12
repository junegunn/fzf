package fzf

import (
	"reflect"
	"testing"

	"github.com/junegunn/fzf/src/algo"
)

func TestParseTermsExtended(t *testing.T) {
	terms := parseTerms(ModeExtended, CaseSmart,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$ ^iii$")
	if len(terms) != 9 ||
		terms[0].typ != termFuzzy || terms[0].inv ||
		terms[1].typ != termExact || terms[1].inv ||
		terms[2].typ != termPrefix || terms[2].inv ||
		terms[3].typ != termSuffix || terms[3].inv ||
		terms[4].typ != termFuzzy || !terms[4].inv ||
		terms[5].typ != termExact || !terms[5].inv ||
		terms[6].typ != termPrefix || !terms[6].inv ||
		terms[7].typ != termSuffix || !terms[7].inv ||
		terms[8].typ != termEqual || terms[8].inv {
		t.Errorf("%s", terms)
	}
	for idx, term := range terms {
		if len(term.text) != 3 {
			t.Errorf("%s", term)
		}
		if idx > 0 && len(term.origText) != 4+idx/5 {
			t.Errorf("%s", term)
		}
	}
}

func TestParseTermsExtendedExact(t *testing.T) {
	terms := parseTerms(ModeExtendedExact, CaseSmart,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0].typ != termExact || terms[0].inv || len(terms[0].text) != 3 ||
		terms[1].typ != termFuzzy || terms[1].inv || len(terms[1].text) != 3 ||
		terms[2].typ != termPrefix || terms[2].inv || len(terms[2].text) != 3 ||
		terms[3].typ != termSuffix || terms[3].inv || len(terms[3].text) != 3 ||
		terms[4].typ != termExact || !terms[4].inv || len(terms[4].text) != 3 ||
		terms[5].typ != termFuzzy || !terms[5].inv || len(terms[5].text) != 3 ||
		terms[6].typ != termPrefix || !terms[6].inv || len(terms[6].text) != 3 ||
		terms[7].typ != termSuffix || !terms[7].inv || len(terms[7].text) != 3 {
		t.Errorf("%s", terms)
	}
}

func TestParseTermsEmpty(t *testing.T) {
	terms := parseTerms(ModeExtended, CaseSmart, "' $ ^ !' !^ !$")
	if len(terms) != 0 {
		t.Errorf("%s", terms)
	}
}

func TestExact(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pattern := BuildPattern(ModeExtended, CaseSmart, true,
		[]Range{}, Delimiter{}, []rune("'abc"))
	sidx, eidx := algo.ExactMatchNaive(
		pattern.caseSensitive, pattern.forward, []rune("aabbcc abc"), pattern.terms[0].text)
	if sidx != 7 || eidx != 10 {
		t.Errorf("%s / %d / %d", pattern.terms, sidx, eidx)
	}
}

func TestEqual(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pattern := BuildPattern(ModeExtended, CaseSmart, true, []Range{}, Delimiter{}, []rune("^AbC$"))

	match := func(str string, sidxExpected int, eidxExpected int) {
		sidx, eidx := algo.EqualMatch(
			pattern.caseSensitive, pattern.forward, []rune(str), pattern.terms[0].text)
		if sidx != sidxExpected || eidx != eidxExpected {
			t.Errorf("%s / %d / %d", pattern.terms, sidx, eidx)
		}
	}
	match("ABC", -1, -1)
	match("AbC", 0, 3)
}

func TestCaseSensitivity(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pat1 := BuildPattern(ModeFuzzy, CaseSmart, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat2 := BuildPattern(ModeFuzzy, CaseSmart, true, []Range{}, Delimiter{}, []rune("Abc"))
	clearPatternCache()
	pat3 := BuildPattern(ModeFuzzy, CaseIgnore, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat4 := BuildPattern(ModeFuzzy, CaseIgnore, true, []Range{}, Delimiter{}, []rune("Abc"))
	clearPatternCache()
	pat5 := BuildPattern(ModeFuzzy, CaseRespect, true, []Range{}, Delimiter{}, []rune("abc"))
	clearPatternCache()
	pat6 := BuildPattern(ModeFuzzy, CaseRespect, true, []Range{}, Delimiter{}, []rune("Abc"))

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
	pattern := BuildPattern(ModeExtended, CaseSmart, true, []Range{}, Delimiter{}, []rune("jg"))
	tokens := Tokenize([]rune("junegunn"), Delimiter{})
	trans := Transform(tokens, []Range{Range{1, 1}})

	origRunes := []rune("junegunn.choi")
	for _, mode := range []Mode{ModeFuzzy, ModeExtended} {
		chunk := Chunk{
			&Item{
				text:        []rune("junegunn"),
				origText:    &origRunes,
				transformed: trans},
		}
		pattern.mode = mode
		matches := pattern.matchChunk(&chunk)
		if string(matches[0].text) != "junegunn" || string(*matches[0].origText) != "junegunn.choi" ||
			matches[0].offsets[0][0] != 0 || matches[0].offsets[0][1] != 5 ||
			!reflect.DeepEqual(matches[0].transformed, trans) {
			t.Error("Invalid match result", matches)
		}
	}
}
