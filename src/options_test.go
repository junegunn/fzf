package fzf

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/junegunn/fzf/src/tui"
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
	tokens := Tokenize("-*--*---**---", delim)
	if delim.regex != nil ||
		tokens[0].text.ToString() != "-*" ||
		tokens[1].text.ToString() != "--*" ||
		tokens[2].text.ToString() != "---*" ||
		tokens[3].text.ToString() != "*" ||
		tokens[4].text.ToString() != "---" {
		t.Errorf("%s %v %d", delim, tokens, len(tokens))
	}
}

func TestDelimiterRegexRegex(t *testing.T) {
	delim := delimiterRegexp("--\\*")
	tokens := Tokenize("-*--*---**---", delim)
	if delim.str != nil ||
		tokens[0].text.ToString() != "-*--*" ||
		tokens[1].text.ToString() != "---*" ||
		tokens[2].text.ToString() != "*---" {
		t.Errorf("%s %d", tokens, len(tokens))
	}
}

func TestSplitNth(t *testing.T) {
	{
		ranges := splitNth("..")
		if len(ranges) != 1 ||
			ranges[0].begin != rangeEllipsis ||
			ranges[0].end != rangeEllipsis {
			t.Errorf("%v", ranges)
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
			t.Errorf("%v", ranges)
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
			t.Errorf("nth should be empty: %v", opts.Nth)
		}
	}
	for _, words := range [][]string{[]string{"--nth", "..,3", "+x"}, []string{"--nth", "3,1..", "+x"}, []string{"--nth", "..-1,1", "+x"}} {
		{
			opts := defaultOptions()
			parseOptions(opts, words)
			postProcessOptions(opts)
			if len(opts.Nth) != 0 {
				t.Errorf("nth should be empty: %v", opts.Nth)
			}
		}
		{
			opts := defaultOptions()
			words = append(words, "-x")
			parseOptions(opts, words)
			postProcessOptions(opts)
			if len(opts.Nth) != 2 {
				t.Errorf("nth should not be empty: %v", opts.Nth)
			}
		}
	}
}

func TestParseKeys(t *testing.T) {
	pairs := parseKeyChords("ctrl-z,alt-z,f2,@,Alt-a,!,ctrl-G,J,g,ctrl-alt-a,ALT-enter,alt-SPACE", "")
	check := func(i int, s string) {
		if pairs[i] != s {
			t.Errorf("%s != %s", pairs[i], s)
		}
	}
	if len(pairs) != 12 {
		t.Error(12)
	}
	check(tui.CtrlZ, "ctrl-z")
	check(tui.AltZ, "alt-z")
	check(tui.F2, "f2")
	check(tui.AltZ+'@', "@")
	check(tui.AltA, "Alt-a")
	check(tui.AltZ+'!', "!")
	check(tui.CtrlA+'g'-'a', "ctrl-G")
	check(tui.AltZ+'J', "J")
	check(tui.AltZ+'g', "g")
	check(tui.CtrlAltA, "ctrl-alt-a")
	check(tui.CtrlAltM, "ALT-enter")
	check(tui.AltSpace, "alt-SPACE")

	// Synonyms
	pairs = parseKeyChords("enter,Return,space,tab,btab,esc,up,down,left,right", "")
	if len(pairs) != 9 {
		t.Error(9)
	}
	check(tui.CtrlM, "Return")
	check(tui.AltZ+' ', "space")
	check(tui.Tab, "tab")
	check(tui.BTab, "btab")
	check(tui.ESC, "esc")
	check(tui.Up, "up")
	check(tui.Down, "down")
	check(tui.Left, "left")
	check(tui.Right, "right")

	pairs = parseKeyChords("Tab,Ctrl-I,PgUp,page-up,pgdn,Page-Down,Home,End,Alt-BS,Alt-BSpace,shift-left,shift-right,btab,shift-tab,return,Enter,bspace", "")
	if len(pairs) != 11 {
		t.Error(11)
	}
	check(tui.Tab, "Ctrl-I")
	check(tui.PgUp, "page-up")
	check(tui.PgDn, "Page-Down")
	check(tui.Home, "Home")
	check(tui.End, "End")
	check(tui.AltBS, "Alt-BSpace")
	check(tui.SLeft, "shift-left")
	check(tui.SRight, "shift-right")
	check(tui.BTab, "shift-tab")
	check(tui.CtrlM, "Enter")
	check(tui.BSpace, "bspace")
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
	check(pairs, tui.AltZ+',', ",")

	pairs = parseKeyChords(",,a,b", "")
	checkN(len(pairs), 3)
	check(pairs, tui.AltZ+'a', "a")
	check(pairs, tui.AltZ+'b', "b")
	check(pairs, tui.AltZ+',', ",")

	pairs = parseKeyChords("a,b,,", "")
	checkN(len(pairs), 3)
	check(pairs, tui.AltZ+'a', "a")
	check(pairs, tui.AltZ+'b', "b")
	check(pairs, tui.AltZ+',', ",")

	pairs = parseKeyChords("a,,,b", "")
	checkN(len(pairs), 3)
	check(pairs, tui.AltZ+'a', "a")
	check(pairs, tui.AltZ+'b', "b")
	check(pairs, tui.AltZ+',', ",")

	pairs = parseKeyChords("a,,,b,c", "")
	checkN(len(pairs), 4)
	check(pairs, tui.AltZ+'a', "a")
	check(pairs, tui.AltZ+'b', "b")
	check(pairs, tui.AltZ+'c', "c")
	check(pairs, tui.AltZ+',', ",")

	pairs = parseKeyChords(",,,", "")
	checkN(len(pairs), 1)
	check(pairs, tui.AltZ+',', ",")
}

