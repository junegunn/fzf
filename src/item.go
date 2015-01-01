package fzf

import (
	"fmt"
	"sort"
)

type Offset [2]int

type Item struct {
	text        *string
	origText    *string
	offsets     []Offset
	index       int
	rank        Rank
	transformed *Transformed
}

type Rank [3]int

var NilRank = Rank{-1, 0, 0}

func (i *Item) Rank() Rank {
	if i.rank[0] > 0 {
		return i.rank
	}
	sort.Sort(ByOrder(i.offsets))
	matchlen := 0
	prevEnd := 0
	for _, offset := range i.offsets {
		begin := offset[0]
		end := offset[1]
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
	i.rank = Rank{matchlen, len(*i.text), i.index}
	return i.rank
}

func (i *Item) Print() {
	if i.origText != nil {
		fmt.Println(*i.origText)
	} else {
		fmt.Println(*i.text)
	}
}

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

type ByRelevance []*Item

func (a ByRelevance) Len() int {
	return len(a)
}

func (a ByRelevance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevance) Less(i, j int) bool {
	irank := a[i].Rank()
	jrank := a[j].Rank()

	return compareRanks(irank, jrank)
}

func compareRanks(irank Rank, jrank Rank) bool {
	for idx := range irank {
		if irank[idx] < jrank[idx] {
			return true
		} else if irank[idx] > jrank[idx] {
			return false
		}
	}
	return true
}

func SortMerge(partialResults [][]*Item) []*Item {
	if len(partialResults) == 1 {
		return partialResults[0]
	}

	merged := []*Item{}

	for len(partialResults) > 0 {
		minRank := Rank{0, 0, 0}
		minIdx := -1

		for idx, partialResult := range partialResults {
			if len(partialResult) > 0 {
				rank := partialResult[0].Rank()
				if minIdx < 0 || compareRanks(rank, minRank) {
					minRank = rank
					minIdx = idx
				}
			}
		}

		if minIdx >= 0 {
			merged = append(merged, partialResults[minIdx][0])
			partialResults[minIdx] = partialResults[minIdx][1:]
		}

		nonEmptyPartialResults := make([][]*Item, 0, len(partialResults))
		for _, partialResult := range partialResults {
			if len(partialResult) > 0 {
				nonEmptyPartialResults = append(nonEmptyPartialResults, partialResult)
			}
		}
		partialResults = nonEmptyPartialResults
	}

	return merged
}
