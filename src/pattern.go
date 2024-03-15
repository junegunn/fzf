package fzf

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// fuzzy
// 'exact
// ^prefix-exact
// suffix-exact$
// !inverse-exact
// !'inverse-fuzzy
// !^inverse-prefix-exact
// !inverse-suffix-exact$

type termType int

const (
	termFuzzy termType = iota
	termExact
	termPrefix
	termSuffix
	termEqual
)

type term struct {
	typ           termType
	inv           bool
	text          []rune
	caseSensitive bool
	normalize     bool
}

// String returns the string representation of a term.
func (t term) String() string {
	return fmt.Sprintf("term{typ: %d, inv: %v, text: []rune(%q), caseSensitive: %v}", t.typ, t.inv, string(t.text), t.caseSensitive)
}

type termSet []term

// Pattern represents search pattern
type Pattern struct {
	fuzzy         bool
	fuzzyAlgo     algo.Algo
	extended      bool
	caseSensitive bool
	normalize     bool
	forward       bool
	withPos       bool
	text          []rune
	termSets      []termSet
	sortable      bool
	cacheable     bool
	cacheKey      string
	delimiter     Delimiter
	nth           []Range
	procFun       map[termType]algo.Algo
}

var (
	_patternCache map[string]*Pattern
	_splitRegex   *regexp.Regexp
	_cache        ChunkCache
)

func init() {
	_splitRegex = regexp.MustCompile(" +")
	clearPatternCache()
	clearChunkCache()
}

func clearPatternCache() {
	// We can uniquely identify the pattern for a given string since
	// search mode and caseMode do not change while the program is running
	_patternCache = make(map[string]*Pattern)
}

func clearChunkCache() {
	_cache = NewChunkCache()
}

// BuildPattern builds Pattern object from the given arguments
func BuildPattern(fuzzy bool, fuzzyAlgo algo.Algo, extended bool, caseMode Case, normalize bool, forward bool,
	withPos bool, cacheable bool, nth []Range, delimiter Delimiter, runes []rune) *Pattern {

	var asString string
	if extended {
		asString = strings.TrimLeft(string(runes), " ")
		for strings.HasSuffix(asString, " ") && !strings.HasSuffix(asString, "\\ ") {
			asString = asString[:len(asString)-1]
		}
	} else {
		asString = string(runes)
	}

	cached, found := _patternCache[asString]
	if found {
		return cached
	}

	caseSensitive := true
	sortable := true
	termSets := []termSet{}

	if extended {
		termSets = parseTerms(fuzzy, caseMode, normalize, asString)
		// We should not sort the result if there are only inverse search terms
		sortable = false
	Loop:
		for _, termSet := range termSets {
			for idx, term := range termSet {
				if !term.inv {
					sortable = true
				}
				// If the query contains inverse search terms or OR operators,
				// we cannot cache the search scope
				if !cacheable || idx > 0 || term.inv || fuzzy && term.typ != termFuzzy || !fuzzy && term.typ != termExact {
					cacheable = false
					if sortable {
						// Can't break until we see at least one non-inverse term
						break Loop
					}
				}
			}
		}
	} else {
		lowerString := strings.ToLower(asString)
		normalize = normalize &&
			lowerString == string(algo.NormalizeRunes([]rune(lowerString)))
		caseSensitive = caseMode == CaseRespect ||
			caseMode == CaseSmart && lowerString != asString
		if !caseSensitive {
			asString = lowerString
		}
	}

	ptr := &Pattern{
		fuzzy:         fuzzy,
		fuzzyAlgo:     fuzzyAlgo,
		extended:      extended,
		caseSensitive: caseSensitive,
		normalize:     normalize,
		forward:       forward,
		withPos:       withPos,
		text:          []rune(asString),
		termSets:      termSets,
		sortable:      sortable,
		cacheable:     cacheable,
		nth:           nth,
		delimiter:     delimiter,
		procFun:       make(map[termType]algo.Algo)}

	ptr.cacheKey = ptr.buildCacheKey()
	ptr.procFun[termFuzzy] = fuzzyAlgo
	ptr.procFun[termEqual] = algo.EqualMatch
	ptr.procFun[termExact] = algo.ExactMatchNaive
	ptr.procFun[termPrefix] = algo.PrefixMatch
	ptr.procFun[termSuffix] = algo.SuffixMatch

	_patternCache[asString] = ptr
	return ptr
}

