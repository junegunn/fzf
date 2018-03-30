package fzf

import (
	"regexp"
	"testing"

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

	// {}, preserve ansi
	result = replacePlaceholder("echo {}", false, Delimiter{}, false, "query", items1)
	check("echo '  foo'\\''bar \x1b[31mbaz\x1b[m'")

	// {}, strip ansi
	result = replacePlaceholder("echo {}", true, Delimiter{}, false, "query", items1)
	check("echo '  foo'\\''bar baz'")

	// {}, with multiple items
	result = replacePlaceholder("echo {}", true, Delimiter{}, false, "query", items2)
	check("echo 'foo'\\''bar baz'")

	// {..}, strip leading whitespaces, preserve ansi
	result = replacePlaceholder("echo {..}", false, Delimiter{}, false, "query", items1)
	check("echo 'foo'\\''bar \x1b[31mbaz\x1b[m'")

	// {..}, strip leading whitespaces, strip ansi
	result = replacePlaceholder("echo {..}", true, Delimiter{}, false, "query", items1)
	check("echo 'foo'\\''bar baz'")

	// {q}
	result = replacePlaceholder("echo {} {q}", true, Delimiter{}, false, "query", items1)
	check("echo '  foo'\\''bar baz' 'query'")

	// {q}, multiple items
	result = replacePlaceholder("echo {+}{q}{+}", true, Delimiter{}, false, "query 'string'", items2)
	check("echo 'foo'\\''bar baz' 'FOO'\\''BAR BAZ''query '\\''string'\\''''foo'\\''bar baz' 'FOO'\\''BAR BAZ'")

	result = replacePlaceholder("echo {}{q}{}", true, Delimiter{}, false, "query 'string'", items2)
	check("echo 'foo'\\''bar baz''query '\\''string'\\''''foo'\\''bar baz'")

	result = replacePlaceholder("echo {1}/{2}/{2,1}/{-1}/{-2}/{}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, false, "query", items1)
	check("echo 'foo'\\''bar'/'baz'/'bazfoo'\\''bar'/'baz'/'foo'\\''bar'/'  foo'\\''bar baz'/'foo'\\''bar baz'/{n.t}/{}/{1}/{q}/''")

	result = replacePlaceholder("echo {1}/{2}/{-1}/{-2}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, false, "query", items2)
	check("echo 'foo'\\''bar'/'baz'/'baz'/'foo'\\''bar'/'foo'\\''bar baz'/{n.t}/{}/{1}/{q}/''")

	result = replacePlaceholder("echo {+1}/{+2}/{+-1}/{+-2}/{+..}/{n.t}/\\{}/\\{1}/\\{q}/{+3}", true, Delimiter{}, false, "query", items2)
	check("echo 'foo'\\''bar' 'FOO'\\''BAR'/'baz' 'BAZ'/'baz' 'BAZ'/'foo'\\''bar' 'FOO'\\''BAR'/'foo'\\''bar baz' 'FOO'\\''BAR BAZ'/{n.t}/{}/{1}/{q}/'' ''")

	// forcePlus
	result = replacePlaceholder("echo {1}/{2}/{-1}/{-2}/{..}/{n.t}/\\{}/\\{1}/\\{q}/{3}", true, Delimiter{}, true, "query", items2)
	check("echo 'foo'\\''bar' 'FOO'\\''BAR'/'baz' 'BAZ'/'baz' 'BAZ'/'foo'\\''bar' 'FOO'\\''BAR'/'foo'\\''bar baz' 'FOO'\\''BAR BAZ'/{n.t}/{}/{1}/{q}/'' ''")

	// Whitespace preserving flag with "'" delimiter
	result = replacePlaceholder("echo {s1}", true, Delimiter{str: &delim}, false, "query", items1)
	check("echo '  foo'")

	result = replacePlaceholder("echo {s2}", true, Delimiter{str: &delim}, false, "query", items1)
	check("echo 'bar baz'")

	result = replacePlaceholder("echo {s}", true, Delimiter{str: &delim}, false, "query", items1)
	check("echo '  foo'\\''bar baz'")

	result = replacePlaceholder("echo {s..}", true, Delimiter{str: &delim}, false, "query", items1)
	check("echo '  foo'\\''bar baz'")

	// Whitespace preserving flag with regex delimiter
	regex = regexp.MustCompile("\\w+")

	result = replacePlaceholder("echo {s1}", true, Delimiter{regex: regex}, false, "query", items1)
	check("echo '  '")

	result = replacePlaceholder("echo {s2}", true, Delimiter{regex: regex}, false, "query", items1)
	check("echo ''\\'''")

	result = replacePlaceholder("echo {s3}", true, Delimiter{regex: regex}, false, "query", items1)
	check("echo ' '")

	// No match
	result = replacePlaceholder("echo {}/{+}", true, Delimiter{}, false, "query", []*Item{nil, nil})
	check("echo /")

	// No match, but with selections
	result = replacePlaceholder("echo {}/{+}", true, Delimiter{}, false, "query", []*Item{nil, item1})
	check("echo /'  foo'\\''bar baz'")

	// String delimiter
	result = replacePlaceholder("echo {}/{1}/{2}", true, Delimiter{str: &delim}, false, "query", items1)
	check("echo '  foo'\\''bar baz'/'foo'/'bar baz'")

	// Regex delimiter
	regex = regexp.MustCompile("[oa]+")
	// foo'bar baz
	result = replacePlaceholder("echo {}/{1}/{3}/{2..3}", true, Delimiter{regex: regex}, false, "query", items1)
	check("echo '  foo'\\''bar baz'/'f'/'r b'/''\\''bar b'")
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
