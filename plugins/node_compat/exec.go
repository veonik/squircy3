package node_compat

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Process struct {
	cmd      *exec.Cmd
	input    bytes.Buffer
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	error    error
	running  int64
	finished chan struct{}
}

func NewProcess(command string, args ...string) *Process {
	cmd := exec.Command(command, args...)
	p := &Process{cmd: cmd, finished: make(chan struct{})}
	return p
}

func (p *Process) Run() {
	if !atomic.CompareAndSwapInt64(&p.running, 0, 1) {
		return
	}
	select {
	case <-p.finished:
		return

	default:
		// not ran yet
	}
	defer func() {
		close(p.finished)
	}()
	p.cmd.Stdout = &p.stdout
	p.cmd.Stderr = &p.stderr
	p.cmd.Stdin = &p.input
	p.error = p.cmd.Run()
}

func (p *Process) Result() (*os.ProcessState, error) {
	select {
	case <-p.finished:
		return p.cmd.ProcessState, p.error
	}
}

func (p *Process) ExitCode() int {
	select {
	case <-p.finished:
		return p.cmd.ProcessState.ExitCode()
	}
}

func (p *Process) Kill() error {
	select {
	case <-p.finished:
		return errors.New("not running")
	default:
		return p.cmd.Process.Kill()
	}
}

func (p *Process) Input(input string) error {
	_, err := fmt.Fprintln(&p.input, input)
	return err
}

func (p *Process) Error() string {
	return p.stderr.String()
}

func (p *Process) Output() string {
	return p.stdout.String()
}
