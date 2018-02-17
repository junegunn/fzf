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
	"github.com/lucasb-eyer/go-colorful"
	"math"
)

// FindColor attempts to find a given color, or the best match possible for it,
// from the palette given.  This is an expensive operation, so results should
// be cached by the caller.
func FindColor(c Color, palette []Color) Color {
	match := ColorDefault
	dist := float64(0)
	r, g, b := c.RGB()
	c1 := colorful.Color{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
	}
	for _, d := range palette {
		r, g, b = d.RGB()
		c2 := colorful.Color{
			R: float64(r) / 255.0,
			G: float64(g) / 255.0,
			B: float64(b) / 255.0,
		}
		// CIE94 is more accurate, but really really expensive.
		nd := c1.DistanceCIE76(c2)
		if math.IsNaN(nd) {
			nd = math.Inf(1)
		}
		if match == ColorDefault || nd < dist {
			match = d
			dist = nd
		}
	}
	return match
}
