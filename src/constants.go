package fzf

import (
	"github.com/junegunn/fzf/src/util"
)

// Current version
const Version = "0.9.0"

// fzf events
const (
	EvtReadNew util.EventType = iota
	EvtReadFin
	EvtSearchNew
	EvtSearchProgress
	EvtSearchFin
	EvtClose
)
