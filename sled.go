package sled

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Avalanche-io/sled/events"
	"github.com/Workiva/go-datastructures/trie/ctrie"
	"github.com/boltdb/bolt"
)

type Sled struct {
	ct                *ctrie.Ctrie
	db                *bolt.DB
	close_wg          *sync.WaitGroup
	loading           chan struct{}
	event_index_lock  []sync.Mutex
	event_subscribers int
	event_logs        [][]*events.Event
}

type KV interface {
	Set(string, interface{})
	Get(key string) interface{}
	Iterator(<-chan struct{}) <-chan Element
	Snapshot() *Sled
	SetNil(string, interface{}) bool
	Delete(string) (interface{}, bool)
}

type Element interface {
	Key() string
	Value() interface{}
}

// A Sledder is a data structure that can can represent itself as key value
// pairs for storage in a Sled structure.
type Sledder interface {
	Keys() []string
	WriteToSled(prefix string, sled *Sled) error
	ReadFromSled(prefix string, sled *Sled) error
}

var (
	KeyCreatedEvent events.Type
	KeyChangedEvent events.Type
	KeyRemovedEvent events.Type
	KeySetEvent     events.Type
	EventTypeCount  int
)

func New() *Sled {
	ct := ctrie.New(nil)
	event_keys(ct)
	wg := sync.WaitGroup{}
	locker := make([]sync.Mutex, EventTypeCount)
	event_logs := make([][]*events.Event, EventTypeCount)
	s := Sled{ct, nil, &wg, nil, locker, 0, event_logs}
	return &s
}

func Open(path string) (*Sled, error) {
	s := New()
	err := s.Open(path)
	if err != nil {
		return nil, err
	}
	s.createBuckets()
	s.loadAllKeys()
	return s, err
}

func (s *Sled) Close() error {
	s.Wait()
	return s.db.Close()
}

func (s *Sled) Wait() {
	if s.close_wg != nil {
		s.close_wg.Wait()
	}
}

func init() {
	KeyCreatedEvent = events.AddType("key-created")
	KeyChangedEvent = events.AddType("key-changed")
	KeyRemovedEvent = events.AddType("key-removed")
	KeySetEvent = events.AddType("key-set")
	EventTypeCount = 4
}

func event_keys(s *ctrie.Ctrie) {
	// s.Insert([]byte(".events/"+string(KeyCreatedEvent)+"/next_id"), 0)
	// s.Insert([]byte(".events/"+string(KeyChangedEvent)+"/next_id"), 0)
	// s.Insert([]byte(".events/"+string(KeyRemovedEvent)+"/next_id"), 0)
	// s.Insert([]byte(".events/"+string(KeySetEvent)+"/next_id"), 0)

}

func event_index_key(t events.Type) []byte {
	return []byte(".events/" + string(t) + "/next_id")
}

func event_key(t events.Type, id int) []byte {
	return []byte(fmt.Sprintf(".events/%s/%d", t, id))
}

func (s *Sled) event_id(t events.Type) int {
	index_key := event_index_key(t)
	v, _ := s.ct.Lookup(index_key)
	id := v.(int)
	s.ct.Insert(index_key, id+1)
	return id
}

func (s *Sled) get_events_for(t events.Type, start_id int) (event_slice []*events.Event, end_id int) {
	s.event_index_lock[t].Lock()
	event_slice = s.event_logs[t][start_id:]
	end_id = len(s.event_logs[t])
	s.event_index_lock[t].Unlock()
	// v, _ := s.ct.Lookup(event_index_key(t))
	// end_id = v.(int)
	// for i := start_id; i < end_id; i++ {
	// 	s.event_index_lock[t].Lock()
	// 	e, _ := s.ct.Lookup(event_key(t, i))
	// 	s.event_index_lock[t].Unlock()

	// 	event_slice = append(event_slice, e.(*events.Event))
	// }
	return
}

type event_subscription struct {
	sled       *Sled
	event_type events.Type
	last_id    int
	event_chan chan *events.Event
	closing    chan chan error
	args       []string
}

func (s *event_subscription) Events() <-chan *events.Event {
	return s.event_chan
}

func (s *event_subscription) Close() error {
	errc := make(chan error)
	s.closing <- errc
	err := <-errc
	s.sled.event_subscribers--
	return err
}

