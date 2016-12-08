package config

import "path/filepath"

// Strongly typed 'filename' type, that can enforce file name only
// without any path prefixes.
type Filename string

// Creates a Filename from a string by removing any path prefixes.
func NewFilename(filename string) *Filename {
	l := len(filename)
	switch {
	case l == 0:
		return nil
	default:
		_, f := filepath.Split(filename)
		fl := Filename(f)
		return &fl
	}
}

// Implements Stringer interface
func (f *Filename) String() string {
	if f == nil {
		return "<nil>"
	}
	return string(*f)
}
