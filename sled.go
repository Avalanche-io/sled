package sled

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/Avalanche-io/sled/config"
	"github.com/Avalanche-io/sled/events"
	"github.com/Avalanche-io/sled/storage"
	"github.com/Workiva/go-datastructures/trie/ctrie"
	"github.com/boltdb/bolt"
	"github.com/etcenter/c4/asset"
)

// Sledder is an interface to a KV store with snapshots, iterators, and
// exclusive Set key functions.
type KV interface {
	Set(string, interface{}) error
	Get(key string) (interface{}, error)
	GetID(key string) *asset.ID
	Iterator(<-chan struct{}) <-chan Element
	Snapshot() *Sled
	SetNil(string, interface{}) bool
	Delete(string) (interface{}, bool)
}

// Sled holds pointers to the configuration, database, and the ctrie
// data structure.  It has no exported data structures.
type Sled struct {
	cfg               *config.Config
	ct                *ctrie.Ctrie
	db                *bolt.DB
	st                storage.IO
	close_wg          *sync.WaitGroup
	loading           chan struct{}
	event_index_lock  []sync.Mutex
	event_subscribers int
	event_logs        [][]*events.Event
}

// A key/value pair interface, for use in range operations and signals.
type Element interface {
	Key() string
	Value() interface{}
}

// A Sledder is an interface to a data structure that can represent
// itself as key value pairs. (wip)
type Sledder interface {
	Keys() []string
	WriteToSled(prefix string, sled *Sled) error
	ReadFromSled(prefix string, sled *Sled) error
}

// Create a new Sled object with optional custom configuration.
func New(configs ...*config.Config) *Sled {
	cfg := config.DefaultConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}
	cfg.Mkdirs()

	ct := ctrie.New(nil)
	event_keys(ct)
	wg := sync.WaitGroup{}
	locker := make([]sync.Mutex, EventTypeCount)
	event_logs := make([][]*events.Event, EventTypeCount)
	st := storage.New(cfg)
	s := Sled{cfg, ct, nil, st, &wg, nil, locker, 0, event_logs}
	if s.cfg.DB != nil {
		err := s.Open(*s.cfg.DbPath())
		if err != nil {
			panic(err)
		}
	}
	return &s
}

// Assigns value to key, replacing any previous values.
func (s *Sled) Set(key string, value interface{}) error {
	var old_value interface{}
	var id *asset.ID
	var err error
	existed := false
	size := reflect.TypeOf(value).Size()
	// fmt.Printf("Set size: %d\n", size)
	save_to_file := size > 64
	save_to_db := s.db != nil
	send_events := s.event_subscribers > 0

	if send_events {
		old_value, existed = s.ct.Lookup([]byte(key))
	}
	// fmt.Printf("Set type: %T\n", value)
	// switch value.(type) {
	// case []byte:
	// 	fmt.Printf("type is byte!! len = %d\n", len(value.([]byte)))
	// 	data = value.([]byte)
	// case string:
	// 	data = []byte(value.(string))
	// default:
	// 	data, err = json.Marshal(value)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	// if len(data) <= 64 {
	// 	fmt.Printf("Set data: %s\n", string(data))
	// } else {
	// 	fmt.Printf("Set data: %d\n", len(data))
	// }

	// id, err = asset.Identify(bytes.NewReader(data))
	// if err != nil {
	// 	return err
	// }
	// fmt.Printf("Set id: %s\n", id.String())

	id, err = s.st.Save(value)
	if err != nil {
		return err
	}
	// fmt.Printf("Set Key/ID: %s, %s\n", key, id)
	// fmt.Printf("Set Value: %T, %v\n", value, value)

	if save_to_db {
		err = s.put_db("assets", key, id)
		if err != nil {
			return SledError(err.Error())
		}
	}

	if save_to_file {
		s.ct.Insert([]byte(key), *id)
	} else {
		s.ct.Insert([]byte(key), value)
	}

	if send_events {
		if !existed {
			s.LogEvent(KeyCreatedEvent, key, value)
		} else {
			s.LogEvent(KeyChangedEvent, key, old_value)
		}
		s.LogEvent(KeySetEvent, key, value)
	}
	return nil
}

// Get return the value stored for the given key, or nil if no value was found.
func (s *Sled) Get(key string) (interface{}, error) {
	var val interface{}
	var ok bool
	val, ok = s.ct.Lookup([]byte(key))
	if val == nil {
		id, err := s.get_db("assets", key)
		if err != nil {
			return nil, err
		}
		size, err := s.st.SizeOf(id)
		if err != nil {
			return nil, err
		}
		_ = size
		// fmt.Println("Get key: %s, %d\n", id, size)
		return s.st.Load(id)
	}

	if ok {
		switch val.(type) {
		case nil:
			return nil, nil
		case asset.ID:
			id := val.(asset.ID)
			// fmt.Println("\n+++++++++++++\n")
			// fmt.Printf("type: %T\n", val)
			// fmt.Println("")
			// fmt.Printf("id.String(): %s\n", (&id).String())
			// fmt.Println("")
			// fmt.Print(id)
			// fmt.Println("\n+++++++++++++\n")
			// fmt.Printf("Get (is asset.ID): len")
			return s.st.Load(&id)
		default:
			return val, nil
		}
	}
	return nil, nil
}

// Snapshot returns a single point in time image of the Sled.
// Snapshot is fast and non blocking.
func (s *Sled) Snapshot() *Sled {
	ct := s.ct.Snapshot()
	event_keys(ct)
	locker := make([]sync.Mutex, EventTypeCount)
	event_logs := make([][]*events.Event, EventTypeCount)

	sl := Sled{s.cfg, ct, nil, s.st, s.close_wg, nil, locker, 0, event_logs}
	return &sl
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

// SetNil is exclusive Set.  It only assigns the value to the key,
// if the key is not already set.  It returns true if the assignment succeed.
func (s *Sled) SetNil(key string, value interface{}) bool {
	if _, existed := s.ct.Lookup([]byte(key)); !existed {
		s.Set(key, value)
	}
	return false
}

func (s *Sled) GetID(key string) *asset.ID {
	val, _ := s.ct.Lookup([]byte(key))
	switch val.(type) {
	case asset.ID:
		id := val.(asset.ID)
		fmt.Printf("GetID: len(val) == %s\n", (&id).String())
		return &id
	default:
		id, err := s.st.Save(val)
		if err != nil {
			panic(err)
		}
		return id
		// panic("Get ID, Unhanded type.")
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
