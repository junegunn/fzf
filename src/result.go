package fzf

import (
	"math"
	"sort"

	"github.com/junegunn/fzf/src/curses"
	"github.com/junegunn/fzf/src/util"
)

// Offset holds two 32-bit integers denoting the offsets of a matched substring
type Offset [2]int32

type colorOffset struct {
	offset [2]int32
	color  int
	bold   bool
	index  int32
}

type rank struct {
	index int32
	// byMatchLen, byBonus, ...
	points [5]uint16
}

type Result struct {
	item    *Item
	offsets []Offset
	rank    rank
}

func buildResult(item *Item, offsets []Offset, bonus int, trimLen int) *Result {
	if len(offsets) > 1 {
		sort.Sort(ByOrder(offsets))
	}

	result := Result{item: item, offsets: offsets, rank: rank{index: item.index}}

	matchlen := 0
	prevEnd := 0
	minBegin := math.MaxInt32
	numChars := item.text.Length()
	for _, offset := range offsets {
		begin := int(offset[0])
		end := int(offset[1])
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

	for idx, criterion := range sortCriteria {
		var val uint16
		switch criterion {
		case byMatchLen:
			if matchlen == 0 {
				val = math.MaxUint16
			} else {
				val = util.AsUint16(matchlen)
			}
		case byBonus:
			// Higher is better
			val = math.MaxUint16 - util.AsUint16(bonus)
		case byLength:
			// If offsets is empty, trimLen will be 0, but we don't care
			val = util.AsUint16(trimLen)
		case byBegin:
			// We can't just look at item.offsets[0][0] because it can be an inverse term
			whitePrefixLen := 0
			for idx := 0; idx < numChars; idx++ {
				r := item.text.Get(idx)
				whitePrefixLen = idx
				if idx == minBegin || r != ' ' && r != '\t' {
					break
				}
			}
			val = util.AsUint16(minBegin - whitePrefixLen)
		case byEnd:
			if prevEnd > 0 {
				val = util.AsUint16(1 + numChars - prevEnd)
			} else {
				// Empty offsets due to inverse terms.
				val = 1
			}
		}
		result.rank.points[idx] = val
	}

	return &result
}

// Sort criteria to use. Never changes once fzf is started.
var sortCriteria []criterion

// Index returns ordinal index of the Item
func (result *Result) Index() int32 {
	return result.item.index
}

func minRank() rank {
	return rank{index: 0, points: [5]uint16{0, math.MaxUint16, 0, 0, 0}}
}

func (result *Result) colorOffsets(color int, bold bool, current bool) []colorOffset {
	itemColors := result.item.Colors()

	if len(itemColors) == 0 {
		var offsets []colorOffset
		for _, off := range result.offsets {

			offsets = append(offsets, colorOffset{offset: [2]int32{off[0], off[1]}, color: color, bold: bold})
		}
		return offsets
	}

	// Find max column
	var maxCol int32
	for _, off := range result.offsets {
		if off[1] > maxCol {
			maxCol = off[1]
		}
	}
	for _, ansi := range itemColors {
		if ansi.offset[1] > maxCol {
			maxCol = ansi.offset[1]
		}
	}
	cols := make([]int, maxCol)

	for colorIndex, ansi := range itemColors {
		for i := ansi.offset[0]; i < ansi.offset[1]; i++ {
			cols[i] = colorIndex + 1 // XXX
		}
	}

	for _, off := range result.offsets {
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
	var colors []colorOffset
	add := func(idx int) {
		if curr != 0 && idx > start {
			if curr == -1 {
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)}, color: color, bold: bold})
			} else {
				ansi := itemColors[curr-1]
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
				colors = append(colors, colorOffset{
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
	return colors
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
type ByRelevance []*Result

func (a ByRelevance) Len() int {
	return len(a)
}

func (a ByRelevance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevance) Less(i, j int) bool {
	return compareRanks((*a[i]).rank, (*a[j]).rank, false)
}

// ByRelevanceTac is for sorting Items
type ByRelevanceTac []*Result

func (a ByRelevanceTac) Len() int {
	return len(a)
}

func (a ByRelevanceTac) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevanceTac) Less(i, j int) bool {
	return compareRanks((*a[i]).rank, (*a[j]).rank, true)
}

func compareRanks(irank rank, jrank rank, tac bool) bool {
	for idx := 0; idx < 5; idx++ {
		left := irank.points[idx]
		right := jrank.points[idx]
		if left < right {
			return true
		} else if left > right {
			return false
		}
	}
	return (irank.index <= jrank.index) != tac
}
