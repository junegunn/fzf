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

func TestDelimiterRegexRegexCaret(t *testing.T) {
	delim := delimiterRegexp(`(^\s*|\s+)`)
	tokens := Tokenize("foo  bar baz", delim)
	if delim.str != nil ||
		len(tokens) != 4 ||
		tokens[0].text.ToString() != "" ||
		tokens[1].text.ToString() != "foo  " ||
		tokens[2].text.ToString() != "bar " ||
		tokens[3].text.ToString() != "baz" {
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
	for _, words := range [][]string{{"--nth", "..,3", "+x"}, {"--nth", "3,1..", "+x"}, {"--nth", "..-1,1", "+x"}} {
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
	checkEvent := func(e tui.Event, s string) {
		if pairs[e] != s {
			t.Errorf("%s != %s", pairs[e], s)
		}
	}
	check := func(et tui.EventType, s string) {
		checkEvent(et.AsEvent(), s)
	}
	if len(pairs) != 12 {
		t.Error(12)
	}
	check(tui.CtrlZ, "ctrl-z")
	check(tui.F2, "f2")
	check(tui.CtrlG, "ctrl-G")
	checkEvent(tui.AltKey('z'), "alt-z")
	checkEvent(tui.Key('@'), "@")
	checkEvent(tui.AltKey('a'), "Alt-a")
	checkEvent(tui.Key('!'), "!")
	checkEvent(tui.Key('J'), "J")
	checkEvent(tui.Key('g'), "g")
	checkEvent(tui.CtrlAltKey('a'), "ctrl-alt-a")
	checkEvent(tui.CtrlAltKey('m'), "ALT-enter")
	checkEvent(tui.AltKey(' '), "alt-SPACE")

	// Synonyms
	pairs = parseKeyChords("enter,Return,space,tab,btab,esc,up,down,left,right", "")
	if len(pairs) != 9 {
		t.Error(9)
	}
	check(tui.CtrlM, "Return")
	checkEvent(tui.Key(' '), "space")
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
	check := func(pairs map[tui.Event]string, e tui.Event, s string) {
		if pairs[e] != s {
			t.Errorf("%s != %s", pairs[e], s)
		}
	}

	pairs := parseKeyChords(",", "")
	checkN(len(pairs), 1)
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords(",,a,b", "")
	checkN(len(pairs), 3)
	check(pairs, tui.Key('a'), "a")
	check(pairs, tui.Key('b'), "b")
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords("a,b,,", "")
	checkN(len(pairs), 3)
	check(pairs, tui.Key('a'), "a")
	check(pairs, tui.Key('b'), "b")
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords("a,,,b", "")
	checkN(len(pairs), 3)
	check(pairs, tui.Key('a'), "a")
	check(pairs, tui.Key('b'), "b")
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords("a,,,b,c", "")
	checkN(len(pairs), 4)
	check(pairs, tui.Key('a'), "a")
	check(pairs, tui.Key('b'), "b")
	check(pairs, tui.Key('c'), "c")
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords(",,,", "")
	checkN(len(pairs), 1)
	check(pairs, tui.Key(','), ",")

	pairs = parseKeyChords(",ALT-,,", "")
	checkN(len(pairs), 1)
	check(pairs, tui.AltKey(','), "ALT-,")
}

func TestBind(t *testing.T) {
	keymap := defaultKeymap()
	check := func(event tui.Event, arg1 string, types ...actionType) {
		if len(keymap[event]) != len(types) {
			t.Errorf("invalid number of actions for %v (%d != %d)",
				event, len(types), len(keymap[event]))
			return
		}
		for idx, action := range keymap[event] {
			if types[idx] != action.t {
				t.Errorf("invalid action type (%d != %d)", types[idx], action.t)
			}
		}
		if len(arg1) > 0 && keymap[event][0].a != arg1 {
			t.Errorf("invalid action argument: (%s != %s)", arg1, keymap[event][0].a)
		}
	}
	check(tui.CtrlA.AsEvent(), "", actBeginningOfLine)
	errorString := ""
	errorFn := func(e string) {
		errorString = e
	}
	parseKeymap(keymap,
		"ctrl-a:kill-line,ctrl-b:toggle-sort+up+down,c:page-up,alt-z:page-down,"+
			"f1:execute(ls {+})+abort+execute(echo \n{+})+select-all,f2:execute/echo {}, {}, {}/,f3:execute[echo '({})'],f4:execute;less {};,"+
			"alt-a:execute-Multi@echo (,),[,],/,:,;,%,{}@,alt-b:execute;echo (,),[,],/,:,@,%,{};,"+
			"x:Execute(foo+bar),X:execute/bar+baz/"+
			",f1:+first,f1:+top"+
			",,:abort,::accept,+:execute:++\nfoobar,Y:execute(baz)+up", errorFn)
	check(tui.CtrlA.AsEvent(), "", actKillLine)
	check(tui.CtrlB.AsEvent(), "", actToggleSort, actUp, actDown)
	check(tui.Key('c'), "", actPageUp)
	check(tui.Key(','), "", actAbort)
	check(tui.Key(':'), "", actAccept)
	check(tui.AltKey('z'), "", actPageDown)
	check(tui.F1.AsEvent(), "ls {+}", actExecute, actAbort, actExecute, actSelectAll, actFirst, actFirst)
	check(tui.F2.AsEvent(), "echo {}, {}, {}", actExecute)
	check(tui.F3.AsEvent(), "echo '({})'", actExecute)
	check(tui.F4.AsEvent(), "less {}", actExecute)
	check(tui.Key('x'), "foo+bar", actExecute)
	check(tui.Key('X'), "bar+baz", actExecute)
	check(tui.AltKey('a'), "echo (,),[,],/,:,;,%,{}", actExecuteMulti)
	check(tui.AltKey('b'), "echo (,),[,],/,:,@,%,{}", actExecute)
	check(tui.Key('+'), "++\nfoobar,Y:execute(baz)+up", actExecute)

	for idx, char := range []rune{'~', '!', '@', '#', '$', '%', '^', '&', '*', '|', ';', '/'} {
		parseKeymap(keymap, fmt.Sprintf("%d:execute%cfoobar%c", idx%10, char, char), errorFn)
		check(tui.Key([]rune(fmt.Sprintf("%d", idx%10))[0]), "foobar", actExecute)
	}

	parseKeymap(keymap, "f1:abort", errorFn)
	check(tui.F1.AsEvent(), "", actAbort)
	if len(errorString) > 0 {
		t.Errorf("error parsing keymap: %s", errorString)
	}
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
	if customized.Fg.Color != 231 || customized.Bg.Color != 232 {
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

func TestDefaultCtrlNP(t *testing.T) {
	check := func(words []string, et tui.EventType, expected actionType) {
		e := et.AsEvent()
		opts := defaultOptions()
		parseOptions(opts, words)
		postProcessOptions(opts)
		if opts.Keymap[e][0].t != expected {
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
	check([]string{hist}, tui.CtrlP, actPrevHistory)

	check([]string{hist, "--bind=ctrl-n:accept"}, tui.CtrlN, actAccept)
	check([]string{hist, "--bind=ctrl-n:accept"}, tui.CtrlP, actPrevHistory)

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
	opts = optsFor("--preview", "cat {}", "--preview-window=left:15,hidden,wrap:+{1}-/2")
	if !(opts.Preview.command == "cat {}" &&
		opts.Preview.hidden == true &&
		opts.Preview.wrap == true &&
		opts.Preview.position == posLeft &&
		opts.Preview.scroll == "+{1}-/2" &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15) {
		t.Error(opts.Preview)
	}
	opts = optsFor("--preview-window=up,15,wrap,hidden,+{1}+3-1-2/2", "--preview-window=down", "--preview-window=cycle")
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == true &&
		opts.Preview.wrap == true &&
		opts.Preview.cycle == true &&
		opts.Preview.position == posDown &&
		opts.Preview.scroll == "+{1}+3-1-2/2" &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15) {
		t.Error(opts.Preview.size.size)
	}
	opts = optsFor("--preview-window=up:15:wrap:hidden")
	if !(opts.Preview.command == "" &&
		opts.Preview.hidden == true &&
		opts.Preview.wrap == true &&
		opts.Preview.position == posUp &&
		opts.Preview.size.percent == false &&
		opts.Preview.size.size == 15) {
		t.Error(opts.Preview)
	}
	opts = optsFor("--preview=foo", "--preview-window=up", "--preview-window=default:70%")
	if !(opts.Preview.command == "foo" &&
		opts.Preview.position == posRight &&
		opts.Preview.size.percent == true &&
		opts.Preview.size.size == 70) {
		t.Error(opts.Preview)
	}
}

func TestAdditiveExpect(t *testing.T) {
	opts := optsFor("--expect=a", "--expect", "b", "--expect=c")
	if len(opts.Expect) != 3 {
		t.Error(opts.Expect)
	}
}

func TestValidateSign(t *testing.T) {
	testCases := []struct {
		inputSign string
		isValid   bool
	}{
		{"> ", true},
		{"아", true},
		{"😀", true},
		{"", false},
		{">>>", false},
	}

	for _, testCase := range testCases {
		err := validateSign(testCase.inputSign, "")
		if testCase.isValid && err != nil {
			t.Errorf("Input sign `%s` caused error", testCase.inputSign)
		}

		if !testCase.isValid && err == nil {
			t.Errorf("Input sign `%s` did not cause error", testCase.inputSign)
		}
	}
}

func TestParseSingleActionList(t *testing.T) {
	actions := parseSingleActionList("Execute@foo+bar,baz@+up+up+reload:down+down", func(string) {})
	if len(actions) != 4 {
		t.Errorf("Invalid number of actions parsed:%d", len(actions))
	}
	if actions[0].t != actExecute || actions[0].a != "foo+bar,baz" {
		t.Errorf("Invalid action parsed: %v", actions[0])
	}
	if actions[1].t != actUp || actions[2].t != actUp {
		t.Errorf("Invalid action parsed: %v / %v", actions[1], actions[2])
	}
	if actions[3].t != actReload || actions[3].a != "down+down" {
		t.Errorf("Invalid action parsed: %v", actions[3])
	}
}

func TestParseSingleActionListError(t *testing.T) {
	err := ""
	parseSingleActionList("change-query(foobar)baz", func(e string) {
		err = e
	})
	if len(err) == 0 {
		t.Errorf("Failed to detect error")
	}
}

func TestMaskActionContents(t *testing.T) {
	original := ":execute((f)(o)(o)(b)(a)(r))+change-query@qu@ry@+up,x:reload:hello:world"
	expected := ":execute                    +change-query       +up,x:reload            "
	masked := maskActionContents(original)
	if masked != expected {
		t.Errorf("Not masked: %s", masked)
	}
}
