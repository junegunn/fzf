/*
Package fzf implements fzf, a command-line fuzzy finder.

The MIT License (MIT)

Copyright (c) 2015 Junegunn Choi

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
	"runtime"
	"time"

	"github.com/junegunn/fzf/src/util"
)

func initProcs() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

/*
Reader   -> EvtReadFin
Reader   -> EvtReadNew        -> Matcher  (restart)
Terminal -> EvtSearchNew:bool -> Matcher  (restart)
Matcher  -> EvtSearchProgress -> Terminal (update info)
Matcher  -> EvtSearchFin      -> Terminal (update list)
Matcher  -> EvtHeader         -> Terminal (update header)
*/

// Run starts fzf
func Run(opts *Options) {
	initProcs()

	sort := opts.Sort > 0
	rankTiebreak = opts.Tiebreak

	if opts.Version {
		fmt.Println(Version)
		os.Exit(0)
	}

	// Event channel
	eventBox := util.NewEventBox()

	// ANSI code processor
	ansiProcessor := func(data *string) (*string, []ansiOffset) {
		// By default, we do nothing
		return data, nil
	}
	if opts.Ansi {
		if opts.Theme != nil {
			var state *ansiState
			ansiProcessor = func(data *string) (*string, []ansiOffset) {
				trimmed, offsets, newState := extractColor(data, state)
				state = newState
				return trimmed, offsets
			}
		} else {
			// When color is disabled but ansi option is given,
			// we simply strip out ANSI codes from the input
			ansiProcessor = func(data *string) (*string, []ansiOffset) {
				trimmed, _, _ := extractColor(data, nil)
				return trimmed, nil
			}
		}
	}

	// Chunk list
	var chunkList *ChunkList
	header := make([]string, 0, opts.HeaderLines)
	if len(opts.WithNth) == 0 {
		chunkList = NewChunkList(func(data *string, index int) *Item {
			if len(header) < opts.HeaderLines {
				header = append(header, *data)
				eventBox.Set(EvtHeader, header)
				return nil
			}
			data, colors := ansiProcessor(data)
			return &Item{
				text:   data,
				index:  uint32(index),
				colors: colors,
				rank:   Rank{0, 0, uint32(index)}}
		})
	} else {
		chunkList = NewChunkList(func(data *string, index int) *Item {
			tokens := Tokenize(data, opts.Delimiter)
			trans := Transform(tokens, opts.WithNth)
			if len(header) < opts.HeaderLines {
				header = append(header, *joinTokens(trans))
				eventBox.Set(EvtHeader, header)
				return nil
			}
			item := Item{
				text:     joinTokens(trans),
				origText: data,
				index:    uint32(index),
				colors:   nil,
				rank:     Rank{0, 0, uint32(index)}}

			trimmed, colors := ansiProcessor(item.text)
			item.text = trimmed
			item.colors = colors
			return &item
		})
	}

	// Reader
	streamingFilter := opts.Filter != nil && !sort && !opts.Tac && !opts.Sync
	if !streamingFilter {
		reader := Reader{func(str string) bool {
			return chunkList.Push(str)
		}, eventBox, opts.ReadZero}
		go reader.ReadSource()
	}

	// Matcher
	patternBuilder := func(runes []rune) *Pattern {
		return BuildPattern(
			opts.Mode, opts.Case, opts.Nth, opts.Delimiter, runes)
	}
	matcher := NewMatcher(patternBuilder, sort, opts.Tac, eventBox)

	// Filtering mode
	if opts.Filter != nil {
		if opts.PrintQuery {
			fmt.Println(*opts.Filter)
		}

		pattern := patternBuilder([]rune(*opts.Filter))

		if streamingFilter {
			reader := Reader{
				func(str string) bool {
					item := chunkList.trans(&str, 0)
					if item != nil && pattern.MatchItem(item) {
						fmt.Println(*item.text)
					}
					return false
				}, eventBox, opts.ReadZero}
			reader.ReadSource()
		} else {
			eventBox.Unwatch(EvtReadNew)
			eventBox.WaitFor(EvtReadFin)

			snapshot, _ := chunkList.Snapshot()
			merger, _ := matcher.scan(MatchRequest{
				chunks:  snapshot,
				pattern: pattern})
			for i := 0; i < merger.Length(); i++ {
				fmt.Println(merger.Get(i).AsString())
			}
		}
		os.Exit(0)
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
			defer events.Clear()
			for evt, value := range *events {
				switch evt {

				case EvtReadNew, EvtReadFin:
					reading = reading && evt == EvtReadNew
					snapshot, count := chunkList.Snapshot()
					terminal.UpdateCount(count, !reading)
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
					terminal.UpdateHeader(value.([]string), opts.HeaderLines)

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
										fmt.Println(opts.Query)
									}
									if len(opts.Expect) > 0 {
										fmt.Println()
									}
									for i := 0; i < count; i++ {
										fmt.Println(val.Get(i).AsString())
									}
									os.Exit(0)
								}
								deferred = false
								terminal.startChan <- true
							}
						}
						terminal.UpdateList(val)
					}
				}
			}
		})
		if delay && reading {
			dur := util.DurWithin(
				time.Duration(ticks)*coordinatorDelayStep,
				0, coordinatorDelayMax)
			time.Sleep(dur)
		}
	}
}
