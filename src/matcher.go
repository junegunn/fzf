package fzf

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/junegunn/fzf/src/util"
)

// MatchRequest represents a search request
type MatchRequest struct {
	chunks   []*Chunk
	pattern  *Pattern
	final    bool
	sort     bool
	revision revision
}

type MatchResult struct {
	merger     *Merger
	passMerger *Merger
	cancelled  bool
}

func (mr MatchResult) cacheable() bool {
	return mr.merger != nil && mr.merger.cacheable()
}

func (mr MatchResult) final() bool {
	return mr.merger != nil && mr.merger.final
}

// Matcher is responsible for performing search
type Matcher struct {
	cache          *ChunkCache
	patternBuilder func([]rune) *Pattern
	sort           bool
	tac            bool
	eventBox       *util.EventBox
	reqBox         *util.EventBox
	partitions     int
	slab           []*util.Slab
	sortBuf        [][]Result
	mergerCache    map[string]MatchResult
	revision       revision
	scanMutex      sync.Mutex
	cancelScan     *util.AtomicBool
}

const (
	reqRetry util.EventType = iota
	reqReset
)

// NewMatcher returns a new Matcher
func NewMatcher(cache *ChunkCache, patternBuilder func([]rune) *Pattern,
	sort bool, tac bool, eventBox *util.EventBox, revision revision, threads int) *Matcher {
	partitions := runtime.NumCPU()
	if threads > 0 {
		partitions = threads
	}
	return &Matcher{
		cache:          cache,
		patternBuilder: patternBuilder,
		sort:           sort,
		tac:            tac,
		eventBox:       eventBox,
		reqBox:         util.NewEventBox(),
		partitions:     partitions,
		slab:           make([]*util.Slab, partitions),
		sortBuf:        make([][]Result, partitions),
		mergerCache:    make(map[string]MatchResult),
		revision:       revision,
		cancelScan:     util.NewAtomicBool(false)}
}

// Loop puts Matcher in action
func (m *Matcher) Loop() {
	prevCount := 0

	for {
		var request MatchRequest

		stop := false
		m.reqBox.Wait(func(events *util.Events) {
			for t, val := range *events {
				if t == reqQuit {
					stop = true
					return
				}
				switch val := val.(type) {
				case MatchRequest:
					request = val
				default:
					panic(fmt.Sprintf("Unexpected type: %T", val))
				}
			}
			events.Clear()
		})
		if stop {
			break
		}

		cacheCleared := false
		if request.sort != m.sort || request.revision != m.revision {
			m.sort = request.sort
			m.mergerCache = make(map[string]MatchResult)
			if !request.revision.compatible(m.revision) {
				m.cache.Clear()
			}
			m.revision = request.revision
			cacheCleared = true
		}

		// Restart search
		patternString := request.pattern.AsString()
		var result MatchResult
		count := CountItems(request.chunks)

		if !cacheCleared {
			if count == prevCount {
				// Look up mergerCache
				if cached, found := m.mergerCache[patternString]; found && cached.final() == request.final {
					result = cached
				}
			} else {
				// Invalidate mergerCache
				prevCount = count
				m.mergerCache = make(map[string]MatchResult)
			}
		}

		if result.merger == nil {
			m.scanMutex.Lock()
			result = m.scan(request)
			m.scanMutex.Unlock()
		}

		if !result.cancelled {
			if result.cacheable() {
				m.mergerCache[patternString] = result
			}
			result.merger.final = request.final
			m.eventBox.Set(EvtSearchFin, result)
		}
	}
}

type partialResult struct {
	index   int
	matches []Result
}

func (m *Matcher) scan(request MatchRequest) MatchResult {
	startedAt := time.Now()

	numChunks := len(request.chunks)
	if numChunks == 0 {
		m := EmptyMerger(request.revision)
		return MatchResult{m, m, false}
	}
	pattern := request.pattern
	passMerger := PassMerger(&request.chunks, m.tac, request.revision, pattern.startIndex)
	if pattern.IsEmpty() {
		return MatchResult{passMerger, passMerger, false}
	}

	minIndex := request.chunks[0].items[0].Index()
	maxIndex := request.chunks[numChunks-1].lastIndex(minIndex)
	cancelled := util.NewAtomicBool(false)

	numWorkers := min(m.partitions, numChunks)
	var nextChunk atomic.Int32
	resultChan := make(chan partialResult, numWorkers)
	countChan := make(chan int, numChunks)
	waitGroup := sync.WaitGroup{}

	for idx := range numWorkers {
		waitGroup.Add(1)
		if m.slab[idx] == nil {
			m.slab[idx] = util.MakeSlab(slab16Size, slab32Size)
		}
		go func(idx int, slab *util.Slab) {
			defer waitGroup.Done()
			var matches []Result
			for {
				ci := int(nextChunk.Add(1)) - 1
				if ci >= numChunks {
					break
				}
				chunkMatches := request.pattern.Match(request.chunks[ci], slab)
				matches = append(matches, chunkMatches...)
				if cancelled.Get() {
					return
				}
				countChan <- len(chunkMatches)
			}
			if m.sort && request.pattern.sortable {
				m.sortBuf[idx] = radixSortResults(matches, m.tac, m.sortBuf[idx])
			}
			resultChan <- partialResult{idx, matches}
		}(idx, m.slab[idx])
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

		if m.cancelScan.Get() || m.reqBox.Peek(reqReset) {
			return MatchResult{nil, nil, wait()}
		}

		if time.Since(startedAt) > progressMinDuration {
			m.eventBox.Set(EvtSearchProgress, float32(count)/float32(numChunks))
		}
	}

	partialResults := make([][]Result, numWorkers)
	for range numWorkers {
		partialResult := <-resultChan
		partialResults[partialResult.index] = partialResult.matches
	}
	merger := NewMerger(pattern, partialResults, m.sort && request.pattern.sortable, m.tac, request.revision, minIndex, maxIndex)
	return MatchResult{merger, passMerger, false}
}

// Reset is called to interrupt/signal the ongoing search
func (m *Matcher) Reset(chunks []*Chunk, patternRunes []rune, cancel bool, final bool, sort bool, revision revision) {
	pattern := m.patternBuilder(patternRunes)

	var event util.EventType
	if cancel {
		event = reqReset
	} else {
		event = reqRetry
	}
	m.reqBox.Set(event, MatchRequest{chunks, pattern, final, sort, revision})
}

// CancelScan cancels any in-flight scan, waits for it to finish,
// and prevents new scans from starting until ResumeScan is called.
// This is used to safely mutate shared items (e.g., during with-nth changes).
func (m *Matcher) CancelScan() {
	m.cancelScan.Set(true)
	m.scanMutex.Lock()
	m.cancelScan.Set(false)
}

// ResumeScan allows scans to proceed again after CancelScan.
func (m *Matcher) ResumeScan() {
	m.scanMutex.Unlock()
}

func (m *Matcher) Stop() {
	m.reqBox.Set(reqQuit, nil)
}
