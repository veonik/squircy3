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
		p.Run()
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

func TestProcess_Run_multipleTimesNoIllEffects(t *testing.T) {
	p := node_compat.NewProcess("echo")
	p.Run()
	p.Run()
	p.Run()
	if p.ExitCode() != 0 {
		t.Fatalf("expected exit code to be 0 but got %d", p.ExitCode())
	}
}

func TestProcess_Kill(t *testing.T) {
	p := node_compat.NewProcess("sleep", "10")
	p.Start()
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

func TestProcess_Kill_notStarted(t *testing.T) {
	p := node_compat.NewProcess("sleep", "10")
	err := p.Kill()
	if err == nil {
		t.Fatalf("expected err but got nil")
	} else if err.Error() != "not started" {
		t.Fatalf("expected process to not be started but got: %s", err)
	}
}

func TestProcess_Kill_alreadyFinished(t *testing.T) {
	p := node_compat.NewProcess("echo")
	p.Start()
	if p.ExitCode() != 0 {
		t.Fatalf("expected exit code of 0 but got %d", p.ExitCode())
	}
	err := p.Kill()
	if err == nil {
		t.Fatalf("expected err but got nil")
	} else if err.Error() != "already finished" {
		t.Fatalf("expected process to already be finished but got: %s", err)
	}
}

func TestProcess_IO(t *testing.T) {
	p := node_compat.NewProcess("cat")
	p.Start()
	if err := p.Input("hello"); err != nil {
		t.Fatalf("error writing to stdin: %s", err)
	}
	<-time.After(100 * time.Millisecond)
	if out := p.Output(); out != "hello" {
		t.Fatalf("expected output to equal \"hello\" but got: %s", out)
	}
	if err := p.Input(", hi there"); err != nil {
		t.Fatalf("error writing to stdin: %s", err)
	}
	<-time.After(100 * time.Millisecond)
	if out := p.Output(); out != "hello, hi there" {
		t.Fatalf("expected output to equal \"hello, hi there\" but got: %s", out)
	}
	if p.Done() != false {
		t.Fatalf("expected process to still be running, but was not")
	}
	if err := p.CloseInput(); err != nil {
		t.Fatalf("expected no error when closing stdin, but got: %s", err)
	}
	if p.ExitCode() != 0 {
		t.Fatalf("expected exit code to be 0 but got %d", p.ExitCode())
	}
}
