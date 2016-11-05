package sled

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

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

type EventController struct {
	wg                    sync.WaitGroup
	indexLock             sync.Mutex
	createEventSignalLock sync.Mutex
	createEventSignaler   *sync.Cond
	// event_channels []chan *Event
	// event_chan chan *Event
	// events		[][]chan *Event
}

type EventHandler func(event *Event)

func NewEventController() *EventController {
	// Create channels for every event type
	// channels := make([]chan *Event, 1)
	// channels[0] = make(chan *Event)
	// return &EventController{event_channels: channels}
	ec := &EventController{}
	ec.createEventSignaler = sync.NewCond(&ec.createEventSignalLock)
	return ec
}

func (t EventType) String() string {
	return strconv.Itoa(int(t))
}

func (e *EventController) Lock() {
	e.indexLock.Lock()
}

func (e *EventController) Unlock() {
	e.indexLock.Unlock()
}

func (e *EventController) LockCreateEventSignaler() {
	e.createEventSignalLock.Lock()
}

func (e *EventController) UnlockCreateEventSignaler() {
	e.createEventSignalLock.Unlock()
}

func (e *EventController) WaitCreateEventSignaler() {
	e.createEventSignaler.Wait()
}

func (e *EventController) HandleEvents() {
	// start event handler thread if not started
	// this is the only event channel reader, signal
	// on the channel is written to all the registered handlers

}

func (e *EventController) Wait() {
	e.wg.Wait()
}

func (s *Sled) On(t EventType, handler EventHandler) {

	go func() {
		for {
			event_key := "/events/" + string(t) + "/latest"
			var event *Event
			valid := false
			s.events.LockCreateEventSignaler()
			for !valid {
				s.events.WaitCreateEventSignaler()
				switch v := s.Get(event_key); v.(type) {
				case nil:
				case *Event:
					event = v.(*Event)
					valid = true
				}
			}
			fmt.Println(event)
			handler(event)
			s.events.UnlockCreateEventSignaler()
		}
	}()

	// e := s.events
	// index_key := "/events/" + string(t) + "/index"
	// handler_index := int(0)
	// e.Lock()
	// switch v := s.Get(index_key); v.(type) {
	// case nil:
	// 	s.Set(index_key, 0)
	// case int:
	// 	handler_index = v.(int)
	// }
	// handler_index++
	// s.Set(index_key, handler_index)
	// e.Unlock()
	// handler_key := "/events/" + string(t) + "/handlers/" + strconv.Itoa(handler_index)
	// s.Set(handler_key, handler)
	// e.HandleEvents()

	// s.event_chan = make(chan *Event)
	// s.events_wg.Add(1)
	// go func() {
	// 	for e := range s.event_chan {
	// 		if e.Type == t {
	// 			handler(e)
	// 		}
	// 	}
	// 	s.events_wg.Done()
	// }()
}

func (e *EventController) Send() {
	e.createEventSignaler.Broadcast()
}

func (s *Sled) SendEvent(t EventType, key string) {
	now := time.Now()
	e := Event{t, key, &now}
	event_key := "/events/" + string(t) + "/latest"
	// s.events.LockCreateEventSignaler()
	s.ct.Insert([]byte(event_key), &e)
	// s.events.UnlockCreateEventSignaler()
	s.events.Send()
	// s.events.event_channels[t] <- &ev
}
