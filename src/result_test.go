package fzf

import (
	"math"
	"sort"
	"testing"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

func withIndex(i *Item, index int) *Item {
	(*i).text.Index = int32(index)
	return i
}

func TestOffsetSort(t *testing.T) {
	offsets := []Offset{
		{3, 5}, {2, 7},
		{1, 3}, {2, 9}}
	sort.Sort(ByOrder(offsets))

	if offsets[0][0] != 1 || offsets[0][1] != 3 ||
		offsets[1][0] != 2 || offsets[1][1] != 7 ||
		offsets[2][0] != 2 || offsets[2][1] != 9 ||
		offsets[3][0] != 3 || offsets[3][1] != 5 {
		t.Error("Invalid order:", offsets)
	}
}

func TestRankComparison(t *testing.T) {
	rank := func(vals ...uint16) Result {
		return Result{
			points: [4]uint16{vals[0], vals[1], vals[2], vals[3]},
			item:   &Item{text: util.Chars{Index: int32(vals[4])}}}
	}
	if compareRanks(rank(3, 0, 0, 0, 5), rank(2, 0, 0, 0, 7), false) ||
		!compareRanks(rank(3, 0, 0, 0, 5), rank(3, 0, 0, 0, 6), false) ||
		!compareRanks(rank(1, 2, 0, 0, 3), rank(1, 3, 0, 0, 2), false) ||
		!compareRanks(rank(0, 0, 0, 0, 0), rank(0, 0, 0, 0, 0), false) {
		t.Error("Invalid order")
	}

	if compareRanks(rank(3, 0, 0, 0, 5), rank(2, 0, 0, 0, 7), true) ||
		!compareRanks(rank(3, 0, 0, 0, 5), rank(3, 0, 0, 0, 6), false) ||
		!compareRanks(rank(1, 2, 0, 0, 3), rank(1, 3, 0, 0, 2), true) ||
		!compareRanks(rank(0, 0, 0, 0, 0), rank(0, 0, 0, 0, 0), false) {
		t.Error("Invalid order (tac)")
	}
}

// Match length, string length, index
func TestResultRank(t *testing.T) {
	// FIXME global
	sortCriteria = []criterion{byScore, byLength}

	str := []rune("foo")
	item1 := buildResult(
		withIndex(&Item{text: util.RunesToChars(str)}, 1), []Offset{}, 2)
	if item1.points[3] != math.MaxUint16-2 || // Bonus
		item1.points[2] != 3 || // Length
		item1.points[1] != 0 || // Unused
		item1.points[0] != 0 || // Unused
		item1.item.Index() != 1 {
		t.Error(item1)
	}
	// Only differ in index
	item2 := buildResult(&Item{text: util.RunesToChars(str)}, []Offset{}, 2)

	items := []Result{item1, item2}
	sort.Sort(ByRelevance(items))
	if items[0] != item2 || items[1] != item1 {
		t.Error(items)
	}

	items = []Result{item2, item1, item1, item2}
	sort.Sort(ByRelevance(items))
	if items[0] != item2 || items[1] != item2 ||
		items[2] != item1 || items[3] != item1 {
		t.Error(items, item1, item1.item.Index(), item2, item2.item.Index())
	}

	// Sort by relevance
	item3 := buildResult(
		withIndex(&Item{}, 2), []Offset{{1, 3}, {5, 7}}, 3)
	item4 := buildResult(
		withIndex(&Item{}, 2), []Offset{{1, 2}, {6, 7}}, 4)
	item5 := buildResult(
		withIndex(&Item{}, 2), []Offset{{1, 3}, {5, 7}}, 5)
	item6 := buildResult(
		withIndex(&Item{}, 2), []Offset{{1, 2}, {6, 7}}, 6)
	items = []Result{item1, item2, item3, item4, item5, item6}
	sort.Sort(ByRelevance(items))
	if !(items[0] == item6 && items[1] == item5 &&
		items[2] == item4 && items[3] == item3 &&
		items[4] == item2 && items[5] == item1) {
		t.Error(items, item1, item2, item3, item4, item5, item6)
	}
}

func TestChunkTiebreak(t *testing.T) {
	// FIXME global
	sortCriteria = []criterion{byScore, byChunk}

	score := 100
	test := func(input string, offset Offset, chunk string) {
		item := buildResult(withIndex(&Item{text: util.RunesToChars([]rune(input))}, 1), []Offset{offset}, score)
		if !(item.points[3] == math.MaxUint16-uint16(score) && item.points[2] == uint16(len(chunk))) {
			t.Error(item.points)
		}
	}
	test("hello foobar goodbye", Offset{8, 9}, "foobar")
	test("hello foobar goodbye", Offset{7, 18}, "foobar goodbye")
	test("hello foobar goodbye", Offset{0, 1}, "hello")
	test("hello foobar goodbye", Offset{5, 7}, "hello foobar") // TBD
}

func TestColorOffset(t *testing.T) {
	// ------------ 20 ----  --  ----
	//   ++++++++        ++++++++++
	// --++++++++--    --++++++++++---

	offsets := []Offset{{5, 15}, {10, 12}, {25, 35}}
	item := Result{
		item: &Item{
			colors: &[]ansiOffset{
				{[2]int32{0, 20}, ansiState{1, 5, 0, -1}},
				{[2]int32{22, 27}, ansiState{2, 6, tui.Bold, -1}},
				{[2]int32{30, 32}, ansiState{3, 7, 0, -1}},
				{[2]int32{33, 40}, ansiState{4, 8, tui.Bold, -1}}}}}

	colBase := tui.NewColorPair(89, 189, tui.AttrUndefined)
	colMatch := tui.NewColorPair(99, 199, tui.AttrUndefined)
	colors := item.colorOffsets(offsets, tui.Dark256, colBase, colMatch, true)
	assert := func(idx int, b int32, e int32, c tui.ColorPair) {
		o := colors[idx]
		if o.offset[0] != b || o.offset[1] != e || o.color != c {
			t.Error(o, b, e, c)
		}
	}
	// [{[0 5] {1 5 0}} {[5 15] {99 199 0}} {[15 20] {1 5 0}}
	//  {[22 25] {2 6 1}} {[25 27] {99 199 1}} {[27 30] {99 199 0}}
	//  {[30 32] {99 199 0}} {[32 33] {99 199 0}} {[33 35] {99 199 1}}
	//  {[35 40] {4 8 1}}]
	assert(0, 0, 5, tui.NewColorPair(1, 5, tui.AttrUndefined))
	assert(1, 5, 15, colMatch)
	assert(2, 15, 20, tui.NewColorPair(1, 5, tui.AttrUndefined))
	assert(3, 22, 25, tui.NewColorPair(2, 6, tui.Bold))
	assert(4, 25, 27, colMatch.WithAttr(tui.Bold))
	assert(5, 27, 30, colMatch)
	assert(6, 30, 32, colMatch)
	assert(7, 32, 33, colMatch) // TODO: Should we merge consecutive blocks?
	assert(8, 33, 35, colMatch.WithAttr(tui.Bold))
	assert(9, 35, 40, tui.NewColorPair(4, 8, tui.Bold))

	colRegular := tui.NewColorPair(-1, -1, tui.AttrUndefined)
	colUnderline := tui.NewColorPair(-1, -1, tui.Underline)
	colors = item.colorOffsets(offsets, tui.Dark256, colRegular, colUnderline, true)

	// [{[0 5] {1 5 0}} {[5 15] {1 5 8}} {[15 20] {1 5 0}}
	//  {[22 25] {2 6 1}} {[25 27] {2 6 9}} {[27 30] {-1 -1 8}}
	//  {[30 32] {3 7 8}} {[32 33] {-1 -1 8}} {[33 35] {4 8 9}}
	//  {[35 40] {4 8 1}}]
	assert(0, 0, 5, tui.NewColorPair(1, 5, tui.AttrUndefined))
	assert(1, 5, 15, tui.NewColorPair(1, 5, tui.Underline))
	assert(2, 15, 20, tui.NewColorPair(1, 5, tui.AttrUndefined))
	assert(3, 22, 25, tui.NewColorPair(2, 6, tui.Bold))
	assert(4, 25, 27, tui.NewColorPair(2, 6, tui.Bold|tui.Underline))
	assert(5, 27, 30, colUnderline)
	assert(6, 30, 32, tui.NewColorPair(3, 7, tui.Underline))
	assert(7, 32, 33, colUnderline)
	assert(8, 33, 35, tui.NewColorPair(4, 8, tui.Bold|tui.Underline))
	assert(9, 35, 40, tui.NewColorPair(4, 8, tui.Bold))
}
