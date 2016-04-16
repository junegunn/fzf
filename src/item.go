package fzf

import (
	"math"

	"github.com/junegunn/fzf/src/curses"
)

// Offset holds three 32-bit integers denoting the offsets of a matched substring
type Offset [3]int32

type colorOffset struct {
	offset [2]int32
	color  int
	bold   bool
}

// Item represents each input line
type Item struct {
	text        []rune
	origText    *[]rune
	transformed []Token
	offsets     []Offset
	colors      []ansiOffset
	rank        [5]int32
	bonus       int32
}

// Sort criteria to use. Never changes once fzf is started.
var sortCriteria []criterion

func isRankValid(rank [5]int32) bool {
	// Exclude ordinal index
	for _, r := range rank[:4] {
		if r > 0 {
			return true
		}
	}
	return false
}

func buildEmptyRank(index int32) [5]int32 {
	return [5]int32{0, 0, 0, 0, index}
}

func (item *Item) Index() int32 {
	return item.rank[4]
}

// Rank calculates rank of the Item
func (item *Item) Rank(cache bool) [5]int32 {
	if cache && isRankValid(item.rank) {
		return item.rank
	}
	matchlen := 0
	prevEnd := 0
	lenSum := 0
	minBegin := math.MaxInt32
	for _, offset := range item.offsets {
		begin := int(offset[0])
		end := int(offset[1])
		trimLen := int(offset[2])
		lenSum += trimLen
		if prevEnd > begin {
			begin = prevEnd
		}
		if end > prevEnd {
			prevEnd = end
		}
		if end > begin {
			if begin < minBegin {
				minBegin = begin
			}
			matchlen += end - begin
		}
	}
	rank := buildEmptyRank(item.Index())
	for idx, criterion := range sortCriteria {
		var val int32
		switch criterion {
		case byMatchLen:
			if matchlen == 0 {
				val = math.MaxInt32
			} else {
				// It is extremely unlikely that bonus exceeds 128
				val = 128*int32(matchlen) - item.bonus
			}
		case byLength:
			// It is guaranteed that .transformed in not null in normal execution
			if item.transformed != nil {
				// If offsets is empty, lenSum will be 0, but we don't care
				val = int32(lenSum)
			} else {
				val = int32(len(item.text))
			}
		case byBegin:
			// We can't just look at item.offsets[0][0] because it can be an inverse term
			whitePrefixLen := 0
			for idx, r := range item.text {
				whitePrefixLen = idx
				if idx == minBegin || r != ' ' && r != '\t' {
					break
				}
			}
			val = int32(minBegin - whitePrefixLen)
		case byEnd:
			if prevEnd > 0 {
				val = int32(1 + len(item.text) - prevEnd)
			} else {
				// Empty offsets due to inverse terms.
				val = 1
			}
		}
		rank[idx] = val
	}
	if cache {
		item.rank = rank
	}
	return rank
}

// AsString returns the original string
func (item *Item) AsString(stripAnsi bool) string {
	return *item.StringPtr(stripAnsi)
}

// StringPtr returns the pointer to the original string
func (item *Item) StringPtr(stripAnsi bool) *string {
	if item.origText != nil {
		if stripAnsi {
			trimmed, _, _ := extractColor(string(*item.origText), nil)
			return &trimmed
		}
		orig := string(*item.origText)
		return &orig
	}
	str := string(item.text)
	return &str
}

func (item *Item) colorOffsets(color int, bold bool, current bool) []colorOffset {
	if len(item.colors) == 0 {
		var offsets []colorOffset
		for _, off := range item.offsets {

			offsets = append(offsets, colorOffset{offset: [2]int32{off[0], off[1]}, color: color, bold: bold})
		}
		return offsets
	}

	// Find max column
	var maxCol int32
	for _, off := range item.offsets {
		if off[1] > maxCol {
			maxCol = off[1]
		}
	}
	for _, ansi := range item.colors {
		if ansi.offset[1] > maxCol {
			maxCol = ansi.offset[1]
		}
	}
	cols := make([]int, maxCol)

	for colorIndex, ansi := range item.colors {
		for i := ansi.offset[0]; i < ansi.offset[1]; i++ {
			cols[i] = colorIndex + 1 // XXX
		}
	}

	for _, off := range item.offsets {
		for i := off[0]; i < off[1]; i++ {
			cols[i] = -1
		}
	}

	// sort.Sort(ByOrder(offsets))

	// Merge offsets
	// ------------  ----  --  ----
	//   ++++++++      ++++++++++
	// --++++++++--  --++++++++++---
	curr := 0
	start := 0
	var offsets []colorOffset
	add := func(idx int) {
		if curr != 0 && idx > start {
			if curr == -1 {
				offsets = append(offsets, colorOffset{
					offset: [2]int32{int32(start), int32(idx)}, color: color, bold: bold})
			} else {
				ansi := item.colors[curr-1]
				fg := ansi.color.fg
				if fg == -1 {
					if current {
						fg = curses.CurrentFG
					} else {
						fg = curses.FG
					}
				}
				bg := ansi.color.bg
				if bg == -1 {
					if current {
						bg = curses.DarkBG
					} else {
						bg = curses.BG
					}
				}
				offsets = append(offsets, colorOffset{
					offset: [2]int32{int32(start), int32(idx)},
					color:  curses.PairFor(fg, bg),
					bold:   ansi.color.bold || bold})
			}
		}
	}
	for idx, col := range cols {
		if col != curr {
			add(idx)
			start = idx
			curr = col
		}
	}
	add(int(maxCol))
	return offsets
}

// ByOrder is for sorting substring offsets
type ByOrder []Offset

func (a ByOrder) Len() int {
	return len(a)
}

func (a ByOrder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByOrder) Less(i, j int) bool {
	ioff := a[i]
	joff := a[j]
	return (ioff[0] < joff[0]) || (ioff[0] == joff[0]) && (ioff[1] <= joff[1])
}

// ByRelevance is for sorting Items
type ByRelevance []*Item

func (a ByRelevance) Len() int {
	return len(a)
}

func (a ByRelevance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevance) Less(i, j int) bool {
	irank := a[i].Rank(true)
	jrank := a[j].Rank(true)

	return compareRanks(irank, jrank, false)
}

// ByRelevanceTac is for sorting Items
type ByRelevanceTac []*Item

func (a ByRelevanceTac) Len() int {
	return len(a)
}

func (a ByRelevanceTac) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevanceTac) Less(i, j int) bool {
	irank := a[i].Rank(true)
	jrank := a[j].Rank(true)

	return compareRanks(irank, jrank, true)
}

func compareRanks(irank [5]int32, jrank [5]int32, tac bool) bool {
	for idx := 0; idx < 4; idx++ {
		left := irank[idx]
		right := jrank[idx]
		if left < right {
			return true
		} else if left > right {
			return false
		}
	}
	return (irank[4] <= jrank[4]) != tac
}
