package fzf

import "fmt"

var EmptyMerger *Merger = NewMerger([][]*Item{}, false)

type Merger struct {
	lists   [][]*Item
	merged  []*Item
	cursors []int
	sorted  bool
	count   int
}

func NewMerger(lists [][]*Item, sorted bool) *Merger {
	mg := Merger{
		lists:   lists,
		merged:  []*Item{},
		cursors: make([]int, len(lists)),
		sorted:  sorted,
		count:   0}

	for _, list := range mg.lists {
		mg.count += len(list)
	}
	return &mg
}

func (mg *Merger) Length() int {
	return mg.count
}

func (mg *Merger) Get(idx int) *Item {
	if len(mg.lists) == 1 {
		return mg.lists[0][idx]
	} else if !mg.sorted {
		for _, list := range mg.lists {
			numItems := len(list)
			if idx < numItems {
				return list[idx]
			}
			idx -= numItems
		}
		panic(fmt.Sprintf("Index out of bounds (unsorted, %d/%d)", idx, mg.count))
	}
	return mg.mergedGet(idx)
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
				if minIdx < 0 || compareRanks(rank, minRank) {
					minRank = rank
					minIdx = listIdx
				}
			}
			mg.cursors[listIdx] = cursor
		}

		if minIdx >= 0 {
			chosen := mg.lists[minIdx]
			mg.merged = append(mg.merged, chosen[mg.cursors[minIdx]])
			mg.cursors[minIdx] += 1
		} else {
			panic(fmt.Sprintf("Index out of bounds (sorted, %d/%d)", i, mg.count))
		}
	}
	return mg.merged[idx]
}
