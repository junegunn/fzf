package fzf

import (
	"bytes"
	"regexp"
	"testing"
	"text/template"

	"github.com/junegunn/fzf/src/util"
)

func newItem(str string) *Item {
	bytes := []byte(str)
	trimmed, _, _ := extractColor(str, nil, nil)
	return &Item{origText: &bytes, text: util.ToChars([]byte(trimmed))}
}

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

// Helper function to parse, execute and convert "text/template" to string. Panics on error.
func templateToString(format string, data interface{}) string {
	bb := &bytes.Buffer{}

	err := template.Must(template.New("").Parse(format)).Execute(bb, data)
	if err != nil {
		panic(err)
	}

	return bb.String()
}
