package fzf

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func assert(t *testing.T, cond bool, msg ...string) {
	if !cond {
		t.Error(msg)
	}
}

func randItem() *Item {
	str := fmt.Sprintf("%d", rand.Uint32())
	offsets := make([]Offset, rand.Int()%3)
	for idx := range offsets {
		sidx := int32(rand.Uint32() % 20)
		eidx := sidx + int32(rand.Uint32()%20)
		offsets[idx] = Offset{sidx, eidx}
	}
	return &Item{
		text:    []rune(str),
		rank:    buildEmptyRank(rand.Int31()),
		offsets: offsets}
}

func TestEmptyMerger(t *testing.T) {
	assert(t, EmptyMerger.Length() == 0, "Not empty")
	assert(t, EmptyMerger.count == 0, "Invalid count")
	assert(t, len(EmptyMerger.lists) == 0, "Invalid lists")
	assert(t, len(EmptyMerger.merged) == 0, "Invalid merged list")
}

func buildLists(partiallySorted bool) ([][]*Item, []*Item) {
	numLists := 4
	lists := make([][]*Item, numLists)
	cnt := 0
	for i := 0; i < numLists; i++ {
		numItems := rand.Int() % 20
		cnt += numItems
		lists[i] = make([]*Item, numItems)
		for j := 0; j < numItems; j++ {
			item := randItem()
			lists[i][j] = item
		}
		if partiallySorted {
			sort.Sort(ByRelevance(lists[i]))
		}
	}
	items := []*Item{}
	for _, list := range lists {
		items = append(items, list...)
	}
	return lists, items
}

func TestMergerUnsorted(t *testing.T) {
	lists, items := buildLists(false)
	cnt := len(items)

	// Not sorted: same order
	mg := NewMerger(lists, false, false)
	assert(t, cnt == mg.Length(), "Invalid Length")
	for i := 0; i < cnt; i++ {
		assert(t, items[i] == mg.Get(i), "Invalid Get")
	}
}

func TestMergerSorted(t *testing.T) {
	lists, items := buildLists(true)
	cnt := len(items)

	// Sorted sorted order
	mg := NewMerger(lists, true, false)
	assert(t, cnt == mg.Length(), "Invalid Length")
	sort.Sort(ByRelevance(items))
	for i := 0; i < cnt; i++ {
		if items[i] != mg.Get(i) {
			t.Error("Not sorted", items[i], mg.Get(i))
		}
	}

	// Inverse order
	mg2 := NewMerger(lists, true, false)
	for i := cnt - 1; i >= 0; i-- {
		if items[i] != mg2.Get(i) {
			t.Error("Not sorted", items[i], mg2.Get(i))
		}
	}
}
