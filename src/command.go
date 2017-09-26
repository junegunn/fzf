package fzf

import (
	"os"
	"strings"

	"github.com/junegunn/fzf/src/util"
)

type Command interface {
	GetPreview(stripAnsi bool, delimiter Delimiter, query string, allItems []*Item) string
	HasPlusFlag() bool
	Execute(withStdio bool, stripAnsi bool, delimiter Delimiter, forcePlus bool, query string, allItems []*Item)
}

type DefaultCommand struct {
	command string
}

func NewDefaultCommand(command string) Command {
	if len(command) == 0 {
		return nil
	}
	return &DefaultCommand{command}
}

func (p *DefaultCommand) GetPreview(stripAnsi bool, delimiter Delimiter, query string, allItems []*Item) string {
	command := replacePlaceholder(p.command,
		stripAnsi, delimiter, false, query, allItems)
	cmd := util.ExecCommand(command)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func (p *DefaultCommand) HasPlusFlag() bool {
	for _, match := range placeholder.FindAllString(p.command, -1) {
		if match[0] == '\\' {
			continue
		}
		if match[1] == '+' {
			return true
		}
	}
	return false
}

func (p *DefaultCommand) Execute(withStdio bool, stripAnsi bool, delimiter Delimiter, forcePlus bool, query string, allItems []*Item) {
	command := replacePlaceholder(p.command, stripAnsi, delimiter, forcePlus, query, allItems)
	cmd := util.ExecCommand(command)
	if withStdio {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Run()
}

func replacePlaceholder(template string, stripAnsi bool, delimiter Delimiter, forcePlus bool, query string, allItems []*Item) string {
	current := allItems[:1]
	selected := allItems[1:]
	if current[0] == nil {
		current = []*Item{}
	}
	if selected[0] == nil {
		selected = []*Item{}
	}
	return placeholder.ReplaceAllStringFunc(template, func(match string) string {
		// Escaped pattern
		if match[0] == '\\' {
			return match[1:]
		}

		// Current query
		if match == "{q}" {
			return quoteEntry(query)
		}

		plusFlag := forcePlus
		if match[1] == '+' {
			match = "{" + match[2:]
			plusFlag = true
		}
		items := current
		if plusFlag {
			items = selected
		}

		replacements := make([]string, len(items))

		if match == "{}" {
			for idx, item := range items {
				replacements[idx] = quoteEntry(item.AsString(stripAnsi))
			}
			return strings.Join(replacements, " ")
		}

		tokens := strings.Split(match[1:len(match)-1], ",")
		ranges := make([]Range, len(tokens))
		for idx, s := range tokens {
			r, ok := ParseRange(&s)
			if !ok {
				// Invalid expression, just return the original string in the template
				return match
			}
			ranges[idx] = r
		}

		for idx, item := range items {
			tokens := Tokenize(item.AsString(stripAnsi), delimiter)
			trans := Transform(tokens, ranges)
			str := string(joinTokens(trans))
			if delimiter.Str != nil {
				str = strings.TrimSuffix(str, *delimiter.Str)
			} else if delimiter.Regex != nil {
				delims := delimiter.Regex.FindAllStringIndex(str, -1)
				if len(delims) > 0 && delims[len(delims)-1][1] == len(str) {
					str = str[:delims[len(delims)-1][0]]
				}
			}
			str = strings.TrimSpace(str)
			replacements[idx] = quoteEntry(str)
		}
		return strings.Join(replacements, " ")
	})
}
