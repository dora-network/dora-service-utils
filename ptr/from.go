package ptr

func From[T any](from T) *T {
	return &from
}
