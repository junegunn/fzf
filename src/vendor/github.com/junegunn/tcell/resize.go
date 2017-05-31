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
	"time"
)

// EventResize is sent when the window size changes.
type EventResize struct {
	t time.Time
	w int
	h int
}

// NewEventResize creates an EventResize with the new updated window size,
// which is given in character cells.
func NewEventResize(width, height int) *EventResize {
	return &EventResize{t: time.Now(), w: width, h: height}
}

// When returns the time when the Event was created.
func (ev *EventResize) When() time.Time {
	return ev.t
}

// Size returns the new window size as width, height in character cells.
func (ev *EventResize) Size() (int, int) {
	return ev.w, ev.h
}
