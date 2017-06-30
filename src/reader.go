package fzf

import (
	"bufio"
	"io"
	"os"

	"github.com/junegunn/fzf/src/util"
)

// Reader reads from command or standard input
type Reader struct {
	pusher   func([]byte) bool
	eventBox *util.EventBox
	delimNil bool
}

// ReadSource reads data from the default command or from standard input
func (r *Reader) ReadSource() {
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
	r.eventBox.Set(EvtReadFin, success)
}

func (r *Reader) feed(src io.Reader) {
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
		if len(bytea) > 0 {
			if err == nil {
				// get rid of carriage return if under Windows:
				if util.IsWindows() && byteaLen >= 2 && bytea[byteaLen-2] == byte('\r') {
					bytea = bytea[:byteaLen-2]
				} else {
					bytea = bytea[:byteaLen-1]
				}
			}
			if r.pusher(bytea) {
				r.eventBox.Set(EvtReadNew, true)
			}
		}
		if err != nil {
			break
		}
	}
}

func (r *Reader) readFromStdin() bool {
	r.feed(os.Stdin)
	return true
}

func (r *Reader) readFromCommand(cmd string) bool {
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
