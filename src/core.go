// Package fzf implements fzf, a command-line fuzzy finder.
package fzf

import (
	"maps"
	"os"
	"sync"
	"time"

	"github.com/junegunn/fzf/src/tui"
	"github.com/junegunn/fzf/src/util"
)

/*
Reader   -> EvtReadFin
Reader   -> EvtReadNew        -> Matcher  (restart)
Terminal -> EvtSearchNew:bool -> Matcher  (restart)
Matcher  -> EvtSearchProgress -> Terminal (update info)
Matcher  -> EvtSearchFin      -> Terminal (update list)
Matcher  -> EvtHeader         -> Terminal (update header)
*/

type revision struct {
	major int
	minor int
}

func (r *revision) bumpMajor() {
	r.major++
	r.minor = 0
}

func (r *revision) bumpMinor() {
	r.minor++
}

func (r revision) compatible(other revision) bool {
	return r.major == other.major
}

func buildItemTransformer(opts *Options) func(*Item) string {
	if opts.AcceptNth != nil {
		fn := opts.AcceptNth(opts.Delimiter)
		return func(item *Item) string {
			return item.acceptNth(opts.Ansi, opts.Delimiter, fn)
		}
	}
	return func(item *Item) string {
		return item.AsString(opts.Ansi)
	}
}

