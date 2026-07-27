package algo

// Equivalence tests for the single- and two-character fast paths against the
// general FuzzyMatchV2 algorithm, which serves as the oracle.
//
// Two complementary strategies:
//   - Exhaustive: every string up to a fixed length over an alphabet that
//     covers all ASCII character classes, so every local scoring branch is
//     exercised (not merely sampled).
//   - Fuzz: coverage-guided, arbitrary length, to reach cases the bounded
//     exhaustive sweep cannot (e.g. long gaps where a high score decays).

import (
	"testing"

	"github.com/junegunn/fzf/src/util"
)

// One char of each ASCII class that affects scoring: lower, upper, delimiter,
// number, non-word, whitespace.
var equivAlphabet = []byte{'a', 'B', '/', '1', '_', ' '}

func samePos(a, b *[]int) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return equalInts(*a, *b)
}

// compareFastPath runs a single input/pattern/params pair through both the
// fast path and the general algorithm and fails on any difference.
func compareFastPath(t *testing.T, chars *util.Chars, pattern []rune, cs, fwd, wp bool, disable *bool, slab *util.Slab) {
	t.Helper()
	*disable = true
	rg, pg := FuzzyMatchV2(cs, false, fwd, chars, pattern, wp, slab)
	*disable = false
	rt, pt := FuzzyMatchV2(cs, false, fwd, chars, pattern, wp, slab)
	if rg != rt || !samePos(pg, pt) {
		t.Fatalf("mismatch item=%q pat=%q cs=%v fwd=%v wp=%v: general %v %v vs fast %v %v",
			chars.ToString(), string(pattern), cs, fwd, wp, rg, pg, rt, pt)
	}
}

// lowerPattern lowercases the pattern for case-insensitive search, matching
// the Algo contract (the pattern is pre-lowercased by BuildPattern).
func lowerPattern(p []rune, cs bool) []rune {
	if cs {
		return p
	}
	out := make([]rune, len(p))
	for i, c := range p {
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return out
}

func runExhaustive(t *testing.T, patLen, maxLen int, disable *bool) {
	// All patterns of length patLen over the alphabet
	var pats [][]rune
	var genPat func(cur []rune)
	genPat = func(cur []rune) {
		if len(cur) == patLen {
			pats = append(pats, append([]rune(nil), cur...))
			return
		}
		for _, c := range equivAlphabet {
			genPat(append(cur, rune(c)))
		}
	}
	genPat(nil)

	slab := util.MakeSlab(100*1024, 2048)
	buf := make([]byte, 0, maxLen)
	var rec func(depth int)
	rec = func(depth int) {
		chars := util.ToChars(append([]byte(nil), buf...))
		for _, cs := range []bool{false, true} {
			for _, fwd := range []bool{true, false} {
				for _, wp := range []bool{false, true} {
					for _, p := range pats {
						compareFastPath(t, &chars, lowerPattern(p, cs), cs, fwd, wp, disable, slab)
					}
				}
			}
		}
		if depth == maxLen {
			return
		}
		for _, c := range equivAlphabet {
			buf = append(buf, c)
			rec(depth + 1)
			buf = buf[:len(buf)-1]
		}
	}
	rec(0)
}

func TestFuzzyMatchV2SingleExhaustive(t *testing.T) {
	runExhaustive(t, 1, 6, &disableSingle)
}

func TestFuzzyMatchV2TwoExhaustive(t *testing.T) {
	runExhaustive(t, 2, 6, &disableTwo)
}

func fuzzFastPath(f *testing.F, patLen int, disable *bool) {
	f.Add("core_color/view/server.txt", "co")
	f.Add("a                        b", "ab")
	f.Add("XyZ/123_abc.def", "z1")
	f.Add("aaaaaaaaaaaaaaaaaaaaaaaa", "aa")
	slab := util.MakeSlab(200*1024, 4096)
	f.Fuzz(func(t *testing.T, input, pat string) {
		r := []rune(pat)
		if len(r) != patLen {
			return
		}
		for _, c := range r {
			if c >= 128 {
				return
			}
		}
		chars := util.ToChars([]byte(input))
		if !chars.IsBytes() {
			return
		}
		for _, cs := range []bool{false, true} {
			for _, fwd := range []bool{true, false} {
				for _, wp := range []bool{false, true} {
					compareFastPath(t, &chars, lowerPattern(r, cs), cs, fwd, wp, disable, slab)
				}
			}
		}
	})
}

func FuzzFuzzyMatchV2Single(f *testing.F) { fuzzFastPath(f, 1, &disableSingle) }
func FuzzFuzzyMatchV2Two(f *testing.F)    { fuzzFastPath(f, 2, &disableTwo) }
