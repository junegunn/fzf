package fzf

import "fmt"

// EmptyMerger is a Merger with no data
func EmptyMerger(revision int) *Merger {
	return NewMerger(nil, [][]Result{}, false, false, revision)
}

// Merger holds a set of locally sorted lists of items and provides the view of
// a single, globally-sorted list
type Merger struct {
	pattern  *Pattern
	lists    [][]Result
	merged   []Result
	chunks   *[]*Chunk
	cursors  []int
	sorted   bool
	tac      bool
	final    bool
	count    int
	pass     bool
	revision int
}

// PassMerger returns a new Merger that simply returns the items in the
// original order
func PassMerger(chunks *[]*Chunk, tac bool, revision int) *Merger {
	mg := Merger{
		pattern:  nil,
		chunks:   chunks,
		tac:      tac,
		count:    0,
		pass:     true,
		revision: revision}

	for _, chunk := range *mg.chunks {
		mg.count += chunk.count
	}
	return &mg
}

// NewMerger returns a new Merger
func NewMerger(pattern *Pattern, lists [][]Result, sorted bool, tac bool, revision int) *Merger {
	mg := Merger{
		pattern:  pattern,
		lists:    lists,
		merged:   []Result{},
		chunks:   nil,
		cursors:  make([]int, len(lists)),
		sorted:   sorted,
		tac:      tac,
		final:    false,
		count:    0,
		revision: revision}

	for _, list := range mg.lists {
		mg.count += len(list)
	}
	return &mg
}

// Revision returns revision number
func (mg *Merger) Revision() int {
	return mg.revision
}

// Length returns the number of items
func (mg *Merger) Length() int {
	return mg.count
}

func (mg *Merger) First() Result {
	if mg.tac && !mg.sorted {
		return mg.Get(mg.count - 1)
	}
	return mg.Get(0)
}

// FindIndex returns the index of the item with the given item index
func (mg *Merger) FindIndex(itemIndex int32) int {
	index := -1
	if mg.pass {
		index = int(itemIndex)
		if mg.tac {
			index = mg.count - index - 1
		}
	} else {
		for i := 0; i < mg.count; i++ {
			if mg.Get(i).item.Index() == itemIndex {
				index = i
				break
			}
		}
	}
	return index
}

// Get returns the pointer to the Result object indexed by the given integer
func (mg *Merger) Get(idx int) Result {
	if mg.chunks != nil {
		if mg.tac {
			idx = mg.count - idx - 1
		}
		chunk := (*mg.chunks)[idx/chunkSize]
		return Result{item: &chunk.items[idx%chunkSize]}
	}

	if mg.sorted {
		return mg.mergedGet(idx)
	}

	if mg.tac {
		idx = mg.count - idx - 1
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

func (mg *Merger) cacheable() bool {
	return mg.count < mergerCacheMax
}

func (mg *Merger) mergedGet(idx int) Result {
	for i := len(mg.merged); i <= idx; i++ {
		minRank := minRank()
		minIdx := -1
		for listIdx, list := range mg.lists {
			cursor := mg.cursors[listIdx]
			if cursor < 0 || cursor == len(list) {
				mg.cursors[listIdx] = -1
				continue
			}
			if cursor >= 0 {
				rank := list[cursor]
				if minIdx < 0 || compareRanks(rank, minRank, mg.tac) {
					minRank = rank
					minIdx = listIdx
				}
			}
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
