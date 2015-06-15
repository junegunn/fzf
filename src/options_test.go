package fzf

import (
	"fmt"
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
	keys := parseKeyChords("ctrl-z,alt-z,f2,@,Alt-a,!,ctrl-G,J,g", "", false)
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

	// Synonyms
	keys = parseKeyChords("enter,return,space,tab,btab,esc,up,down,left,right", "", true)
	check(len(keys), 10)
	check(keys[0], curses.CtrlM)
	check(keys[1], curses.CtrlM)
	check(keys[2], curses.AltZ+' ')
	check(keys[3], curses.Tab)
	check(keys[4], curses.BTab)
	check(keys[5], curses.ESC)
	check(keys[6], curses.Up)
	check(keys[7], curses.Down)
	check(keys[8], curses.Left)
	check(keys[9], curses.Right)
}

func TestParseKeysWithComma(t *testing.T) {
	check := func(key int, expected int) {
		if key != expected {
			t.Errorf("%d != %d", key, expected)
		}
	}

	keys := parseKeyChords(",", "", false)
	check(len(keys), 1)
	check(keys[0], curses.AltZ+',')

	keys = parseKeyChords(",,a,b", "", false)
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,b,,", "", false)
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,,,b", "", false)
	check(len(keys), 3)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+',')

	keys = parseKeyChords("a,,,b,c", "", false)
	check(len(keys), 4)
	check(keys[0], curses.AltZ+'a')
	check(keys[1], curses.AltZ+'b')
	check(keys[2], curses.AltZ+'c')
	check(keys[3], curses.AltZ+',')

	keys = parseKeyChords(",,,", "", false)
	check(len(keys), 1)
	check(keys[0], curses.AltZ+',')
}

func TestBind(t *testing.T) {
	check := func(action actionType, expected actionType) {
		if action != expected {
			t.Errorf("%d != %d", action, expected)
		}
	}
	checkString := func(action string, expected string) {
		if action != expected {
			t.Errorf("%d != %d", action, expected)
		}
	}
	keymap := defaultKeymap()
	execmap := make(map[int]string)
	check(actBeginningOfLine, keymap[curses.CtrlA])
	keymap, execmap, toggleSort :=
		parseKeymap(keymap, execmap, false,
			"ctrl-a:kill-line,ctrl-b:toggle-sort,c:page-up,alt-z:page-down,"+
				"f1:execute(ls {}),f2:execute/echo {}, {}, {}/,f3:execute[echo '({})'],f4:execute;less {};,"+
				"alt-a:execute@echo (,),[,],/,:,;,%,{}@,alt-b:execute;echo (,),[,],/,:,@,%,{};"+
				",X:execute:\nfoobar,Y:execute(baz)")
	if !toggleSort {
		t.Errorf("toggleSort not set")
	}
	check(actKillLine, keymap[curses.CtrlA])
	check(actToggleSort, keymap[curses.CtrlB])
	check(actPageUp, keymap[curses.AltZ+'c'])
	check(actPageDown, keymap[curses.AltZ])
	check(actExecute, keymap[curses.F1])
	check(actExecute, keymap[curses.F2])
	check(actExecute, keymap[curses.F3])
	check(actExecute, keymap[curses.F4])
	checkString("ls {}", execmap[curses.F1])
	checkString("echo {}, {}, {}", execmap[curses.F2])
	checkString("echo '({})'", execmap[curses.F3])
	checkString("less {}", execmap[curses.F4])
	checkString("echo (,),[,],/,:,;,%,{}", execmap[curses.AltA])
	checkString("echo (,),[,],/,:,@,%,{}", execmap[curses.AltB])
	checkString("\nfoobar,Y:execute(baz)", execmap[curses.AltZ+'X'])

	for idx, char := range []rune{'~', '!', '@', '#', '$', '%', '^', '&', '*', '|', ';', '/'} {
		keymap, execmap, toggleSort =
			parseKeymap(keymap, execmap, false, fmt.Sprintf("%d:execute%cfoobar%c", idx%10, char, char))
		checkString("foobar", execmap[curses.AltZ+int([]rune(fmt.Sprintf("%d", idx%10))[0])])
	}

	keymap, execmap, toggleSort = parseKeymap(keymap, execmap, false, "f1:abort")
	if toggleSort {
		t.Errorf("toggleSort set")
	}
	check(actAbort, keymap[curses.F1])
}

func TestColorSpec(t *testing.T) {
	theme := curses.Dark256
	dark := parseTheme(theme, "dark")
	if *dark != *theme {
		t.Errorf("colors should be equivalent")
	}
	if dark == theme {
		t.Errorf("point should not be equivalent")
	}

	light := parseTheme(theme, "dark,light")
	if *light == *theme {
		t.Errorf("should not be equivalent")
	}
	if *light != *curses.Light256 {
		t.Errorf("colors should be equivalent")
	}
	if light == theme {
		t.Errorf("point should not be equivalent")
	}

	customized := parseTheme(theme, "fg:231,bg:232")
	if customized.Fg != 231 || customized.Bg != 232 {
		t.Errorf("color not customized")
	}
	if *curses.Dark256 == *customized {
		t.Errorf("colors should not be equivalent")
	}
	customized.Fg = curses.Dark256.Fg
	customized.Bg = curses.Dark256.Bg
	if *curses.Dark256 == *customized {
		t.Errorf("colors should now be equivalent")
	}

	customized = parseTheme(theme, "fg:231,dark,bg:232")
	if customized.Fg != curses.Dark256.Fg || customized.Bg == curses.Dark256.Bg {
		t.Errorf("color not customized")
	}
	if customized.UseDefault {
		t.Errorf("not using default colors")
	}
	if !curses.Dark256.UseDefault {
		t.Errorf("using default colors")
	}
}