func TestBind(t *testing.T) {
	keymap := defaultKeymap()
	check := func(keyName int, arg1 string, types ...actionType) {
		if len(keymap[keyName]) != len(types) {
			t.Errorf("invalid number of actions (%d != %d)", len(types), len(keymap[keyName]))
			return
		}
		for idx, action := range keymap[keyName] {
			if types[idx] != action.t {
				t.Errorf("invalid action type (%d != %d)", types[idx], action.t)
			}
		}
		if len(arg1) > 0 && keymap[keyName][0].a != arg1 {
			t.Errorf("invalid action argument: (%s != %s)", arg1, keymap[keyName][0].a)
		}
	}
	check(tui.CtrlA, "", actBeginningOfLine)
	parseKeymap(keymap,
		"ctrl-a:kill-line,ctrl-b:toggle-sort+up+down,c:page-up,alt-z:page-down,"+
			"f1:execute(ls {+})+abort+execute(echo {+})+select-all,f2:execute/echo {}, {}, {}/,f3:execute[echo '({})'],f4:execute;less {};,"+
			"alt-a:execute-Multi@echo (,),[,],/,:,;,%,{}@,alt-b:execute;echo (,),[,],/,:,@,%,{};,"+
			"x:Execute(foo+bar),X:execute/bar+baz/"+
			",f1:+top,f1:+top"+
			",,:abort,::accept,+:execute:++\nfoobar,Y:execute(baz)+up")
	check(tui.CtrlA, "", actKillLine)
	check(tui.CtrlB, "", actToggleSort, actUp, actDown)
	check(tui.AltZ+'c', "", actPageUp)
	check(tui.AltZ+',', "", actAbort)
	check(tui.AltZ+':', "", actAccept)
	check(tui.AltZ, "", actPageDown)
	check(tui.F1, "ls {+}", actExecute, actAbort, actExecute, actSelectAll, actTop, actTop)
	check(tui.F2, "echo {}, {}, {}", actExecute)
	check(tui.F3, "echo '({})'", actExecute)
	check(tui.F4, "less {}", actExecute)
	check(tui.AltZ+'x', "foo+bar", actExecute)
	check(tui.AltZ+'X', "bar+baz", actExecute)
	check(tui.AltA, "echo (,),[,],/,:,;,%,{}", actExecuteMulti)
	check(tui.AltB, "echo (,),[,],/,:,@,%,{}", actExecute)
	check(tui.AltZ+'+', "++\nfoobar,Y:execute(baz)+up", actExecute)

	for idx, char := range []rune{'~', '!', '@', '#', '$', '%', '^', '&', '*', '|', ';', '/'} {
		parseKeymap(keymap, fmt.Sprintf("%d:execute%cfoobar%c", idx%10, char, char))
		check(tui.AltZ+int([]rune(fmt.Sprintf("%d", idx%10))[0]), "foobar", actExecute)
	}

	parseKeymap(keymap, "f1:abort")
	check(tui.F1, "", actAbort)
}

