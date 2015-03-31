package fzf

import (
	"testing"

	"github.com/junegunn/fzf/src/curses"
)

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
			ranges[0].begin != rangeEllipsis ||
			ranges[0].end != rangeEllipsis {
			t.Errorf("%s", ranges)
		}
	}
	{
		ranges := splitNth("..3,1..,2..3,4..-1,-3..-2,..,2,-2,2..-2,1..-1")
		if len(ranges) != 10 ||
			ranges[0].begin != rangeEllipsis || ranges[0].end != 3 ||
			ranges[1].begin != rangeEllipsis || ranges[1].end != rangeEllipsis ||
			ranges[2].begin != 2 || ranges[2].end != 3 ||
			ranges[3].begin != 4 || ranges[3].end != rangeEllipsis ||
			ranges[4].begin != -3 || ranges[4].end != -2 ||
			ranges[5].begin != rangeEllipsis || ranges[5].end != rangeEllipsis ||
			ranges[6].begin != 2 || ranges[6].end != 2 ||
			ranges[7].begin != -2 || ranges[7].end != -2 ||
			ranges[8].begin != 2 || ranges[8].end != -2 ||
			ranges[9].begin != rangeEllipsis || ranges[9].end != rangeEllipsis {
			t.Errorf("%s", ranges)
		}
	}
}

func TestIrrelevantNth(t *testing.T) {
	{
		opts := defaultOptions()
		words := []string{"--nth", "..", "-x"}
		parseOptions(opts, words)
		if len(opts.Nth) != 0 {
			t.Errorf("nth should be empty: %s", opts.Nth)
		}
	}
	for _, words := range [][]string{[]string{"--nth", "..,3"}, []string{"--nth", "3,1.."}, []string{"--nth", "..-1,1"}} {
		{
			opts := defaultOptions()
			parseOptions(opts, words)
			if len(opts.Nth) != 0 {
				t.Errorf("nth should be empty: %s", opts.Nth)
			}
		}
		{
			opts := defaultOptions()
			words = append(words, "-x")
			parseOptions(opts, words)
			if len(opts.Nth) != 2 {
				t.Errorf("nth should not be empty: %s", opts.Nth)
			}
		}
	}
}

func TestParseKeys(t *testing.T) {
	keys := parseKeyChords("ctrl-z,alt-z,f2,@,Alt-a,!,ctrl-G,J,g", "")
	check := func(key int, expected int) {
		if key != expected {
			t.Errorf("%d != %d", key, expected)
		}
	}
	check(len(keys), 9)
	check(keys[0], curses.CtrlZ)
	check(keys[1], curses.AltZ)
	check(keys[2], curses.F2)
	check(keys[3], curses.AltZ+'@')
	check(keys[4], curses.AltA)
	check(keys[5], curses.AltZ+'!')
	check(keys[6], curses.CtrlA+'g'-'a')
	check(keys[7], curses.AltZ+'J')
	check(keys[8], curses.AltZ+'g')
}

func TestParseKeysWithComma(t *testing.T) {
	check := func(key int, expected int) {
		if key != expected {
			t.Errorf("%d != %d", key, expected)
		}
	}

	keys := parseKeyChords(",", "")
	check(len(keys), 1)
	check(keys[0], curses.AltZ+',')

	keys = parseKeyChords(",,a,b", "")
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,b,,", "")
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,,,b", "")
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,,,b,c", "")
	check(len(keys), 4)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+'c')
	check(keys[3], curses.AltZ+',')

	keys = parseKeyChords(",,,", "")
	check(len(keys), 1)
	check(keys[0], curses.AltZ+',')
}
