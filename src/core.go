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
		if opts.Theme != nil {
			ansiProcessor = func(data []byte) (util.Chars, *[]ansiOffset) {
				prevLineAnsiState = lineAnsiState
				trimmed, offsets, newState := extractColor(string(data), lineAnsiState, nil)
				lineAnsiState = newState
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
			if opts.Ansi && opts.Theme != nil && len(tokens) > 1 {
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
			item.text, item.colors = ansiProcessor([]byte(transformed))
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
		go reader.ReadSource()
	}

	// Matcher
	forward := true
	for _, cri := range opts.Criteria[1:] {
		if cri == byEnd {
			forward = false
			break
		}
		if cri == byBegin {
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
	clearCache := util.Once(false)
	clearSelection := util.Once(false)
	ticks := 0
	var nextCommand *string
	restart := func(command string) {
		reading = true
		clearCache = util.Once(true)
		clearSelection = util.Once(true)
		chunkList.Clear()
		header = make([]string, 0, opts.HeaderLines)
		go reader.restart(command)
	}
	eventBox.Watch(EvtReadNew)
	for {
		delay := true
		ticks++
		input := func() []rune {
			if opts.Phony {
				return []rune{}
			}
			return []rune(terminal.Input())
		}
		eventBox.Wait(func(events *util.Events) {
			if _, fin := (*events)[EvtReadFin]; fin {
				delete(*events, EvtReadNew)
			}
			for evt, value := range *events {
				switch evt {

				case EvtReadNew, EvtReadFin:
					if evt == EvtReadFin && nextCommand != nil {
						restart(*nextCommand)
						nextCommand = nil
						break
					} else {
						reading = reading && evt == EvtReadNew
					}
					snapshot, count := chunkList.Snapshot()
					terminal.UpdateCount(count, !reading, value.(*string))
					if opts.Sync {
						opts.Sync = false
						terminal.UpdateList(PassMerger(&snapshot, opts.Tac), false)
					}
					matcher.Reset(snapshot, input(), false, !reading, sort, clearCache())

				case EvtSearchNew:
					var command *string
					switch val := value.(type) {
					case searchRequest:
						sort = val.sort
						command = val.command
					}
					if command != nil {
						if reading {
							reader.terminate()
							nextCommand = command
						} else {
							restart(*command)
						}
						break
					}
					snapshot, _ := chunkList.Snapshot()
					matcher.Reset(snapshot, input(), true, !reading, sort, clearCache())
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
						terminal.UpdateList(val, clearSelection())
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
