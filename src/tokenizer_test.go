package fzf

import "testing"

func TestParseRange(t *testing.T) {
	{
		i := ".."
		r, _ := ParseRange(&i)
		if r.begin != RANGE_ELLIPSIS || r.end != RANGE_ELLIPSIS {
			t.Errorf("%s", r)
		}
	}
	{
		i := "3.."
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != RANGE_ELLIPSIS {
			t.Errorf("%s", r)
		}
	}
	{
		i := "3..5"
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != 5 {
			t.Errorf("%s", r)
		}
	}
	{
		i := "-3..-5"
		r, _ := ParseRange(&i)
		if r.begin != -3 || r.end != -5 {
			t.Errorf("%s", r)
		}
	}
	{
		i := "3"
		r, _ := ParseRange(&i)
		if r.begin != 3 || r.end != 3 {
			t.Errorf("%s", r)
		}
	}
}

func TestTokenize(t *testing.T) {
	// AWK-style
	input := "  abc:  def:  ghi  "
	tokens := Tokenize(&input, nil)
	if *tokens[0].text != "abc:  " || tokens[0].prefixLength != 2 {
		t.Errorf("%s", tokens)
	}

	// With delimiter
	tokens = Tokenize(&input, delimiterRegexp(":"))
	if *tokens[0].text != "  abc:" || tokens[0].prefixLength != 0 {
		t.Errorf("%s", tokens)
	}
}

func TestTransform(t *testing.T) {
	input := "  abc:  def:  ghi:  jkl"
	{
		tokens := Tokenize(&input, nil)
		{
			ranges := splitNth("1,2,3")
			tx := Transform(tokens, ranges)
			if *tx.whole != "abc:  def:  ghi:  " {
				t.Errorf("%s", *tx)
			}
		}
		{
			ranges := splitNth("1..2,3,2..,1")
			tx := Transform(tokens, ranges)
			if *tx.whole != "abc:  def:  ghi:  def:  ghi:  jklabc:  " ||
				len(tx.parts) != 4 ||
				*tx.parts[0].text != "abc:  def:  " || tx.parts[0].prefixLength != 2 ||
				*tx.parts[1].text != "ghi:  " || tx.parts[1].prefixLength != 14 ||
				*tx.parts[2].text != "def:  ghi:  jkl" || tx.parts[2].prefixLength != 8 ||
				*tx.parts[3].text != "abc:  " || tx.parts[3].prefixLength != 2 {
				t.Errorf("%s", *tx)
			}
		}
	}
	{
		tokens := Tokenize(&input, delimiterRegexp(":"))
		{
			ranges := splitNth("1..2,3,2..,1")
			tx := Transform(tokens, ranges)
			if *tx.whole != "  abc:  def:  ghi:  def:  ghi:  jkl  abc:" ||
				len(tx.parts) != 4 ||
				*tx.parts[0].text != "  abc:  def:" || tx.parts[0].prefixLength != 0 ||
				*tx.parts[1].text != "  ghi:" || tx.parts[1].prefixLength != 12 ||
				*tx.parts[2].text != "  def:  ghi:  jkl" || tx.parts[2].prefixLength != 6 ||
				*tx.parts[3].text != "  abc:" || tx.parts[3].prefixLength != 0 {
				t.Errorf("%s", *tx)
			}
		}
	}
}
