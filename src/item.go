package fzf

import (
	"github.com/junegunn/fzf/src/curses"
)

// Offset holds two 32-bit integers denoting the offsets of a matched substring
type Offset [2]int32

type ColorOffset struct {
	offset [2]int32
	color  int
	bold   bool
}

// Item represents each input line
type Item struct {
	text        *string
	origText    *string
	transformed *Transformed
	index       uint32
	offsets     []Offset
	colors      []AnsiOffset
	rank        Rank
}

// Rank is used to sort the search result
type Rank struct {
	matchlen uint16
	strlen   uint16
	index    uint32
}

// Rank calculates rank of the Item
func (i *Item) Rank(cache bool) Rank {
	if cache && (i.rank.matchlen > 0 || i.rank.strlen > 0) {
		return i.rank
	}
	matchlen := 0
	prevEnd := 0
	for _, offset := range i.offsets {
		begin := int(offset[0])
		end := int(offset[1])
		if prevEnd > begin {
			begin = prevEnd
		}
		if end > prevEnd {
			prevEnd = end
		}
		if end > begin {
			matchlen += end - begin
		}
	}
	rank := Rank{uint16(matchlen), uint16(len(*i.text)), i.index}
	if cache {
		i.rank = rank
	}
	return rank
}

// AsString returns the original string
func (i *Item) AsString() string {
	if i.origText != nil {
		return *i.origText
	}
	return *i.text
}

func (item *Item) ColorOffsets(color int, bold bool, current bool) []ColorOffset {
	if len(item.colors) == 0 {
		offsets := make([]ColorOffset, 0)
		for _, off := range item.offsets {
			offsets = append(offsets, ColorOffset{offset: off, color: color, bold: bold})
		}
		return offsets
	}

	// Find max column
	var maxCol int32 = 0
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
	offsets := make([]ColorOffset, 0)
	add := func(idx int) {
		if curr != 0 && idx > start {
			if curr == -1 {
				offsets = append(offsets, ColorOffset{
					offset: Offset{int32(start), int32(idx)}, color: color, bold: bold})
			} else {
				ansi := item.colors[curr-1]
				bg := ansi.color.bg
				if current {
					bg = int(curses.DarkBG)
				}
				offsets = append(offsets, ColorOffset{
					offset: Offset{int32(start), int32(idx)},
					color:  curses.PairFor(ansi.color.fg, bg),
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

func compareRanks(irank Rank, jrank Rank, tac bool) bool {
	if irank.matchlen < jrank.matchlen {
		return true
	} else if irank.matchlen > jrank.matchlen {
		return false
	}

	if irank.strlen < jrank.strlen {
		return true
	} else if irank.strlen > jrank.strlen {
		return false
	}

	return (irank.index <= jrank.index) != tac
}
