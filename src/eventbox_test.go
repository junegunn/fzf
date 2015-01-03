package fzf

import "testing"

func TestEventBox(t *testing.T) {
	eb := NewEventBox()

	// Wait should return immediately
	ch := make(chan bool)

	go func() {
		eb.Set(EVT_READ_NEW, 10)
		ch <- true
		<-ch
		eb.Set(EVT_SEARCH_NEW, 10)
		eb.Set(EVT_SEARCH_NEW, 15)
		eb.Set(EVT_SEARCH_NEW, 20)
		eb.Set(EVT_SEARCH_PROGRESS, 30)
		ch <- true
		<-ch
		eb.Set(EVT_SEARCH_FIN, 40)
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
		count += 1
	}

	if count != 3 {
		t.Error("Invalid number of events", count)
	}
	if sum != 100 {
		t.Error("Invalid sum", sum)
	}
}
