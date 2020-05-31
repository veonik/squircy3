package vm_test

import (
	"testing"

	"code.dopame.me/veonik/squircy3/vm"
)

var registry = vm.NewRegistry("../testdata")

func TestAsyncResult_withNonPromise(t *testing.T) {
	v, err := vm.New(registry)
	if err != nil {
		t.Fatalf("unexpected error creating VM: %s", err)
	}
	if err := v.Start(); err != nil {
		t.Errorf("failed to start v: %s", err)
		return
	}
	res, err := v.RunString(`"hello, world!"`).Await()
	if err != nil {
		t.Errorf("failed to run script: %s", err)
		return
	}
	if res.String() != "hello, world!" {
		t.Errorf("expected: hello, world!\ngot: %s", res.String())
		return
	}
}

func TestAsyncResult_withPromise(t *testing.T) {
	v, err := vm.New(registry)
	if err != nil {
		t.Fatalf("unexpected error creating VM: %s", err)
	}
	if err := v.Start(); err != nil {
		t.Errorf("failed to start v: %s", err)
		return
	}
	res, err := v.RunString(`
this.Promise = require('./es6-promise').Promise;
new Promise(function(resolve) { resolve("hello, world!"); });`).Await()
	if err != nil {
		t.Errorf("failed to run script: %s", err)
		return
	}
	if res.String() != "hello, world!" {
		t.Errorf("expected: hello, world!\ngot: %s", res.String())
		return
	}
}

func TestAsyncResult_StopVM(t *testing.T) {
	v, err := vm.New(registry)
	if err != nil {
		t.Fatalf("unexpected error creating VM: %s", err)
	}
	if err := v.Start(); err != nil {
		t.Errorf("failed to start v: %s", err)
		return
	}
	r := v.RunString(`
this.Promise = require('./es6-promise').Promise;
new Promise(function(resolve) { 
  setTimeout(function() { 
	console.log('wot');
    resolve("hello, world!"); 
  }, 100); 
});`)
	// <-time.After(10 * time.Millisecond)
	if err := v.Shutdown(); err != nil {
		t.Errorf("failed to shutdown VM: %s", err)
		return
	}
	res, err := r.Await()
	if err == nil {
		t.Errorf("expected error, got nil")
		t.Errorf("value: %T %T", res, res.Export())
		return
	}
	if err.Error() != "execution cancelled" {
		t.Errorf("expected: cancelled\ngot: %s", err.Error())
		return
	}
}
