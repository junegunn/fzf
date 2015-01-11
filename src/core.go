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
)

const COORDINATOR_DELAY_MAX time.Duration = 100 * time.Millisecond
const COORDINATOR_DELAY_STEP time.Duration = 10 * time.Millisecond

func initProcs() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

/*
Reader   -> EVT_READ_FIN
Reader   -> EVT_READ_NEW        -> Matcher  (restart)
Terminal -> EVT_SEARCH_NEW      -> Matcher  (restart)
Matcher  -> EVT_SEARCH_PROGRESS -> Terminal (update info)
Matcher  -> EVT_SEARCH_FIN      -> Terminal (update list)
*/

func Run(options *Options) {
	initProcs()

	opts := ParseOptions()

	if opts.Version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	// Event channel
	eventBox := NewEventBox()

	// Chunk list
	var chunkList *ChunkList
	if len(opts.WithNth) == 0 {
		chunkList = NewChunkList(func(data *string, index int) *Item {
			return &Item{text: data, rank: Rank{0, 0, uint32(index)}}
		})
	} else {
		chunkList = NewChunkList(func(data *string, index int) *Item {
			tokens := Tokenize(data, opts.Delimiter)
			item := Item{
				text:     Transform(tokens, opts.WithNth).whole,
				origText: data,
				rank:     Rank{0, 0, uint32(index)}}
			return &item
		})
	}

	// Reader
	reader := Reader{func(str string) { chunkList.Push(str) }, eventBox}
	go reader.ReadSource()

	// Matcher
	patternBuilder := func(runes []rune) *Pattern {
		return BuildPattern(
			opts.Mode, opts.Case, opts.Nth, opts.Delimiter, runes)
	}
	matcher := NewMatcher(patternBuilder, opts.Sort > 0, eventBox)

	// Defered-interactive / Non-interactive
	//   --select-1 | --exit-0 | --filter
	if filtering := opts.Filter != nil; filtering || opts.Select1 || opts.Exit0 {
		limit := 0
		var patternString string
		if filtering {
			patternString = *opts.Filter
		} else {
			if opts.Select1 || opts.Exit0 {
				limit = 1
			}
			patternString = opts.Query
		}
		pattern := patternBuilder([]rune(patternString))

		looping := true
		eventBox.Unwatch(EVT_READ_NEW)
		for looping {
			eventBox.Wait(func(events *Events) {
				for evt, _ := range *events {
					switch evt {
					case EVT_READ_FIN:
						looping = false
						return
					}
				}
			})
		}

		snapshot, _ := chunkList.Snapshot()
		merger, cancelled := matcher.scan(MatchRequest{
			chunks:  snapshot,
			pattern: pattern}, limit)

		if !cancelled && (filtering ||
			opts.Exit0 && merger.Length() == 0 ||
			opts.Select1 && merger.Length() == 1) {
			if opts.PrintQuery {
				fmt.Println(patternString)
			}
			for i := 0; i < merger.Length(); i++ {
				merger.Get(i).Print()
			}
			os.Exit(0)
		}
	}

	// Go interactive
	go matcher.Loop()

	// Terminal I/O
	terminal := NewTerminal(opts, eventBox)
	go terminal.Loop()

	// Event coordination
	reading := true
	ticks := 0
	eventBox.Watch(EVT_READ_NEW)
	for {
		delay := true
		ticks += 1
		eventBox.Wait(func(events *Events) {
			defer events.Clear()
			for evt, value := range *events {
				switch evt {

				case EVT_READ_NEW, EVT_READ_FIN:
					reading = reading && evt == EVT_READ_NEW
					snapshot, count := chunkList.Snapshot()
					terminal.UpdateCount(count, !reading)
					matcher.Reset(snapshot, terminal.Input(), false)

				case EVT_SEARCH_NEW:
					snapshot, _ := chunkList.Snapshot()
					matcher.Reset(snapshot, terminal.Input(), true)
					delay = false

				case EVT_SEARCH_PROGRESS:
					switch val := value.(type) {
					case float32:
						terminal.UpdateProgress(val)
					}

				case EVT_SEARCH_FIN:
					switch val := value.(type) {
					case *Merger:
						terminal.UpdateList(val)
					}
				}
			}
		})
		if delay && reading {
			dur := DurWithin(
				time.Duration(ticks)*COORDINATOR_DELAY_STEP,
				0, COORDINATOR_DELAY_MAX)
			time.Sleep(dur)
		}
	}
}