// Run starts fzf
func Run(opts *Options) (int, error) {
	if opts.Filter == nil {
		if opts.useTmux() {
			return runTmux(os.Args, opts)
		}

		if needWinpty(opts) {
			return runWinpty(os.Args, opts)
		}
	}

	if err := postProcessOptions(opts); err != nil {
		return ExitError, err
	}

	defer util.RunAtExitFuncs()

	// Output channel given
	if opts.Output != nil {
		opts.Printer = func(str string) {
			opts.Output <- str
		}
	}

	sort := opts.Sort > 0
	sortCriteria = opts.Criteria

	// Event channel
	eventBox := util.NewEventBox()

	// ANSI code processor
	ansiProcessor := func(data []byte) (util.Chars, *[]ansiOffset) {
		return util.ToChars(data), nil
	}

	var lineAnsiState, prevLineAnsiState *ansiState
	if opts.Ansi {
		ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
			prevLineAnsiState = lineAnsiState
			trimmed, offsets, newState := extractColor(byteString(data), lineAnsiState, nil)
			lineAnsiState = newState

			// Full line background is found. Add a special marker.
			if offsets != nil && newState != nil && len(*offsets) > 0 && newState.lbg >= 0 {
				marker := (*offsets)[len(*offsets)-1]
				marker.offset[0] = marker.offset[1]
				marker.color.bg = newState.lbg
				marker.color.attr = marker.color.attr | tui.FullBg
				newOffsets := append(*offsets, marker)
				offsets = &newOffsets

				// Reset the full-line background color
				lineAnsiState.lbg = -1
			}
			return util.ToChars(stringBytes(trimmed)), offsets
		}
	}

	// Chunk list
	cache := NewChunkCache()
	var chunkList *ChunkList
	var itemIndex int32
	header := make([]string, 0, opts.HeaderLines)
	if opts.WithNth == nil {
		chunkList = NewChunkList(cache, func(item *Item, data []byte) bool {
			if len(header) < opts.HeaderLines {
				header = append(header, byteString(data))
				eventBox.Set(EvtHeader, header)
				return false
			}
			item.text, item.colors = ansiProcessor(data)
			item.text.Index = itemIndex
			itemIndex++
			return true
		})
	} else {
		nthTransformer := opts.WithNth(opts.Delimiter)
		chunkList = NewChunkList(cache, func(item *Item, data []byte) bool {
			tokens := Tokenize(byteString(data), opts.Delimiter)
			if opts.Ansi && len(tokens) > 1 {
				var ansiState *ansiState
				if prevLineAnsiState != nil {
					ansiStateDup := *prevLineAnsiState
					ansiState = &ansiStateDup
				}
				for _, token := range tokens {
					prevAnsiState := ansiState
					_, _, ansiState = extractColor(token.text.ToString(), ansiState, nil)
					if prevAnsiState != nil {
						token.text.Prepend("\x1b[m" + prevAnsiState.ToString())
					} else {
						token.text.Prepend("\x1b[m")
					}
				}
			}
			transformed := nthTransformer(tokens, itemIndex)
			if len(header) < opts.HeaderLines {
				header = append(header, transformed)
				eventBox.Set(EvtHeader, header)
				return false
			}
			item.text, item.colors = ansiProcessor(stringBytes(transformed))

			// We should not trim trailing whitespaces with background colors
			var maxColorOffset int32
			if item.colors != nil {
				for _, ansi := range *item.colors {
					if ansi.color.bg >= 0 {
						maxColorOffset = max(maxColorOffset, ansi.offset[1])
					}
				}
			}
			item.text.TrimTrailingWhitespaces(int(maxColorOffset))
			item.text.Index = itemIndex
			item.origText = &data
			itemIndex++
			return true
		})
	}

	// Process executor
	executor := util.NewExecutor(opts.WithShell)

	// Terminal I/O
	var terminal *Terminal
	var err error
	var initialEnv []string
	initialReload := opts.extractReloadOnStart()
	if opts.Filter == nil {
		terminal, err = NewTerminal(opts, eventBox, executor)
		if err != nil {
			return ExitError, err
		}
		if len(initialReload) > 0 {
			var temps []string
			initialReload, temps = terminal.replacePlaceholderInInitialCommand(initialReload)
			initialEnv = terminal.environ()
			defer removeFiles(temps)
		}
	}

	// Reader
	streamingFilter := opts.Filter != nil && !sort && !opts.Tac && !opts.Sync
	var reader *Reader
	if !streamingFilter {
		reader = NewReader(func(data []byte) bool {
			return chunkList.Push(data)
		}, eventBox, executor, opts.ReadZero, opts.Filter == nil)

		readyChan := make(chan bool)
		go reader.ReadSource(opts.Input, opts.WalkerRoot, opts.WalkerOpts, opts.WalkerSkip, initialReload, initialEnv, readyChan)
		<-readyChan
	}

	// Matcher
	forward := true
	withPos := false
	for idx := len(opts.Criteria) - 1; idx > 0; idx-- {
		switch opts.Criteria[idx] {
		case byChunk:
			withPos = true
		case byEnd:
			forward = false
		case byBegin:
			forward = true
		case byPathname:
			withPos = true
			forward = false
		}
	}

	nth := opts.Nth
	inputRevision := revision{}
	snapshotRevision := revision{}
	patternCache := make(map[string]*Pattern)
	denyMutex := sync.Mutex{}
	denylist := make(map[int32]struct{})
	clearDenylist := func() {
		denyMutex.Lock()
		if len(denylist) > 0 {
			patternCache = make(map[string]*Pattern)
		}
		denylist = make(map[int32]struct{})
		denyMutex.Unlock()
	}
	patternBuilder := func(runes []rune) *Pattern {
		denyMutex.Lock()
		denylistCopy := maps.Clone(denylist)
		denyMutex.Unlock()
		return BuildPattern(cache, patternCache,
			opts.Fuzzy, opts.FuzzyAlgo, opts.Extended, opts.Case, opts.Normalize, forward, withPos,
			opts.Filter == nil, nth, opts.Delimiter, inputRevision, runes, denylistCopy)
	}
	matcher := NewMatcher(cache, patternBuilder, sort, opts.Tac, eventBox, inputRevision)

	// Filtering mode
	if opts.Filter != nil {
		if opts.PrintQuery {
			opts.Printer(*opts.Filter)
		}

		pattern := patternBuilder([]rune(*opts.Filter))
		matcher.sort = pattern.sortable

		transformer := buildItemTransformer(opts)

		found := false
		if streamingFilter {
			slab := util.MakeSlab(slab16Size, slab32Size)
			mutex := sync.Mutex{}
			reader := NewReader(
				func(runes []byte) bool {
					item := Item{}
					if chunkList.trans(&item, runes) {
						mutex.Lock()
						if result, _, _ := pattern.MatchItem(&item, false, slab); result != nil {
							opts.Printer(transformer(&item))
							found = true
						}
						mutex.Unlock()
					}
					return false
				}, eventBox, executor, opts.ReadZero, false)
			reader.ReadSource(opts.Input, opts.WalkerRoot, opts.WalkerOpts, opts.WalkerSkip, initialReload, initialEnv, nil)
		} else {
			eventBox.Unwatch(EvtReadNew)
			eventBox.WaitFor(EvtReadFin)

			// NOTE: Streaming filter is inherently not compatible with --tail
			snapshot, _, _ := chunkList.Snapshot(opts.Tail)
			result := matcher.scan(MatchRequest{
				chunks:  snapshot,
				pattern: pattern})
			for i := 0; i < result.merger.Length(); i++ {
				opts.Printer(transformer(result.merger.Get(i).item))
				found = true
			}
		}
		if found {
			return ExitOk, nil
		}
		return ExitNoMatch, nil
	}

	// Synchronous search
	if opts.Sync {
		eventBox.Unwatch(EvtReadNew)
		eventBox.WaitFor(EvtReadFin)
	}

	// Go interactive
	go matcher.Loop()
	defer matcher.Stop()

	// Handling adaptive height
	maxFit := 0 // Maximum number of items that can fit on screen
	padHeight := 0
	heightUnknown := opts.Height.auto
	if heightUnknown {
		maxFit, padHeight = terminal.MaxFitAndPad()
	}
	deferred := opts.Select1 || opts.Exit0 || opts.Sync
	go terminal.Loop()
	if !deferred && !heightUnknown {
		// Start right away
		terminal.startChan <- fitpad{-1, -1}
	}

	// Event coordination
	reading := true
	ticks := 0
	startTick := 0
	var nextCommand *commandSpec
	var nextEnviron []string
	eventBox.Watch(EvtReadNew)
	total := 0
	query := []rune{}
	determine := func(final bool) {
		if heightUnknown {
			if total >= maxFit || final {
				deferred = false
				heightUnknown = false
				terminal.startChan <- fitpad{min(total, maxFit), padHeight}
			}
		} else if deferred {
			deferred = false
			terminal.startChan <- fitpad{-1, -1}
		}
	}

	useSnapshot := false
	var snapshot []*Chunk
	var count int
	restart := func(command commandSpec, environ []string) {
		if !useSnapshot {
			clearDenylist()
		}
		reading = true
		startTick = ticks
		chunkList.Clear()
		itemIndex = 0
		inputRevision.bumpMajor()
		header = make([]string, 0, opts.HeaderLines)
		readyChan := make(chan bool)
		go reader.restart(command, environ, readyChan)
		<-readyChan
	}

	exitCode := ExitOk
	stop := false
	for {
		delay := true
		ticks++
		input := func() []rune {
			paused, input := terminal.Input()
			if !paused {
				query = input
			}
			return query
		}
		eventBox.Wait(func(events *util.Events) {
			if _, fin := (*events)[EvtReadFin]; fin {
				delete(*events, EvtReadNew)
			}
			for evt, value := range *events {
				switch evt {
				case EvtQuit:
					if reading {
						reader.terminate()
					}
					quitSignal := value.(quitSignal)
					exitCode = quitSignal.code
					err = quitSignal.err
					stop = true
					return
				case EvtReadNew, EvtReadFin:
					if evt == EvtReadFin && nextCommand != nil {
						restart(*nextCommand, nextEnviron)
						nextCommand = nil
						nextEnviron = nil
						break
					} else {
						reading = reading && evt == EvtReadNew
					}
					if useSnapshot && evt == EvtReadFin { // reload-sync
						clearDenylist()
						useSnapshot = false
					}
					if !useSnapshot {
						if !snapshotRevision.compatible(inputRevision) {
							query = []rune{}
						}
						var changed bool
						snapshot, count, changed = chunkList.Snapshot(opts.Tail)
						if changed {
							inputRevision.bumpMinor()
						}
						snapshotRevision = inputRevision
					}
					total = count
					terminal.UpdateCount(total, !reading, value.(*string))
					if heightUnknown && !deferred {
						determine(!reading)
					}
					matcher.Reset(snapshot, input(), false, !reading, sort, snapshotRevision)

				case EvtSearchNew:
					var command *commandSpec
					var environ []string
					var changed bool
					switch val := value.(type) {
					case searchRequest:
						sort = val.sort
						command = val.command
						environ = val.environ
						changed = val.changed
						bump := false
						if len(val.denylist) > 0 && val.revision.compatible(inputRevision) {
							denyMutex.Lock()
							for _, itemIndex := range val.denylist {
								denylist[itemIndex] = struct{}{}
							}
							denyMutex.Unlock()
							bump = true
						}
						if val.nth != nil {
							// Change nth and clear caches
							nth = *val.nth
							bump = true
						}
						if bump {
							patternCache = make(map[string]*Pattern)
							cache.Clear()
							inputRevision.bumpMinor()
						}
						if command != nil {
							useSnapshot = val.sync
						}
					}
					if command != nil {
						if reading {
							reader.terminate()
							nextCommand = command
							nextEnviron = environ
						} else {
							restart(*command, environ)
						}
					}
					if !changed {
						break
					}
					if !useSnapshot {
						newSnapshot, newCount, changed := chunkList.Snapshot(opts.Tail)
						if changed {
							inputRevision.bumpMinor()
						}
						// We want to avoid showing empty list when reload is triggered
						// and the query string is changed at the same time i.e. command != nil && changed
						if command == nil || newCount > 0 {
							if snapshotRevision != inputRevision {
								query = []rune{}
							}
							snapshot = newSnapshot
							snapshotRevision = inputRevision
						}
					}
					matcher.Reset(snapshot, input(), true, !reading, sort, snapshotRevision)
					delay = false

				case EvtSearchProgress:
					switch val := value.(type) {
					case float32:
						terminal.UpdateProgress(val)
					}

				case EvtHeader:
					headerPadded := make([]string, opts.HeaderLines)
					copy(headerPadded, value.([]string))
					terminal.UpdateHeader(headerPadded)

				case EvtSearchFin:
					switch val := value.(type) {
					case MatchResult:
						merger := val.merger
						if deferred {
							count := merger.Length()
							if opts.Select1 && count > 1 || opts.Exit0 && !opts.Select1 && count > 0 {
								determine(merger.final)
							} else if merger.final {
								if opts.Exit0 && count == 0 || opts.Select1 && count == 1 {
									if opts.PrintQuery {
										opts.Printer(opts.Query)
									}
									if len(opts.Expect) > 0 {
										opts.Printer("")
									}
									transformer := buildItemTransformer(opts)
									for i := range count {
										opts.Printer(transformer(merger.Get(i).item))
									}
									if count == 0 {
										exitCode = ExitNoMatch
									}
									stop = true
									return
								}
								determine(merger.final)
							}
						}
						terminal.UpdateList(val)
					}
				}
			}
			events.Clear()
		})
		if stop {
			break
		}
		if delay && reading {
			dur := util.Constrain(
				time.Duration(ticks-startTick)*coordinatorDelayStep,
				0, coordinatorDelayMax)
			time.Sleep(dur)
		}
	}
	return exitCode, err
}
