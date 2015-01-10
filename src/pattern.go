package fzf

import (
	"regexp"
	"strings"
)

const UPPERCASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// fuzzy
// 'exact
// ^exact-prefix
// exact-suffix$
// !not-fuzzy
// !'not-exact
// !^not-exact-prefix
// !not-exact-suffix$

type TermType int

const (
	TERM_FUZZY TermType = iota
	TERM_EXACT
	TERM_PREFIX
	TERM_SUFFIX
)

type Term struct {
	typ      TermType
	inv      bool
	text     []rune
	origText []rune
}

type Pattern struct {
	mode          Mode
	caseSensitive bool
	text          []rune
	terms         []Term
	hasInvTerm    bool
	delimiter     *regexp.Regexp
	nth           []Range
	procFun       map[TermType]func(bool, *string, []rune) (int, int)
}

var (
	_patternCache map[string]*Pattern
	_splitRegex   *regexp.Regexp
	_cache        ChunkCache
)

func init() {
	// We can uniquely identify the pattern for a given string since
	// mode and caseMode do not change while the program is running
	_patternCache = make(map[string]*Pattern)
	_splitRegex = regexp.MustCompile("\\s+")
	_cache = NewChunkCache()
}

func clearPatternCache() {
	_patternCache = make(map[string]*Pattern)
}

func BuildPattern(mode Mode, caseMode Case,
	nth []Range, delimiter *regexp.Regexp, runes []rune) *Pattern {

	var asString string
	switch mode {
	case MODE_EXTENDED, MODE_EXTENDED_EXACT:
		asString = strings.Trim(string(runes), " ")
	default:
		asString = string(runes)
	}

	cached, found := _patternCache[asString]
	if found {
		return cached
	}

	caseSensitive, hasInvTerm := true, false
	terms := []Term{}

	switch caseMode {
	case CASE_SMART:
		if !strings.ContainsAny(asString, UPPERCASE) {
			runes, caseSensitive = []rune(strings.ToLower(asString)), false
		}
	case CASE_IGNORE:
		runes, caseSensitive = []rune(strings.ToLower(asString)), false
	}

	switch mode {
	case MODE_EXTENDED, MODE_EXTENDED_EXACT:
		terms = parseTerms(mode, string(runes))
		for _, term := range terms {
			if term.inv {
				hasInvTerm = true
			}
		}
	}

	ptr := &Pattern{
		mode:          mode,
		caseSensitive: caseSensitive,
		text:          runes,
		terms:         terms,
		hasInvTerm:    hasInvTerm,
		nth:           nth,
		delimiter:     delimiter,
		procFun:       make(map[TermType]func(bool, *string, []rune) (int, int))}

	ptr.procFun[TERM_FUZZY] = FuzzyMatch
	ptr.procFun[TERM_EXACT] = ExactMatchNaive
	ptr.procFun[TERM_PREFIX] = PrefixMatch
	ptr.procFun[TERM_SUFFIX] = SuffixMatch

	_patternCache[asString] = ptr
	return ptr
}

func parseTerms(mode Mode, str string) []Term {
	tokens := _splitRegex.Split(str, -1)
	terms := []Term{}
	for _, token := range tokens {
		typ, inv, text := TERM_FUZZY, false, token
		origText := []rune(text)
		if mode == MODE_EXTENDED_EXACT {
			typ = TERM_EXACT
		}

		if strings.HasPrefix(text, "!") {
			inv = true
			text = text[1:]
		}

		if strings.HasPrefix(text, "'") {
			if mode == MODE_EXTENDED {
				typ = TERM_EXACT
				text = text[1:]
			}
		} else if strings.HasPrefix(text, "^") {
			typ = TERM_PREFIX
			text = text[1:]
		} else if strings.HasSuffix(text, "$") {
			typ = TERM_SUFFIX
			text = text[:len(text)-1]
		}

		if len(text) > 0 {
			terms = append(terms, Term{
				typ:      typ,
				inv:      inv,
				text:     []rune(text),
				origText: origText})
		}
	}
	return terms
}

