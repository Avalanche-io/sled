package config_test

import (
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled/config"
)

func TestFilename(t *testing.T) {
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
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("/foo.bar"),
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("foo.bar/"),
			Expct: addrString(""),
		},
		{
			In:    addrString("bat/foo.bar"),
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("/bat/foo.bar"),
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("http://bat/foo.bar"),
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("/a/b/baz/bat/foo.bar"),
			Expct: addrString("foo.bar"),
		},
		{
			In:    addrString("/a/b/baz/bat/foo/bar"),
			Expct: addrString("bar"),
		},
	}

	// nothing to do

	// check
	for _, t := range tests {
		var fn *config.Filename
		if t.In != nil {
			fn = config.NewFilename(*t.In)
		}
		is.Equal(fn.String(), *t.Expct)
	}
}
