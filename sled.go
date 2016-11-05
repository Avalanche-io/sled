package sled

import (
	"encoding/json"
	"sync"

	"github.com/Workiva/go-datastructures/trie/ctrie"
	"github.com/boltdb/bolt"
)

type Sled struct {
	ct       *ctrie.Ctrie
	db       *bolt.DB
	close_wg *sync.WaitGroup
	loading  chan struct{}
	events   *EventController
}

func New() *Sled {
	ct := ctrie.New(nil)
	wg := sync.WaitGroup{}
	ec := NewEventController()
	s := Sled{ct, nil, &wg, nil, ec}
	return &s
}

func Open(path string) (*Sled, error) {
	s := New()
	err := s.Open(path)
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
	s.events.Wait()
}

func (s *Sled) Set(key string, value interface{}) {
	s.ct.Insert([]byte(key), value)
	s.SendEvent(CreateEvent, key)
	if s.db != nil {
		s.close_wg.Add(1)
		go func() {
			defer s.close_wg.Done()
			value_json, err := json.Marshal(value)
			if err != nil {
				panic(err)
			}
			err = s.put_db(s.db, "assets", []byte(key), []byte(value_json))
			if err != nil {
				panic(err)
			}
		}()
	}
}

func (s *Sled) Get(key string) interface{} {
	if s.loading != nil {
		<-s.loading
	}
	val, ok := s.ct.Lookup([]byte(key))
	if ok {
		return val
	}
	return nil

}
