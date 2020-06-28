package vm

import (
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// runtime is a wrapper for goja that intends to increase concurrency safety.
type runtime struct {
	inner *goja.Runtime

	mu sync.Mutex
}

func (r *runtime) do(fn func(*goja.Runtime)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn(r.inner)
}

type job func(*goja.Runtime)

type deferredJob struct {
	fn     goja.Callable
	args   []goja.Value
	repeat bool

	cancelled chan struct{}
	sdone     chan struct{}
}

// scheduler handles the javascript event loop and evaluating javascript code.
type scheduler struct {
	runtime  *runtime
	registry *Registry

	jobs    chan job
	done    chan struct{}
	running bool
	mu      sync.Mutex

	initHandlers []func(r *goja.Runtime)
}

func newScheduler(registry *Registry) *scheduler {
	s := &scheduler{
		runtime:  nil,
		registry: registry,
		jobs:     make(chan job, 256),
	}
	return s
}

func (s *scheduler) initRuntime() error {
	sh := []func(*goja.Runtime){
		s.registry.Enable,
		func(r *goja.Runtime) {
			console := r.NewObject()
			err := console.Set("log", func(call goja.FunctionCall) goja.Value {
				var vals []interface{}
				for _, v := range call.Arguments {
					vals = append(vals, v.Export())
				}
				logrus.Infoln(vals...)
				return goja.Undefined()
			})
			if err != nil {
				panic(err)
			}
			r.Set("console", console)
		},
		func(r *goja.Runtime) {
			r.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
				return s.deferred(call, false)
			})
			r.Set("setInterval", func(call goja.FunctionCall) goja.Value {
				return s.deferred(call, true)
			})
			r.Set("setImmediate", func(call goja.FunctionCall) goja.Value {
				args := call.Arguments[1:]
				call.Arguments = append([]goja.Value{call.Arguments[0], r.ToValue(1 * time.Microsecond)}, args...)
				return s.deferred(call, false)
			})
			r.Set("clearTimeout", cancelDeferredJob)
			r.Set("clearInterval", cancelDeferredJob)
		}}
	s.mu.Lock()
	sh = append(sh, s.initHandlers...)
	s.mu.Unlock()
	for _, h := range sh {
		h(s.runtime.inner)
	}
	return nil
}

func (s *scheduler) onRuntimeInit(h ...func(r *goja.Runtime)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initHandlers = append(s.initHandlers, h...)
}

func (s *scheduler) prependRuntimeInit(h ...func(r *goja.Runtime)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initHandlers = append(h, s.initHandlers...)
}

func (s *scheduler) worker() {
	for {
		s.mu.Lock()
		done := s.done
		s.mu.Unlock()
		select {
		case <-done:
			return
		case j := <-s.jobs:
			s.runtime.do(j)
		}
	}
}

func (s *scheduler) run(j job) {
	s.jobs <- j
}

func (s *scheduler) interrupt(v interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.runtime.inner.Interrupt(v)
}

func (s *scheduler) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return errors.New("already started")
	}
	s.done = make(chan struct{})
	s.runtime = &runtime{inner: goja.New()}
	s.running = true
	s.run(func(r *goja.Runtime) {
		err := s.initRuntime()
		if err != nil {
			logrus.Warnln("error initializing runtime", err)
		}
	})
	go s.worker()
	return nil
}

func (s *scheduler) stop() error {
	if !s.running {
		return errors.New("not started")
	}
	stop := func(gr *goja.Runtime) {
		// actually change the state inside this job
		// after this is executed, no further jobs will run
		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = false
		close(s.done)
	}
	s.run(stop)
	select {
	case <-time.After(500 * time.Millisecond):
		// soft timeout, try emptying the jobs queue and interrupting execution
		logrus.Warnln("vm soft time out expired, flushing remaining jobs without done them")
		s.drain()
		s.runtime.inner.Interrupt("vm is shutting down")
		// requeue the stop job since we just flushed it down the drain
		s.run(stop)

	case <-s.done:
		return nil
	}
	select {
	case <-time.After(time.Second):
		// hard time out, give up
		return errors.New("timed out waiting to stop")
	case <-s.done:
		return nil
	}
}

// drain empties the jobs channel.
func (s *scheduler) drain() {
	for len(s.jobs) > 0 {
		<-s.jobs
	}
}

// deferred defers a function invocation.
func (s *scheduler) deferred(call goja.FunctionCall, repeating bool) goja.Value {
	if fn, ok := goja.AssertFunction(call.Argument(0)); ok {
		delay := call.Argument(1).ToInteger()
		var args []goja.Value
		if len(call.Arguments) > 2 {
			args = call.Arguments[2:]
		}
		return s.runtime.inner.ToValue(newDeferred(s, fn, time.Duration(delay)*time.Millisecond, repeating, args...))
	}
	panic(s.runtime.inner.NewTypeError("argument 0 must be a function, got %s", call.Argument(0).ExportType()))
}

func newDeferred(s *scheduler, fn goja.Callable, delay time.Duration, repeat bool, args ...goja.Value) *deferredJob {
	t := &deferredJob{fn: fn, args: args, repeat: repeat, cancelled: make(chan struct{}), sdone: s.done}
	go func() {
		for {
			select {
			case <-t.sdone:
				return
			case <-t.cancelled:
				return

			case <-time.After(delay):
				s.run(func(*goja.Runtime) {
					if _, err := t.fn(nil, t.args...); err != nil {
						logrus.Errorln("error handling deferred job:", err)
					}
				})

				if !t.repeat {
					return
				}
			}
		}
	}()
	return t
}

func (j *deferredJob) cancel() {
	select {
	case <-j.cancelled:
		return

	default:
		close(j.cancelled)
	}
}

func cancelDeferredJob(j *deferredJob) {
	j.cancel()
}
