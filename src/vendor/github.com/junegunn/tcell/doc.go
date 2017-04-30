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

// Package tcell provides a lower-level, portable API for building
// programs that interact with terminals or consoles.  It works with
// both common (and many uncommon!) terminals or terminal emulators,
// and Windows console implementations.
//
// It provides support for up to 256 colors, text attributes, and box drawing
// elements.  A database of terminals built from a real terminfo database
// is provided, along with code to generate new database entries.
//
// Tcell offers very rich support for mice, dependent upon the terminal
// of course.  (Windows, XTerm, and iTerm 2 are known to work very well.)
//
// If the environment is not Unicode by default, such as an ISO8859 based
// locale or GB18030, Tcell can convert input and outupt, so that your
// terminal can operate in whatever locale is most convenient, while the
// application program can just assume "everything is UTF-8".  Reasonable
// defaults are used for updating characters to something suitable for
// display.  Unicode box drawing characters will be converted to use the
// alternate character set of your terminal, if native conversions are
// not available.  If no ACS is available, then some ASCII fallbacks will
// be used.
//
// A rich set of keycodes is supported, with support for up to 65 function
// keys, and various other special keys.
//
package tcell
