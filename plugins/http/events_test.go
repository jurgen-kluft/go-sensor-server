package httpplugin

import "testing"

func TestEventHubDropsForSlowSubscriberWithoutBlocking(t *testing.T) {
	hub := newEventHub()
	events, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	for index := 0; index < eventQueueCapacity+10; index++ {
		hub.Publish("test", index)
	}
	if got := len(events); got != eventQueueCapacity {
		t.Fatalf("queued events = %d, want %d", got, eventQueueCapacity)
	}
}
