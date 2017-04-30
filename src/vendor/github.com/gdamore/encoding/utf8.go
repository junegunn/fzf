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

type validUtf8 struct{}

// UTF8 is an encoding for UTF-8.  All it does is verify that the UTF-8
// in is valid.  The main reason for its existence is that it will detect
// and report ErrSrcShort or ErrDstShort, whereas the Nop encoding just
// passes every byte, blithely.
var UTF8 encoding.Encoding = validUtf8{}

func (validUtf8) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: encoding.UTF8Validator}
}

func (validUtf8) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: encoding.UTF8Validator}
}
