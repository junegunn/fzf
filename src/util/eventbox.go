package util

import "sync"

// EventType is the type for fzf events
type EventType int

// Events is a type that associates EventType to any data
type Events map[EventType]any

// EventBox is used for coordinating events
type EventBox struct {
	events Events
	cond   *sync.Cond
	ignore map[EventType]bool
}

// NewEventBox returns a new EventBox
func NewEventBox() *EventBox {
	return &EventBox{
		events: make(Events),
		cond:   sync.NewCond(&sync.Mutex{}),
		ignore: make(map[EventType]bool)}
}

// Wait blocks the goroutine until signaled
func (b *EventBox) Wait(callback func(*Events)) {
	b.cond.L.Lock()

	if len(b.events) == 0 {
		b.cond.Wait()
	}

	callback(&b.events)
	b.cond.L.Unlock()
}

// Set turns on the event type on the box
func (b *EventBox) Set(event EventType, value any) {
	b.cond.L.Lock()
	b.events[event] = value
	if _, found := b.ignore[event]; !found {
		b.cond.Broadcast()
	}
	b.cond.L.Unlock()
}

// Clear clears the events
// Unsynchronized; should be called within Wait routine
func (events *Events) Clear() {
	for event := range *events {
		delete(*events, event)
	}
}

// Peek peeks at the event box if the given event is set
func (b *EventBox) Peek(event EventType) bool {
	b.cond.L.Lock()
	_, ok := b.events[event]
	b.cond.L.Unlock()
	return ok
}

// Watch deletes the events from the ignore list
func (b *EventBox) Watch(events ...EventType) {
	b.cond.L.Lock()
	for _, event := range events {
		delete(b.ignore, event)
	}
	b.cond.L.Unlock()
}

// Unwatch adds the events to the ignore list
func (b *EventBox) Unwatch(events ...EventType) {
	b.cond.L.Lock()
	for _, event := range events {
		b.ignore[event] = true
	}
	b.cond.L.Unlock()
}

// WaitFor blocks the execution until the event is received
func (b *EventBox) WaitFor(event EventType) {
	looping := true
	for looping {
		b.Wait(func(events *Events) {
			for evt := range *events {
				switch evt {
				case event:
					looping = false
					return
				}
			}
		})
	}
}
