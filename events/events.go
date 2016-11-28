package events

import (
	"sync"
	"time"
)

var (
	initialized bool
	type_index  Type
	to_type     map[string]Type
	from_type   []string
)

type Type int

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

func (t Type) String() string {
	return TypeToString(t)
}

func AddType(key string) Type {
	if !initialized {
		type_index = 0
		to_type = make(map[string]Type)
		initialized = true
	}
	from_type = append(from_type, key)
	to_type[key] = type_index
	i := type_index
	type_index++
	return i
}

func StringToType(key string) Type {
	return to_type[key]
}

func TypeToString(t Type) string {
	return from_type[int(t)]
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
