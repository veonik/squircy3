package event_test

import (
	"fmt"
	"testing"
	"time"

	"code.dopame.me/veonik/squircy3/event"
)

func TestDispatcher(t *testing.T) {
	expect := func(v, ex string) {
		if v != ex {
			t.Fatalf("expected %s\ngot %s", v, ex)
		}
	}

	d := event.NewDispatcher()
	go d.Loop()
	defer d.Stop()
	greetings := make(chan string, 5)
	h := event.HandlerFunc(func(ev *event.Event) {
		greetings <- fmt.Sprintf("Hello, %s!", ev.Data["name"])
	})
	d.Bind("test.event", h)
	d.Emit("test.event", map[string]interface{}{"name": "Steve"})
	d.Emit("test.event", map[string]interface{}{"name": "Jack"})
	// allow time for the event to be handled
	<-time.After(100 * time.Microsecond)
	d.Unbind("test.event", h)
	d.Emit("test.event", map[string]interface{}{"name": "Jill"})
	// allow time for the event to be handled
	<-time.After(100 * time.Microsecond)

	expect(<-greetings, "Hello, Steve!")
	expect(<-greetings, "Hello, Jack!")
	select {
	case v := <-greetings:
		t.Fatalf("unexpected greeting %s", v)
	default:
		// no more greetings
	}
}

func TestEvent_StopPropagation(t *testing.T) {
	// create an unbuffered dispatcher so that only one event may
	// be emitted at a time.
	d := event.NewDispatcherLimit(0)
	go d.Loop()
	defer d.Stop()
	first := false
	second := false
	third := false
	d.Bind("test.event", event.HandlerFunc(func(ev *event.Event) {
		// should fire
		first = true
	}))
	d.Bind("test.event", event.HandlerFunc(func(ev *event.Event) {
		// should fire and stop further firing
		ev.StopPropagation()
		second = true
	}))
	d.Bind("test.event", event.HandlerFunc(func(ev *event.Event) {
		// should not fire
		third = true
	}))
	d.Emit("test.event", nil)
	// since only one event can be emitted at a time, this will block
	// until the "test.event" event is finished being emitted.
	d.Emit("unknown", nil)
	if !first {
		t.Fatal("expected first event to fire")
	}
	if !second {
		t.Fatal("expected second event to fire")
	}
	if third {
		t.Fatal("did not expect third event to fire")
	}
}
