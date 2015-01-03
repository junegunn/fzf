package fzf

import "sync"

type EventType int

type Events map[EventType]interface{}

type EventBox struct {
	events Events
	cond   *sync.Cond
	ignore map[EventType]bool
}

func NewEventBox() *EventBox {
	return &EventBox{
		events: make(Events),
		cond:   sync.NewCond(&sync.Mutex{}),
		ignore: make(map[EventType]bool)}
}

func (b *EventBox) Wait(callback func(*Events)) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	if len(b.events) == 0 {
		b.cond.Wait()
	}

	callback(&b.events)
}

func (b *EventBox) Set(event EventType, value interface{}) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	b.events[event] = value
	if _, found := b.ignore[event]; !found {
		b.cond.Broadcast()
	}
}

// Unsynchronized; should be called within Wait routine
func (events *Events) Clear() {
	for event := range *events {
		delete(*events, event)
	}
}

func (b *EventBox) Peak(event EventType) bool {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	_, ok := b.events[event]
	return ok
}

func (b *EventBox) Watch(events ...EventType) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	for _, event := range events {
		delete(b.ignore, event)
	}
}

func (b *EventBox) Unwatch(events ...EventType) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	for _, event := range events {
		b.ignore[event] = true
	}
}
