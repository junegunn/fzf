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
	match  bool
	url    *url
}

func (co colorOffset) IsFullBgMarker(at int32) bool {
	return at == co.offset[0] && at == co.offset[1] && co.color.Attr()&tui.FullBg > 0
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
			minBegin = min(b, minBegin)
			minEnd = min(e, minEnd)
			maxEnd = max(e, maxEnd)
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
			if validOffsetFound {
				b := minBegin
				e := maxEnd
				for ; b >= 1; b-- {
					if unicode.IsSpace(item.text.Get(b - 1)) {
						break
					}
				}
				for ; e < numChars; e++ {
					if unicode.IsSpace(item.text.Get(e)) {
						break
					}
				}
				val = util.AsUint16(e - b)
			}
		case byLength:
			val = item.TrimLength()
		case byPathname:
			if validOffsetFound {
				// lastDelim := strings.LastIndexByte(item.text.ToString(), '/')
				lastDelim := -1
				s := item.text.ToString()
				for i := len(s) - 1; i >= 0; i-- {
					if s[i] == '/' || s[i] == '\\' {
						lastDelim = i
						break
					}
				}
				if lastDelim <= minBegin {
					val = util.AsUint16(minBegin - lastDelim)
				}
			}
		case byBegin, byEnd:
			if validOffsetFound {
				whitePrefixLen := 0
				for idx := range numChars {
					r := item.text.Get(idx)
					whitePrefixLen = idx
					if idx == minBegin || !unicode.IsSpace(r) {
						break
					}
				}
				if criterion == byBegin {
					val = util.AsUint16(minEnd - whitePrefixLen)
				} else {
					val = util.AsUint16(math.MaxUint16 - math.MaxUint16*(maxEnd-whitePrefixLen)/(int(item.TrimLength())+1))
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

func (result *Result) colorOffsets(matchOffsets []Offset, nthOffsets []Offset, theme *tui.ColorTheme, colBase tui.ColorPair, colMatch tui.ColorPair, attrNth tui.Attr, hidden bool) []colorOffset {
	itemColors := result.item.Colors()

	// No ANSI codes
	if len(itemColors) == 0 && len(nthOffsets) == 0 {
		offsets := make([]colorOffset, len(matchOffsets))
		for i, off := range matchOffsets {
			offsets[i] = colorOffset{offset: [2]int32{off[0], off[1]}, color: colMatch, match: true}
		}
		return offsets
	}

	// Find max column
	var maxCol int32
	for _, off := range append(matchOffsets, nthOffsets...) {
		if off[1] > maxCol {
			maxCol = off[1]
		}
	}
	for _, ansi := range itemColors {
		if ansi.offset[1] > maxCol {
			maxCol = ansi.offset[1]
		}
	}

	type cellInfo struct {
		index int
		color bool
		match bool
		nth   bool
		fbg   tui.Color
	}

	cols := make([]cellInfo, maxCol+1)
	for idx := range cols {
		cols[idx].fbg = -1
	}
	for colorIndex, ansi := range itemColors {
		if ansi.offset[0] == ansi.offset[1] && ansi.color.attr&tui.FullBg > 0 {
			cols[ansi.offset[0]].fbg = ansi.color.lbg
		} else {
			for i := ansi.offset[0]; i < ansi.offset[1]; i++ {
				cols[i] = cellInfo{colorIndex, true, false, false, cols[i].fbg}
			}
		}
	}

	for _, off := range matchOffsets {
		for i := off[0]; i < off[1]; i++ {
			cols[i].match = true
		}
	}

	for _, off := range nthOffsets {
		for i := off[0]; i < off[1]; i++ {
			cols[i].nth = true
		}
	}

	// sort.Sort(ByOrder(offsets))

	// Merge offsets
	// ------------  ----  --  ----
	//   ++++++++      ++++++++++
	// --++++++++--  --++++++++++---
	curr := cellInfo{0, false, false, false, -1}
	start := 0
	ansiToColorPair := func(ansi ansiOffset, base tui.ColorPair) tui.ColorPair {
		if !theme.Colored {
			return tui.NewColorPair(-1, -1, ansi.color.attr).MergeAttr(base)
		}
		// fd --color always | fzf --ansi --delimiter / --nth -1 --color fg:dim:strip,nth:regular
		if base.ShouldStripColors() {
			return base
		}
		fg := ansi.color.fg
		bg := ansi.color.bg
		if fg == -1 {
			fg = colBase.Fg()
		}
		if bg == -1 {
			bg = colBase.Bg()
		}
		return tui.NewColorPair(fg, bg, ansi.color.attr).MergeAttr(base)
	}
	var colors []colorOffset
	add := func(idx int) {
		if curr.fbg >= 0 {
			colors = append(colors, colorOffset{
				offset: [2]int32{int32(start), int32(start)},
				color:  tui.NewColorPair(-1, curr.fbg, tui.FullBg),
				match:  false,
				url:    nil})
		}
		if (curr.color || curr.nth || curr.match) && idx > start {
			if curr.match {
				var color tui.ColorPair
				if curr.nth {
					color = colBase.WithAttr(attrNth).Merge(colMatch)
				} else {
					color = colBase.Merge(colMatch)
				}
				var url *url
				if curr.color {
					ansi := itemColors[curr.index]
					url = ansi.color.url
					origColor := ansiToColorPair(ansi, colMatch)
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
						if curr.nth {
							color = color.WithAttr(attrNth &^ tui.AttrRegular)
						}
					} else {
						color = origColor.MergeNonDefault(color)
					}
				}
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)}, color: color, match: true, url: url})
			} else if curr.color {
				ansi := itemColors[curr.index]
				base := colBase
				if curr.nth {
					base = base.WithAttr(attrNth)
				}
				if hidden {
					base = base.WithFg(theme.Nomatch)
				}
				color := ansiToColorPair(ansi, base)
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)},
					color:  color,
					match:  false,
					url:    ansi.color.url})
			} else {
				color := colBase.WithAttr(attrNth)
				if hidden {
					color = color.WithFg(theme.Nomatch)
				}
				colors = append(colors, colorOffset{
					offset: [2]int32{int32(start), int32(idx)},
					color:  color,
					match:  false,
					url:    nil})
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
