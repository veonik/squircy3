package vm // import "code.dopame.me/veonik/squircy3/vm"

import (
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/pkg/errors"
)

type VM struct {
	registry  *Registry
	scheduler *scheduler

	babel bool

	done chan struct{}
	mu   sync.Mutex
}

func New(registry *Registry) (*VM, error) {
	return &VM{registry: registry, scheduler: newScheduler(registry)}, nil
}

func (vm *VM) SetModule(module *Module) {
	vm.registry.SetModule(module)
}

func (vm *VM) PrependRuntimeInit(h func(*goja.Runtime)) {
	vm.scheduler.prependRuntimeInit(h)
}

func (vm *VM) OnRuntimeInit(h func(*goja.Runtime)) {
	vm.scheduler.onRuntimeInit(h)
}

func (vm *VM) SetTransformer(fn func(in string) (string, error)) {
	vm.registry.Transform = fn
}

func (vm *VM) Compile(name, in string) (*goja.Program, error) {
	p, err := parser.ParseFile(nil, name, in, parser.Mode(0))
	if err != nil {
		if vm.registry.Transform != nil {
			in, err = vm.registry.Transform(in)
			if err != nil {
				return nil, err
			}
			return goja.Compile(name, in, true)
		}
		return nil, err
	}
	return goja.CompileAST(p, true)
}

func (vm *VM) Start() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if vm.done != nil {
		select {
		case <-vm.done:
			// closed; not running, nothing to do
		default:
			return nil
		}
	}
	vm.done = make(chan struct{})
	return vm.scheduler.start()
}

func (vm *VM) Shutdown() (err error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	go func(done chan struct{}) {
		select {
		case <-done:
			// do nothing, already closed
		default:
			err = vm.scheduler.stop()
			close(done)
		}
	}(vm.done)
	select {
	case <-vm.done:
		// all done, nothing to do
	case <-time.After(time.Second):
		err = errors.New("timed out waiting for vm to shutdown")
	}
	return err
}

// Result is the output from executing some code in the VM.
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

	cancel chan struct{}
}

func newResult(vm *VM) *Result {
	r := &Result{Ready: make(chan struct{}), cancel: make(chan struct{})}
	go func() {
		for {
			select {
			case <-r.Ready:
				// close the cancel channel if we need to
				select {
				case <-r.cancel:
					// do nothing

				default:
					close(r.cancel)
				}
				return

			case <-r.cancel:
				// signal to cancel received, resolve with an error
				r.resolve(nil, errors.New("execution cancelled"))

			case <-vm.done:
				// VM shutdown without resolving, cancel execution
				close(r.cancel)
			}
		}
	}()
	return r
}

// resolve populates the result with the given value or error.
func (r *Result) resolve(v goja.Value, err error) {
	select {
	case <-r.Ready:
		fmt.Println("resolve called on already finished Result")

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

func (vm *VM) RunString(in string) *Result {
	res := newResult(vm)
	vm.scheduler.run(func(r *goja.Runtime) {
		p, err := vm.Compile("<eval>", in)
		if err != nil {
			res.resolve(nil, err)
		} else {
			res.resolve(r.RunProgram(p))
		}
	})
	return res
}

func (vm *VM) RunScript(name, in string) *Result {
	res := newResult(vm)
	vm.scheduler.run(func(r *goja.Runtime) {
		p, err := vm.Compile(name, in)
		if err != nil {
			res.resolve(nil, err)
		} else {
			res.resolve(r.RunProgram(p))
		}
	})
	return res
}

func (vm *VM) RunProgram(p *goja.Program) *Result {
	res := newResult(vm)
	vm.scheduler.run(func(r *goja.Runtime) {
		res.resolve(r.RunProgram(p))
	})
	return res
}

func (vm *VM) Do(fn func(*goja.Runtime)) {
	vm.scheduler.run(fn)
}
