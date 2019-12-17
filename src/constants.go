package fzf

import (
	"math"
	"os"
	"time"

	"github.com/junegunn/fzf/src/util"
)

const (
	// Current version
	version = "0.20.0"

	// Core
	coordinatorDelayMax  time.Duration = 100 * time.Millisecond
	coordinatorDelayStep time.Duration = 10 * time.Millisecond

	// Reader
	readerBufferSize       = 64 * 1024
	readerPollIntervalMin  = 10 * time.Millisecond
	readerPollIntervalStep = 5 * time.Millisecond
	readerPollIntervalMax  = 50 * time.Millisecond

	// Terminal
	initialDelay      = 20 * time.Millisecond
	initialDelayTac   = 100 * time.Millisecond
	spinnerDuration   = 200 * time.Millisecond
	previewCancelWait = 500 * time.Millisecond
	maxPatternLength  = 300
	maxMulti          = math.MaxInt32

	// Matcher
	numPartitionsMultiplier = 8
	maxPartitions           = 32
	progressMinDuration     = 200 * time.Millisecond

	// Capacity of each chunk
	chunkSize int = 100

	// Pre-allocated memory slices to minimize GC
	slab16Size int = 100 * 1024 // 200KB * 32 = 12.8MB
	slab32Size int = 2048       // 8KB * 32 = 256KB

	// Do not cache results of low selectivity queries
	queryCacheMax int = chunkSize / 5

	// Not to cache mergers with large lists
	mergerCacheMax int = 100000

	// History
	defaultHistoryMax int = 1000

	// Jump labels
	defaultJumpLabels string = "asdfghjklqwertyuiopzxcvbnm1234567890ASDFGHJKLQWERTYUIOPZXCVBNM`~;:,<.>/?'\"!@#$%^&*()[{]}-_=+"
)

var defaultCommand string

func init() {
	if !util.IsWindows() {
		defaultCommand = `set -o pipefail; command find -L . -mindepth 1 \( -path '*/\.*' -o -fstype 'sysfs' -o -fstype 'devfs' -o -fstype 'devtmpfs' -o -fstype 'proc' \) -prune -o -type f -print -o -type l -print 2> /dev/null | cut -b3-`
	} else if os.Getenv("TERM") == "cygwin" {
		defaultCommand = `sh -c "command find -L . -mindepth 1 -path '*/\.*' -prune -o -type f -print -o -type l -print 2> /dev/null | cut -b3-"`
	} else {
		defaultCommand = `for /r %P in (*) do @(set "_curfile=%P" & set "_curfile=!_curfile:%__CD__%=!" & echo !_curfile!)`
	}
}

// fzf events
const (
	EvtReadNew util.EventType = iota
	EvtReadFin
	EvtSearchNew
	EvtSearchProgress
	EvtSearchFin
	EvtHeader
	EvtReady
)

const (
	exitCancel    = -1
	exitOk        = 0
	exitNoMatch   = 1
	exitError     = 2
	exitInterrupt = 130
)
