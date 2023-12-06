package fzf

import (
	"bytes"
	"io"
	"os"
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
		type quotes struct{ O, I, S string } // outer, inner quotes, print separator
		unixStyle := quotes{`'`, `'\''`, "\n"}
		windowsStyle := quotes{`^"`, `'`, "\n"}
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

	/*
		Test multiple placeholders and the function parameters.
	*/

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

	/*
		Test single placeholders, but focus on the placeholders' parameters (e.g. flags).
		see: TestParsePlaceholder
	*/
	items3 := []*Item{
		// single line
		newItem("1a 1b 1c 1d 1e 1f"),
		// multi line
		newItem("1a 1b 1c 1d 1e 1f"),
		newItem("2a 2b 2c 2d 2e 2f"),
		newItem("3a 3b 3c 3d 3e 3f"),
		newItem("4a 4b 4c 4d 4e 4f"),
		newItem("5a 5b 5c 5d 5e 5f"),
		newItem("6a 6b 6c 6d 6e 6f"),
		newItem("7a 7b 7c 7d 7e 7f"),
	}
	stripAnsi := false
	printsep = "\n"
	forcePlus := false
	query := "sample query"

	templateToOutput := make(map[string]string)
	templateToFile := make(map[string]string) // same as above, but the file contents will be matched
	// I. item type placeholder
	templateToOutput[`{}`] = `{{.O}}1a 1b 1c 1d 1e 1f{{.O}}`
	templateToOutput[`{+}`] = `{{.O}}1a 1b 1c 1d 1e 1f{{.O}} {{.O}}2a 2b 2c 2d 2e 2f{{.O}} {{.O}}3a 3b 3c 3d 3e 3f{{.O}} {{.O}}4a 4b 4c 4d 4e 4f{{.O}} {{.O}}5a 5b 5c 5d 5e 5f{{.O}} {{.O}}6a 6b 6c 6d 6e 6f{{.O}} {{.O}}7a 7b 7c 7d 7e 7f{{.O}}`
	templateToOutput[`{n}`] = `0`
	templateToOutput[`{+n}`] = `0 0 0 0 0 0 0`
	templateToFile[`{f}`] = `1a 1b 1c 1d 1e 1f{{.S}}`
	templateToFile[`{+f}`] = `1a 1b 1c 1d 1e 1f{{.S}}2a 2b 2c 2d 2e 2f{{.S}}3a 3b 3c 3d 3e 3f{{.S}}4a 4b 4c 4d 4e 4f{{.S}}5a 5b 5c 5d 5e 5f{{.S}}6a 6b 6c 6d 6e 6f{{.S}}7a 7b 7c 7d 7e 7f{{.S}}`
	templateToFile[`{nf}`] = `0{{.S}}`
	templateToFile[`{+nf}`] = `0{{.S}}0{{.S}}0{{.S}}0{{.S}}0{{.S}}0{{.S}}0{{.S}}`

	// II. token type placeholders
	templateToOutput[`{..}`] = templateToOutput[`{}`]
	templateToOutput[`{1..}`] = templateToOutput[`{}`]
	templateToOutput[`{..2}`] = `{{.O}}1a 1b{{.O}}`
	templateToOutput[`{1..2}`] = templateToOutput[`{..2}`]
	templateToOutput[`{-2..-1}`] = `{{.O}}1e 1f{{.O}}`
	// shorthand for x..x range
	templateToOutput[`{1}`] = `{{.O}}1a{{.O}}`
	templateToOutput[`{1..1}`] = templateToOutput[`{1}`]
	templateToOutput[`{-6}`] = templateToOutput[`{1}`]
	// multiple ranges
	templateToOutput[`{1,2}`] = templateToOutput[`{1..2}`]
	templateToOutput[`{1,2,4}`] = `{{.O}}1a 1b 1d{{.O}}`
	templateToOutput[`{1,2..4}`] = `{{.O}}1a 1b 1c 1d{{.O}}`
	templateToOutput[`{1..2,-4..-3}`] = `{{.O}}1a 1b 1c 1d{{.O}}`
	// flags
	templateToOutput[`{+1}`] = `{{.O}}1a{{.O}} {{.O}}2a{{.O}} {{.O}}3a{{.O}} {{.O}}4a{{.O}} {{.O}}5a{{.O}} {{.O}}6a{{.O}} {{.O}}7a{{.O}}`
	templateToOutput[`{+-1}`] = `{{.O}}1f{{.O}} {{.O}}2f{{.O}} {{.O}}3f{{.O}} {{.O}}4f{{.O}} {{.O}}5f{{.O}} {{.O}}6f{{.O}} {{.O}}7f{{.O}}`
	templateToOutput[`{s1}`] = `{{.O}}1a {{.O}}`
	templateToFile[`{f1}`] = `1a{{.S}}`
	templateToOutput[`{+s1..2}`] = `{{.O}}1a 1b {{.O}} {{.O}}2a 2b {{.O}} {{.O}}3a 3b {{.O}} {{.O}}4a 4b {{.O}} {{.O}}5a 5b {{.O}} {{.O}}6a 6b {{.O}} {{.O}}7a 7b {{.O}}`
	templateToFile[`{+sf1..2}`] = `1a 1b {{.S}}2a 2b {{.S}}3a 3b {{.S}}4a 4b {{.S}}5a 5b {{.S}}6a 6b {{.S}}7a 7b {{.S}}`

	// III. query type placeholder
	// query flag is not removed after parsing, so it gets doubled
	// while the double q is invalid, it is useful here for testing purposes
	templateToOutput[`{q}`] = "{{.O}}" + query + "{{.O}}"

	// IV. escaping placeholder
	templateToOutput[`\{}`] = `{}`
	templateToOutput[`\{++}`] = `{++}`
	templateToOutput[`{++}`] = templateToOutput[`{+}`]

	for giveTemplate, wantOutput := range templateToOutput {
		result = replacePlaceholder(giveTemplate, stripAnsi, Delimiter{}, printsep, forcePlus, query, items3)
		checkFormat(wantOutput)
	}
	for giveTemplate, wantOutput := range templateToFile {
		path := replacePlaceholder(giveTemplate, stripAnsi, Delimiter{}, printsep, forcePlus, query, items3)

		data, err := readFile(path)
		if err != nil {
			t.Errorf("Cannot read the content of the temp file %s.", path)
		}
		result = string(data)

		checkFormat(wantOutput)
	}
}

