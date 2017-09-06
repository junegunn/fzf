/*
Package fzf implements fzf, a command-line fuzzy finder.

The MIT License (MIT)

Copyright (c) 2017 Junegunn Choi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package fzf

import (
	"fmt"
	"os"
	"time"

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

// Run starts fzf
func Run(opts *Options, revision string) {
	postProcessOptions(opts)

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
	if opts.Ansi {
		if opts.Theme != nil {
			var state *ansiState
			ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
				trimmed, offsets, newState := extractColor(string(data), state, nil)
				state = newState
				return util.ToChars([]byte(trimmed)), offsets
			}
		} else {
			// When color is disabled but ansi option is given,
			// we simply strip out ANSI codes from the input
			ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
				trimmed, _, _ := extractColor(string(data), nil, nil)
				return util.ToChars([]byte(trimmed)), nil
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
				header = append(header, string(data))
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
			tokens := Tokenize(string(data), opts.Delimiter)
			trans := Transform(tokens, opts.WithNth)
			transformed := joinTokens(trans)
			if len(header) < opts.HeaderLines {
				header = append(header, transformed)
				eventBox.Set(EvtHeader, header)
				return false
			}
			item.text, item.colors = ansiProcessor([]byte(transformed))
			item.text.Index = itemIndex
			item.origText = &data
			itemIndex++
			return true
		})
	}

	// Reader
	streamingFilter := opts.Filter != nil && !sort && !opts.Tac && !opts.Sync
	if !streamingFilter {
		reader := opts.ReaderFactory(func(data []byte) bool {
			return chunkList.Push(data)
		}, eventBox, opts.ReadZero)
		go reader.ReadSource()
	}

	// Matcher
	forward := true
	for _, cri := range opts.Criteria[1:] {
		if cri == CriterionByEnd {
			forward = false
			break
		}
		if cri == CriterionByBegin {
			break
		}
	}
	patternBuilder := func(runes []rune) *Pattern {
		return BuildPattern(
			opts.Fuzzy, opts.FuzzyAlgo, opts.Extended, opts.Case, opts.Normalize, forward,
			opts.Filter == nil, opts.Nth, opts.Delimiter, runes)
	}
	matcher := NewMatcher(patternBuilder, sort, opts.Tac, eventBox)

	// Filtering mode
	if opts.Filter != nil {
		if opts.PrintQuery {
			opts.Printer(*opts.Filter)
		}

		pattern := patternBuilder([]rune(*opts.Filter))

		found := false
		if streamingFilter {
			slab := util.MakeSlab(slab16Size, slab32Size)
			reader := opts.ReaderFactory(
				func(runes []byte) bool {
					item := Item{}
					if chunkList.trans(&item, runes) {
						if result, _, _ := pattern.MatchItem(&item, false, slab); result != nil {
							opts.Printer(item.text.ToString())
							found = true
						}
					}
					return false
				}, eventBox, opts.ReadZero)
			reader.ReadSource()
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
	deferred := opts.Select1 || opts.Exit0
	go terminal.Loop()
	if !deferred {
		terminal.startChan <- true
	}

	// Event coordination
	reading := true
	ticks := 0
	eventBox.Watch(EvtReadNew)
	for {
		delay := true
		ticks++
		eventBox.Wait(func(events *util.Events) {
			if _, fin := (*events)[EvtReadFin]; fin {
				delete(*events, EvtReadNew)
			}
			for evt, value := range *events {
				switch evt {

				case EvtReadNew, EvtReadFin:
					reading = reading && evt == EvtReadNew
					snapshot, count := chunkList.Snapshot()
					terminal.UpdateCount(count, !reading, value.(bool))
					if opts.Sync {
						terminal.UpdateList(PassMerger(&snapshot, opts.Tac))
					}
					matcher.Reset(snapshot, terminal.Input(), false, !reading, sort)

				case EvtSearchNew:
					switch val := value.(type) {
					case bool:
						sort = val
					}
					snapshot, _ := chunkList.Snapshot()
					matcher.Reset(snapshot, terminal.Input(), true, !reading, sort)
					delay = false

				case EvtSearchProgress:
					switch val := value.(type) {
					case float32:
						terminal.UpdateProgress(val)
					}

				case EvtHeader:
					terminal.UpdateHeader(value.([]string))

				case EvtSearchFin:
					switch val := value.(type) {
					case *Merger:
						if deferred {
							count := val.Length()
							if opts.Select1 && count > 1 || opts.Exit0 && !opts.Select1 && count > 0 {
								deferred = false
								terminal.startChan <- true
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
								deferred = false
								terminal.startChan <- true
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
