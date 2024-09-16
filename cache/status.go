package cache

type Status int

const (
	StatusNotReady Status = iota
	StatusReady
	StatusRunning
	StatusStopped
)
