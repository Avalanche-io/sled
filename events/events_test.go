package events_test

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
	"github.com/Avalanche-io/sled/events"
)

func TestSubscribeToEvent(t *testing.T) {
	is := is.New(t)
	sl := sled.New()

	sub := sl.Subscribe(events.StringToType("key-set"))
	go func() {
		time.Sleep(1 * time.Second)
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("/foo_%04d", i)
			sl.Set(key, i)
		}
	}()

	count := 0
	done := make(chan struct{})
	go func() {
		<-done
		err := sub.Close()
		is.NoErr(err)
	}()

	for e := range sub.Events() {
		is.Equal(e.Type, events.StringToType("key-set"))
		count++
		if count == 1000 {
			close(done)
		}
	}
	is.Equal(count, 1000)

}

func TestSubscribeToSpacificEvents(t *testing.T) {
	is := is.New(t)
	sl := sled.New()

	tests := []struct {
		Type   events.Type
		Key    string
		Values []string
		Count  int
	}{
		{
			Type:   events.StringToType("key-created"),
			Key:    "/foo",
			Values: []string{"bar"},
			Count:  1,
		},
		{
			Type:   events.StringToType("key-changed"),
			Key:    "/foo",
			Values: []string{"bar"}, // the old value is returned
			Count:  1,
		},
		{
			Type:   events.StringToType("key-removed"),
			Key:    "/foo",
			Values: []string{"bat"},
			Count:  1,
		},
		{
			Type:   events.StringToType("key-set"),
			Key:    "/foo",
			Values: []string{"bar", "bat"},
			Count:  2,
		},
	}
	var start_wg sync.WaitGroup
	var event_listeners_wg sync.WaitGroup
	subscriptions := make([]events.Subscription, 0, len(tests))
	start_wg.Add(len(tests))
	event_listeners_wg.Add(len(tests))
	event_type_counts := make(map[events.Type]int)
	for _, t := range tests {
		event_type_counts[t.Type] = t.Count
	}
	var event_type_mutex sync.Mutex
	for _, x := range tests {
		sub := sl.Subscribe(x.Type, x.Key)
		subscriptions = append(subscriptions, sub)
		go func(expected_values []string, expected_count int) {
			defer event_listeners_wg.Done()
			start_wg.Done()
			for e := range sub.Events() {
				t.Log("event:", e)
				event_type_mutex.Lock()
				event_type_counts[e.Type] = event_type_counts[e.Type] - 1
				event_type_mutex.Unlock()

				// is.Equal(e.Value, expected_values[count])
			}
			t.Log("sub.Events() - done")
		}(x.Values, x.Count)
	}
	start_wg.Wait()
	sl.Set("/foo", "bar")
	sl.Set("/foo", "bat")
	sl.Delete("/foo")

	time.AfterFunc(4*time.Millisecond, func() {
		for _, s := range subscriptions {
			err := s.Close()
			is.NoErr(err)
		}
	})
	time.Sleep(10 * time.Millisecond)
	event_type_mutex.Lock()
	for i, v := range event_type_counts {
		if v != 0 {
			t.Log("Test", i, "failed:", tests)
			t.Log("Event: ", tests[i].Type.String())
			t.Log("event_type_counts:", event_type_counts)
		}
		is.Equal(v, 0)
	}
	event_type_mutex.Unlock()
	event_listeners_wg.Wait()
}

func TestMultithreadEvents(t *testing.T) {
	is := is.New(t)
	sl := sled.New()

	threads := 8
	keys := 100000 / threads

	sub := sl.Subscribe(events.StringToType("key-set"))
	var wg sync.WaitGroup
	done := make(chan struct{})
	set_start := make([]time.Time, threads)
	set_times := make([]time.Duration, threads)
	wall_clock_start := time.Now()
	for thread := 0; thread < threads; thread++ {
		wg.Add(1)
		go func(thread int) {
			set_start[thread] = time.Now()
			for i := 0; i < keys; i++ {
				key := fmt.Sprintf("/foo/thread_%04d/%04d", thread, i)
				sl.Set(key, i)
			}
			set_times[thread] = time.Now().Sub(set_start[thread])
			wg.Done()
		}(thread)
	}
	wg.Wait()
	with_time := time.Now().Sub(wall_clock_start)

	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		<-done
		err := sub.Close()
		is.NoErr(err)
		wg2.Done()
	}()

	count := 0
	re := regexp.MustCompile(`^/foo/thread_([0-9]+)/([0-9]+)$`)
	counts := make([]int, threads)
	for e := range sub.Events() {
		is.Equal(e.Type, events.StringToType("key-set"))
		path_strings := re.FindStringSubmatch(e.Key)
		thread, err := strconv.Atoi(path_strings[1])
		is.NoErr(err)
		id, err := strconv.Atoi(path_strings[2])
		is.NoErr(err)
		counts[thread]++
		_ = thread
		_ = id
		count++
		if count == (threads * keys) {

			close(done)
		}
	}

	for i, p := range set_times {
		t.Log("w  events: ", i, p)
	}

	// benchmark without events
	sl_no_events := sled.New()
	set_start_no_events := make([]time.Time, threads)
	set_times_no_events := make([]time.Duration, threads)
	wall_clock_start = time.Now()
	for thread := 0; thread < threads; thread++ {
		wg.Add(1)
		go func(thread int) {
			set_start_no_events[thread] = time.Now()
			for i := 0; i < keys; i++ {
				key := fmt.Sprintf("/foo/thread_%04d/%04d", thread, i)
				sl_no_events.Set(key, i)
			}
			set_times_no_events[thread] = time.Now().Sub(set_start_no_events[thread])
			wg.Done()
		}(thread)
	}
	wg.Wait()
	wo_time := time.Now().Sub(wall_clock_start)

	for i, p := range set_times_no_events {
		t.Log("wo events: ", i, p)
	}

	// benchmark for locking map
	locking_map := make(map[string]interface{})
	set_start_map := make([]time.Time, threads)
	set_times_map := make([]time.Duration, threads)
	var map_locker sync.Mutex
	wall_clock_start = time.Now()
	for thread := 0; thread < threads; thread++ {
		wg.Add(1)
		go func(thread int) {
			set_start_map[thread] = time.Now()
			for i := 0; i < keys; i++ {
				key := fmt.Sprintf("/foo/thread_%04d/%04d", thread, i)
				map_locker.Lock()
				locking_map[key] = i
				map_locker.Unlock()
			}
			set_times_map[thread] = time.Now().Sub(set_start_map[thread])
			wg.Done()
		}(thread)
	}
	wg.Wait()
	map_time := time.Now().Sub(wall_clock_start)
	for i, p := range set_times_map {
		t.Log("map:", i, p)
	}

	total_events := (threads * keys)
	t.Log("Threads:", threads, "Total Keys:", keys, "Total Events: ", total_events)
	t.Log("With Events")
	t.Log("  Keys Per Second:", int(float64(keys)/with_time.Seconds()))
	t.Log("  Events Per Second:", int(float64(total_events)/with_time.Seconds()))
	t.Log("Without Events")
	t.Log("  Keys Per Second:", int(float64(keys)/wo_time.Seconds()))
	t.Log("Map[string]interface{} with Locks")
	t.Log("  Keys Per Second:", int(float64(keys)/map_time.Seconds()))
	t.Log("With Events:", with_time, "Without Events:", wo_time, "Locking Map", map_time)
}
