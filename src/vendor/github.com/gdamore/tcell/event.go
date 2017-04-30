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

// Event is a generic interface used for passing around Events.
// Concrete types follow.
type Event interface {
	// When reports the time when the event was generated.
	When() time.Time
}

// EventTime is a simple base event class, suitable for easy reuse.
// It can be used to deliver actual timer events as well.
type EventTime struct {
	when time.Time
}

// When returns the time stamp when the event occurred.
func (e *EventTime) When() time.Time {
	return e.when
}

// SetEventTime sets the time of occurrence for the event.
func (e *EventTime) SetEventTime(t time.Time) {
	e.when = t
}

// SetEventNow sets the time of occurrence for the event to the current time.
func (e *EventTime) SetEventNow() {
	e.SetEventTime(time.Now())
}

// EventHandler is anything that handles events.  If the handler has
// consumed the event, it should return true.  False otherwise.
type EventHandler interface {
	HandleEvent(Event) bool
}
