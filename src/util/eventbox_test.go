package util

import "testing"

// fzf events
const (
	EvtReadNew EventType = iota
	EvtReadFin
	EvtSearchNew
	EvtSearchProgress
	EvtSearchFin
	EvtClose
)

func TestEventBox(t *testing.T) {
	eb := NewEventBox()

	// Wait should return immediately
	ch := make(chan bool)

	go func() {
		eb.Set(EvtReadNew, 10)
		ch <- true
		<-ch
		eb.Set(EvtSearchNew, 10)
		eb.Set(EvtSearchNew, 15)
		eb.Set(EvtSearchNew, 20)
		eb.Set(EvtSearchProgress, 30)
		ch <- true
		<-ch
		eb.Set(EvtSearchFin, 40)
		ch <- true
		<-ch
	}()

	count := 0
	sum := 0
	looping := true
	for looping {
		<-ch
		eb.Wait(func(events *Events) {
			for _, value := range *events {
				switch val := value.(type) {
				case int:
					sum += val
					looping = sum < 100
				}
			}
			events.Clear()
		})
		ch <- true
		count++
	}

	if count != 3 {
		t.Error("Invalid number of events", count)
	}
	if sum != 100 {
		t.Error("Invalid sum", sum)
	}
}
