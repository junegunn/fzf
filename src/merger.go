package fzf

import "fmt"

// Merger with no data
var EmptyMerger = NewMerger([][]*Item{}, false, false)

// Merger holds a set of locally sorted lists of items and provides the view of
// a single, globally-sorted list
type Merger struct {
	lists   [][]*Item
	merged  []*Item
	cursors []int
	sorted  bool
	tac     bool
	final   bool
	count   int
}

// NewMerger returns a new Merger
func NewMerger(lists [][]*Item, sorted bool, tac bool) *Merger {
	mg := Merger{
		lists:   lists,
		merged:  []*Item{},
		cursors: make([]int, len(lists)),
		sorted:  sorted,
		tac:     tac,
		final:   false,
		count:   0}

	for _, list := range mg.lists {
		mg.count += len(list)
	}
	return &mg
}

// Length returns the number of items
func (mg *Merger) Length() int {
	return mg.count
}

// Get returns the pointer to the Item object indexed by the given integer
func (mg *Merger) Get(idx int) *Item {
	if mg.sorted {
		return mg.mergedGet(idx)
	}

	if mg.tac {
		idx = mg.Length() - idx - 1
	}
	for _, list := range mg.lists {
		numItems := len(list)
		if idx < numItems {
			return list[idx]
		}
		idx -= numItems
	}
	panic(fmt.Sprintf("Index out of bounds (unsorted, %d/%d)", idx, mg.count))
}

func (mg *Merger) mergedGet(idx int) *Item {
	for i := len(mg.merged); i <= idx; i++ {
		minRank := Rank{0, 0, 0}
		minIdx := -1
		for listIdx, list := range mg.lists {
			cursor := mg.cursors[listIdx]
			if cursor < 0 || cursor == len(list) {
				mg.cursors[listIdx] = -1
				continue
			}
			if cursor >= 0 {
				rank := list[cursor].Rank(false)
				if minIdx < 0 || compareRanks(rank, minRank, mg.tac) {
					minRank = rank
					minIdx = listIdx
				}
			}
			mg.cursors[listIdx] = cursor
		}

		if minIdx >= 0 {
			chosen := mg.lists[minIdx]
			mg.merged = append(mg.merged, chosen[mg.cursors[minIdx]])
			mg.cursors[minIdx]++
		} else {
			panic(fmt.Sprintf("Index out of bounds (sorted, %d/%d)", i, mg.count))
		}
	}
	return mg.merged[idx]
}
