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

// EventMouse is a mouse event.  It is sent on either mouse up or mouse down
// events.  It is also sent on mouse motion events - if the terminal supports
// it.  We make every effort to ensure that mouse release events are delivered.
// Hence, click drag can be identified by a motion event with the mouse down,
// without any intervening button release.  On some terminals only the initiating
// press and terminating release event will be delivered.
//
// Mouse wheel events, when reported, may appear on their own as individual
// impulses; that is, there will normally not be a release event delivered
// for mouse wheel movements.
//
// Most terminals cannot report the state of more than one button at a time --
// and some cannot report motion events unless a button is pressed.
//
// Applications can inspect the time between events to resolve double or
// triple clicks.
type EventMouse struct {
	t   time.Time
	btn ButtonMask
	mod ModMask
	x   int
	y   int
}

// When returns the time when this EventMouse was created.
func (ev *EventMouse) When() time.Time {
	return ev.t
}

// Buttons returns the list of buttons that were pressed or wheel motions.
func (ev *EventMouse) Buttons() ButtonMask {
	return ev.btn
}

// Modifiers returns a list of keyboard modifiers that were pressed
// with the mouse button(s).
func (ev *EventMouse) Modifiers() ModMask {
	return ev.mod
}

// Position returns the mouse position in character cells.  The origin
// 0, 0 is at the upper left corner.
func (ev *EventMouse) Position() (int, int) {
	return ev.x, ev.y
}

// NewEventMouse is used to create a new mouse event.  Applications
// shouldn't need to use this; its mostly for screen implementors.
func NewEventMouse(x, y int, btn ButtonMask, mod ModMask) *EventMouse {
	return &EventMouse{t: time.Now(), x: x, y: y, btn: btn, mod: mod}
}

// ButtonMask is a mask of mouse buttons and wheel events.  Mouse button presses
// are normally delivered as both press and release events.  Mouse wheel events
// are normally just single impulse events.  Windows supports up to eight
// separate buttons plus all four wheel directions, but XTerm can only support
// mouse buttons 1-3 and wheel up/down.  Its not unheard of for terminals
// to support only one or two buttons (think Macs).  Old terminals, and true
// emulations (such as vt100) won't support mice at all, of course.
type ButtonMask int16

// These are the actual button values.
const (
	Button1 ButtonMask = 1 << iota // Usually left mouse button.
	Button2                        // Usually the middle mouse button.
	Button3                        // Usually the right mouse button.
	Button4                        // Often a side button (thumb/next).
	Button5                        // Often a side button (thumb/prev).
	Button6
	Button7
	Button8
	WheelUp                   // Wheel motion up/away from user.
	WheelDown                 // Wheel motion down/towards user.
	WheelLeft                 // Wheel motion to left.
	WheelRight                // Wheel motion to right.
	ButtonNone ButtonMask = 0 // No button or wheel events.
)
