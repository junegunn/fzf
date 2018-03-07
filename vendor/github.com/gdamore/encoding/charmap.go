// Copyright 2015 Garrett D'Amore
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package encoding

import (
	"sync"
	"unicode/utf8"

	"golang.org/x/text/transform"
	"golang.org/x/text/encoding"
)

const (
	// RuneError is an alias for the UTF-8 replacement rune, '\uFFFD'.
	RuneError = '\uFFFD'

	// RuneSelf is the rune below which UTF-8 and the Unicode values are
	// identical.  Its also the limit for ASCII.
	RuneSelf = 0x80

	// ASCIISub is the ASCII substitution character.
	ASCIISub = '\x1a'
)

// Charmap is a structure for setting up encodings for 8-bit character sets,
// for transforming between UTF8 and that other character set.  It has some
// ideas borrowed from golang.org/x/text/encoding/charmap, but it uses a
// different implementation.  This implementation uses maps, and supports
// user-defined maps.
//
// We do assume that a character map has a reasonable substitution character,
// and that valid encodings are stable (exactly a 1:1 map) and stateless
// (that is there is no shift character or anything like that.)  Hence this
// approach will not work for many East Asian character sets.
//
// Measurement shows little or no measurable difference in the performance of
// the two approaches.  The difference was down to a couple of nsec/op, and
// no consistent pattern as to which ran faster.  With the conversion to
// UTF-8 the code takes about 25 nsec/op.  The conversion in the reverse
// direction takes about 100 nsec/op.  (The larger cost for conversion
// from UTF-8 is most likely due to the need to convert the UTF-8 byte stream
// to a rune before conversion.
//
type Charmap struct {
	transform.NopResetter
	bytes map[rune]byte
	runes [256][]byte
	once  sync.Once

	// The map between bytes and runes.  To indicate that a specific
	// byte value is invalid for a charcter set, use the rune
	// utf8.RuneError.  Values that are absent from this map will
	// be assumed to have the identity mapping -- that is the default
	// is to assume ISO8859-1, where all 8-bit characters have the same
	// numeric value as their Unicode runes.  (Not to be confused with
	// the UTF-8 values, which *will* be different for non-ASCII runes.)
	//
	// If no values less than RuneSelf are changed (or have non-identity
	// mappings), then the character set is assumed to be an ASCII
	// superset, and certain assumptions and optimizations become
	// available for ASCII bytes.
	Map map[byte]rune

	// The ReplacementChar is the byte value to use for substitution.
	// It should normally be ASCIISub for ASCII encodings.  This may be
	// unset (left to zero) for mappings that are strictly ASCII supersets.
	// In that case ASCIISub will be assumed instead.
	ReplacementChar byte
}

type cmapDecoder struct {
	transform.NopResetter
	runes [256][]byte
}

type cmapEncoder struct {
	transform.NopResetter
	bytes   map[rune]byte
	replace byte
}

// Init initializes internal values of a character map.  This should
// be done early, to minimize the cost of allocation of transforms
// later.  It is not strictly necessary however, as the allocation
// functions will arrange to call it if it has not already been done.
func (c *Charmap) Init() {
	c.once.Do(c.initialize)
}

func (c *Charmap) initialize() {
	c.bytes = make(map[rune]byte)
	ascii := true

	for i := 0; i < 256; i++ {
		r, ok := c.Map[byte(i)]
		if !ok {
			r = rune(i)
		}
		if r < 128 && r != rune(i) {
			ascii = false
		}
		if r != RuneError {
			c.bytes[r] = byte(i)
		}
		utf := make([]byte, utf8.RuneLen(r))
		utf8.EncodeRune(utf, r)
		c.runes[i] = utf
	}
	if ascii && c.ReplacementChar == '\x00' {
		c.ReplacementChar = ASCIISub
	}
}

// NewDecoder returns a Decoder the converts from the 8-bit
// character set to UTF-8.  Unknown mappings, if any, are mapped
// to '\uFFFD'.
func (c *Charmap) NewDecoder() *encoding.Decoder {
	c.Init()
	return &encoding.Decoder{Transformer: &cmapDecoder{runes: c.runes}}
}

// NewEncoder returns a Transformer that converts from UTF8 to the
// 8-bit character set.  Unknown mappings are mapped to 0x1A.
func (c *Charmap) NewEncoder() *encoding.Encoder {
	c.Init()
	return &encoding.Encoder{Transformer:
	    &cmapEncoder{bytes: c.bytes, replace: c.ReplacementChar}}
}

func (d *cmapDecoder) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	var e error
	var ndst, nsrc int

	for _, c := range src {
		b := d.runes[c]
		l := len(b)

		if ndst+l > len(dst) {
			e = transform.ErrShortDst
			break
		}
		for i := 0; i < l; i++ {
			dst[ndst] = b[i]
			ndst++
		}
		nsrc++
	}
	return ndst, nsrc, e
}

func (d *cmapEncoder) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	var e error
	var ndst, nsrc int
	for nsrc < len(src) {
		if ndst >= len(dst) {
			e = transform.ErrShortDst
			break
		}

		r, sz := utf8.DecodeRune(src[nsrc:])
		if r == utf8.RuneError && sz == 1 {
			// If its inconclusive due to insufficient data in
			// in the source, report it
			if !atEOF && !utf8.FullRune(src[nsrc:]) {
				e = transform.ErrShortSrc
				break
			}
		}

		if c, ok := d.bytes[r]; ok {
			dst[ndst] = c
		} else {
			dst[ndst] = d.replace
		}
		nsrc += sz
		ndst++
	}

	return ndst, nsrc, e
}
