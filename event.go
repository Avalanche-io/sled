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
	// wg        sync.WaitGroup
	indexLock sync.Mutex
	eventwait sync.WaitGroup
	lockers   []*sync.Mutex
	signalers []*sync.Cond
	// createEventSignaler   *sync.Cond
	// event_channels []chan *Event
	// event_chan chan *Event
	// events		[][]chan *Event
}

type EventHandler func(event *Event)

// sync.Cond needs a Locker interface, but we don't *grin*
type nilLocker struct{}

func (nl *nilLocker) Lock() {
}

func (nl *nilLocker) Unlock() {
}

func NewEventController() *EventController {
	ec := &EventController{}
	lockers := make([]*sync.Mutex, 1)
	signalers := make([]*sync.Cond, 1)
	lockers[CreateEvent] = &sync.Mutex{}
	// signalers[CreateEvent] = sync.NewCond(lockers[CreateEvent])
	signalers[CreateEvent] = sync.NewCond(&nilLocker{})
	ec.lockers = lockers
	ec.signalers = signalers
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

func (e *EventController) LockEvents(t EventType) {
	e.lockers[t].Lock()
}

func (e *EventController) UnlockEvents(t EventType) {
	e.lockers[t].Unlock()
}

func (e *EventController) WaitSignaler(t EventType) {
	e.signalers[t].Wait()
}

func (e *EventController) Wait() {
	e.eventwait.Wait()
}

func (e *EventController) WGadd(delta int) {
	e.eventwait.Add(delta)
}

func (e *EventController) WGdone() {
	e.eventwait.Done()
}

func (s *Sled) EventReady(key string) (bool, *Event) {
	switch v := s.Get(key); v.(type) {
	case nil:
		return false, nil
	case *Event:
		return true, v.(*Event)
	default:
		return false, nil
	}
	return false, nil
}

func (s *Sled) On(t EventType, name string, handler EventHandler) {
	ready := make(chan struct{})
	go func() {
		for {
			// var event *Event
			// s.events.LockEvents(t)
			if ready != nil {
				close(ready)
				ready = nil
			}
			// s.events.WGadd(1)
			fmt.Println("Waiting", name)
			s.events.WaitSignaler(t)
			event := s.Get("/events/" + string(t) + "/latest").(*Event)
			// s.events.UnlockEvents(t)
			fmt.Println("Running", name)
			handler(event)
			fmt.Println("Finished", name)

			// s.events.WGdone()

			// for valid, ev := s.EventReady(event_key); !valid; {
			// 	if ready != nil {
			// 		close(ready)
			// 		ready = nil
			// 	}
			// 	fmt.Println("Waiting", name)
			// 	s.events.WaitSignaler(t)
			// 	event = ev
			// }

		}
	}()
	// don't return until the event listener is ready
	select {
	case <-ready:
	}
}

func (e *EventController) Trigger(t EventType) {
	e.signalers[t].Broadcast()
}

var count int

func init() {
	count = 0
}

func (s *Sled) SendEvent(t EventType, key string) {
	count++
	fmt.Println("SendEvent", count)
	now := time.Now()
	e := Event{t, key, &now}
	event_key := "/events/" + string(t) + "/latest"
	// Insert without an event, so can't use s.Set()
	// s.events.LockEvents(t)
	s.ct.Insert([]byte(event_key), &e)
	// s.events.UnlockEvents(t)
	s.events.Trigger(t)
}
