package fzf

import (
	"sort"
	"testing"
)

func TestOffsetSort(t *testing.T) {
	offsets := []Offset{
		Offset{3, 5}, Offset{2, 7},
		Offset{1, 3}, Offset{2, 9}}
	sort.Sort(ByOrder(offsets))

	if offsets[0][0] != 1 || offsets[0][1] != 3 ||
		offsets[1][0] != 2 || offsets[1][1] != 7 ||
		offsets[2][0] != 2 || offsets[2][1] != 9 ||
		offsets[3][0] != 3 || offsets[3][1] != 5 {
		t.Error("Invalid order:", offsets)
	}
}

func TestRankComparison(t *testing.T) {
	if compareRanks(Rank{3, 0, 5}, Rank{2, 0, 7}) ||
		!compareRanks(Rank{3, 0, 5}, Rank{3, 0, 6}) ||
		!compareRanks(Rank{1, 2, 3}, Rank{1, 3, 2}) ||
		!compareRanks(Rank{0, 0, 0}, Rank{0, 0, 0}) {
		t.Error("Invalid order")
	}
}

// Match length, string length, index
func TestItemRank(t *testing.T) {
	strs := []string{"foo", "foobar", "bar", "baz"}
	item1 := Item{text: &strs[0], rank: Rank{0, 0, 1}, offsets: []Offset{}}
	rank1 := item1.Rank()
	if rank1.matchlen != 0 || rank1.strlen != 3 || rank1.index != 1 {
		t.Error(item1.Rank())
	}
	// Only differ in index
	item2 := Item{text: &strs[0], rank: Rank{0, 0, 0}, offsets: []Offset{}}

	items := []*Item{&item1, &item2}
	sort.Sort(ByRelevance(items))
	if items[0] != &item2 || items[1] != &item1 {
		t.Error(items)
	}

	items = []*Item{&item2, &item1, &item1, &item2}
	sort.Sort(ByRelevance(items))
	if items[0] != &item2 || items[1] != &item2 ||
		items[2] != &item1 || items[3] != &item1 {
		t.Error(items)
	}

	// Sort by relevance
	item3 := Item{text: &strs[1], rank: Rank{0, 0, 2}, offsets: []Offset{Offset{1, 3}, Offset{5, 7}}}
	item4 := Item{text: &strs[1], rank: Rank{0, 0, 2}, offsets: []Offset{Offset{1, 2}, Offset{6, 7}}}
	item5 := Item{text: &strs[2], rank: Rank{0, 0, 2}, offsets: []Offset{Offset{1, 3}, Offset{5, 7}}}
	item6 := Item{text: &strs[2], rank: Rank{0, 0, 2}, offsets: []Offset{Offset{1, 2}, Offset{6, 7}}}
	items = []*Item{&item1, &item2, &item3, &item4, &item5, &item6}
	sort.Sort(ByRelevance(items))
	if items[0] != &item2 || items[1] != &item1 ||
		items[2] != &item6 || items[3] != &item4 ||
		items[4] != &item5 || items[5] != &item3 {
		t.Error(items)
	}
}