func TestQuoteEntry(t *testing.T) {
	type quotes struct{ E, O, SQ, DQ, BS string } // standalone escape, outer, single and double quotes, backslash
	unixStyle := quotes{``, `'`, `'\''`, `"`, `\`}
	windowsStyle := quotes{`^`, `^"`, `'`, `\^"`, `\\`}
	var effectiveStyle quotes

	if util.IsWindows() {
		effectiveStyle = windowsStyle
	} else {
		effectiveStyle = unixStyle
	}

	tests := map[string]string{
		`'`:     `{{.O}}{{.SQ}}{{.O}}`,
		`"`:     `{{.O}}{{.DQ}}{{.O}}`,
		`\`:     `{{.O}}{{.BS}}{{.O}}`,
		`\"`:    `{{.O}}{{.BS}}{{.DQ}}{{.O}}`,
		`"\\\"`: `{{.O}}{{.DQ}}{{.BS}}{{.BS}}{{.BS}}{{.DQ}}{{.O}}`,

		`$`:       `{{.O}}${{.O}}`,
		`$HOME`:   `{{.O}}$HOME{{.O}}`,
		`'$HOME'`: `{{.O}}{{.SQ}}$HOME{{.SQ}}{{.O}}`,

		`&`:                       `{{.O}}{{.E}}&{{.O}}`,
		`|`:                       `{{.O}}{{.E}}|{{.O}}`,
		`<`:                       `{{.O}}{{.E}}<{{.O}}`,
		`>`:                       `{{.O}}{{.E}}>{{.O}}`,
		`(`:                       `{{.O}}{{.E}}({{.O}}`,
		`)`:                       `{{.O}}{{.E}}){{.O}}`,
		`@`:                       `{{.O}}{{.E}}@{{.O}}`,
		`^`:                       `{{.O}}{{.E}}^{{.O}}`,
		`%`:                       `{{.O}}{{.E}}%{{.O}}`,
		`!`:                       `{{.O}}{{.E}}!{{.O}}`,
		`%USERPROFILE%`:           `{{.O}}{{.E}}%USERPROFILE{{.E}}%{{.O}}`,
		`C:\Program Files (x86)\`: `{{.O}}C:{{.BS}}Program Files {{.E}}(x86{{.E}}){{.BS}}{{.O}}`,
		`"C:\Program Files"`:      `{{.O}}{{.DQ}}C:{{.BS}}Program Files{{.DQ}}{{.O}}`,
	}

	for input, expected := range tests {
		escaped := quoteEntry(input)
		expected = templateToString(expected, effectiveStyle)
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
		// (not necessarily unexpected)

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
		// (not necessarily unexpected)

		// notepad++'s parser can't handle `-n"12"` generate by fzf, expects `-n12`
		{give{`notepad++ -n{1} {2}`, ``, newItems(`12	C:\Work\Test Folder\File.txt`)}, want{output: `notepad++ -n^"12^" ^"C:\\Work\\Test Folder\\File.txt^"`}},

		// cat is parsing `\"` as a part of the file path, double quote is illegal character for paths on Windows
		// cat: "C:\\test.txt: Invalid argument
		{give{`cat {}`, ``, newItems(`"C:\test.txt"`)}, want{output: `cat ^"\^"C:\\test.txt\^"^"`}},
		// cat: "C:\\test.txt": Invalid argument
		{give{`cmd /c {}`, ``, newItems(`cat "C:\test.txt"`)}, want{output: `cmd /c ^"cat \^"C:\\test.txt\^"^"`}},

		// the "file" flag in the pattern won't create *.bat or *.cmd file so the command in the output tries to edit the file, instead of executing it
		// the temp file contains: `cat "C:\test.txt"`
		// TODO this should actually work
		{give{`cmd /c {f}`, ``, newItems(`cat "C:\test.txt"`)}, want{match: `^cmd /c .*\fzf-preview-[0-9]{9}$`}},
	}
	testCommands(t, tests)
}