func parseTerms(fuzzy bool, caseMode Case, normalize bool, str string) []termSet {
	str = strings.Replace(str, "\\ ", "\t", -1)
	tokens := _splitRegex.Split(str, -1)
	sets := []termSet{}
	set := termSet{}
	switchSet := false
	afterBar := false
	for _, token := range tokens {
		typ, inv, text := termFuzzy, false, strings.Replace(token, "\t", " ", -1)
		lowerText := strings.ToLower(text)
		caseSensitive := caseMode == CaseRespect ||
			caseMode == CaseSmart && text != lowerText
		normalizeTerm := normalize &&
			lowerText == string(algo.NormalizeRunes([]rune(lowerText)))
		if !caseSensitive {
			text = lowerText
		}
		if !fuzzy {
			typ = termExact
		}

		if len(set) > 0 && !afterBar && text == "|" {
			switchSet = false
			afterBar = true
			continue
		}
		afterBar = false

		if strings.HasPrefix(text, "!") {
			inv = true
			typ = termExact
			text = text[1:]
		}

		if text != "$" && strings.HasSuffix(text, "$") {
			typ = termSuffix
			text = text[:len(text)-1]
		}

		if strings.HasPrefix(text, "'") {
			// Flip exactness
			if fuzzy && !inv {
				typ = termExact
			} else {
				typ = termFuzzy
			}
			text = text[1:]
		} else if strings.HasPrefix(text, "^") {
			if typ == termSuffix {
				typ = termEqual
			} else {
				typ = termPrefix
			}
			text = text[1:]
		}

		if len(text) > 0 {
			if switchSet {
				sets = append(sets, set)
				set = termSet{}
			}
			textRunes := []rune(text)
			if normalizeTerm {
				textRunes = algo.NormalizeRunes(textRunes)
			}
			set = append(set, term{
				typ:           typ,
				inv:           inv,
				text:          textRunes,
				caseSensitive: caseSensitive,
				normalize:     normalizeTerm})
			switchSet = true
		}
	}
	if len(set) > 0 {
		sets = append(sets, set)
	}
	return sets
}

// IsEmpty returns true if the pattern is effectively empty
func (p *Pattern) IsEmpty() bool {
	if !p.extended {
		return len(p.text) == 0
	}
	return len(p.termSets) == 0
}

// AsString returns the search query in string type
func (p *Pattern) AsString() string {
	return string(p.text)
}

func (p *Pattern) buildCacheKey() string {
	if !p.extended {
		return p.AsString()
	}
	cacheableTerms := []string{}
	for _, termSet := range p.termSets {
		if len(termSet) == 1 && !termSet[0].inv && (p.fuzzy || termSet[0].typ == termExact) {
			cacheableTerms = append(cacheableTerms, string(termSet[0].text))
		}
	}
	return strings.Join(cacheableTerms, "\t")
}

// CacheKey is used to build string to be used as the key of result cache
func (p *Pattern) CacheKey() string {
	return p.cacheKey
}

// Match returns the list of matches Items in the given Chunk
func (p *Pattern) Match(chunk *Chunk, slab *util.Slab) []Result {
	// ChunkCache: Exact match
	cacheKey := p.CacheKey()
	if p.cacheable {
		if cached := _cache.Lookup(chunk, cacheKey); cached != nil {
			return cached
		}
	}

	// Prefix/suffix cache
	space := _cache.Search(chunk, cacheKey)

	matches := p.matchChunk(chunk, space, slab)

	if p.cacheable {
		_cache.Add(chunk, cacheKey, matches)
	}
	return matches
}

