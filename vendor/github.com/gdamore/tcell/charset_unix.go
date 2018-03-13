// +build !windows,!nacl,!plan9

// Copyright 2016 The TCell Authors
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
	"os"
	"strings"
)

func getCharset() string {
	// Determine the character set.  This can help us later.
	// Per POSIX, we search for LC_ALL first, then LC_CTYPE, and
	// finally LANG.  First one set wins.
	locale := ""
	if locale = os.Getenv("LC_ALL"); locale == "" {
		if locale = os.Getenv("LC_CTYPE"); locale == "" {
			locale = os.Getenv("LANG")
		}
	}
	if locale == "POSIX" || locale == "C" {
		return "US-ASCII"
	}
	if i := strings.IndexRune(locale, '@'); i >= 0 {
		locale = locale[:i]
	}
	if i := strings.IndexRune(locale, '.'); i >= 0 {
		locale = locale[i+1:]
	} else {
		// Default assumption, and on Linux we can see LC_ALL
		// without a character set, which we assume implies UTF-8.
		return "UTF-8"
	}
	// XXX: add support for aliases
	return locale
}
