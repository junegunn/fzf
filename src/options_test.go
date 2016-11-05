package fzf

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
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
	tokens := Tokenize(util.RunesToChars([]rune("-*--*---**---")), delim)
	if delim.regex != nil ||
		tokens[0].text.ToString() != "-*" ||
		tokens[1].text.ToString() != "--*" ||
		tokens[2].text.ToString() != "---*" ||
		tokens[3].text.ToString() != "*" ||
		tokens[4].text.ToString() != "---" {
		t.Errorf("%s %s %d", delim, tokens, len(tokens))
	}
}

func TestDelimiterRegexRegex(t *testing.T) {
	delim := delimiterRegexp("--\\*")
	tokens := Tokenize(util.RunesToChars([]rune("-*--*---**---")), delim)
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
	check(tui.CtrlZ, "ctrl-z")
	check(tui.AltZ, "alt-z")
	check(tui.F2, "f2")
	check(tui.AltZ+'@', "@")
	check(tui.AltA, "Alt-a")
	check(tui.AltZ+'!', "!")
	check(tui.CtrlA+'g'-'a', "ctrl-G")
	check(tui.AltZ+'J', "J")
	check(tui.AltZ+'g', "g")
	check(tui.AltEnter, "ALT-enter")
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
	check(actBeginningOfLine, keymap[tui.CtrlA])
	parseKeymap(keymap, execmap,
		"ctrl-a:kill-line,ctrl-b:toggle-sort,c:page-up,alt-z:page-down,"+
			"f1:execute(ls {}),f2:execute/echo {}, {}, {}/,f3:execute[echo '({})'],f4:execute;less {};,"+
			"alt-a:execute@echo (,),[,],/,:,;,%,{}@,alt-b:execute;echo (,),[,],/,:,@,%,{};"+
			",,:abort,::accept,X:execute:\nfoobar,Y:execute(baz)")
	check(actKillLine, keymap[tui.CtrlA])
	check(actToggleSort, keymap[tui.CtrlB])
	check(actPageUp, keymap[tui.AltZ+'c'])
	check(actAbort, keymap[tui.AltZ+','])
	check(actAccept, keymap[tui.AltZ+':'])
	check(actPageDown, keymap[tui.AltZ])
	check(actExecute, keymap[tui.F1])
	check(actExecute, keymap[tui.F2])
	check(actExecute, keymap[tui.F3])
	check(actExecute, keymap[tui.F4])
	checkString("ls {}", execmap[tui.F1])
	checkString("echo {}, {}, {}", execmap[tui.F2])
	checkString("echo '({})'", execmap[tui.F3])
	checkString("less {}", execmap[tui.F4])
	checkString("echo (,),[,],/,:,;,%,{}", execmap[tui.AltA])
	checkString("echo (,),[,],/,:,@,%,{}", execmap[tui.AltB])
	checkString("\nfoobar,Y:execute(baz)", execmap[tui.AltZ+'X'])

	for idx, char := range []rune{'~', '!', '@', '#', '$', '%', '^', '&', '*', '|', ';', '/'} {
		parseKeymap(keymap, execmap, fmt.Sprintf("%d:execute%cfoobar%c", idx%10, char, char))
		checkString("foobar", execmap[tui.AltZ+int([]rune(fmt.Sprintf("%d", idx%10))[0])])
	}

	parseKeymap(keymap, execmap, "f1:abort")
	check(actAbort, keymap[tui.F1])
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
		if opts.Keymap[key] != expected {
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
		opts.Preview.position == posRight &&
		opts.Preview.size.percent == true &&
		opts.Preview.size.size == 50) {
		t.Error()
	}
	opts = optsFor("--preview", "cat {}", "--preview-window=left:15:hidden")
	if !(opts.Preview.command == "cat {}" &&
		opts.Preview.hidden == true &&
		opts.Preview.position == posLeft &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15+2) {
		t.Error(opts.Preview)
	}

	opts = optsFor("--preview-window=left:15:hidden", "--preview-window=down")
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == false &&
		opts.Preview.position == posDown &&
		opts.Preview.size.percent == true &&
		opts.Preview.size.size == 50) {
		t.Error(opts.Preview)
	}
}
