package fzf

import (
	"math"
	"sort"
	"unicode"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

// Offset holds two 32-bit integers denoting the offsets of a matched substring
type Offset [2]int32

type colorOffset struct {
	offset [2]int32
	color  tui.ColorPair
}

type Result struct {
	item   *Item
	points [4]uint16
}

func buildResult(item *Item, offsets []Offset, score int) Result {
	if len(offsets) > 1 {
		sort.Sort(ByOrder(offsets))
	}

	result := Result{item: item}
	numChars := item.text.Length()
	minBegin := math.MaxUint16
	minEnd := math.MaxUint16
	maxEnd := 0
	validOffsetFound := false
	for _, offset := range offsets {
		b, e := int(offset[0]), int(offset[1])
		if b < e {
			minBegin = util.Min(b, minBegin)
			minEnd = util.Min(e, minEnd)
			maxEnd = util.Max(e, maxEnd)
			validOffsetFound = true
		}
	}

	for idx, criterion := range sortCriteria {
		val := uint16(math.MaxUint16)
		switch criterion {
		case byScore:
			// Higher is better
			val = math.MaxUint16 - util.AsUint16(score)
		case byChunk:
			b := minBegin
			e := maxEnd
			l := item.text.Length()
			for ; b >= 1; b-- {
				if unicode.IsSpace(item.text.Get(b - 1)) {
					break
				}
			}
			for ; e < l; e++ {
				if unicode.IsSpace(item.text.Get(e)) {
					break
				}
			}
			val = util.AsUint16(e - b)
		case byLength:
			val = item.TrimLength()
		case byBegin, byEnd:
			if validOffsetFound {
				whitePrefixLen := 0
				for idx := 0; idx < numChars; idx++ {
					r := item.text.Get(idx)
					whitePrefixLen = idx
					if idx == minBegin || !unicode.IsSpace(r) {
						break
					}
				}
				if criterion == byBegin {
					val = util.AsUint16(minEnd - whitePrefixLen)
				} else {
					val = util.AsUint16(math.MaxUint16 - math.MaxUint16*(maxEnd-whitePrefixLen)/int(item.TrimLength()))
				}
			}
		}
		result.points[3-idx] = val
	}

	return result
}

// Sort criteria to use. Never changes once fzf is started.
var sortCriteria []criterion

// Index returns ordinal index of the Item
func (result *Result) Index() int32 {
	return result.item.Index()
}

func minRank() Result {
	return Result{item: &minItem, points: [4]uint16{math.MaxUint16, 0, 0, 0}}
}

func (result *Result) colorOffsets(matchOffsets []Offset, theme *tui.ColorTheme, colBase tui.ColorPair, colMatch tui.ColorPair, current bool) []colorOffset {
	itemColors := result.item.Colors()

	// No ANSI codes
	if len(itemColors) == 0 {
		var offsets []colorOffset
		for _, off := range matchOffsets {
			offsets = append(offsets, colorOffset{offset: [2]int32{off[0], off[1]}, color: colMatch})
		}
		return offsets
	}

	// Find max column
	var maxCol int32
	for _, off := range matchOffsets {
		if off[1] > maxCol {
			maxCol = off[1]
		}
	}
	for _, ansi := range itemColors {
		if ansi.offset[1] > maxCol {
			maxCol = ansi.offset[1]
		}
	}

	cols := make([]int, maxCol)
	for colorIndex, ansi := range itemColors {
		for i := ansi.offset[0]; i < ansi.offset[1]; i++ {
			cols[i] = colorIndex + 1 // 1-based index of itemColors
		}
	}

	for _, off := range matchOffsets {
		for i := off[0]; i < off[1]; i++ {
			// Negative of 1-based index of itemColors
			// - The extra -1 means highlighted
			cols[i] = cols[i]*-1 - 1
		}
	}

	// sort.Sort(ByOrder(offsets))

	// Merge offsets
	// ------------  ----  --  ----
	//   ++++++++      ++++++++++
	// --++++++++--  --++++++++++---
	curr := 0
	start := 0
	ansiToColorPair := func(ansi ansiOffset, base tui.ColorPair) tui.ColorPair {
		fg := ansi.color.fg
		bg := ansi.color.bg
		if fg == -1 {
			if current {
				fg = theme.Current.Color
			} else {
				fg = theme.Fg.Color
			}
		}
		if bg == -1 {
			if current {
				bg = theme.DarkBg.Color
			} else {
				bg = theme.Bg.Color
			}
		}
		return tui.NewColorPair(fg, bg, ansi.color.attr).MergeAttr(base)
	}
	var colors []colorOffset
	add := func(idx int) {
		if curr != 0 && idx > start {
			if curr < 0 {
				color := colMatch
				if curr < -1 && theme.Colored {
					origColor := ansiToColorPair(itemColors[-curr-2], colMatch)
					// hl or hl+ only sets the foreground color, so colMatch is the
					// combination of either [hl and bg] or [hl+ and bg+].
					//
					// If the original text already has background color, and the
					// foreground color of colMatch is -1, we shouldn't only apply the
					// background color of colMatch.
					// e.g. echo -e "\x1b[32;7mfoo\x1b[mbar" | fzf --ansi --color bg+:1,hl+:-1:underline
					//      echo -e "\x1b[42mfoo\x1b[mbar" | fzf --ansi --color bg+:1,hl+:-1:underline
					if color.Fg().IsDefault() && origColor.HasBg() {
						color = origColor
					} else {
						color = origColor.MergeNonDefault(color)
					}
				}
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)}, color: color})
			} else {
				ansi := itemColors[curr-1]
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)},
					color:  ansiToColorPair(ansi, colBase)})
			}
		}
	}
	for idx, col := range cols {
		if col != curr {
			add(idx)
			start = idx
			curr = col
		}
	}
	add(int(maxCol))
	return colors
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
type ByRelevance []Result

func (a ByRelevance) Len() int {
	return len(a)
}

func (a ByRelevance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevance) Less(i, j int) bool {
	return compareRanks(a[i], a[j], false)
}

// ByRelevanceTac is for sorting Items
type ByRelevanceTac []Result

func (a ByRelevanceTac) Len() int {
	return len(a)
}

func (a ByRelevanceTac) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByRelevanceTac) Less(i, j int) bool {
	return compareRanks(a[i], a[j], true)
}