// purpose of this test is to demonstrate some shortcomings of fzf's templating system on Windows in Powershell
func TestPowershellCommands(t *testing.T) {
	if !util.IsWindows() {
		t.SkipNow()
	}

	tests := []testCase{
		// reference: give{template, query, items}, want{output OR match}

		/*
			You can read each line in the following table as a pipeline that
			consist of series of parsers that act upon your input (col. 1) and
			each cell represents the output value.

			For example:
			 - exec.Command("program.exe", `\''`)
			   - goes to win32 api which will process it transparently as it contains no special characters, see [CommandLineToArgvW][].
			     - powershell command will receive it as is, that is two arguments: a literal backslash and empty string in single quotes
			     - native command run via/from powershell will receive only one argument: a literal backslash. Because extra parsing rules apply, see [NativeCallsFromPowershell][].
			       - some¹ apps have internal parser, that requires one more level of escaping (yes, this is completely application-specific, but see terminal_test.go#TestWindowsCommands)

			Character⁰   CommandLineToArgvW   Powershell commands              Native commands from Powershell   Apps requiring escapes¹    | Being tested below
			----------   ------------------   ------------------------------   -------------------------------   -------------------------- | ------------------
			"            empty string²        missing argument error           ...                               ...                        |
			\"           literal "            unbalanced quote error           ...                               ...                        |
			'\"'         literal '"'          literal "                        empty string                      empty string (match all)   | yes
			'\\\"'       literal '\"'         literal \"                       literal "                         literal "                  |
			----------   ------------------   ------------------------------   -------------------------------   -------------------------- | ------------------
			\            transparent          transparent                      transparent                       regex error                |
			'\'          transparent          literal \                        literal \                         regex error                | yes
			\\           transparent          transparent                      transparent                       literal \                  |
			'\\'         transparent          literal \\                       literal \\                        literal \                  |
			----------   ------------------   ------------------------------   -------------------------------   -------------------------- | ------------------
			'            transparent          unbalanced quote error           ...                               ...                        |
			\'           transparent          literal \ and unb. quote error   ...                               ...                        |
			\''          transparent          literal \ and empty string       literal \                         regex error                | no, but given as example above
			'''          transparent          unbalanced quote error           ...                               ...                        |
			''''         transparent          literal '                        literal '                         literal '                  | yes
			----------   ------------------   ------------------------------   -------------------------------   -------------------------- | ------------------

			⁰: charatecter or characters 'x' as an argument to a program in go's call: exec.Command("program.exe", `x`)
			¹: native commands like grep, git grep, ripgrep
			²: interpreted as a grouping quote, affects argument parser and gets removed from the result

			[CommandLineToArgvW]: https://docs.microsoft.com/en-gb/windows/win32/api/shellapi/nf-shellapi-commandlinetoargvw#remarks
			[NativeCallsFromPowershell]: https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_parsing?view=powershell-7.1#passing-arguments-that-contain-quote-characters
		*/

		// 1) working examples

		{give{`Get-Content {}`, ``, newItems(`C:\test.txt`)}, want{output: `Get-Content 'C:\test.txt'`}},
		{give{`rg -- "package" {}`, ``, newItems(`.\test.go`)}, want{output: `rg -- "package" '.\test.go'`}},

		// example of escaping single quotes
		{give{`rg -- {}`, ``, newItems(`'foobar'`)}, want{output: `rg -- '''foobar'''`}},

		// chaining powershells
		{give{`powershell -NoProfile -Command {}`, ``, newItems(`cat "C:\test.txt"`)}, want{output: `powershell -NoProfile -Command 'cat \"C:\test.txt\"'`}},

		// 2) problematic examples
		// (not necessarily unexpected)

		// looking for a path string will only work with escaped backslashes
		{give{`rg -- {}`, ``, newItems(`C:\test.txt`)}, want{output: `rg -- 'C:\test.txt'`}},
		// looking for a literal double quote will only work with triple escaped double quotes
		{give{`rg -- {}`, ``, newItems(`"C:\test.txt"`)}, want{output: `rg -- '\"C:\test.txt\"'`}},

		// Get-Content (i.e. cat alias) is parsing `"` as a part of the file path, returns an error:
		// Get-Content : Cannot find drive. A drive with the name '"C:' does not exist.
		{give{`cat {}`, ``, newItems(`"C:\test.txt"`)}, want{output: `cat '\"C:\test.txt\"'`}},

		// the "file" flag in the pattern won't create *.ps1 file so the powershell will offload this "unknown" filetype
		// to explorer, which will prompt user to pick editing program for the fzf-preview file
		// the temp file contains: `cat "C:\test.txt"`
		// TODO this should actually work
		{give{`powershell -NoProfile -Command {f}`, ``, newItems(`cat "C:\test.txt"`)}, want{match: `^powershell -NoProfile -Command .*\fzf-preview-[0-9]{9}$`}},
	}

	// to force powershell-style escaping we temporarily set environment variable that fzf honors
	shellBackup := os.Getenv("SHELL")
	os.Setenv("SHELL", "powershell")
	testCommands(t, tests)
	os.Setenv("SHELL", shellBackup)
}

