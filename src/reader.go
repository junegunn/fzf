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
	if util.IsTty() {
		cmd := os.Getenv("FZF_DEFAULT_COMMAND")
		if len(cmd) == 0 {
			cmd = defaultCommand
		}
		r.readFromCommand(cmd)
	} else {
		r.readFromStdin()
	}
	r.eventBox.Set(EvtReadFin, nil)
}

func (r *Reader) feed(src io.Reader) {
	delim := byte('\n')
	if r.delimNil {
		delim = '\000'
	}
	reader := bufio.NewReader(src)
	for {
		// ReadBytes returns err != nil if and only if the returned data does not
		// end in delim.
		bytea, err := reader.ReadBytes(delim)
		if len(bytea) > 0 {
			if err == nil {
				bytea = bytea[:len(bytea)-1]
			}
			if r.pusher(bytea) {
				r.eventBox.Set(EvtReadNew, nil)
			}
		}
		if err != nil {
			break
		}
	}
}

func (r *Reader) readFromStdin() {
	r.feed(os.Stdin)
}

func (r *Reader) readFromCommand(cmd string) {
	listCommand := util.ExecCommand(cmd)
	out, err := listCommand.StdoutPipe()
	if err != nil {
		return
	}
	err = listCommand.Start()
	if err != nil {
		return
	}
	defer listCommand.Wait()
	r.feed(out)
}
