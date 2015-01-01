package fzf

import "sync"

type EventType int

type Events map[EventType]interface{}

type EventBox struct {
	events Events
	cond   *sync.Cond
}

func NewEventBox() *EventBox {
	return &EventBox{make(Events), sync.NewCond(&sync.Mutex{})}
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
	b.cond.Broadcast()
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
