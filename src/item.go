package fzf

// Offset holds two 32-bit integers denoting the offsets of a matched substring
type Offset [2]int32

// Item represents each input line
type Item struct {
	text        *string
	origText    *string
	transformed *Transformed
	index       uint32
	offsets     []Offset
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
