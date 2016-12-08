package sled

type SledError string

func (s SledError) Error() string {
	return "Sled error: " + string(s)
}

const (
	dbOpenError = SledError("Db already open.")
)
