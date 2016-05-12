package fzf

import (
	"fmt"
	"testing"

	"github.com/junegunn/fzf/src/curses"
)

func TestDelimiterRegex(t *testing.T) {
	// Valid regex
	delim := delimiterRegexp(".")
	if delim.regex == nil || delim.str != nil {
		t.Error(delim)
	}
	// Broken regex -> string
	delim = delimiterRegexp("[0-9")
	if delim.regex != nil || *delim.str != "[0-9" {
		t.Error(delim)
	}
	// Valid regex
	delim = delimiterRegexp("[0-9]")
	if delim.regex.String() != "[0-9]" || delim.str != nil {
		t.Error(delim)
	}
	// Tab character
	delim = delimiterRegexp("\t")
	if delim.regex != nil || *delim.str != "\t" {
		t.Error(delim)
	}
	// Tab expression
	delim = delimiterRegexp("\\t")
	if delim.regex != nil || *delim.str != "\t" {
		t.Error(delim)
	}
	// Tabs -> regex
	delim = delimiterRegexp("\t+")
	if delim.regex == nil || delim.str != nil {
		t.Error(delim)
	}
}

func TestDelimiterRegexString(t *testing.T) {
	delim := delimiterRegexp("*")
	tokens := Tokenize([]rune("-*--*---**---"), delim)
	if delim.regex != nil ||
		string(tokens[0].text) != "-*" ||
		string(tokens[1].text) != "--*" ||
		string(tokens[2].text) != "---*" ||
		string(tokens[3].text) != "*" ||
		string(tokens[4].text) != "---" {
		t.Errorf("%s %s %d", delim, tokens, len(tokens))
	}
}

func TestDelimiterRegexRegex(t *testing.T) {
	delim := delimiterRegexp("--\\*")
	tokens := Tokenize([]rune("-*--*---**---"), delim)
	if delim.str != nil ||
		string(tokens[0].text) != "-*--*" ||
		string(tokens[1].text) != "---*" ||
		string(tokens[2].text) != "*---" {
		t.Errorf("%s %d", tokens, len(tokens))
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
		postProcessOptions(opts)
		if len(opts.Nth) != 0 {
			t.Errorf("nth should be empty: %s", opts.Nth)
		}
	}
	for _, words := range [][]string{[]string{"--nth", "..,3", "+x"}, []string{"--nth", "3,1..", "+x"}, []string{"--nth", "..-1,1", "+x"}} {
		{
			opts := defaultOptions()
			parseOptions(opts, words)
			postProcessOptions(opts)
			if len(opts.Nth) != 0 {
				t.Errorf("nth should be empty: %s", opts.Nth)
			}
		}
		{
			opts := defaultOptions()
			words = append(words, "-x")
			parseOptions(opts, words)
			postProcessOptions(opts)
			if len(opts.Nth) != 2 {
				t.Errorf("nth should not be empty: %s", opts.Nth)
			}
		}
	}
}

func TestParseKeys(t *testing.T) {
	pairs := parseKeyChords("ctrl-z,alt-z,f2,@,Alt-a,!,ctrl-G,J,g,ALT-enter,alt-SPACE", "")
	check := func(i int, s string) {
		if pairs[i] != s {
			t.Errorf("%s != %s", pairs[i], s)
		}
	}
	if len(pairs) != 11 {
		t.Error(11)
	}
	check(curses.CtrlZ, "ctrl-z")
	check(curses.AltZ, "alt-z")
	check(curses.F2, "f2")
	check(curses.AltZ+'@', "@")
	check(curses.AltA, "Alt-a")
	check(curses.AltZ+'!', "!")
	check(curses.CtrlA+'g'-'a', "ctrl-G")
	check(curses.AltZ+'J', "J")
	check(curses.AltZ+'g', "g")
	check(curses.AltEnter, "ALT-enter")
	check(curses.AltSpace, "alt-SPACE")

	// Synonyms
	pairs = parseKeyChords("enter,Return,space,tab,btab,esc,up,down,left,right", "")
	if len(pairs) != 9 {
		t.Error(9)
	}
	check(curses.CtrlM, "Return")
	check(curses.AltZ+' ', "space")
	check(curses.Tab, "tab")
	check(curses.BTab, "btab")
	check(curses.ESC, "esc")
	check(curses.Up, "up")
	check(curses.Down, "down")
	check(curses.Left, "left")
	check(curses.Right, "right")

	pairs = parseKeyChords("Tab,Ctrl-I,PgUp,page-up,pgdn,Page-Down,Home,End,Alt-BS,Alt-BSpace,shift-left,shift-right,btab,shift-tab,return,Enter,bspace", "")
	if len(pairs) != 11 {
		t.Error(11)
	}
	check(curses.Tab, "Ctrl-I")
	check(curses.PgUp, "page-up")
	check(curses.PgDn, "Page-Down")
	check(curses.Home, "Home")
	check(curses.End, "End")
	check(curses.AltBS, "Alt-BSpace")
	check(curses.SLeft, "shift-left")
	check(curses.SRight, "shift-right")
	check(curses.BTab, "shift-tab")
	check(curses.CtrlM, "Enter")
	check(curses.BSpace, "bspace")
}

