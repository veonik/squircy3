// Package vm is an extendable, embeddable javascript interpreter for squircy3.
//
// This package embeds goja (https://github.com/dop251/goja) as the javascript
// parser and executor, improving on it with a concurrency-safe API, intuitive
// API for dealing with asynchronous results, and basic compatibility with
// NodeJS's require() function.
package vm // import "code.dopame.me/veonik/squircy3/vm"

import (
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/pkg/errors"
)

// A VM manages the state and environment of a javascript interpreter.
type VM struct {
	registry  *Registry
	scheduler *scheduler

	// done is initialized when the VM is started and closed when it is stopped.
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
			// closed; not done, nothing to do
		default:
			return nil
		}
	}
	vm.done = make(chan struct{})
	return vm.scheduler.start()
}

func (vm *VM) Shutdown() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	done := vm.done
	var err error
	go func() {
		select {
		case <-done:
			// do nothing, already closed
		default:
			err = vm.scheduler.stop()
			close(done)
		}
	}()
	select {
	case <-done:
		// all done, nothing to do
	case <-time.After(2 * time.Second):
		return errors.New("timed out waiting for vm to shutdown")
	}
	return err
}

func (vm *VM) doneChan() chan struct{} {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	return vm.done
}

func (vm *VM) RunString(in string) *AsyncResult {
	return vm.RunScript("<eval>", in)
}

func (vm *VM) RunScript(name, in string) *AsyncResult {
	vmdone := vm.doneChan()
	res := newResult(vmdone)
	vm.scheduler.run(func(r *goja.Runtime) {
		p, err := vm.Compile(name, in)
		if err != nil {
			res.resolve(nil, err)
		} else {
			res.resolve(r.RunProgram(p))
		}
	})
	return newAsyncResult(res, vmdone, vm.Do)
}

func (vm *VM) RunProgram(p *goja.Program) *AsyncResult {
	vmdone := vm.doneChan()
	res := newResult(vmdone)
	vm.scheduler.run(func(r *goja.Runtime) {
		res.resolve(r.RunProgram(p))
	})
	return newAsyncResult(res, vmdone, vm.Do)
}

func (vm *VM) Do(fn func(*goja.Runtime)) {
	vm.scheduler.run(fn)
}
