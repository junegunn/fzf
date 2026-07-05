package fzf

import "testing"

func TestEscapeTmuxTitle(t *testing.T) {
	for _, tc := range []struct {
		given    string
		expected string
	}{
		{"", ""},
		{" fzf ", " fzf "},
		{"#", "##"},
		{"##", "####"},
		{" C# notes #S ", " C## notes ##S "},
		{"100%", "100%"},
		{";", `\;`},
		{"; rm", "; rm"},
		{" ; ", " ; "},
	} {
		if actual := escapeTmuxTitle(tc.given); actual != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, actual)
		}
	}
}

func TestEscapeTmuxTitleSeparator(t *testing.T) {
	for _, tc := range []struct {
		given    string
		expected string
	}{
		{"#;", "##;"},
		{";#", ";##"},
		{";;", ";;"},
	} {
		if actual := escapeTmuxTitle(tc.given); actual != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, actual)
		}
	}
}
