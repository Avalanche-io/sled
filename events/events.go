package events

import (
	"strconv"
	"sync"
	"time"
)

type Type int

func (t Type) String() string {
	return strconv.Itoa(int(t))
}

type Event struct {
	Id    int
	Type  Type
	Key   string
	Value interface{}
	Time  *time.Time
}

func New(id int, t Type, key string, value interface{}, event_time time.Time) *Event {
	e := Event{
		Id:    id,
		Type:  t,
		Key:   key,
		Value: value,
		Time:  &event_time,
	}
	return &e
}

type Subscription interface {
	Events() <-chan *Event
	Close() error
}

type Fetcher interface {
	Fetch() (events []*Event, err error)
}

var initialized bool
var type_index Type
var type_map map[string]Type

func AddType(key string) Type {
	if !initialized {
		type_index = 0
		type_map = make(map[string]Type)
		initialized = true
	}
	type_map[key] = type_index
	i := type_index
	type_index = type_index + 1
	return i
}

func StringToType(key string) Type {
	return type_map[key]
}

type sub struct {
	fetcher Fetcher
	events  chan *Event
	closing chan chan error
}

func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		events:  make(chan *Event),
		closing: make(chan chan error),
	}
	go s.loop()
	return s
}

func (s *sub) loop() {
	// mutable state
	var err error
	var pending []*Event // appended by fetch; consumed by send
	var next time.Time
	i := 0
	for {
		// channels for cases
		i++
		var first *Event
		var events chan *Event // is nil by default
		if len(pending) > 0 {
			first = pending[0]
			events = s.events // enables send by setting events not non nil value
		}
		var fetchDelay time.Duration
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)
		select {
		case <-startFetch:
			var fetched []*Event
			fetched, err = s.fetcher.Fetch()
			next = time.Now().Add(1 * time.Millisecond)
			if err != nil {
				break
			}
			pending = append(pending, fetched...)
		case events <- first:
			pending = pending[1:]
		case errc := <-s.closing:
			errc <- err
			close(s.events)
			return
		}
	}
}

func (s *sub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}

func (s *sub) Events() <-chan *Event {
	return s.events
}

type mergedsub struct {
	events  chan *Event
	closing chan chan error
}

func (s *mergedsub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}

func (s *mergedsub) Events() <-chan *Event {
	return s.events
}

func Merge(subs ...Subscription) Subscription {
	s := &mergedsub{
		events:  make(chan *Event),
		closing: make(chan chan error),
	}
	var wg sync.WaitGroup

	output := func(c Subscription) {
		for n := range c.Events() {
			s.events <- n
		}
		wg.Done()
	}

	wg.Add(len(subs))
	for _, c := range subs {
		go output(c)
	}
	go func() {
		errc := <-s.closing
		var err error
		for _, c := range subs {
			if e := c.Close(); e != nil {
				err = e
			}
		}
		errc <- err
		close(s.events)
	}()

	return s
}
