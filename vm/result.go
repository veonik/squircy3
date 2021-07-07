package vm

import (
	"math/rand"
	"regexp"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var ErrExecutionCancelled = errors.New("execution cancelled")

// A Result is the output from executing synchronous code on a VM.
type Result struct {
	// Closed when the result is ready. Read from this channel to detect when
	// the result has been populated and is safe to inspect.
	Ready chan struct{}
	// Error associated with the result, if any. Only read from this after
	// the result is ready.
	Error error
	// Value associated with the result if there is no error. Only read from
	// this after the result is ready.
	Value goja.Value

	// vmdone is a copy of the VM's done channel at the time Run* is called.
	// This removes the need to synchronize when reading from the channel
	// since the copy is made while the VM is locked.
	vmdone chan struct{}
	// cancel is closed to signal that the result is no longer needed.
	cancel chan struct{}
}

// resolve populates the result with the given value or error and signals ready.
func newResult(vmdone chan struct{}) *Result {
	r := &Result{Ready: make(chan struct{}), cancel: make(chan struct{}), vmdone: vmdone}
	go func() {
		for {
			select {
			case <-r.Ready:
				// close the cancel channel if we need to
				r.Cancel()
				return

			case <-r.cancel:
				// signal to cancel received, resolve with an error
				r.resolve(nil, ErrExecutionCancelled)

			case <-r.vmdone:
				// VM shutdown without resolving, cancel execution
				r.Cancel()
			}
		}
	}()
	return r
}

// resolve populates the result with the given value or error and signals ready.
func (r *Result) resolve(v goja.Value, err error) {
	select {
	case <-r.Ready:
		logrus.Debugln("resolve called on already finished Result")

	default:
		r.Error = err
		r.Value = v
		close(r.Ready)
	}
}

// Await blocks until the result is ready and returns the result or error.
func (r *Result) Await() (goja.Value, error) {
	<-r.Ready
	return r.Value, r.Error
}

// Cancel the result to halt execution.
func (r *Result) Cancel() {
	select {
	case <-r.cancel:
		// already cancelled, don't bother

	default:
		close(r.cancel)
	}
}

// runFunc is a proxy for the Do method on a VM.
type runFunc func(func(*goja.Runtime))

// AsyncResult handles invocations of asynchronous code that returns promises.
// An AsyncResult accepts any goja.Value; non-promises are supported so this
// is safe (if maybe a bit inefficient) to wrap all results produced by using
// one of the Run* methods on a VM.
type AsyncResult struct {
	// Closed when the result is ready. Read from this channel to detect when
	// the result has been populated and is safe to inspect.
	Ready chan struct{}
	// Error associated with the result, if any. Only read from this after
	// the result is ready.
	Error error
	// Value associated with the result if there is no error. Only read from
	// this after the result is ready.
	Value goja.Value

	// syncResult contains the original synchronous result.
	// Its value may contain a Promise although other types are also handled.
	syncResult *Result

	// Unique javascript variable containing the result.
	stateVar string

	// vmdone is a copy of the VM's done channel at the time Run* is called.
	// This removes the need to synchronize when reading from the channel
	// since the copy is made while the VM is locked.
	vmdone chan struct{}
	// vmdo will be a pointer to the run method on a scheduler.
	// This isn't strictly necessary (a pointer to VM would be fine) but
	// this forced indirection reduces the chance of a Result trying to do
	// something it shouldn't.
	vmdo runFunc

	// cancel is closed to signal that the result is no longer needed.
	cancel chan struct{}
	// waiting is initialized in the run method and used to synchronize
	// the result-ready check.
	waiting chan struct{}
	// done is closed when the run method returns.
	// The goroutine spawned by newAsyncResult waits to return until this
	// channel is closed.
	done chan struct{}
}

func uniqueResultIdentifier() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABZDEFGHIJKLMNOPQRSTUVWXYZ12345678890"
	b := make([]byte, 20)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "____squircy3_await_" + string(b)
}

func newAsyncResult(sr *Result, vmdone chan struct{}, vmdo runFunc) *AsyncResult {
	r := &AsyncResult{
		Ready:      make(chan struct{}),
		syncResult: sr,
		stateVar:   uniqueResultIdentifier(),
		vmdo:       vmdo,
		vmdone:     vmdone,
		cancel:     make(chan struct{}),
		done:       make(chan struct{}),
	}
	go func() {
		// wait until the original Result is ready
		<-sr.Ready
		if sr.Error != nil {
			// already have an error, resolve immediately
			r.resolve(sr.Value, sr.Error)
			return
		}
		// schedule the job to resolve the value
		r.vmdo(r.run)

		// note that the scheduler may have been stopped since the
		// original Result was ready.

		// block until the result is cancelled, the VM is shut down,
		// or the result is ready.
		select {
		case <-r.cancel:
			r.resolve(nil, ErrExecutionCancelled)
			return
		case <-r.vmdone:
			r.resolve(nil, ErrExecutionCancelled)
			return
		case <-r.done:
			// carry on
		}
	}()
	return r
}

