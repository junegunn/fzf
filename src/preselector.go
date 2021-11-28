package fzf

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

type Preselector struct {
	path        string
	prepointed  string
	preselected []string
}

func fileFmtError(path string, e error) error {
	if os.IsPermission(e) {
		return errors.New("permission denied: " + path)
	}
	return errors.New("invalid preselector file: " + e.Error())
}

func NewPreselector(path string) (*Preselector, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		// If it doesn't exist, check if we can create a file with the name
		if os.IsNotExist(err) {
			data = []byte{}
			if err := ioutil.WriteFile(path, data, 0600); err != nil {
				return nil, fileFmtError(path, err)
			}
		} else {
			return nil, fileFmtError(path, err)
		}
	}
	// Split lines and limit the maximum number of lines
	lines := strings.Split(strings.Trim(string(data), "\n"), "\n")
	return &Preselector{
		path:        path,
		prepointed:  lines[0],
		preselected: lines[1:]}, nil
}

func (p *Preselector) apply(t *Terminal) {
	var itemRes Result
	for i := 0; i < t.merger.Length(); i++ {
		itemRes = t.merger.Get(i)
		// find prepointed if any
		if p.prepointed != "" &&
			itemRes.item.AsString(true) == p.prepointed {
			t.vset(int(itemRes.item.Index()))
			p.prepointed = ""
		}
		// find preselected if any
		if p.preselected != nil {
			for j := 0; j < len(p.preselected); j++ {
				if itemRes.item.AsString(true) == p.preselected[j] {
					t.selectItem(itemRes.item)
					p.preselected[j] = "" // TODO: remove deleted ones
				}
			}
		}
	}
}

func (p *Preselector) save(t *Terminal) error {
	current := t.currentItem()
	if current == nil {
		return errors.New("current item not found")
	}
	res := make([]string, 0, len(t.selected)+1)
	res = append(res, current.AsString(true))
	for _, sel := range t.selected {
		res = append(res, sel.item.AsString(true))
	}

	data := strings.Join(res, "\n")
	if err := ioutil.WriteFile(p.path, []byte(data), 0600); err != nil {
		return fileFmtError(p.path, err)
	}

	return nil
}
