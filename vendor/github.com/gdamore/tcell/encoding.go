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

package tcell

import (
	"strings"
	"sync"

	"golang.org/x/text/encoding"

	gencoding "github.com/gdamore/encoding"
)

var encodings map[string]encoding.Encoding
var encodingLk sync.Mutex
var encodingFallback EncodingFallback = EncodingFallbackFail

// RegisterEncoding may be called by the application to register an encoding.
// The presence of additional encodings will facilitate application usage with
// terminal environments where the I/O subsystem does not support Unicode.
//
// Windows systems use Unicode natively, and do not need any of the encoding
// subsystem when using Windows Console screens.
//
// Please see the Go documentation for golang.org/x/text/encoding -- most of
// the common ones exist already as stock variables.  For example, ISO8859-15
// can be registered using the following code:
//
//   import "golang.org/x/text/encoding/charmap"
//
//     ...
//     RegisterEncoding("ISO8859-15", charmap.ISO8859_15)
//
// Aliases can be registered as well, for example "8859-15" could be an alias
// for "ISO8859-15".
//
// For POSIX systems, the tcell package will check the environment variables
// LC_ALL, LC_CTYPE,  and LANG (in that order) to determine the character set.
// These are expected to have the following pattern:
//
//	 $language[.$codeset[@$variant]
//
// We extract only the $codeset part, which will usually be something like
// UTF-8 or ISO8859-15 or KOI8-R.  Note that if the locale is either "POSIX"
// or "C", then we assume US-ASCII (the POSIX 'portable character set'
// and assume all other characters are somehow invalid.)
//
// Modern POSIX systems and terminal emulators may use UTF-8, and for those
// systems, this API is also unnecessary.  For example, Darwin (MacOS X) and
// modern Linux running modern xterm generally will out of the box without
// any of this.  Use of UTF-8 is recommended when possible, as it saves
// quite a lot processing overhead.
//
// Note that some encodings are quite large (for example GB18030 which is a
// superset of Unicode) and so the application size can be expected ot
// increase quite a bit as each encoding is added.  The East Asian encodings
// have been seen to add 100-200K per encoding to the application size.
//
func RegisterEncoding(charset string, enc encoding.Encoding) {
	encodingLk.Lock()
	charset = strings.ToLower(charset)
	encodings[charset] = enc
	encodingLk.Unlock()
}

// EncodingFallback describes how the system behavees when the locale
// requires a character set that we do not support.  The system always
// supports UTF-8 and US-ASCII. On Windows consoles, UTF-16LE is also
// supported automatically.  Other character sets must be added using the
// RegisterEncoding API.  (A large group of nearly all of them can be
// added using the RegisterAll function in the encoding sub package.)
type EncodingFallback int

const (
	// EncodingFallbackFail behavior causes GetEncoding to fail
	// when it cannot find an encoding.
	EncodingFallbackFail = iota

	// EncodingFallbackASCII behaviore causes GetEncoding to fall back
	// to a 7-bit ASCII encoding, if no other encoding can be found.
	EncodingFallbackASCII

	// EncodingFallbackUTF8 behavior causes GetEncoding to assume
	// UTF8 can pass unmodified upon failure.  Note that this behavior
	// is not recommended, unless you are sure your terminal can cope
	// with real UTF8 sequences.
	EncodingFallbackUTF8
)

// SetEncodingFallback changes the behavior of GetEncoding when a suitable
// encoding is not found.  The default is EncodingFallbackFail, which
// causes GetEncoding to simply return nil.
func SetEncodingFallback(fb EncodingFallback) {
	encodingLk.Lock()
	encodingFallback = fb
	encodingLk.Unlock()
}

// GetEncoding is used by Screen implementors who want to locate an encoding
// for the given character set name.  Note that this will return nil for
// either the Unicode (UTF-8) or ASCII encodings, since we don't use
// encodings for them but instead have our own native methods.
func GetEncoding(charset string) encoding.Encoding {
	charset = strings.ToLower(charset)
	encodingLk.Lock()
	defer encodingLk.Unlock()
	if enc, ok := encodings[charset]; ok {
		return enc
	}
	switch encodingFallback {
	case EncodingFallbackASCII:
		return gencoding.ASCII
	case EncodingFallbackUTF8:
		return encoding.Nop
	}
	return nil
}

func init() {
	// We always support UTF-8 and ASCII.
	encodings = make(map[string]encoding.Encoding)
	encodings["utf-8"] = gencoding.UTF8
	encodings["utf8"] = gencoding.UTF8
	encodings["us-ascii"] = gencoding.ASCII
	encodings["ascii"] = gencoding.ASCII
	encodings["iso646"] = gencoding.ASCII
}
