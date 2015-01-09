package fzf

var EmptyMerger *Merger = NewMerger([][]*Item{}, false)

type Merger struct {
	lists   [][]*Item
	merged  []*Item
	cursors []int
	done    bool
}

func NewMerger(lists [][]*Item, sorted bool) *Merger {
	mg := Merger{
		lists:   lists,
		merged:  []*Item{},
		cursors: make([]int, len(lists)),
		done:    false}
	if !sorted {
		for _, list := range lists {
			mg.merged = append(mg.merged, list...)
		}
		mg.done = true
	}
	return &mg
}

func (mg *Merger) Length() int {
	cnt := 0
	for _, list := range mg.lists {
		cnt += len(list)
	}
	return cnt
}

func (mg *Merger) Get(idx int) *Item {
	if mg.done {
		return mg.merged[idx]
	} else if len(mg.lists) == 1 {
		return mg.lists[0][idx]
	}
	mg.buildUpto(idx)
	return mg.merged[idx]
}

func (mg *Merger) buildUpto(upto int) {
	numBuilt := len(mg.merged)
	if numBuilt > upto {
		return
	}

	for i := numBuilt; i <= upto; i++ {
		minRank := Rank{0, 0, 0}
		minIdx := -1
		for listIdx, list := range mg.lists {
			cursor := mg.cursors[listIdx]
			if cursor < 0 || cursor == len(list) {
				mg.cursors[listIdx] = -1
				continue
			}
			if cursor >= 0 {
				rank := list[cursor].Rank()
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
			mg.done = true
			return
		}
	}
}
