package event // import "code.dopame.me/veonik/squircy3/event"

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// An Event is used to pass data between an emitter and handler.
type Event struct {
	Name string
	Data map[string]interface{}

	handled bool
}

func (e *Event) StopPropagation() {
	e.handled = true
}

// A Handler is a uniquely identifiable function for handling an emitted event.
type Handler interface {
	ID() string
	Handle(ev *Event)
}

type handlerFunc struct {
	h func(ev *Event)
}

// HandlerFunc returns a valid Handler using the given function.
func HandlerFunc(h func(ev *Event)) Handler {
	return &handlerFunc{h}
}

func (h *handlerFunc) ID() string {
	return fmt.Sprintf("%p", h)
}

func (h *handlerFunc) Handle(ev *Event) {
	h.h(ev)
}

// Dispatcher binds functions to be called indirectly by unrelated code.
type Dispatcher struct {
	handlersIndex map[string]map[string]struct{}
	handlers      map[string][]Handler

	mu sync.RWMutex

	emitting chan *Event
	quit     chan struct{}
}

// NewDispatcher returns an event dispatcher, ready for use.
func NewDispatcher() *Dispatcher {
	return NewDispatcherLimit(1024)
}

// NewDispatcherLimit returns an event dispatcher with a buffer size of limit.
func NewDispatcherLimit(limit int) *Dispatcher {
	return &Dispatcher{
		handlersIndex: make(map[string]map[string]struct{}),
		handlers:      make(map[string][]Handler),
		emitting:      make(chan *Event, limit),
		quit:          make(chan struct{}),
	}
}

// Stop signals to the dispatcher to stop emitting events.
//
// All workers started with Loop will stop processing without draining the
// pending events.
func (d *Dispatcher) Stop() {
	d.mu.RLock()
	defer d.mu.RUnlock()
	select {
	case <-d.quit:
		return
	default:
		// not already closed
	}
	close(d.quit)
}

// Loop emits events in an infinite loop until the dispatcher is stopped.
//
// If the Dispatcher is not running when Loop is called, it will be started.
// This method should be called in a separate goroutine. More than one worker
// can be started by calling this method multiple times in separate goroutines.
func (d *Dispatcher) Loop() {
	d.mu.Lock()
	select {
	case <-d.quit:
		// closed, need to recreate
		d.quit = make(chan struct{})
	default:
		// already started
	}
	// avoid data race by reading this inside the lock
	quit := d.quit
	// "emitting" is not necessary to protect in this way as nothing
	// ever writes to the "emitting" field.
	d.mu.Unlock()
	for {
		select {
		case <-quit:
			// quit signal received, stop asap
			return

		case ev := <-d.emitting:
			for _, h := range d.handlersForEvent(ev.Name) {
				h.Handle(ev)
				if ev.handled {
					break
				}
			}
		}
	}
}

// handlersForEvent returns a copy of the handlersIndex for the given event.
func (d *Dispatcher) handlersForEvent(name string) []Handler {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Handler(nil), d.handlers[name]...)
}

// Emit will call bound handlersIndex for the given in event.
//
// This method does not block unless the underlying channel becomes full.
// The map received by this method is not copied; avoid writing to it once
// it has been passed into this method.
func (d *Dispatcher) Emit(name string, data map[string]interface{}) {
	d.emitting <- &Event{Name: name, Data: data}
}

// Bind adds the given handler to the list of handlersIndex for the event.
func (d *Dispatcher) Bind(name string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := handler.ID()
	if _, ok := d.handlersIndex[name]; !ok {
		d.handlersIndex[name] = make(map[string]struct{})
	}
	if _, ok := d.handlersIndex[name][id]; ok {
		logrus.Warnln("rebinding handler for", name, id)
	} else {
		logrus.Debugln("binding handler for", name, id)
	}
	d.handlersIndex[name][id] = struct{}{}
	d.handlers[name] = append(d.handlers[name], handler)
}

// Unbind removes the given handler from the list of handlersIndex for the event.
func (d *Dispatcher) Unbind(name string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := handler.ID()
	if _, ok := d.handlersIndex[name]; !ok {
		logrus.Debugln("not unbinding anything for", name, id)
		return
	}
	if _, ok := d.handlersIndex[name][id]; !ok {
		logrus.Debugln("not unbinding anything for", name, id)
		return
	}
	logrus.Debugln("unbinding handler for", name, id)
	delete(d.handlersIndex[name], id)
	hi := -1
	for i, h := range d.handlers[name] {
		if h.ID() == id {
			hi = i
		}
	}
	if hi < 0 {
		return
	}
	d.handlers[name] = append(d.handlers[name][:hi], d.handlers[name][hi+1:]...)
}
