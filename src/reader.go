package fzf

import (
	"bufio"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/junegunn/fzf/src/util"
)

type Reader interface {
	ReadSource()
}

type ReaderFactory func(pusher func([]byte) bool, eventBox *util.EventBox, delimNil bool) Reader

// Reader reads from command or standard input
type DefaultReader struct {
	pusher   func([]byte) bool
	eventBox *util.EventBox
	delimNil bool
	event    int32
}

// NewReader returns new Reader object
func NewDefaultReader(pusher func([]byte) bool, eventBox *util.EventBox, delimNil bool) Reader {
	return &DefaultReader{pusher, eventBox, delimNil, int32(EvtReady)}
}

func (r *DefaultReader) startEventPoller() {
	go func() {
		ptr := &r.event
		pollInterval := readerPollIntervalMin
		for {
			if atomic.CompareAndSwapInt32(ptr, int32(EvtReadNew), int32(EvtReady)) {
				r.eventBox.Set(EvtReadNew, true)
				pollInterval = readerPollIntervalMin
			} else if atomic.LoadInt32(ptr) == int32(EvtReadFin) {
				return
			} else {
				pollInterval += readerPollIntervalStep
				if pollInterval > readerPollIntervalMax {
					pollInterval = readerPollIntervalMax
				}
			}
			time.Sleep(pollInterval)
		}
	}()
}

func (r *DefaultReader) fin(success bool) {
	atomic.StoreInt32(&r.event, int32(EvtReadFin))
	r.eventBox.Set(EvtReadFin, success)
}

// ReadSource reads data from the default command or from standard input
func (r *DefaultReader) ReadSource() {
	r.startEventPoller()
	var success bool
	if util.IsTty() {
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			cmd = defaultCommand
		}
		success = r.readFromCommand(cmd)
	} else {
		success = r.readFromStdin()
	}
	r.fin(success)
}

func (r *DefaultReader) feed(src io.Reader) {
	delim := byte('\n')
	if r.delimNil {
		delim = '\000'
	}
	reader := bufio.NewReaderSize(src, readerBufferSize)
	for {
		// ReadBytes returns err != nil if and only if the returned data does not
		// end in delim.
		bytea, err := reader.ReadBytes(delim)
		byteaLen := len(bytea)
		if byteaLen > 0 {
			if err == nil {
				// get rid of carriage return if under Windows:
				if util.IsWindows() && byteaLen >= 2 && bytea[byteaLen-2] == byte('\r') {
					bytea = bytea[:byteaLen-2]
				} else {
					bytea = bytea[:byteaLen-1]
				}
			}
			if r.pusher(bytea) {
				atomic.StoreInt32(&r.event, int32(EvtReadNew))
			}
		}
		if err != nil {
			break
		}
	}
}

func (r *DefaultReader) readFromStdin() bool {
	r.feed(os.Stdin)
	return true
}

func (r *DefaultReader) readFromCommand(cmd string) bool {
	listCommand := util.ExecCommand(cmd)
	out, err := listCommand.StdoutPipe()
	if err != nil {
		return false
	}
	err = listCommand.Start()
	if err != nil {
		return false
	}
	r.feed(out)
	return listCommand.Wait() == nil
}

type ChannelReader struct {
	channel  <-chan []byte
	pusher   func([]byte) bool
	eventBox *util.EventBox
	event    int32
}

func NewChannelReader(channel <-chan []byte) ReaderFactory {
	return func(pusher func([]byte) bool, eventBox *util.EventBox, delimNil bool) Reader {
		return &ChannelReader{
			channel:  channel,
			pusher:   pusher,
			eventBox: eventBox,
			event:    int32(EvtReady),
		}
	}
}

func (r *ChannelReader) ReadSource() {
	r.startEventPoller()
	for bytes := range r.channel {
		if r.pusher(bytes) {
			atomic.StoreInt32(&r.event, int32(EvtReadNew))
		}
	}
	r.fin()
}

func (r *ChannelReader) startEventPoller() {
	go func() {
		ptr := &r.event
		pollInterval := readerPollIntervalMin
		for {
			if atomic.CompareAndSwapInt32(ptr, int32(EvtReadNew), int32(EvtReady)) {
				r.eventBox.Set(EvtReadNew, true)
				pollInterval = readerPollIntervalMin
			} else if atomic.LoadInt32(ptr) == int32(EvtReadFin) {
				return
			} else {
				pollInterval += readerPollIntervalStep
				if pollInterval > readerPollIntervalMax {
					pollInterval = readerPollIntervalMax
				}
			}
			time.Sleep(pollInterval)
		}
	}()
}

func (r *ChannelReader) fin() {
	atomic.StoreInt32(&r.event, int32(EvtReadFin))
	r.eventBox.Set(EvtReadFin, true)
}
