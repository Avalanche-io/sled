package config_test

import (
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled/config"
)

func addrString(s string) *string {
	return &s
}

func TestPath(t *testing.T) {
	// init
	is := is.New(t)
	tests := []struct {
		In    *string
		Expct *string
	}{
		{
			In:    nil,
			Expct: addrString("<nil>"),
		},
		{
			In:    addrString(""),
			Expct: addrString("<nil>"),
		},
		{
			In:    addrString("foo.bar"),
			Expct: addrString("foo.bar/"),
		},
		{
			In:    addrString("/foo.bar"),
			Expct: addrString("/"),
		},
		{
			In:    addrString("foo.bar/"),
			Expct: addrString("foo.bar/"),
		},
		{
			In:    addrString("bat/foo.bar"),
			Expct: addrString("bat/"),
		},
		{
			In:    addrString("/bat/foo.bar"),
			Expct: addrString("/bat/"),
		},
		{
			In:    addrString("http://bat/foo.bar"),
			Expct: addrString("/bat/"),
		},
		{
			In:    addrString("/a/b/baz/bat/foo.bar"),
			Expct: addrString("/a/b/baz/bat/"),
		},
		{
			In:    addrString("/a/b/baz/bat/foo/bar"),
			Expct: addrString("/a/b/baz/bat/foo/"),
		},
	}

	// nothing to do

	// check
	for _, t := range tests {
		var path *config.Path
		if t.In != nil {
			path, _ = config.NewPath(*t.In)
		}
		is.Equal(path.String(), *t.Expct)
	}
}

func TestAppend(t *testing.T) {
	// init
	is := is.New(t)
	p1, err := config.NewPath("/foo/bar/")
	is.NoErr(err)

	p2, err := config.NewPath("/some/sub/path/")
	is.NoErr(err)
	p3, err := config.NewPath("")
	is.Equal(err.Error(), "Nil path.")

	is.Equal(p1.Append(nil).String(), "/foo/bar/")
	is.Equal(p1.Append(p3).String(), "/foo/bar/")
	is.Equal(p1.Append(p2).String(), "/foo/bar/some/sub/path/")
}
