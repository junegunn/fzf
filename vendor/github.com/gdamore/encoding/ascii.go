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
	"golang.org/x/text/encoding"
)

// ASCII represents the 7-bit US-ASCII scheme.  It decodes directly to
// UTF-8 without change, as all ASCII values are legal UTF-8.
// Unicode values less than 128 (i.e. 7 bits) map 1:1 with ASCII.
// It encodes runes outside of that to 0x1A, the ASCII substitution character.
var ASCII encoding.Encoding

func init() {
	amap := make(map[byte]rune)
	for i := 128; i <= 255; i++ {
		amap[byte(i)] = RuneError
	}

	cm := &Charmap{Map: amap}
	cm.Init()
	ASCII = cm
}