func TestColorSpec(t *testing.T) {
	theme := tui.Dark256
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
	if *light != *tui.Light256 {
		t.Errorf("colors should be equivalent")
	}
	if light == theme {
		t.Errorf("point should not be equivalent")
	}

	customized := parseTheme(theme, "fg:231,bg:232")
	if customized.Fg != 231 || customized.Bg != 232 {
		t.Errorf("color not customized")
	}
	if *tui.Dark256 == *customized {
		t.Errorf("colors should not be equivalent")
	}
	customized.Fg = tui.Dark256.Fg
	customized.Bg = tui.Dark256.Bg
	if *tui.Dark256 != *customized {
		t.Errorf("colors should now be equivalent: %v, %v", tui.Dark256, customized)
	}

	customized = parseTheme(theme, "fg:231,dark,bg:232")
	if customized.Fg != tui.Dark256.Fg || customized.Bg == tui.Dark256.Bg {
		t.Errorf("color not customized")
	}
}

func TestParseNilTheme(t *testing.T) {
	var theme *tui.ColorTheme
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
		if opts.Keymap[key][0].t != expected {
			t.Error()
		}
	}
	check([]string{}, tui.CtrlN, actDown)
	check([]string{}, tui.CtrlP, actUp)

	check([]string{"--bind=ctrl-n:accept"}, tui.CtrlN, actAccept)
	check([]string{"--bind=ctrl-p:accept"}, tui.CtrlP, actAccept)

	f, _ := ioutil.TempFile("", "fzf-history")
	f.Close()
	hist := "--history=" + f.Name()
	check([]string{hist}, tui.CtrlN, actNextHistory)
	check([]string{hist}, tui.CtrlP, actPreviousHistory)

	check([]string{hist, "--bind=ctrl-n:accept"}, tui.CtrlN, actAccept)
	check([]string{hist, "--bind=ctrl-n:accept"}, tui.CtrlP, actPreviousHistory)

	check([]string{hist, "--bind=ctrl-p:accept"}, tui.CtrlN, actNextHistory)
	check([]string{hist, "--bind=ctrl-p:accept"}, tui.CtrlP, actAccept)
}

func optsFor(words ...string) *Options {
	opts := defaultOptions()
	parseOptions(opts, words)
	postProcessOptions(opts)
	return opts
}

func TestToggle(t *testing.T) {
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

func TestPreviewOpts(t *testing.T) {
	opts := optsFor()
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == false &&
		opts.Preview.wrap == false &&
		opts.Preview.position == posRight &&
		opts.Preview.size.percent == true &&
		opts.Preview.size.size == 50) {
		t.Error()
	}
	opts = optsFor("--preview", "cat {}", "--preview-window=left:15:hidden:wrap")
	if !(opts.Preview.command == "cat {}" &&
		opts.Preview.hidden == true &&
		opts.Preview.wrap == true &&
		opts.Preview.position == posLeft &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15+2+2) {
		t.Error(opts.Preview)
	}
	opts = optsFor("--preview-window=up:15:wrap:hidden", "--preview-window=down")
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == false &&
		opts.Preview.wrap == false &&
		opts.Preview.position == posDown &&
		opts.Preview.size.percent == true &&
		opts.Preview.size.size == 50) {
		t.Error(opts.Preview)
	}
	opts = optsFor("--preview-window=up:15:wrap:hidden")
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == true &&
		opts.Preview.wrap == true &&
		opts.Preview.position == posUp &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15+2) {
		t.Error(opts.Preview)
	}
}

func TestAdditiveExpect(t *testing.T) {
	opts := optsFor("--expect=a", "--expect", "b", "--expect=c")
	if len(opts.Expect) != 3 {
		t.Error(opts.Expect)
	}
}
