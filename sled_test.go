package sled_test

import (
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
)

func TestNewGetSet(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	sl := sled.New()
	type TestStruct struct {
		Foo string
	}

	tests := []struct {
		Key   string
		Value interface{}
	}{
		{
			Key:   "foo",
			Value: "bar",
		},
		{
			Key:   "bar",
			Value: 12,
		},
		{
			Key:   "baz",
			Value: TestStruct{"bat"},
		},
		{
			Key:   "foo2",
			Value: strptr("bar"),
		},
		{
			Key:   "bar2",
			Value: intptr(12),
		},
		{
			Key:   "baz2",
			Value: &TestStruct{"bat"},
		},
		{
			Key: "bat",
			Value: struct {
				Question string
				Answer   int
			}{
				Question: "What is the meaning of life, the universe, everything?",
				Answer:   42,
			},
		},
	}

	t.Log("check")
	for _, tt := range tests {
		switch v := tt.Value.(type) {
		// case string:
		// 	sl.Set(tt.Key, v)
		// case int:
		// 	sl.Set(tt.Key, v)
		default:
			sl.Set(tt.Key, v)
		}
		switch tt.Value.(type) {
		case string:
			var output string
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		case int:
			var output int
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		case *string:
			var output *string
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		case *int:
			var output *int
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		case TestStruct:
			var output TestStruct
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		case *TestStruct:
			var output *TestStruct
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		default:
			var output interface{}
			err := sl.Get(tt.Key, &output)
			is.NoErr(err)
			is.Equal(output, tt.Value)
		}
	}
}

func strptr(value string) *string {
	return &value
}

func intptr(value int) *int {
	return &value
}

func indexOf(list []string, key string) int {
	for i, k := range list {
		if k == key {
			return i
		}
	}
	return -1
}

func TestIterate(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	sl := sled.New()
	keys := []string{"foo", "bar", "baz"}
	values := []string{"value 1", "value 2", "value 3"}
	t.Log("do")
	for i := 0; i < len(keys); i++ {
		sl.Set(keys[i], values[i])
	}

	t.Log("check")
	for elem := range sl.Iterate(nil) {
		i := indexOf(keys, elem.Key())
		is.NotEqual(i, -1)
		is.Equal(elem.Value(), values[i])
		elem.Close()
	}
}

func TestSetIfNil(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	sl := sled.New()

	t.Log("do")
	sl.Set("foo", "bar")

	t.Log("check")
	is.False(sl.SetIfNil("foo", "bar"))
	is.True(sl.SetIfNil("baz", "bat"))
}

func TestDelete(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	sl := sled.New()

	t.Log("do")
	sl.Set("foo", "bar")
	baz, baz_not_ok := sl.Delete("baz")
	foo, foo_ok := sl.Delete("foo")

	t.Log("check")
	is.OK(!baz_not_ok)
	is.Nil(baz)
	is.OK(foo_ok)
	is.NotNil(foo)
	is.Equal(foo.(string), "bar")
	var nil_value interface{}
	err := sl.Get("foo", nil_value)
	is.Nil(nil_value)
	is.Equal(err.Error(), "key does not exist")
}

func TestSnapshot(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	sl := sled.New()

	t.Log("do")
	sl.Set("foo", "bar")
	snap := sl.Snapshot()
	sl.Set("bat", "baz")

	t.Log("check")
	// snap should have "foo"
	var bar_value string
	err := snap.Get("foo", &bar_value)
	is.NoErr(err)
	is.Equal(bar_value, "bar")

	// but not have "bat"
	var nil_value interface{}
	err = snap.Get("bat", nil_value)
	is.Err(err)
	is.Nil(nil_value)
}
