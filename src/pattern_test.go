package fzf

import (
	"testing"

	"github.com/junegunn/fzf/src/algo"
)

func TestParseTermsExtended(t *testing.T) {
	terms := parseTerms(ModeExtended,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0].typ != termFuzzy || terms[0].inv ||
		terms[1].typ != termExact || terms[1].inv ||
		terms[2].typ != termPrefix || terms[2].inv ||
		terms[3].typ != termSuffix || terms[3].inv ||
		terms[4].typ != termFuzzy || !terms[4].inv ||
		terms[5].typ != termExact || !terms[5].inv ||
		terms[6].typ != termPrefix || !terms[6].inv ||
		terms[7].typ != termSuffix || !terms[7].inv {
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
	terms := parseTerms(ModeExtendedExact,
		"aaa 'bbb ^ccc ddd$ !eee !'fff !^ggg !hhh$")
	if len(terms) != 8 ||
		terms[0].typ != termExact || terms[0].inv || len(terms[0].text) != 3 ||
		terms[1].typ != termExact || terms[1].inv || len(terms[1].text) != 4 ||
		terms[2].typ != termPrefix || terms[2].inv || len(terms[2].text) != 3 ||
		terms[3].typ != termSuffix || terms[3].inv || len(terms[3].text) != 3 ||
		terms[4].typ != termExact || !terms[4].inv || len(terms[4].text) != 3 ||
		terms[5].typ != termExact || !terms[5].inv || len(terms[5].text) != 4 ||
		terms[6].typ != termPrefix || !terms[6].inv || len(terms[6].text) != 3 ||
		terms[7].typ != termSuffix || !terms[7].inv || len(terms[7].text) != 3 {
		t.Errorf("%s", terms)
	}
}

func TestParseTermsEmpty(t *testing.T) {
	terms := parseTerms(ModeExtended, "' $ ^ !' !^ !$")
	if len(terms) != 0 {
		t.Errorf("%s", terms)
	}
}

func TestExact(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pattern := BuildPattern(ModeExtended, CaseSmart,
		[]Range{}, nil, []rune("'abc"))
	runes := []rune("aabbcc abc")
	sidx, eidx := algo.ExactMatchNaive(pattern.caseSensitive, &runes, pattern.terms[0].text)
	if sidx != 7 || eidx != 10 {
		t.Errorf("%s / %d / %d", pattern.terms, sidx, eidx)
	}
}

func TestCaseSensitivity(t *testing.T) {
	defer clearPatternCache()
	clearPatternCache()
	pat1 := BuildPattern(ModeFuzzy, CaseSmart, []Range{}, nil, []rune("abc"))
	clearPatternCache()
	pat2 := BuildPattern(ModeFuzzy, CaseSmart, []Range{}, nil, []rune("Abc"))
	clearPatternCache()
	pat3 := BuildPattern(ModeFuzzy, CaseIgnore, []Range{}, nil, []rune("abc"))
	clearPatternCache()
	pat4 := BuildPattern(ModeFuzzy, CaseIgnore, []Range{}, nil, []rune("Abc"))
	clearPatternCache()
	pat5 := BuildPattern(ModeFuzzy, CaseRespect, []Range{}, nil, []rune("abc"))
	clearPatternCache()
	pat6 := BuildPattern(ModeFuzzy, CaseRespect, []Range{}, nil, []rune("Abc"))

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
	strptr := func(str string) *string {
		return &str
	}
	pattern := BuildPattern(ModeExtended, CaseSmart, []Range{}, nil, []rune("jg"))
	tokens := Tokenize(strptr("junegunn"), nil)
	trans := Transform(tokens, []Range{Range{1, 1}})

	for _, mode := range []Mode{ModeFuzzy, ModeExtended} {
		chunk := Chunk{
			&Item{
				text:        strptr("junegunn"),
				origText:    strptr("junegunn.choi"),
				transformed: trans},
		}
		pattern.mode = mode
		matches := pattern.matchChunk(&chunk)
		if *matches[0].text != "junegunn" || *matches[0].origText != "junegunn.choi" ||
			matches[0].offsets[0][0] != 0 || matches[0].offsets[0][1] != 5 ||
			matches[0].transformed != trans {
			t.Error("Invalid match result", matches)
		}
	}
}
