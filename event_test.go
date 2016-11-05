package sled_test

import (
	"testing"
	"time"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
)

func TestEventCreate(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	sl.On(sled.CreateEvent, func(event *sled.Event) {
		t.Log("event received on: ", event.Time)
		is.Equal(event.Key, "foo")
	})

	sl.Set("foo", "bar")
	value := sl.Get("foo")
	is.Equal(value.(string), "bar")
	time.Sleep(2 * time.Second)
	sl.Wait()
}
