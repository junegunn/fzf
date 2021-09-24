package fzf

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/junegunn/fzf/src/util"
)

func TestReplacePlaceholder(t *testing.T) {
	item1 := newItem("  foo'bar \x1b[31mbaz\x1b[m")
	items1 := []*Item{item1, item1}
	items2 := []*Item{
		newItem("foo'bar \x1b[31mbaz\x1b[m"),
		newItem("foo'bar \x1b[31mbaz\x1b[m"),
		newItem("FOO'BAR \x1b[31mBAZ\x1b[m")}

	delim := "'"
	var regex *regexp.Regexp

	var result string
	check := func(expected string) {
		if result != expected {
			t.Errorf("expected: %s, actual: %s", expected, result)
		}
	}
	// helper function that converts template format into string and carries out the check()
	checkFormat := func(format string) {
		type quotes struct{ O, I string } // outer, inner quotes
		unixStyle := quotes{"'", "'\\''"}
		windowsStyle := quotes{"^\"", "'"}
		var effectiveStyle quotes

		if util.IsWindows() {
			effectiveStyle = windowsStyle
		} else {
			effectiveStyle = unixStyle
		}

		expected := templateToString(format, effectiveStyle)
		check(expected)
	}
	printsep := "\n"
	// {}, preserve ansi
	result = replacePlaceholder("echo {}", false, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar \x1b[31mbaz\x1b[m{{.O}}")

	// {}, strip ansi
	result = replacePlaceholder("echo {}", true, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}}")

	// {}, with multiple items
	result = replacePlaceholder("echo {}", true, Delimiter{}, printsep, false, "query", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar baz{{.O}}")

	// {..}, strip leading whitespaces, preserve ansi
	result = replacePlaceholder("echo {..}", false, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}foo{{.I}}bar \x1b[31mbaz\x1b[m{{.O}}")

	// {..}, strip leading whitespaces, strip ansi
	result = replacePlaceholder("echo {..}", true, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}foo{{.I}}bar baz{{.O}}")

	// {q}
	result = replacePlaceholder("echo {} {q}", true, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}} {{.O}}query{{.O}}")

	// {q}, multiple items
	result = replacePlaceholder("echo {+}{q}{+}", true, Delimiter{}, printsep, false, "query 'string'", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar baz{{.O}} {{.O}}FOO{{.I}}BAR BAZ{{.O}}{{.O}}query {{.I}}string{{.I}}{{.O}}{{.O}}foo{{.I}}bar baz{{.O}} {{.O}}FOO{{.I}}BAR BAZ{{.O}}")

	result = replacePlaceholder("echo {}{q}{}", true, Delimiter{}, printsep, false, "query 'string'", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar baz{{.O}}{{.O}}query {{.I}}string{{.I}}{{.O}}{{.O}}foo{{.I}}bar baz{{.O}}")

	result = replacePlaceholder("echo {1}/{2}/{2,1}/{-1}/{-2}/{}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}foo{{.I}}bar{{.O}}/{{.O}}baz{{.O}}/{{.O}}bazfoo{{.I}}bar{{.O}}/{{.O}}baz{{.O}}/{{.O}}foo{{.I}}bar{{.O}}/{{.O}}  foo{{.I}}bar baz{{.O}}/{{.O}}foo{{.I}}bar baz{{.O}}/{n.t}/{}/{1}/{q}/{{.O}}{{.O}}")

	result = replacePlaceholder("echo {1}/{2}/{-1}/{-2}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, printsep, false, "query", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar{{.O}}/{{.O}}baz{{.O}}/{{.O}}baz{{.O}}/{{.O}}foo{{.I}}bar{{.O}}/{{.O}}foo{{.I}}bar baz{{.O}}/{n.t}/{}/{1}/{q}/{{.O}}{{.O}}")

	result = replacePlaceholder("echo {+1}/{+2}/{+-1}/{+-2}/{+..}/{n.t}/\\{}/\\{1}/\\{q}/{+3}", true, Delimiter{}, printsep, false, "query", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar{{.O}} {{.O}}FOO{{.I}}BAR{{.O}}/{{.O}}baz{{.O}} {{.O}}BAZ{{.O}}/{{.O}}baz{{.O}} {{.O}}BAZ{{.O}}/{{.O}}foo{{.I}}bar{{.O}} {{.O}}FOO{{.I}}BAR{{.O}}/{{.O}}foo{{.I}}bar baz{{.O}} {{.O}}FOO{{.I}}BAR BAZ{{.O}}/{n.t}/{}/{1}/{q}/{{.O}}{{.O}} {{.O}}{{.O}}")

	// forcePlus
	result = replacePlaceholder("echo {1}/{2}/{-1}/{-2}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, printsep, true, "query", items2)
	checkFormat("echo {{.O}}foo{{.I}}bar{{.O}} {{.O}}FOO{{.I}}BAR{{.O}}/{{.O}}baz{{.O}} {{.O}}BAZ{{.O}}/{{.O}}baz{{.O}} {{.O}}BAZ{{.O}}/{{.O}}foo{{.I}}bar{{.O}} {{.O}}FOO{{.I}}BAR{{.O}}/{{.O}}foo{{.I}}bar baz{{.O}} {{.O}}FOO{{.I}}BAR BAZ{{.O}}/{n.t}/{}/{1}/{q}/{{.O}}{{.O}} {{.O}}{{.O}}")

	// Whitespace preserving flag with "'" delimiter
	result = replacePlaceholder("echo {s1}", true, Delimiter{str: &delim}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.O}}")

	result = replacePlaceholder("echo {s2}", true, Delimiter{str: &delim}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}bar baz{{.O}}")

	result = replacePlaceholder("echo {s}", true, Delimiter{str: &delim}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}}")

	result = replacePlaceholder("echo {s..}", true, Delimiter{str: &delim}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}}")

	// Whitespace preserving flag with regex delimiter
	regex = regexp.MustCompile(`\w+`)

	result = replacePlaceholder("echo {s1}", true, Delimiter{regex: regex}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  {{.O}}")

	result = replacePlaceholder("echo {s2}", true, Delimiter{regex: regex}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}{{.I}}{{.O}}")

	result = replacePlaceholder("echo {s3}", true, Delimiter{regex: regex}, printsep, false, "query", items1)
	checkFormat("echo {{.O}} {{.O}}")

	// No match
	result = replacePlaceholder("echo {}/{+}", true, Delimiter{}, printsep, false, "query", []*Item{nil, nil})
	check("echo /")

	// No match, but with selections
	result = replacePlaceholder("echo {}/{+}", true, Delimiter{}, printsep, false, "query", []*Item{nil, item1})
	checkFormat("echo /{{.O}}  foo{{.I}}bar baz{{.O}}")

	// String delimiter
	result = replacePlaceholder("echo {}/{1}/{2}", true, Delimiter{str: &delim}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}}/{{.O}}foo{{.O}}/{{.O}}bar baz{{.O}}")

	// Regex delimiter
	regex = regexp.MustCompile("[oa]+")
	// foo'bar baz
	result = replacePlaceholder("echo {}/{1}/{3}/{2..3}", true, Delimiter{regex: regex}, printsep, false, "query", items1)
	checkFormat("echo {{.O}}  foo{{.I}}bar baz{{.O}}/{{.O}}f{{.O}}/{{.O}}r b{{.O}}/{{.O}}{{.I}}bar b{{.O}}")
}

func TestQuoteEntryCmd(t *testing.T) {
	tests := map[string]string{
		`"`:                       `^"\^"^"`,
		`\`:                       `^"\\^"`,
		`\"`:                      `^"\\\^"^"`,
		`"\\\"`:                   `^"\^"\\\\\\\^"^"`,
		`&|<>()@^%!`:              `^"^&^|^<^>^(^)^@^^^%^!^"`,
		`%USERPROFILE%`:           `^"^%USERPROFILE^%^"`,
		`C:\Program Files (x86)\`: `^"C:\\Program Files ^(x86^)\\^"`,
	}

	for input, expected := range tests {
		escaped := quoteEntryCmd(input)
		if escaped != expected {
			t.Errorf("Input: %s, expected: %s, actual %s", input, expected, escaped)
		}
	}
}

// purpose of this test is to demonstrate some shortcomings of fzf's templating system on Unix
func TestUnixCommands(t *testing.T) {
	if util.IsWindows() {
		t.SkipNow()
	}
	tests := []testCase{
		// reference: give{template, query, items}, want{output OR match}

		// 1) working examples

		// paths that does not have to evaluated will work fine, when quoted
		{give{`grep foo {}`, ``, newItems(`test`)}, want{output: `grep foo 'test'`}},
		{give{`grep foo {}`, ``, newItems(`/home/user/test`)}, want{output: `grep foo '/home/user/test'`}},
		{give{`grep foo {}`, ``, newItems(`./test`)}, want{output: `grep foo './test'`}},

		// only placeholders are escaped as data, this will lookup tilde character in a test file in your home directory
		// quoting the tilde is required (to be treated as string)
		{give{`grep {} ~/test`, ``, newItems(`~`)}, want{output: `grep '~' ~/test`}},

		// 2) problematic examples

		// paths that need to expand some part of it won't work (special characters and variables)
		{give{`cat {}`, ``, newItems(`~/test`)}, want{output: `cat '~/test'`}},
		{give{`cat {}`, ``, newItems(`$HOME/test`)}, want{output: `cat '$HOME/test'`}},
	}
	testCommands(t, tests)
}

// purpose of this test is to demonstrate some shortcomings of fzf's templating system on Windows
func TestWindowsCommands(t *testing.T) {
	if !util.IsWindows() {
		t.SkipNow()
	}
	tests := []testCase{
		// reference: give{template, query, items}, want{output OR match}

		// 1) working examples

		// example of redundantly escaped backslash in the output, besides looking bit ugly, it won't cause any issue
		{give{`type {}`, ``, newItems(`C:\test.txt`)}, want{output: `type ^"C:\\test.txt^"`}},
		{give{`rg -- "package" {}`, ``, newItems(`.\test.go`)}, want{output: `rg -- "package" ^".\\test.go^"`}},
		// example of mandatorily escaped backslash in the output, otherwise `rg -- "C:\test.txt"` is matching for tabulator
		{give{`rg -- {}`, ``, newItems(`C:\test.txt`)}, want{output: `rg -- ^"C:\\test.txt^"`}},
		// example of mandatorily escaped double quote in the output, otherwise `rg -- ""C:\\test.txt""` is not matching for the double quotes around the path
		{give{`rg -- {}`, ``, newItems(`"C:\test.txt"`)}, want{output: `rg -- ^"\^"C:\\test.txt\^"^"`}},

		// 2) problematic examples

		// notepad++'s parser can't handle `-n"12"` generate by fzf, expects `-n12`
		{give{`notepad++ -n{1} {2}`, ``, newItems(`12	C:\Work\Test Folder\File.txt`)}, want{output: `notepad++ -n^"12^" ^"C:\\Work\\Test Folder\\File.txt^"`}},

		// cat is parsing `\"` as a part of the file path, double quote is illegal character for paths on Windows
		// cat: "C:\\test.txt: Invalid argument
		{give{`cat {}`, ``, newItems(`"C:\test.txt"`)}, want{output: `cat ^"\^"C:\\test.txt\^"^"`}},
		// cat: "C:\\test.txt": Invalid argument
		{give{`cmd /c {}`, ``, newItems(`cat "C:\test.txt"`)}, want{output: `cmd /c ^"cat \^"C:\\test.txt\^"^"`}},

		// the "file" flag in the pattern won't create *.bat or *.cmd file so the command in the output tries to edit the file, instead of executing it
		// the temp file contains: `cat "C:\test.txt"`
		{give{`cmd /c {f}`, ``, newItems(`cat "C:\test.txt"`)}, want{match: `^cmd /c .*\fzf-preview-[0-9]{9}$`}},
	}
	testCommands(t, tests)
}

/* utilities section */

// Item represents one line in fzf UI. Usually it is relative path to files and folders.
func newItem(str string) *Item {
	bytes := []byte(str)
	trimmed, _, _ := extractColor(str, nil, nil)
	return &Item{origText: &bytes, text: util.ToChars([]byte(trimmed))}
}

// Functions tested in this file require array of items (allItems). The array needs
// to consist of at least two nils. This is helper function.
func newItems(str ...string) []*Item {
	result := make([]*Item, util.Max(len(str), 2))
	for i, s := range str {
		result[i] = newItem(s)
	}
	return result
}

// (for logging purposes)
func (item *Item) String() string {
	return item.AsString(true)
}

// Helper function to parse, execute and convert "text/template" to string. Panics on error.
func templateToString(format string, data interface{}) string {
	bb := &bytes.Buffer{}

	err := template.Must(template.New("").Parse(format)).Execute(bb, data)
	if err != nil {
		panic(err)
	}

	return bb.String()
}

// ad hoc types for test cases
type give struct {
	template string
	query    string
	allItems []*Item
}
type want struct {
	/*
		Unix:
		The `want.output` string is supposed to be formatted for evaluation by
		`sh -c command` system call.

		Windows:
		The `want.output` string is supposed to be formatted for evaluation by
		`cmd.exe /s /c "command"` system call. The `/s` switch enables so called old
		behaviour, which is more favourable for nesting (possibly escaped)
		special characters. This is the relevant section of `help cmd`:

		...old behavior is to see if the first character is
		a quote character and if so, strip the leading character and
		remove the last quote character on the command line, preserving
		any text after the last quote character.
	*/
	output string // literal output
	match  string // output is matched against this regex (when output is empty string)
}
type testCase struct {
	give
	want
}

func testCommands(t *testing.T, tests []testCase) {
	// common test parameters
	delim := "\t"
	delimiter := Delimiter{str: &delim}
	printsep := ""
	stringAnsi := false
	forcePlus := false

	// evaluate the test cases
	for idx, test := range tests {
		gotOutput := replacePlaceholder(
			test.give.template, stringAnsi, delimiter, printsep, forcePlus,
			test.give.query,
			test.give.allItems)
		switch {
		case test.want.output != "":
			if gotOutput != test.want.output {
				t.Errorf("tests[%v]:\ngave{\n\ttemplate: '%s',\n\tquery: '%s',\n\tallItems: %s}\nand got '%s',\nbut want '%s'",
					idx,
					test.give.template, test.give.query, test.give.allItems,
					gotOutput, test.want.output)
			}
		case test.want.match != "":
			wantMatch := strings.ReplaceAll(test.want.match, "\\", "\\\\")
			wantRegex := regexp.MustCompile(wantMatch)
			if !wantRegex.MatchString(gotOutput) {
				t.Errorf("tests[%v]:\ngave{\n\ttemplate: '%s',\n\tquery: '%s',\n\tallItems: %s}\nand got '%s',\nbut want '%s'",
					idx,
					test.give.template, test.give.query, test.give.allItems,
					gotOutput, test.want.match)
			}
		default:
			t.Errorf("tests[%v]: test case does not describe 'want' property", idx)
		}
	}
}
