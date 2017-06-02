package fzf

import (
	"github.com/junegunn/fzf/src/util"
)

// Item represents each input line
type Item struct {
	index       int32
	trimLength  int32
	text        util.Chars
	origText    *[]byte
	colors      *[]ansiOffset
	transformed []Token
}

// Index returns ordinal index of the Item
func (item *Item) Index() int32 {
	return item.index
}

func (item *Item) TrimLength() int32 {
	if item.trimLength >= 0 {
		return item.trimLength
	}
	item.trimLength = int32(item.text.TrimLength())
	return item.trimLength
}

// Colors returns ansiOffsets of the Item
func (item *Item) Colors() []ansiOffset {
	if item.colors == nil {
		return []ansiOffset{}
	}
	return *item.colors
}

// AsString returns the original string
func (item *Item) AsString(stripAnsi bool) string {
	if item.origText != nil {
		if stripAnsi {
			trimmed, _, _ := extractColor(string(*item.origText), nil, nil)
			return trimmed
		}
		return string(*item.origText)
	}
	return item.text.ToString()
}
