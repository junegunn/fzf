package fzf

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

func assert(t *testing.T, cond bool, msg ...string) {
	if !cond {
		t.Error(msg)
	}
}

func randResult() Result {
	str := fmt.Sprintf("%d", rand.Uint32())
	chars := util.ToChars([]byte(str))
	chars.Index = rand.Int31()
	return Result{item: &Item{text: chars}}
}

func TestEmptyMerger(t *testing.T) {
	assert(t, EmptyMerger(0).Length() == 0, "Not empty")
	assert(t, EmptyMerger(0).count == 0, "Invalid count")
	assert(t, len(EmptyMerger(0).lists) == 0, "Invalid lists")
	assert(t, len(EmptyMerger(0).merged) == 0, "Invalid merged list")
}

func buildLists(partiallySorted bool) ([][]Result, []Result) {
	numLists := 4
	lists := make([][]Result, numLists)
	cnt := 0
	for i := 0; i < numLists; i++ {
		numResults := rand.Int() % 20
		cnt += numResults
		lists[i] = make([]Result, numResults)
		for j := 0; j < numResults; j++ {
			item := randResult()
			lists[i][j] = item
		}
		if partiallySorted {
			sort.Sort(ByRelevance(lists[i]))
		}
	}
	items := []Result{}
	for _, list := range lists {
		items = append(items, list...)
	}
	return lists, items
}

func TestMergerUnsorted(t *testing.T) {
	lists, items := buildLists(false)
	cnt := len(items)

	// Not sorted: same order
	mg := NewMerger(nil, lists, false, false, 0)
	assert(t, cnt == mg.Length(), "Invalid Length")
	for i := 0; i < cnt; i++ {
		assert(t, items[i] == mg.Get(i), "Invalid Get")
	}
}

func TestMergerSorted(t *testing.T) {
	lists, items := buildLists(true)
	cnt := len(items)

	// Sorted sorted order
	mg := NewMerger(nil, lists, true, false, 0)
	assert(t, cnt == mg.Length(), "Invalid Length")
	sort.Sort(ByRelevance(items))
	for i := 0; i < cnt; i++ {
		if items[i] != mg.Get(i) {
			t.Error("Not sorted", items[i], mg.Get(i))
		}
	}

	// Inverse order
	mg2 := NewMerger(nil, lists, true, false, 0)
	for i := cnt - 1; i >= 0; i-- {
		if items[i] != mg2.Get(i) {
			t.Error("Not sorted", items[i], mg2.Get(i))
		}
	}
}
