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

func TestEventHandlerLists(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	call_count := 0

	// handler 1
	sl.On(sled.CreateEvent, func(event *sled.Event) {
		t.Log("event received on: ", event.Time)
		is.Equal(event.Key, "foo")
		call_count++
	})

	// handler 2
	sl.On(sled.CreateEvent, func(event *sled.Event) {
		t.Log("event received on: ", event.Time)
		is.Equal(event.Key, "foo")
		call_count++
	})

	// handler 2
	sl.On(sled.CreateEvent, func(event *sled.Event) {
		t.Log("event received on: ", event.Time)
		is.Equal(event.Key, "foo")
		call_count++
	})

	sl.Set("foo", "bar")
	value := sl.Get("foo")
	is.Equal(value.(string), "bar")
	sl.Wait()
	time.Sleep(1 * time.Second)
	is.Equal(call_count, 3)
}
