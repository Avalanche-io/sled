package sled

import "time"

// type Sled struct {
//   ct       *ctrie.Ctrie
//   db       *bolt.DB
//   close_wg *sync.WaitGroup
//   loading  chan struct{}
// }

type EventType int

const (
	CreateEvent EventType = iota
)

type Event struct {
	Type EventType
	Key  string
	Time *time.Time
}

type EventHandler func(event *Event)

func (s *Sled) On(t EventType, handler EventHandler) {
	s.event_chan = make(chan *Event)
	s.events_wg.Add(1)
	go func() {
		for e := range s.event_chan {
			if e.Type == t {
				handler(e)
			}
		}
		s.events_wg.Done()
	}()
}

func (s *Sled) send_event(t EventType, key string) {
	now := time.Now()
	if s.event_chan != nil {
		go func() {
			e := Event{t, key, &now}
			s.event_chan <- &e
		}()
	}
}
