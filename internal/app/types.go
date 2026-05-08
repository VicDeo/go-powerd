package app

type iconGetter interface {
	Get(percent int, charging bool) (icon []byte, fromCache bool)
}

type actionTrigger interface {
	HandleUpdate(capacity int, isPluggedIn bool)
}