func TestParseKeysWithComma(t *testing.T) {
	checkN := func(a int, b int) {
		if a != b {
			t.Errorf("%d != %d", a, b)
		}
	}
	check := func(pairs map[int]string, i int, s string) {
		if pairs[i] != s {
			t.Errorf("%s != %s", pairs[i], s)
		}
	}

	pairs := parseKeyChords(",", "")
	checkN(len(pairs), 1)
	check(pairs, curses.AltZ+',', ",")

	pairs = parseKeyChords(",,a,b", "")
	checkN(len(pairs), 3)
	check(pairs, curses.AltZ+'a', "a")
	check(pairs, curses.AltZ+'b', "b")
	check(pairs, curses.AltZ+',', ",")

	pairs = parseKeyChords("a,b,,", "")
	checkN(len(pairs), 3)
	check(pairs, curses.AltZ+'a', "a")
	check(pairs, curses.AltZ+'b', "b")
	check(pairs, curses.AltZ+',', ",")

	pairs = parseKeyChords("a,,,b", "")
	checkN(len(pairs), 3)
	check(pairs, curses.AltZ+'a', "a")
	check(pairs, curses.AltZ+'b', "b")
	check(pairs, curses.AltZ+',', ",")

	pairs = parseKeyChords("a,,,b,c", "")
	checkN(len(pairs), 4)
	check(pairs, curses.AltZ+'a', "a")
	check(pairs, curses.AltZ+'b', "b")
	check(pairs, curses.AltZ+'c', "c")
	check(pairs, curses.AltZ+',', ",")

	pairs = parseKeyChords(",,,", "")
	checkN(len(pairs), 1)
	check(pairs, curses.AltZ+',', ",")
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
	parseKeymap(keymap, execmap,
		"ctrl-a:kill-line,ctrl-b:toggle-sort,c:page-up,alt-z:page-down,"+
			"f1:execute(ls {}),f2:execute/echo {}, {}, {}/,f3:execute[echo '({})'],f4:execute;less {};,"+
			"alt-a:execute@echo (,),[,],/,:,;,%,{}@,alt-b:execute;echo (,),[,],/,:,@,%,{};"+
			",,:abort,::accept,X:execute:\nfoobar,Y:execute(baz)")
	check(actKillLine, keymap[curses.CtrlA])
	check(actToggleSort, keymap[curses.CtrlB])
	check(actPageUp, keymap[curses.AltZ+'c'])
	check(actAbort, keymap[curses.AltZ+','])
	check(actAccept, keymap[curses.AltZ+':'])
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
		parseKeymap(keymap, execmap, fmt.Sprintf("%d:execute%cfoobar%c", idx%10, char, char))
		checkString("foobar", execmap[curses.AltZ+int([]rune(fmt.Sprintf("%d", idx%10))[0])])
	}

	parseKeymap(keymap, execmap, "f1:abort")
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

func TestParseNilTheme(t *testing.T) {
	var theme *curses.ColorTheme
	newTheme := parseTheme(theme, "prompt:12")
	if newTheme != nil {
		t.Errorf("color is disabled. keep it that way.")
	}
	newTheme = parseTheme(theme, "prompt:12,dark,prompt:13")
	if newTheme.Prompt != 13 {
		t.Errorf("color should now be enabled and customized")
	}
}

func TestDefaultCtrlNP(t *testing.T) {
	check := func(words []string, key int, expected actionType) {
		opts := defaultOptions()
		parseOptions(opts, words)
		postProcessOptions(opts)
		if opts.Keymap[key] != expected {
			t.Error()
		}
	}
	check([]string{}, curses.CtrlN, actDown)
	check([]string{}, curses.CtrlP, actUp)

	check([]string{"--bind=ctrl-n:accept"}, curses.CtrlN, actAccept)
	check([]string{"--bind=ctrl-p:accept"}, curses.CtrlP, actAccept)

	hist := "--history=/tmp/foo"
	check([]string{hist}, curses.CtrlN, actNextHistory)
	check([]string{hist}, curses.CtrlP, actPreviousHistory)

	check([]string{hist, "--bind=ctrl-n:accept"}, curses.CtrlN, actAccept)
	check([]string{hist, "--bind=ctrl-n:accept"}, curses.CtrlP, actPreviousHistory)

	check([]string{hist, "--bind=ctrl-p:accept"}, curses.CtrlN, actNextHistory)
	check([]string{hist, "--bind=ctrl-p:accept"}, curses.CtrlP, actAccept)
}

func TestToggle(t *testing.T) {
	optsFor := func(words ...string) *Options {
		opts := defaultOptions()
		parseOptions(opts, words)
		postProcessOptions(opts)
		return opts
	}

	opts := optsFor()
	if opts.ToggleSort {
		t.Error()
	}

	opts = optsFor("--bind=a:toggle-sort")
	if !opts.ToggleSort {
		t.Error()
	}

	opts = optsFor("--bind=a:toggle-sort", "--bind=a:up")
	if opts.ToggleSort {
		t.Error()
	}
}
