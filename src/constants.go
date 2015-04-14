package fzf

import (
	"github.com/junegunn/fzf/src/util"
)

// Current version
const Version = "0.9.8"

// fzf events
const (
	EvtReadNew util.EventType = iota
	EvtReadFin
	EvtSearchNew
	EvtSearchProgress
	EvtSearchFin
	EvtClose
)
