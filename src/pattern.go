package fzf

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/junegunn/fzf/src/algo"
)

// fuzzy
// 'exact
// ^exact-prefix
// exact-suffix$
// !not-fuzzy
// !'not-exact
// !^not-exact-prefix
// !not-exact-suffix$

type termType int

const (
	termFuzzy termType = iota
	termExact
	termPrefix
	termSuffix
)

type term struct {
	typ      termType
	inv      bool
	text     []rune
	origText []rune
}

// Pattern represents search pattern
type Pattern struct {
	mode          Mode
	caseSensitive bool
	text          []rune
	terms         []term
	hasInvTerm    bool
	delimiter     *regexp.Regexp
	nth           []Range
	procFun       map[termType]func(bool, *[]rune, []rune) (int, int)
}

var (
	_patternCache map[string]*Pattern
	_splitRegex   *regexp.Regexp
	_cache        ChunkCache
)

func init() {
	_splitRegex = regexp.MustCompile("\\s+")
	clearPatternCache()
	clearChunkCache()
}

func clearPatternCache() {
	// We can uniquely identify the pattern for a given string since
	// mode and caseMode do not change while the program is running
	_patternCache = make(map[string]*Pattern)
}

func clearChunkCache() {
	_cache = NewChunkCache()
}

// BuildPattern builds Pattern object from the given arguments
func BuildPattern(mode Mode, caseMode Case,
	nth []Range, delimiter *regexp.Regexp, runes []rune) *Pattern {

	var asString string
	switch mode {
	case ModeExtended, ModeExtendedExact:
		asString = strings.Trim(string(runes), " ")
	default:
		asString = string(runes)
	}

	cached, found := _patternCache[asString]
	if found {
		return cached
	}

	caseSensitive, hasInvTerm := true, false
	terms := []term{}

	switch caseMode {
	case CaseSmart:
		hasUppercase := false
		for _, r := range runes {
			if unicode.IsUpper(r) {
				hasUppercase = true
				break
			}
		}
		if !hasUppercase {
			runes, caseSensitive = []rune(strings.ToLower(asString)), false
		}
	case CaseIgnore:
		runes, caseSensitive = []rune(strings.ToLower(asString)), false
	}

	switch mode {
	case ModeExtended, ModeExtendedExact:
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
		procFun:       make(map[termType]func(bool, *[]rune, []rune) (int, int))}

	ptr.procFun[termFuzzy] = algo.FuzzyMatch
	ptr.procFun[termExact] = algo.ExactMatchNaive
	ptr.procFun[termPrefix] = algo.PrefixMatch
	ptr.procFun[termSuffix] = algo.SuffixMatch

	_patternCache[asString] = ptr
	return ptr
}

func parseTerms(mode Mode, str string) []term {
	tokens := _splitRegex.Split(str, -1)
	terms := []term{}
	for _, token := range tokens {
		typ, inv, text := termFuzzy, false, token
		origText := []rune(text)
		if mode == ModeExtendedExact {
			typ = termExact
		}

		if strings.HasPrefix(text, "!") {
			inv = true
			text = text[1:]
		}

		if strings.HasPrefix(text, "'") {
			if mode == ModeExtended {
				typ = termExact
				text = text[1:]
			}
		} else if strings.HasPrefix(text, "^") {
			typ = termPrefix
			text = text[1:]
		} else if strings.HasSuffix(text, "$") {
			typ = termSuffix
			text = text[:len(text)-1]
		}

		if len(text) > 0 {
			terms = append(terms, term{
				typ:      typ,
				inv:      inv,
				text:     []rune(text),
				origText: origText})
		}
	}
	return terms
}

// IsEmpty returns true if the pattern is effectively empty
func (p *Pattern) IsEmpty() bool {
	if p.mode == ModeFuzzy {
		return len(p.text) == 0
	}
	return len(p.terms) == 0
}

// AsString returns the search query in string type
func (p *Pattern) AsString() string {
	return string(p.text)
}

