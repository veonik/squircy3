package vm_test

import (
	"testing"
	"time"

	"code.dopame.me/veonik/squircy3/vm"
)

func TestVM_Restart(t *testing.T) {
	v, err := vm.New(vm.NewRegistry("."))
	if err != nil {
		t.Errorf("failed to create v: %s", err)
		return
	}
	if err := v.Start(); err != nil {
		t.Errorf("failed to start v: %s", err)
		return
	}
	res, err := v.RunString("10 + 5").Await()
	if err != nil {
		t.Errorf("error evaluating string: %s", err)
		return
	}
	if ri := res.ToInteger(); ri != 15 {
		t.Errorf("expected expression to result in 15, got %d", ri)
		return
	}
	time.Sleep(10 * time.Millisecond)
	if err := v.Shutdown(); err != nil {
		t.Errorf("failed to shutdown v: %s", err)
		return
	}
	time.Sleep(10 * time.Millisecond)
	if err := v.Start(); err != nil {
		t.Errorf("failed to start v: %s", err)
		return
	}
	res, err = v.RunString("15 + 10").Await()
	if err != nil {
		t.Errorf("error evaluating string: %s", err)
		return
	}
	if ri := res.ToInteger(); ri != 25 {
		t.Errorf("expected expression to result in 25, got %d", ri)
		return
	}
	time.Sleep(10 * time.Millisecond)
	if err := v.Shutdown(); err != nil {
		t.Errorf("failed to shutdown v: %s", err)
		return
	}
}