func (s *event_subscription) loop() {
	var err error
	var pending []*events.Event // appended by fetch; consumed by send
	var next time.Time
	i := 0
	for {
		// channels for cases
		i++
		var first *events.Event
		var events chan *events.Event // is nil by default
		if len(pending) > 0 {
			first = pending[0]
			events = s.event_chan // enables send by setting events not non nil value
		}
		var fetchDelay time.Duration
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)
		select {
		case <-startFetch:
			// var fetched []*events.Event
			fetched, end_id := s.sled.get_events_for(s.event_type, s.last_id)
			next = time.Now().Add(1 * time.Millisecond)
			if err != nil {
				break
			}
			pending = append(pending, fetched...)
			s.last_id = end_id
		case events <- first:
			pending = pending[1:]
		case errc := <-s.closing:
			errc <- err
			close(s.event_chan)
			return
		}
	}
}

func (s *Sled) LogEvent(t events.Type, key string, value interface{}) {
	go func() {
		now := time.Now().UTC()
		// TODO: remove index locking
		s.event_index_lock[t].Lock()
		id := len(s.event_logs[t])
		// id := s.event_id(t)
		e := events.New(id, t, key, value, now) // Event{id, t, key, value, &now}
		// event_key := fmt.Sprintf(".events/%s/%d", t, id)
		// s.ct.Insert([]byte(event_key), &e)
		s.event_logs[t] = append(s.event_logs[t], e)
		s.event_index_lock[t].Unlock()
	}()
}

func (s *Sled) Subscribe(t events.Type, args ...string) events.Subscription {
	sub := &event_subscription{
		sled:       s,
		event_type: t,
		last_id:    0,
		event_chan: make(chan *events.Event),
		closing:    make(chan chan error),
		args:       args,
	}
	s.event_subscribers++
	go sub.loop()
	return sub
}

// Delete removes a key and value, and returns it's previous value with
// an existed flat that will be true if the key was not empty.
func (s *Sled) Delete(key string) (value interface{}, existed bool) {
	value, existed = s.ct.Remove([]byte(key))
	if s.event_subscribers > 0 {
		if existed {
			s.LogEvent(KeyRemovedEvent, key, value)
		}
	}
	return
}

// SetNil is exclusive Set.  It only assigns the value to key,
// if the key is not set already.  It returns true if the key was
// empty and the value assigned, otherwise false.
func (s *Sled) SetNil(key string, value interface{}) bool {
	_, existed := s.ct.Lookup([]byte(key))
	if existed {
		return false
	}
	s.ct.Insert([]byte(key), value)
	if s.event_subscribers > 0 {
		s.LogEvent(KeyCreatedEvent, key, value)
		s.LogEvent(KeySetEvent, key, value)
	}
	if s.db != nil {
		value_json, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		err = s.put_db(s.db, "assets", []byte(key), []byte(value_json))
		if err != nil {
			panic(err)
		}
	}
	return true
}

// Set stores value in key.
func (s *Sled) Set(key string, value interface{}) {
	var old_value interface{}
	existed := false
	if s.event_subscribers > 0 {
		old_value, existed = s.ct.Lookup([]byte(key))
		s.ct.Insert([]byte(key), value)
		if !existed {
			s.LogEvent(KeyCreatedEvent, key, value)
		} else {
			s.LogEvent(KeyChangedEvent, key, old_value)
		}
		s.LogEvent(KeySetEvent, key, value)
	} else {
		s.ct.Insert([]byte(key), value)
	}

	if s.db != nil {
		value_json, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		err = s.put_db(s.db, "assets", []byte(key), []byte(value_json))
		if err != nil {
			panic(err)
		}
	}
}

// Snapshot returns a single point in time image of the Sled.
// Snapshot is fast and non blocking.
func (s *Sled) Snapshot() *Sled {
	ct := s.ct.Snapshot()
	event_keys(ct)
	locker := make([]sync.Mutex, EventTypeCount)
	event_logs := make([][]*events.Event, EventTypeCount)
	sl := Sled{ct, nil, s.close_wg, nil, locker, 0, event_logs}
	return &sl
}

// Get return the value stored for the given key, or nil if no value was found.
func (s *Sled) Get(key string) interface{} {
	val, ok := s.ct.Lookup([]byte(key))
	if ok {
		return val
	}
	return nil
}

type ele struct {
	k string
	v interface{}
}

func (e *ele) Key() string {
	return e.k
}

func (e *ele) Value() interface{} {
	return e.v
}

func (s *Sled) Iterator(cancel <-chan struct{}) <-chan Element {
	out := make(chan Element)
	c := make(chan struct{})
	go func() {
		defer close(out)
		for e := range s.ct.Iterator(c) {
			entry := ele{
				string(e.Key),
				e.Value,
			}
			select {
			case out <- &entry:
			case <-cancel:
				close(c)
			}
		}

	}()
	return out
}
