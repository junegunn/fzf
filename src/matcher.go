package fzf

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

type MatchRequest struct {
	chunks  []*Chunk
	pattern *Pattern
}

type Matcher struct {
	patternBuilder func([]rune) *Pattern
	sort           bool
	eventBox       *EventBox
	reqBox         *EventBox
	partitions     int
	mergerCache    map[string]*Merger
}

const (
	REQ_RETRY EventType = iota
	REQ_RESET
)

const (
	STAT_CANCELLED int = iota
	STAT_QCH
	STAT_CHUNKS
)

const (
	PROGRESS_MIN_DURATION = 200 * time.Millisecond
)

func NewMatcher(patternBuilder func([]rune) *Pattern,
	sort bool, eventBox *EventBox) *Matcher {
	return &Matcher{
		patternBuilder: patternBuilder,
		sort:           sort,
		eventBox:       eventBox,
		reqBox:         NewEventBox(),
		partitions:     runtime.NumCPU(),
		mergerCache:    make(map[string]*Merger)}
}

func (m *Matcher) Loop() {
	prevCount := 0

	for {
		var request MatchRequest

		m.reqBox.Wait(func(events *Events) {
			for _, val := range *events {
				switch val := val.(type) {
				case MatchRequest:
					request = val
				default:
					panic(fmt.Sprintf("Unexpected type: %T", val))
				}
			}
			events.Clear()
		})

		// Restart search
		patternString := request.pattern.AsString()
		var merger *Merger
		cancelled := false
		count := CountItems(request.chunks)

		foundCache := false
		if count == prevCount {
			// Look up mergerCache
			if cached, found := m.mergerCache[patternString]; found {
				foundCache = true
				merger = cached
			}
		} else {
			// Invalidate mergerCache
			prevCount = count
			m.mergerCache = make(map[string]*Merger)
		}

		if !foundCache {
			merger, cancelled = m.scan(request, 0)
		}

		if !cancelled {
			m.mergerCache[patternString] = merger
			m.eventBox.Set(EVT_SEARCH_FIN, merger)
		}
	}
}

func (m *Matcher) sliceChunks(chunks []*Chunk) [][]*Chunk {
	perSlice := len(chunks) / m.partitions

	// No need to parallelize
	if perSlice == 0 {
		return [][]*Chunk{chunks}
	}

	slices := make([][]*Chunk, m.partitions)
	for i := 0; i < m.partitions; i++ {
		start := i * perSlice
		end := start + perSlice
		if i == m.partitions-1 {
			end = len(chunks)
		}
		slices[i] = chunks[start:end]
	}
	return slices
}

type partialResult struct {
	index   int
	matches []*Item
}

func (m *Matcher) scan(request MatchRequest, limit int) (*Merger, bool) {
	startedAt := time.Now()

	numChunks := len(request.chunks)
	if numChunks == 0 {
		return EmptyMerger, false
	}
	pattern := request.pattern
	empty := pattern.IsEmpty()
	cancelled := NewAtomicBool(false)

	slices := m.sliceChunks(request.chunks)
	numSlices := len(slices)
	resultChan := make(chan partialResult, numSlices)
	countChan := make(chan int, numChunks)
	waitGroup := sync.WaitGroup{}

	for idx, chunks := range slices {
		waitGroup.Add(1)
		go func(idx int, chunks []*Chunk) {
			defer func() { waitGroup.Done() }()
			sliceMatches := []*Item{}
			for _, chunk := range chunks {
				var matches []*Item
				if empty {
					matches = *chunk
				} else {
					matches = request.pattern.Match(chunk)
				}
				sliceMatches = append(sliceMatches, matches...)
				if cancelled.Get() {
					return
				}
				countChan <- len(matches)
			}
			if !empty && m.sort {
				sort.Sort(ByRelevance(sliceMatches))
			}
			resultChan <- partialResult{idx, sliceMatches}
		}(idx, chunks)
	}

	wait := func() bool {
		cancelled.Set(true)
		waitGroup.Wait()
		return true
	}

	count := 0
	matchCount := 0
	for matchesInChunk := range countChan {
		count += 1
		matchCount += matchesInChunk

		if limit > 0 && matchCount > limit {
			return nil, wait() // For --select-1 and --exit-0
		}

		if count == numChunks {
			break
		}

		if !empty && m.reqBox.Peak(REQ_RESET) {
			return nil, wait()
		}

		if time.Now().Sub(startedAt) > PROGRESS_MIN_DURATION {
			m.eventBox.Set(EVT_SEARCH_PROGRESS, float32(count)/float32(numChunks))
		}
	}

	partialResults := make([][]*Item, numSlices)
	for range slices {
		partialResult := <-resultChan
		partialResults[partialResult.index] = partialResult.matches
	}
	return NewMerger(partialResults, !empty && m.sort), false
}

func (m *Matcher) Reset(chunks []*Chunk, patternRunes []rune, cancel bool) {
	pattern := m.patternBuilder(patternRunes)

	var event EventType
	if cancel {
		event = REQ_RESET
	} else {
		event = REQ_RETRY
	}
	m.reqBox.Set(event, MatchRequest{chunks, pattern})
}
