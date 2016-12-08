package storage_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/cheekybits/is"
	"github.com/etcenter/c4/asset"

	"github.com/Avalanche-io/sled"
)

func randomdata(amount int) ([]byte, error) {
	data := make([]byte, amount)
	_, err := rand.Read(data)
	return data, err
}

func TestStoreByteSlice(t *testing.T) {
	// init
	amount := 1024
	is := is.New(t)
	sl := sled.New()
	data, err := randomdata(amount)
	is.NoErr(err)

	// do
	sl.Set("/foo", data)
	value, err := sl.Get("/foo")
	is.NoErr(err)

	// check
	cnt := 0
	for i, b := range data {
		cnt++
		is.Equal(value.([]byte)[i], b)
	}
	is.Equal(cnt, amount)
}

func TestKeyID(t *testing.T) {
	// init
	amount := 1024
	is := is.New(t)
	sl := sled.New()
	data, err := randomdata(amount)
	is.NoErr(err)
	id, err := asset.Identify(bytes.NewReader(data))
	is.NoErr(err)

	// do
	sl.Set("/foo", data)
	foo_id := sl.GetID("/foo")
	is.NotNil(foo_id)

	// check
	is.Equal(id.String(), foo_id.String())
}
