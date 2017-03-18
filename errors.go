package sled

type ErrCanceled struct{}

func (ErrCanceled) Error() string {
	return "canceled"
}

type ErrGetType struct {
	a string
	b string
}

func (e ErrGetType) Error() string {
	return "value of type " + e.a + " is not assignable to type " + e.b
}
