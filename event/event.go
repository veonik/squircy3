package event // import "code.dopame.me/veonik/squircy3/event"

import (
	"fmt"
	"sync"
)

type Event struct {
	Name string
	Data map[string]interface{}

	handled bool
}

func (e *Event) StopPropagation() {
	e.handled = true
}

type Handler func(ev *Event)

type Dispatcher struct {
	handlers map[string][]Handler

	mu sync.RWMutex

	emitting chan *Event
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[string][]Handler), emitting: make(chan *Event, 8)}
}

func (d *Dispatcher) Loop() {
	for {
		select {
		case ev, ok := <-d.emitting:
			if !ok {
				// closed channel, return
				return
			}
			evn := ev.Name
			d.mu.RLock()
			handlers, ok := d.handlers[evn]
			handlers = append([]Handler{}, handlers...)
			d.mu.RUnlock()
			if !ok || len(handlers) == 0 {
				// nothing to do
				continue
			}
			for _, h := range handlers {
				h(ev)
				if ev.handled {
					break
				}
			}
		}
	}
}

func (d *Dispatcher) Bind(name string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[name] = append(d.handlers[name], handler)
}

func (d *Dispatcher) Unbind(name string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	hi := fmt.Sprintf("%v", handler)
	hs, ok := d.handlers[name]
	if !ok {
		return
	}
	i := -1
	for j, h := range hs {
		ohi := fmt.Sprintf("%v", h)
		if hi == ohi {
			i = j
			break
		}
	}
	if i < 0 {
		return
	}
	d.handlers[name] = append(hs[:i], hs[i+1:]...)
}

func (d *Dispatcher) UnbindAll(name string) error {
	return nil
}

func (d *Dispatcher) UnbindAllHandlers() error {
	return nil
}

func (d *Dispatcher) Emit(name string, data map[string]interface{}) {
	d.emitting <- &Event{Name: name, Data: data}
}
