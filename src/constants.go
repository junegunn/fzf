package fzf

// Current version
const Version = "0.9.0"

// EventType is the type for fzf events
type EventType int

// fzf events
const (
	EvtReadNew EventType = iota
	EvtReadFin
	EvtSearchNew
	EvtSearchProgress
	EvtSearchFin
	EvtClose
)
