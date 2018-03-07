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

// EventInterrupt is a generic wakeup event.  Its can be used to
// to request a redraw.  It can carry an arbitrary payload, as well.
type EventInterrupt struct {
	t time.Time
	v interface{}
}

// When returns the time when this event was created.
func (ev *EventInterrupt) When() time.Time {
	return ev.t
}

// Data is used to obtain the opaque event payload.
func (ev *EventInterrupt) Data() interface{} {
	return ev.v
}

// NewEventInterrupt creates an EventInterrupt with the given payload.
func NewEventInterrupt(data interface{}) *EventInterrupt {
	return &EventInterrupt{t: time.Now(), v: data}
}
