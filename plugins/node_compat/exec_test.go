package node_compat_test

import (
	"strings"
	"testing"
	"time"

	"code.dopame.me/veonik/squircy3/plugins/node_compat"
)

type test struct {
	expected string
	command  string
	args     []string
	prepare  func(p *node_compat.Process) error
}

var tests = map[string]test{
	"simple stdout": {"hello, world", "echo", []string{"hello, world"}, nil},
	"stdin": {"hi there, world", "cat", nil, func(p *node_compat.Process) error {
		return p.Input("hi there, world")
	}},
}

func TestNewProcess(t *testing.T) {
	for name, test := range tests {
		p := node_compat.NewProcess(test.command, test.args...)
		if test.prepare != nil {
			if err := test.prepare(p); err != nil {
				t.Errorf("%s: error preparing test: %s", name, err)
				continue
			}
		}
		go p.Run()
		s, err := p.Result()
		if err != nil {
			t.Errorf("%s: error running test: %s", name, err)
			continue
		}
		if !s.Success() {
			t.Errorf("%s: exited with code: %d", name, s.ExitCode())
			continue
		}
		o := strings.TrimSpace(p.Output())
		if o != test.expected {
			t.Errorf("%s: expected %s, but got: %s", name, test.expected, o)
			continue
		}
	}
}

func TestProcess_Kill(t *testing.T) {
	p := node_compat.NewProcess("sleep", "10")
	go p.Run()
	<-time.After(100 * time.Millisecond)
	if err := p.Kill(); err != nil {
		t.Fatalf("failed to kill process: %s", err)
	}
	res, err := p.Result()
	if res != nil && res.Success() {
		t.Fatalf("expected process to exit with error, but exited successfully")
	}
	if err == nil {
		t.Fatalf("expected error from killed process, got nil")
	}
}
