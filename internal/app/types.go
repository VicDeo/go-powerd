package app

import "context"

type iconGetter interface {
	Get(percent int, charging bool) (icon []byte, fromCache bool)
}

type actionTrigger interface {
	HandleUpdate(capacity int, isPluggedIn bool)
}

type debouncer interface {
	Start(func())
	Trigger()
}

type statusProvider interface {
	Load() error
	Len() int
	IsPluggedIn() bool
	Capacity() int
	Tooltip(string) string
	Log()
}

type eventWatcher interface {
	Watch(context.Context, func()) error
}
