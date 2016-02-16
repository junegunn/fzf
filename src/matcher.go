package fzf

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/junegunn/fzf/src/util"
)

// MatchRequest represents a search request
type MatchRequest struct {
	chunks  []*Chunk
	pattern *Pattern
	final   bool
	sort    bool
}

// Matcher is responsible for performing search
type Matcher struct {
	patternBuilder func([]rune) *Pattern
	sort           bool
	tac            bool
	eventBox       *util.EventBox
	reqBox         *util.EventBox
	partitions     int
	mergerCache    map[string]*Merger
}

const (
	reqRetry util.EventType = iota
	reqReset
)

// NewMatcher returns a new Matcher
func NewMatcher(patternBuilder func([]rune) *Pattern,
	sort bool, tac bool, eventBox *util.EventBox) *Matcher {
	return &Matcher{
		patternBuilder: patternBuilder,
		sort:           sort,
		tac:            tac,
		eventBox:       eventBox,
		reqBox:         util.NewEventBox(),
		partitions:     runtime.NumCPU(),
		mergerCache:    make(map[string]*Merger)}
}

// Loop puts Matcher in action
func (m *Matcher) Loop() {
	prevCount := 0

	for {
		var request MatchRequest

		m.reqBox.Wait(func(events *util.Events) {
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

		if request.sort != m.sort {
			m.sort = request.sort
			m.mergerCache = make(map[string]*Merger)
			clearChunkCache()
		}

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
			merger, cancelled = m.scan(request)
		}

		if !cancelled {
			if merger.cacheable() {
				m.mergerCache[patternString] = merger
			}
			merger.final = request.final
			m.eventBox.Set(EvtSearchFin, merger)
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

func (m *Matcher) scan(request MatchRequest) (*Merger, bool) {
	startedAt := time.Now()

	numChunks := len(request.chunks)
	if numChunks == 0 {
		return EmptyMerger, false
	}
	pattern := request.pattern
	if pattern.IsEmpty() {
		return PassMerger(&request.chunks, m.tac), false
	}

	cancelled := util.NewAtomicBool(false)

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
				matches := request.pattern.Match(chunk)
				sliceMatches = append(sliceMatches, matches...)
				if cancelled.Get() {
					return
				}
				countChan <- len(matches)
			}
			if m.sort {
				if m.tac {
					sort.Sort(ByRelevanceTac(sliceMatches))
				} else {
					sort.Sort(ByRelevance(sliceMatches))
				}
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
		count++
		matchCount += matchesInChunk

		if count == numChunks {
			break
		}

		if m.reqBox.Peek(reqReset) {
			return nil, wait()
		}

		if time.Now().Sub(startedAt) > progressMinDuration {
			m.eventBox.Set(EvtSearchProgress, float32(count)/float32(numChunks))
		}
	}

	partialResults := make([][]*Item, numSlices)
	for _, _ = range slices {
		partialResult := <-resultChan
		partialResults[partialResult.index] = partialResult.matches
	}
	return NewMerger(partialResults, m.sort, m.tac), false
}

// Reset is called to interrupt/signal the ongoing search
func (m *Matcher) Reset(chunks []*Chunk, patternRunes []rune, cancel bool, final bool, sort bool) {
	pattern := m.patternBuilder(patternRunes)

	var event util.EventType
	if cancel {
		event = reqReset
	} else {
		event = reqRetry
	}
	m.reqBox.Set(event, MatchRequest{chunks, pattern, final, sort})
}
