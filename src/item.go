package fzf

import (
	"fmt"
	"sort"
)

type Offset [2]int32

type Item struct {
	text        *string
	origText    *string
	transformed *Transformed
	offsets     []Offset
	rank        Rank
}

type Rank struct {
	matchlen uint16
	strlen   uint16
	index    uint32
}

func (i *Item) Rank() Rank {
	if i.rank.matchlen > 0 || i.rank.strlen > 0 {
		return i.rank
	}
	sort.Sort(ByOrder(i.offsets))
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
	i.rank = Rank{uint16(matchlen), uint16(len(*i.text)), i.rank.index}
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

	if irank.index <= jrank.index {
		return true
	}
	return false
}
