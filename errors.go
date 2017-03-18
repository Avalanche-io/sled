package sled

type ErrCanceled struct{}

func (ErrCanceled) Error() string {
	return "canceled"
}