/*
Test typical valid placeholders and parsing of them.

Also since the parser assumes the input is matched with `placeholder` regex,
the regex is tested here as well.
*/
func TestParsePlaceholder(t *testing.T) {
	// give, want pairs
	templates := map[string]string{
		// I. item type placeholder
		`{}`:    `{}`,
		`{+}`:   `{+}`,
		`{n}`:   `{n}`,
		`{+n}`:  `{+n}`,
		`{f}`:   `{f}`,
		`{+nf}`: `{+nf}`,

		// II. token type placeholders
		`{..}`:     `{..}`,
		`{1..}`:    `{1..}`,
		`{..2}`:    `{..2}`,
		`{1..2}`:   `{1..2}`,
		`{-2..-1}`: `{-2..-1}`,
		// shorthand for x..x range
		`{1}`:    `{1}`,
		`{1..1}`: `{1..1}`,
		`{-6}`:   `{-6}`,
		// multiple ranges
		`{1,2}`:         `{1,2}`,
		`{1,2,4}`:       `{1,2,4}`,
		`{1,2..4}`:      `{1,2..4}`,
		`{1..2,-4..-3}`: `{1..2,-4..-3}`,
		// flags
		`{+1}`:      `{+1}`,
		`{+-1}`:     `{+-1}`,
		`{s1}`:      `{s1}`,
		`{f1}`:      `{f1}`,
		`{+s1..2}`:  `{+s1..2}`,
		`{+sf1..2}`: `{+sf1..2}`,

		// III. query type placeholder
		// query flag is not removed after parsing, so it gets doubled
		// while the double q is invalid, it is useful here for testing purposes
		`{q}`: `{qq}`,

		// IV. escaping placeholder
		`\{}`:   `{}`,
		`\{++}`: `{++}`,
		`{++}`:  `{+}`,
	}

	for giveTemplate, wantTemplate := range templates {
		if !placeholder.MatchString(giveTemplate) {
			t.Errorf(`given placeholder %s does not match placeholder regex, so attempt to parse it is unexpected`, giveTemplate)
			continue
		}

		_, placeholderWithoutFlags, flags := parsePlaceholder(giveTemplate)
		gotTemplate := placeholderWithoutFlags[:1] + flags.encodePlaceholder() + placeholderWithoutFlags[1:]

		if gotTemplate != wantTemplate {
			t.Errorf(`parsed placeholder "%s" into "%s", but want "%s"`, giveTemplate, gotTemplate, wantTemplate)
		}
	}
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
	stripAnsi := false
	forcePlus := false

	// evaluate the test cases
	for idx, test := range tests {
		gotOutput := replacePlaceholder(
			test.give.template, stripAnsi, delimiter, printsep, forcePlus,
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
			wantMatch := strings.ReplaceAll(test.want.match, `\`, `\\`)
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

// naive encoder of placeholder flags
func (flags placeholderFlags) encodePlaceholder() string {
	encoded := ""
	if flags.plus {
		encoded += "+"
	}
	if flags.preserveSpace {
		encoded += "s"
	}
	if flags.number {
		encoded += "n"
	}
	if flags.file {
		encoded += "f"
	}
	if flags.query {
		encoded += "q"
	}
	return encoded
}

// can be replaced with os.ReadFile() in go 1.16+
func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data := make([]byte, 0, 128)
	for {
		if len(data) >= cap(data) {
			d := append(data[:cap(data)], 0)
			data = d[:len(data)]
		}

		n, err := file.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, err
		}
	}
}
