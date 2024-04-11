// Package fzf implements fzf, a command-line fuzzy finder.
package fzf

import (
	"fmt"
	"os"
	"time"
	"unsafe"

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

func ustring(data []byte) string {
	return unsafe.String(unsafe.SliceData(data), len(data))
}

func sbytes(data string) []byte {
	return unsafe.Slice(unsafe.StringData(data), len(data))
}

// Run starts fzf
func Run(opts *Options, version string, revision string) {
	sort := opts.Sort > 0
	sortCriteria = opts.Criteria

	if opts.Version {
		if len(revision) > 0 {
			fmt.Printf("%s (%s)\n", version, revision)
		} else {
			fmt.Println(version)
		}
		os.Exit(exitOk)
	}

	// Event channel
	eventBox := util.NewEventBox()

	// ANSI code processor
	ansiProcessor := func(data []byte) (util.Chars, *[]ansiOffset) {
		return util.ToChars(data), nil
	}

	var lineAnsiState, prevLineAnsiState *ansiState
	if opts.Ansi {
		if opts.Theme.Colored {
			ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
				prevLineAnsiState = lineAnsiState
				trimmed, offsets, newState := extractColor(ustring(data), lineAnsiState, nil)
				lineAnsiState = newState
				return util.ToChars(sbytes(trimmed)), offsets
			}
		} else {
			// When color is disabled but ansi option is given,
			// we simply strip out ANSI codes from the input
			ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
				trimmed, _, _ := extractColor(ustring(data), nil, nil)
				return util.ToChars(sbytes(trimmed)), nil
			}
		}
	}

	// Chunk list
	var chunkList *ChunkList
	var itemIndex int32
	header := make([]string, 0, opts.HeaderLines)
	if len(opts.WithNth) == 0 {
		chunkList = NewChunkList(func(item *Item, data []byte) bool {
			if len(header) < opts.HeaderLines {
				header = append(header, ustring(data))
				eventBox.Set(EvtHeader, header)
				return false
			}
			item.text, item.colors = ansiProcessor(data)
			item.text.Index = itemIndex
			itemIndex++
			return true
		})
	} else {
		chunkList = NewChunkList(func(item *Item, data []byte) bool {
			tokens := Tokenize(ustring(data), opts.Delimiter)
			if opts.Ansi && opts.Theme.Colored && len(tokens) > 1 {
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
			trans := Transform(tokens, opts.WithNth)
			transformed := joinTokens(trans)
			if len(header) < opts.HeaderLines {
				header = append(header, transformed)
				eventBox.Set(EvtHeader, header)
				return false
			}
			item.text, item.colors = ansiProcessor(sbytes(transformed))
			item.text.TrimTrailingWhitespaces()
			item.text.Index = itemIndex
			item.origText = &data
			itemIndex++
			return true
		})
	}

	// Reader
	streamingFilter := opts.Filter != nil && !sort && !opts.Tac && !opts.Sync
	var reader *Reader
	if !streamingFilter {
		reader = NewReader(func(data []byte) bool {
			return chunkList.Push(data)
		}, eventBox, opts.ReadZero, opts.Filter == nil)
		go reader.ReadSource(opts.WalkerRoot, opts.WalkerOpts, opts.WalkerSkip)
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
		}
	}
	patternBuilder := func(runes []rune) *Pattern {
		return BuildPattern(
			opts.Fuzzy, opts.FuzzyAlgo, opts.Extended, opts.Case, opts.Normalize, forward, withPos,
			opts.Filter == nil, opts.Nth, opts.Delimiter, runes)
	}
	inputRevision := 0
	snapshotRevision := 0
	matcher := NewMatcher(patternBuilder, sort, opts.Tac, eventBox, inputRevision)

	// Filtering mode
	if opts.Filter != nil {
		if opts.PrintQuery {
			opts.Printer(*opts.Filter)
		}

		pattern := patternBuilder([]rune(*opts.Filter))
		matcher.sort = pattern.sortable

		found := false
		if streamingFilter {
			slab := util.MakeSlab(slab16Size, slab32Size)
			reader := NewReader(
				func(runes []byte) bool {
					item := Item{}
					if chunkList.trans(&item, runes) {
						if result, _, _ := pattern.MatchItem(&item, false, slab); result != nil {
							opts.Printer(item.text.ToString())
							found = true
						}
					}
					return false
				}, eventBox, opts.ReadZero, false)
			reader.ReadSource(opts.WalkerRoot, opts.WalkerOpts, opts.WalkerSkip)
		} else {
			eventBox.Unwatch(EvtReadNew)
			eventBox.WaitFor(EvtReadFin)

			snapshot, _ := chunkList.Snapshot()
			merger, _ := matcher.scan(MatchRequest{
				chunks:  snapshot,
				pattern: pattern})
			for i := 0; i < merger.Length(); i++ {
				opts.Printer(merger.Get(i).item.AsString(opts.Ansi))
				found = true
			}
		}
		if found {
			os.Exit(exitOk)
		}
		os.Exit(exitNoMatch)
	}

	// Synchronous search
	if opts.Sync {
		eventBox.Unwatch(EvtReadNew)
		eventBox.WaitFor(EvtReadFin)
	}

	// Go interactive
	go matcher.Loop()

	// Terminal I/O
	terminal := NewTerminal(opts, eventBox)
	maxFit := 0 // Maximum number of items that can fit on screen
	padHeight := 0
	heightUnknown := opts.Height.auto
	if heightUnknown {
		maxFit, padHeight = terminal.MaxFitAndPad()
	}
	deferred := opts.Select1 || opts.Exit0
	go terminal.Loop()
	if !deferred && !heightUnknown {
		// Start right away
		terminal.startChan <- fitpad{-1, -1}
	}

	// Event coordination
	reading := true
	ticks := 0
	var nextCommand *string
	var nextEnviron []string
	eventBox.Watch(EvtReadNew)
	total := 0
	query := []rune{}
	determine := func(final bool) {
		if heightUnknown {
			if total >= maxFit || final {
				deferred = false
				heightUnknown = false
				terminal.startChan <- fitpad{util.Min(total, maxFit), padHeight}
			}
		} else if deferred {
			deferred = false
			terminal.startChan <- fitpad{-1, -1}
		}
	}

	useSnapshot := false
	var snapshot []*Chunk
	var count int
	restart := func(command string, environ []string) {
		reading = true
		chunkList.Clear()
		itemIndex = 0
		inputRevision++
		header = make([]string, 0, opts.HeaderLines)
		go reader.restart(command, environ)
	}
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
					os.Exit(value.(int))
				case EvtReadNew, EvtReadFin:
					if evt == EvtReadFin && nextCommand != nil {
						restart(*nextCommand, nextEnviron)
						nextCommand = nil
						nextEnviron = nil
						break
					} else {
						reading = reading && evt == EvtReadNew
					}
					if useSnapshot && evt == EvtReadFin {
						useSnapshot = false
					}
					if !useSnapshot {
						if snapshotRevision != inputRevision {
							query = []rune{}
						}
						snapshot, count = chunkList.Snapshot()
						snapshotRevision = inputRevision
					}
					total = count
					terminal.UpdateCount(total, !reading, value.(*string))
					if opts.Sync {
						opts.Sync = false
						terminal.UpdateList(PassMerger(&snapshot, opts.Tac, snapshotRevision))
					}
					if heightUnknown && !deferred {
						determine(!reading)
					}
					matcher.Reset(snapshot, input(), false, !reading, sort, snapshotRevision)

				case EvtSearchNew:
					var command *string
					var environ []string
					var changed bool
					switch val := value.(type) {
					case searchRequest:
						sort = val.sort
						command = val.command
						environ = val.environ
						changed = val.changed
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
						newSnapshot, newCount := chunkList.Snapshot()
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
					case *Merger:
						if deferred {
							count := val.Length()
							if opts.Select1 && count > 1 || opts.Exit0 && !opts.Select1 && count > 0 {
								determine(val.final)
							} else if val.final {
								if opts.Exit0 && count == 0 || opts.Select1 && count == 1 {
									if opts.PrintQuery {
										opts.Printer(opts.Query)
									}
									if len(opts.Expect) > 0 {
										opts.Printer("")
									}
									for i := 0; i < count; i++ {
										opts.Printer(val.Get(i).item.AsString(opts.Ansi))
									}
									if count > 0 {
										os.Exit(exitOk)
									}
									os.Exit(exitNoMatch)
								}
								determine(val.final)
							}
						}
						terminal.UpdateList(val)
					}
				}
			}
			events.Clear()
		})
		if delay && reading {
			dur := util.DurWithin(
				time.Duration(ticks)*coordinatorDelayStep,
				0, coordinatorDelayMax)
			time.Sleep(dur)
		}
	}
}