func (p *Pattern) matchChunk(chunk *Chunk, space []Result, slab *util.Slab) []Result {
	matches := []Result{}

	if space == nil {
		for idx := 0; idx < chunk.count; idx++ {
			if match, _, _ := p.MatchItem(&chunk.items[idx], p.withPos, slab); match != nil {
				matches = append(matches, *match)
			}
		}
	} else {
		for _, result := range space {
			if match, _, _ := p.MatchItem(result.item, p.withPos, slab); match != nil {
				matches = append(matches, *match)
			}
		}
	}
	return matches
}

// MatchItem returns true if the Item is a match
func (p *Pattern) MatchItem(item *Item, withPos bool, slab *util.Slab) (*Result, []Offset, *[]int) {
	if p.extended {
		if offsets, bonus, pos := p.extendedMatch(item, withPos, slab); len(offsets) == len(p.termSets) {
			result := buildResult(item, offsets, bonus)
			return &result, offsets, pos
		}
		return nil, nil, nil
	}
	offset, bonus, pos := p.basicMatch(item, withPos, slab)
	if sidx := offset[0]; sidx >= 0 {
		offsets := []Offset{offset}
		result := buildResult(item, offsets, bonus)
		return &result, offsets, pos
	}
	return nil, nil, nil
}

func (p *Pattern) basicMatch(item *Item, withPos bool, slab *util.Slab) (Offset, int, *[]int) {
	var input []Token
	if len(p.nth) == 0 {
		input = []Token{{text: &item.text, prefixLength: 0}}
	} else {
		input = p.transformInput(item)
	}
	if p.fuzzy {
		return p.iter(p.fuzzyAlgo, input, p.caseSensitive, p.normalize, p.forward, p.text, withPos, slab)
	}
	return p.iter(algo.ExactMatchNaive, input, p.caseSensitive, p.normalize, p.forward, p.text, withPos, slab)
}

func (p *Pattern) extendedMatch(item *Item, withPos bool, slab *util.Slab) ([]Offset, int, *[]int) {
	var input []Token
	if len(p.nth) == 0 {
		input = []Token{{text: &item.text, prefixLength: 0}}
	} else {
		input = p.transformInput(item)
	}
	offsets := []Offset{}
	var totalScore int
	var allPos *[]int
	if withPos {
		allPos = &[]int{}
	}
	for _, termSet := range p.termSets {
		var offset Offset
		var currentScore int
		matched := false
		for _, term := range termSet {
			pfun := p.procFun[term.typ]
			off, score, pos := p.iter(pfun, input, term.caseSensitive, term.normalize, p.forward, term.text, withPos, slab)
			if sidx := off[0]; sidx >= 0 {
				if term.inv {
					continue
				}
				offset, currentScore = off, score
				matched = true
				if withPos {
					if pos != nil {
						*allPos = append(*allPos, *pos...)
					} else {
						for idx := off[0]; idx < off[1]; idx++ {
							*allPos = append(*allPos, int(idx))
						}
					}
				}
				break
			} else if term.inv {
				offset, currentScore = Offset{0, 0}, 0
				matched = true
				continue
			}
		}
		if matched {
			offsets = append(offsets, offset)
			totalScore += currentScore
		}
	}
	return offsets, totalScore, allPos
}

func (p *Pattern) transformInput(item *Item) []Token {
	if item.transformed != nil {
		return *item.transformed
	}

	tokens := Tokenize(item.text.ToString(), p.delimiter)
	ret := Transform(tokens, p.nth)
	item.transformed = &ret
	return ret
}

func (p *Pattern) iter(pfun algo.Algo, tokens []Token, caseSensitive bool, normalize bool, forward bool, pattern []rune, withPos bool, slab *util.Slab) (Offset, int, *[]int) {
	for _, part := range tokens {
		if res, pos := pfun(caseSensitive, normalize, forward, part.text, pattern, withPos, slab); res.Start >= 0 {
			sidx := int32(res.Start) + part.prefixLength
			eidx := int32(res.End) + part.prefixLength
			if pos != nil {
				for idx := range *pos {
					(*pos)[idx] += int(part.prefixLength)
				}
			}
			return Offset{sidx, eidx}, res.Score, pos
		}
	}
	return Offset{-1, -1}, 0, nil
}
