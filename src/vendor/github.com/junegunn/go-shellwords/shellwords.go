package shellwords

import (
	"errors"
	"os"
	"regexp"
)

var (
	ParseEnv      bool = false
	ParseBacktick bool = false
)

var envRe = regexp.MustCompile(`\$({[a-zA-Z0-9_]+}|[a-zA-Z0-9_]+)`)

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func replaceEnv(s string) string {
	return envRe.ReplaceAllStringFunc(s, func(s string) string {
		s = s[1:]
		if s[0] == '{' {
			s = s[1 : len(s)-1]
		}
		return os.Getenv(s)
	})
}

type Parser struct {
	ParseEnv      bool
	ParseBacktick bool
	Position      int
}

func NewParser() *Parser {
	return &Parser{ParseEnv, ParseBacktick, 0}
}

func (p *Parser) Parse(line string) ([]string, error) {
	args := []string{}
	buf := ""
	var escaped, doubleQuoted, singleQuoted, backQuote bool
	backtick := ""

	pos := -1
	got := false

loop:
	for i, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if r == '\\' {
			if singleQuoted {
				buf += string(r)
			} else {
				escaped = true
			}
			continue
		}

		if isSpace(r) {
			if singleQuoted || doubleQuoted || backQuote {
				buf += string(r)
				backtick += string(r)
			} else if got {
				if p.ParseEnv {
					buf = replaceEnv(buf)
				}
				args = append(args, buf)
				buf = ""
				got = false
			}
			continue
		}

		switch r {
		case '`':
			if !singleQuoted && !doubleQuoted {
				if p.ParseBacktick {
					if backQuote {
						out, err := shellRun(backtick)
						if err != nil {
							return nil, err
						}
						buf = out
					}
					backtick = ""
					backQuote = !backQuote
					continue
				}
				backtick = ""
				backQuote = !backQuote
			}
		case '"':
			if !singleQuoted {
				doubleQuoted = !doubleQuoted
				continue
			}
		case '\'':
			if !doubleQuoted {
				singleQuoted = !singleQuoted
				continue
			}
		case ';', '&', '|', '<', '>':
			if !(escaped || singleQuoted || doubleQuoted || backQuote) {
				pos = i
				break loop
			}
		}

		got = true
		buf += string(r)
		if backQuote {
			backtick += string(r)
		}
	}

	if got {
		if p.ParseEnv {
			buf = replaceEnv(buf)
		}
		args = append(args, buf)
	}

	if escaped || singleQuoted || doubleQuoted || backQuote {
		return nil, errors.New("invalid command line string")
	}

	p.Position = pos

	return args, nil
}

func Parse(line string) ([]string, error) {
	return NewParser().Parse(line)
}
