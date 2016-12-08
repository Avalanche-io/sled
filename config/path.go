package config

import (
	"errors"
	"path/filepath"
	"strings"
)

// Strongly typed 'path' type, that can enforce "/" ending
type Path string

// Create a Path from a string, enforcing trailing slash
func NewPath(path string) (*Path, error) {
	path = removeUrlPrefix(path)
	path = deduplicateSlash(path)

	l := len(path)
	switch {
	case l == 0:
		return nil, errors.New("Nil path.")
	case path[l-1] == '/':
		p := Path(path)
		return &p, nil
	case strings.Index(path, "/") == -1:
		p := Path(path + "/")
		return &p, errors.New("No slashes at all. Appending slash to end of string.")
	default:
		d, _ := filepath.Split(path)
		p := Path(d)
		return &p, errors.New("No trailing slash. Removed last item in path.")
	}
}

// Append sub-path to path managing slashes appropriately.
func (p *Path) Append(subpath *Path) *Path {
	if subpath == nil {
		return p
	}
	path, _ := NewPath(p.String() + "/" + subpath.String())
	return path
}

// Implements Stringer interface
func (p *Path) String() string {
	if p == nil {
		return "<nil>"
	}
	return string(*p)
}

func deduplicateSlash(path string) string {
	p := path
	i := strings.Index(p, "//")

	for i != -1 {
		// p = p[:i] + p[i+1:]
		p = strings.Replace(p, "//", "/", -1)
		i = strings.Index(p, "//")
	}
	return p
}

func removeUrlPrefix(path string) string {
	p := path
	if i := strings.LastIndex(path, ":"); i != -1 {
		p = path[i+1:]
	}
	return p
}
