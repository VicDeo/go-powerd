package app

type iconGetter interface {
	Get(percent int, charging bool) (icon []byte, fromCache bool)
}
