package event // import "code.dopame.me/veonik/squircy3/event"

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
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
		logrus.Debugln("not unbinding anything for", name, handler, hi)
		return
	}
	i := -1
	for j, h := range hs {
		ohi := fmt.Sprintf("%v", h)
		if hi == ohi {
			i = j
			logrus.Debugln("unbinding", name, handler, hi)
			break
		}
	}
	if i < 0 {
		logrus.Debugln("not unbinding anything for", name, handler, hi)
		return
	}
	d.handlers[name] = append(hs[:i], hs[i+1:]...)
}

func (d *Dispatcher) Emit(name string, data map[string]interface{}) {
	d.emitting <- &Event{Name: name, Data: data}
}
