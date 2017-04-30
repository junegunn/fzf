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

// ISO8859_9 represents the 8-bit ISO8859-9 scheme.
var ISO8859_9 encoding.Encoding

func init() {
	cm := &Charmap{Map: map[byte]rune{
		0xD0: 'Ğ',
		0xDD: 'İ',
		0xDE: 'Ş',
		0xF0: 'ğ',
		0xFD: 'ı',
		0xFE: 'ş',
	}}
	cm.Init()
	ISO8859_9 = cm
}