// CacheKey is used to build string to be used as the key of result cache
func (p *Pattern) CacheKey() string {
	if p.mode == ModeFuzzy {
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

// Match returns the list of matches Items in the given Chunk
func (p *Pattern) Match(chunk *Chunk) []*Item {
	space := chunk

	// ChunkCache: Exact match
	cacheKey := p.CacheKey()
	if !p.hasInvTerm { // Because we're excluding Inv-term from cache key
		if cached, found := _cache.Find(chunk, cacheKey); found {
			return cached
		}
	}

	// ChunkCache: Prefix/suffix match
Loop:
	for idx := 1; idx < len(cacheKey); idx++ {
		// [---------| ] | [ |---------]
		// [--------|  ] | [  |--------]
		// [-------|   ] | [   |-------]
		prefix := cacheKey[:len(cacheKey)-idx]
		suffix := cacheKey[idx:]
		for _, substr := range [2]*string{&prefix, &suffix} {
			if cached, found := _cache.Find(chunk, *substr); found {
				cachedChunk := Chunk(cached)
				space = &cachedChunk
				break Loop
			}
		}
	}

	matches := p.matchChunk(space)

	if !p.hasInvTerm {
		_cache.Add(chunk, cacheKey, matches)
	}
	return matches
}

func (p *Pattern) matchChunk(chunk *Chunk) []*Item {
	matches := []*Item{}
	if p.mode == ModeFuzzy {
		for _, item := range *chunk {
			if sidx, eidx := p.fuzzyMatch(item); sidx >= 0 {
				matches = append(matches,
					dupItem(item, []Offset{Offset{int32(sidx), int32(eidx)}}))
			}
		}
	} else {
		for _, item := range *chunk {
			if offsets := p.extendedMatch(item); len(offsets) == len(p.terms) {
				matches = append(matches, dupItem(item, offsets))
			}
		}
	}
	return matches
}

// MatchItem returns true if the Item is a match
func (p *Pattern) MatchItem(item *Item) bool {
	if p.mode == ModeFuzzy {
		sidx, _ := p.fuzzyMatch(item)
		return sidx >= 0
	}
	offsets := p.extendedMatch(item)
	return len(offsets) == len(p.terms)
}

func dupItem(item *Item, offsets []Offset) *Item {
	sort.Sort(ByOrder(offsets))
	return &Item{
		text:        item.text,
		origText:    item.origText,
		transformed: item.transformed,
		index:       item.index,
		offsets:     offsets,
		colors:      item.colors,
		rank:        Rank{0, 0, item.index}}
}

func (p *Pattern) fuzzyMatch(item *Item) (int, int) {
	input := p.prepareInput(item)
	return p.iter(algo.FuzzyMatch, input, p.text)
}

func (p *Pattern) extendedMatch(item *Item) []Offset {
	input := p.prepareInput(item)
	offsets := []Offset{}
	for _, term := range p.terms {
		pfun := p.procFun[term.typ]
		if sidx, eidx := p.iter(pfun, input, term.text); sidx >= 0 {
			if term.inv {
				break
			}
			offsets = append(offsets, Offset{int32(sidx), int32(eidx)})
		} else if term.inv {
			offsets = append(offsets, Offset{0, 0})
		}
	}
	return offsets
}

func (p *Pattern) prepareInput(item *Item) *[]Token {
	if item.transformed != nil {
		return item.transformed
	}

	var ret *[]Token
	if len(p.nth) > 0 {
		tokens := Tokenize(item.text, p.delimiter)
		ret = Transform(tokens, p.nth)
	} else {
		runes := []rune(*item.text)
		trans := []Token{Token{text: &runes, prefixLength: 0}}
		ret = &trans
	}
	item.transformed = ret
	return ret
}

func (p *Pattern) iter(pfun func(bool, *[]rune, []rune) (int, int),
	tokens *[]Token, pattern []rune) (int, int) {
	for _, part := range *tokens {
		prefixLength := part.prefixLength
		if sidx, eidx := pfun(p.caseSensitive, part.text, pattern); sidx >= 0 {
			return sidx + prefixLength, eidx + prefixLength
		}
	}
	return -1, -1
}
