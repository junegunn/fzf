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

// ISO8859_1 represents the 8-bit ISO8859-1 scheme.  It decodes directly to
// UTF-8 without change, as all ISO8859-1 values are legal UTF-8.
// Unicode values less than 256 (i.e. 8 bits) map 1:1 with 8859-1.
// It encodes runes outside of that to 0x1A, the ASCII substitution character.
var ISO8859_1 encoding.Encoding

func init() {
	cm := &Charmap{}
	cm.Init()

	// 8859-1 is the 8-bit identity map for Unicode.
	ISO8859_1 = cm
}
