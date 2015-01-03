package fzf

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

const COORDINATOR_DELAY time.Duration = 100 * time.Millisecond

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
			return &Item{text: data, index: index}
		})
	} else {
		chunkList = NewChunkList(func(data *string, index int) *Item {
			item := Item{text: data, index: index}
			tokens := Tokenize(item.text, opts.Delimiter)
			item.origText = item.text
			item.text = Transform(tokens, opts.WithNth).whole
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

		matches, cancelled := matcher.scan(MatchRequest{
			chunks:  chunkList.Snapshot(),
			pattern: pattern}, limit)

		if !cancelled && (filtering ||
			opts.Exit0 && len(matches) == 0 || opts.Select1 && len(matches) == 1) {
			if opts.PrintQuery {
				fmt.Println(patternString)
			}
			for _, item := range matches {
				item.Print()
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
					terminal.UpdateCount(chunkList.Count(), !reading)
					matcher.Reset(chunkList.Snapshot(), terminal.Input(), false)

				case EVT_SEARCH_NEW:
					matcher.Reset(chunkList.Snapshot(), terminal.Input(), true)
					delay = false

				case EVT_SEARCH_PROGRESS:
					switch val := value.(type) {
					case float32:
						terminal.UpdateProgress(val)
					}

				case EVT_SEARCH_FIN:
					switch val := value.(type) {
					case []*Item:
						terminal.UpdateList(val)
					}
				}
			}
		})
		if ticks > 3 && delay && reading {
			time.Sleep(COORDINATOR_DELAY)
		}
	}
}
