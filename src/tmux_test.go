package fzf

import "testing"

func TestEscapeTmuxFormat(t *testing.T) {
	for _, tc := range []struct {
		given    string
		expected string
	}{
		{"", ""},
		{" fzf ", " fzf "},
		{"100%", "100%"},
		// '#[' would be taken as a style directive when the label is drawn
		{"#[fg=red]x", "##[fg=red]x"},
		// '#{' is already literal in a substituted value, but doubling it
		// keeps a label that fzf renders itself looking the same
		{"#{pane_id}", "##{pane_id}"},
		{" C# notes #S ", " C## notes ##S "},
		{";", `\;`},
		{"; rm", "; rm"},
	} {
		if actual := escapeTmuxFormat(tc.given); actual != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, actual)
		}
	}
}

func TestEscapeTmuxSeparator(t *testing.T) {
	for _, tc := range []struct {
		given    string
		expected string
	}{
		{"", ""},
		{" fzf ", " fzf "},
		{"#", "#"},
		{"#{pane_id}", "#{pane_id}"},
		{"100%", "100%"},
		{";", `\;`},
		{"; rm", "; rm"},
		{" ; ", " ; "},
		{";;", ";;"},
	} {
		if actual := escapeTmuxSeparator(tc.given); actual != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, actual)
		}
	}
}
