// Copyright 2015 The TCell Authors
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
	"github.com/gdamore/encoding"
	"github.com/gdamore/tcell"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// Register registers all known encodings.  This is a short-cut to
// add full character set support to your program.  Note that this can
// add several megabytes to your program's size, because some of the encoodings
// are rather large (particularly those from East Asia.)
func Register() {
	// We supply latin1 and latin5, because Go doesn't
	tcell.RegisterEncoding("ISO8859-1", encoding.ISO8859_1)
	tcell.RegisterEncoding("ISO8859-9", encoding.ISO8859_9)

	tcell.RegisterEncoding("ISO8859-10", charmap.ISO8859_10)
	tcell.RegisterEncoding("ISO8859-13", charmap.ISO8859_13)
	tcell.RegisterEncoding("ISO8859-14", charmap.ISO8859_14)
	tcell.RegisterEncoding("ISO8859-15", charmap.ISO8859_15)
	tcell.RegisterEncoding("ISO8859-16", charmap.ISO8859_16)
	tcell.RegisterEncoding("ISO8859-2", charmap.ISO8859_2)
	tcell.RegisterEncoding("ISO8859-3", charmap.ISO8859_3)
	tcell.RegisterEncoding("ISO8859-4", charmap.ISO8859_4)
	tcell.RegisterEncoding("ISO8859-5", charmap.ISO8859_5)
	tcell.RegisterEncoding("ISO8859-6", charmap.ISO8859_6)
	tcell.RegisterEncoding("ISO8859-7", charmap.ISO8859_7)
	tcell.RegisterEncoding("ISO8859-8", charmap.ISO8859_8)
	tcell.RegisterEncoding("KOI8-R", charmap.KOI8R)
	tcell.RegisterEncoding("KOI8-U", charmap.KOI8U)

	// Asian stuff
	tcell.RegisterEncoding("EUC-JP", japanese.EUCJP)
	tcell.RegisterEncoding("SHIFT_JIS", japanese.ShiftJIS)
	tcell.RegisterEncoding("ISO2022JP", japanese.ISO2022JP)

	tcell.RegisterEncoding("EUC-KR", korean.EUCKR)

	tcell.RegisterEncoding("GB18030", simplifiedchinese.GB18030)
	tcell.RegisterEncoding("GB2312", simplifiedchinese.HZGB2312)
	tcell.RegisterEncoding("GBK", simplifiedchinese.GBK)

	tcell.RegisterEncoding("Big5", traditionalchinese.Big5)

	// Common aliaess
	aliases := map[string]string{
		"8859-1":      "ISO8859-1",
		"ISO-8859-1":  "ISO8859-1",
		"8859-13":     "ISO8859-13",
		"ISO-8859-13": "ISO8859-13",
		"8859-14":     "ISO8859-14",
		"ISO-8859-14": "ISO8859-14",
		"8859-15":     "ISO8859-15",
		"ISO-8859-15": "ISO8859-15",
		"8859-16":     "ISO8859-16",
		"ISO-8859-16": "ISO8859-16",
		"8859-2":      "ISO8859-2",
		"ISO-8859-2":  "ISO8859-2",
		"8859-3":      "ISO8859-3",
		"ISO-8859-3":  "ISO8859-3",
		"8859-4":      "ISO8859-4",
		"ISO-8859-4":  "ISO8859-4",
		"8859-5":      "ISO8859-5",
		"ISO-8859-5":  "ISO8859-5",
		"8859-6":      "ISO8859-6",
		"ISO-8859-6":  "ISO8859-6",
		"8859-7":      "ISO8859-7",
		"ISO-8859-7":  "ISO8859-7",
		"8859-8":      "ISO8859-8",
		"ISO-8859-8":  "ISO8859-8",
		"8859-9":      "ISO8859-9",
		"ISO-8859-9":  "ISO8859-9",

		"SJIS":        "Shift_JIS",
		"EUCJP":       "EUC-JP",
		"2022-JP":     "ISO2022JP",
		"ISO-2022-JP": "ISO2022JP",

		"EUCKR": "EUC-KR",

		// ISO646 isn't quite exactly ASCII, but the 1991 IRV
		// (international reference version) is so.  This helps
		// some older systems that may use "646" for POSIX locales.
		"646":    "US-ASCII",
		"ISO646": "US-ASCII",

		// Other names for UTF-8
		"UTF8": "UTF-8",
	}
	for n, v := range aliases {
		if enc := tcell.GetEncoding(v); enc != nil {
			tcell.RegisterEncoding(n, enc)
		}
	}
}