func (p *Pattern) IsEmpty() bool {
	if p.mode == MODE_FUZZY {
		return len(p.text) == 0
	} else {
		return len(p.terms) == 0
	}
}

func (p *Pattern) AsString() string {
	return string(p.text)
}

func (p *Pattern) CacheKey() string {
	if p.mode == MODE_FUZZY {
		return p.AsString()
	}
	cacheableTerms := []string{}
	for _, term := range p.terms {
		if term.inv {
			continue
		}
		cacheableTerms = append(cacheableTerms, string(term.origText))
	}
	return strings.Join(cacheableTerms, " ")
}

func (p *Pattern) Match(chunk *Chunk) []*Item {
	space := chunk

	// ChunkCache: Exact match
	cacheKey := p.CacheKey()
	if !p.hasInvTerm { // Because we're excluding Inv-term from cache key
		if cached, found := _cache.Find(chunk, cacheKey); found {
			return cached
		}
	}

	// ChunkCache: Prefix match
	foundPrefixCache := false
	for idx := len(cacheKey) - 1; idx > 0; idx-- {
		if cached, found := _cache.Find(chunk, cacheKey[:idx]); found {
			cachedChunk := Chunk(cached)
			space = &cachedChunk
			foundPrefixCache = true
			break
		}
	}

	// ChunkCache: Suffix match
	if !foundPrefixCache {
		for idx := 1; idx < len(cacheKey); idx++ {
			if cached, found := _cache.Find(chunk, cacheKey[idx:]); found {
				cachedChunk := Chunk(cached)
				space = &cachedChunk
				break
			}
		}
	}

	var matches []*Item
	if p.mode == MODE_FUZZY {
		matches = p.fuzzyMatch(space)
	} else {
		matches = p.extendedMatch(space)
	}

	if !p.hasInvTerm {
		_cache.Add(chunk, cacheKey, matches)
	}
	return matches
}

func (p *Pattern) fuzzyMatch(chunk *Chunk) []*Item {
	matches := []*Item{}
	for _, item := range *chunk {
		input := p.prepareInput(item)
		if sidx, eidx := p.iter(FuzzyMatch, input, p.text); sidx >= 0 {
			matches = append(matches, &Item{
				text:     item.text,
				origText: item.origText,
				offsets:  []Offset{Offset{int32(sidx), int32(eidx)}},
				rank:     Rank{0, 0, item.rank.index}})
		}
	}
	return matches
}

func (p *Pattern) extendedMatch(chunk *Chunk) []*Item {
	matches := []*Item{}
	for _, item := range *chunk {
		input := p.prepareInput(item)
		offsets := []Offset{}
	Loop:
		for _, term := range p.terms {
			pfun := p.procFun[term.typ]
			if sidx, eidx := p.iter(pfun, input, term.text); sidx >= 0 {
				if term.inv {
					break Loop
				}
				offsets = append(offsets, Offset{int32(sidx), int32(eidx)})
			} else if term.inv {
				offsets = append(offsets, Offset{0, 0})
			}
		}
		if len(offsets) == len(p.terms) {
			matches = append(matches, &Item{
				text:     item.text,
				origText: item.origText,
				offsets:  offsets,
				rank:     Rank{0, 0, item.rank.index}})
		}
	}
	return matches
}

func (p *Pattern) prepareInput(item *Item) *Transformed {
	if item.transformed != nil {
		return item.transformed
	}

	var ret *Transformed
	if len(p.nth) > 0 {
		tokens := Tokenize(item.text, p.delimiter)
		ret = Transform(tokens, p.nth)
	} else {
		trans := Transformed{
			whole: item.text,
			parts: []Token{Token{text: item.text, prefixLength: 0}}}
		ret = &trans
	}
	item.transformed = ret
	return ret
}

func (p *Pattern) iter(pfun func(bool, *string, []rune) (int, int),
	inputs *Transformed, pattern []rune) (int, int) {
	for _, part := range inputs.parts {
		prefixLength := part.prefixLength
		if sidx, eidx := pfun(p.caseSensitive, part.text, pattern); sidx >= 0 {
			return sidx + prefixLength, eidx + prefixLength
		}
	}
	return -1, -1
}
