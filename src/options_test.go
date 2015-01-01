package fzf

import "testing"

func TestDelimiterRegex(t *testing.T) {
	rx := delimiterRegexp("*")
	tokens := rx.FindAllString("-*--*---**---", -1)
	if tokens[0] != "-*" || tokens[1] != "--*" || tokens[2] != "---*" ||
		tokens[3] != "*" || tokens[4] != "---" {
		t.Errorf("%s %s %d", rx, tokens, len(tokens))
	}
}

func TestSplitNth(t *testing.T) {
	{
		ranges := splitNth("..")
		if len(ranges) != 1 ||
			ranges[0].begin != RANGE_ELLIPSIS ||
			ranges[0].end != RANGE_ELLIPSIS {
			t.Errorf("%s", ranges)
		}
	}
	{
		ranges := splitNth("..3,1..,2..3,4..-1,-3..-2,..,2,-2")
		if len(ranges) != 8 ||
			ranges[0].begin != RANGE_ELLIPSIS || ranges[0].end != 3 ||
			ranges[1].begin != 1 || ranges[1].end != RANGE_ELLIPSIS ||
			ranges[2].begin != 2 || ranges[2].end != 3 ||
			ranges[3].begin != 4 || ranges[3].end != -1 ||
			ranges[4].begin != -3 || ranges[4].end != -2 ||
			ranges[5].begin != RANGE_ELLIPSIS || ranges[5].end != RANGE_ELLIPSIS ||
			ranges[6].begin != 2 || ranges[6].end != 2 ||
			ranges[7].begin != -2 || ranges[7].end != -2 {
			t.Errorf("%s", ranges)
		}
	}
}
