package sled_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
)

func TestEventCreate(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	sl.On(sled.CreateEvent, "other", func(event *sled.Event) {
		t.Log("event received on: ", event.Time)
		is.Equal(event.Key, "foo")
	})

	sl.Set("foo", "bar")
	value := sl.Get("foo")
	is.Equal(value.(string), "bar")
	// sl.Wait()
}

func TestEventHandlerLists(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	call_count := 0

	// var wg sync.WaitGroup
	// wg.Add(3)
	// handler 1
	sl.On(sled.CreateEvent, "1", func(event *sled.Event) {
		t.Log("Handler 1 received event on: ", event.Time)
		// is.Equal(event.Key, "foo")
		call_count++
		// wg.Done()
	})

	// handler 2
	sl.On(sled.CreateEvent, "2", func(event *sled.Event) {
		t.Log("Handler 2 received event on: ", event.Time)
		// is.Equal(event.Key, "foo")
		call_count++
		// wg.Done()
	})

	// handler 3
	sl.On(sled.CreateEvent, "3", func(event *sled.Event) {
		t.Log("Handler 3 received event on: ", event.Time)
		// is.Equal(event.Key, "foo")
		call_count++
		// wg.Done()
	})

	for i := 0; i < 10; i++ {
		sl.Set("foo"+strconv.Itoa(i), "bar"+strconv.Itoa(i))
	}
	// wg.Wait()
	time.Sleep(10 * time.Second)
	// sl.Wait()
	is.Equal(call_count, 3*10)
}
