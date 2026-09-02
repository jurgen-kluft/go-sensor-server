package httpplugin

import (
	"encoding/json"
	"sync"
)

const eventQueueCapacity = 32

type Event struct {
	Name string
	Data json.RawMessage
}

type eventHub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]chan Event
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[uint64]chan Event)}
}

func (hub *eventHub) Subscribe() (<-chan Event, func()) {
	hub.mu.Lock()
	hub.nextID++
	id := hub.nextID
	events := make(chan Event, eventQueueCapacity)
	hub.subscribers[id] = events
	hub.mu.Unlock()

	var once sync.Once
	return events, func() {
		once.Do(func() {
			hub.mu.Lock()
			delete(hub.subscribers, id)
			hub.mu.Unlock()
		})
	}
}

func (hub *eventHub) Publish(name string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	event := Event{Name: name, Data: data}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for _, events := range hub.subscribers {
		select {
		case events <- event:
		default:
		}
	}
}
