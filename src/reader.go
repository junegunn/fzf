package fzf

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"unicode/utf8"

	"github.com/junegunn/fzf/src/util"
)

// Reader reads from command or standard input
type Reader struct {
	pusher   func([]rune) bool
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
			runes := make([]rune, 0, len(bytea))
			for i := 0; i < len(bytea); {
				if bytea[i] < utf8.RuneSelf {
					runes = append(runes, rune(bytea[i]))
					i++
				} else {
					r, sz := utf8.DecodeRune(bytea[i:])
					i += sz
					runes = append(runes, r)
				}
			}
			if err == nil {
				runes = runes[:len(runes)-1]
			}
			if r.pusher(runes) {
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
