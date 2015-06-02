package fzf

import (
	"bufio"
	"io"
	"os"
	"os/exec"

	"github.com/junegunn/fzf/src/util"
)

// Reader reads from command or standard input
type Reader struct {
	pusher   func(string)
	eventBox *util.EventBox
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
	reader := bufio.NewReader(src)
	eof := false
Loop:
	for !eof {
		buf := []byte{}
		iter := 0 // TODO: max size?
		for {
			// "ReadLine either returns a non-nil line or it returns an error, never both"
			line, isPrefix, err := reader.ReadLine()
			eof = err == io.EOF
			if eof {
				break
			} else if err != nil {
				break Loop
			}
			iter++
			buf = append(buf, line...)
			if !isPrefix {
				break
			}
		}
		if iter > 0 {
			r.pusher(string(buf))
			r.eventBox.Set(EvtReadNew, nil)
		}
	}
}

func (r *Reader) readFromStdin() {
	r.feed(os.Stdin)
}

func (r *Reader) readFromCommand(cmd string) {
	listCommand := exec.Command("sh", "-c", cmd)
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