// resolve populates the result with the given value or error and signals ready.
func (r *AsyncResult) resolve(v goja.Value, err error) {
	select {
	case <-r.Ready:
		logrus.Warnln("resolve called on already finished Result")

	default:
		r.Error = err
		r.Value = v
		close(r.Ready)
		r.vmdo(r.cleanup)
	}
}

// Await blocks until the result is ready and returns the result or error.
func (r *AsyncResult) Await() (goja.Value, error) {
	<-r.Ready
	return r.Value, r.Error
}

// Cancel the result to halt execution.
func (r *AsyncResult) Cancel() {
	select {
	case <-r.cancel:
		// already cancelled, don't bother

	default:
		close(r.cancel)
	}
}

func getOrCreateResultHandler(gr *goja.Runtime) goja.Callable {
	if v := gr.Get("____squircy3_handle_result"); v != nil {
		fn, _ := goja.AssertFunction(v)
		return fn
	}
	// this script handles the resolution of the Promise if the value is a
	// Promise, sets the error if the value is an Error, or sets the result
	// as the value for anything else.
	v, err := gr.RunString(`
(function(state) {
	if(typeof Promise !== 'undefined' && state.value instanceof Promise) {
	  state.value
		.then(function(result) { state.result = result; })
		.catch(function(error) { state.error = error; })
		.finally(function() { state.done = true; });
	} else if(state.error instanceof Error) {
	  state.error = state.value;
	  state.done = true;
	} else {
	  state.result = state.value;
	  state.done = true;
	}
})`)
	if err != nil {
		logrus.Warnln("unable to set async result handler:", err.Error())
		return nil
	}
	gr.Set("____squircy3_handle_result", v)
	fn, _ := goja.AssertFunction(v)
	return fn
}

func (r *AsyncResult) run(gr *goja.Runtime) {
	defer func() {
		select {
		case <-r.done:
			// already closed
		default:
			close(r.done)
		}
	}()
	o := gr.NewObject()
	_ = o.Set("value", r.syncResult.Value)
	_ = o.Set("result", goja.Undefined())
	_ = o.Set("error", goja.Undefined())
	_ = o.Set("done", false)
	gr.Set(r.stateVar, o)
	hr := getOrCreateResultHandler(gr)
	if hr == nil {
		r.resolve(nil, errors.New("unable to get result handler"))
		return
	}
	_, err := hr(nil, gr.Get(r.stateVar))
	if err != nil {
		r.resolve(nil, err)
		return
	}
	go r.loop()
}

func (r *AsyncResult) cleanup(gr *goja.Runtime) {
	gr.Set(r.stateVar, goja.Undefined())
}

func (r *AsyncResult) loop() {
	defer func() {
		select {
		case <-r.Ready:
			// already closed, no need to close it again
			return
		default:
			close(r.Ready)
		}
	}()
	// delay is how long to wait until we check for a result
	delay := 10 * time.Microsecond
	for {
		if delay < 100*time.Millisecond {
			// backoff sharply at first but stop at 100ms between checks
			delay = delay * 10
		}
		select {
		case <-r.cancel:
			return
		case <-r.vmdone:
			// VM shutdown without resolving, cancel execution
			r.Cancel()
			continue
		case <-time.After(delay):
			r.waiting = make(chan struct{})
			r.vmdo(r.check)
			<-r.waiting
		}
		select {
		case <-r.Ready:
			return
		default:
		}
	}
}

func (r *AsyncResult) check(gr *goja.Runtime) {
	defer close(r.waiting)
	o := gr.Get(r.stateVar).ToObject(gr)
	if !o.Get("done").ToBoolean() {
		// result is not yet ready
		return
	}
	// result is ready, get it out of the vm
	var res goja.Value
	var err error
	v := o.Get("error")
	if goja.IsUndefined(v) {
		res = o.Get("result")
	} else {
		if cv, ok := v.Export().(error); ok {
			err = cv
		} else {
			// seems like many errors will not actually get exported to error interface,
			// so this matches the string representation of the value if it looks like
			// an error message.
			// this will match strings like:
			//   Error: some message
			//   TypeError: some message
			//   Exception: hello, world
			//   SomeException: hi there
			vs := v.String()
			if ok, err2 := regexp.MatchString("^([a-zA-Z0-9]+?)?(Error|Exception):", vs); err2 == nil && ok {
				err = errors.New(vs)
			}
		}
		if err == nil {
			err = errors.Errorf("received non-Error from rejected Promise: %s %s", v.String(), v.ExportType())
		}
	}
	r.resolve(res, err)
}
