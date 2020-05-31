package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// A safeBuffer is a buffer that is safe to use in concurrent contexts.
type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

func (b *safeBuffer) Read(p []byte) (n int, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Read(p)
}

func (b *safeBuffer) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.String()
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

type Process struct {
	cmd      *exec.Cmd
	stdio    io.WriteCloser
	stdout   safeBuffer
	stderr   safeBuffer
	error    error
	running  int64
	finished chan struct{}
}

func NewProcess(command string, args ...string) *Process {
	cmd := exec.Command(command, args...)
	p := &Process{cmd: cmd, finished: make(chan struct{})}
	p.cmd.Stdout = &p.stdout
	p.cmd.Stderr = &p.stderr
	wc, err := cmd.StdinPipe()
	if err != nil {
		p.error = err
		close(p.finished)
	}
	p.stdio = wc
	return p
}

// start starts the process, leaving stdin open for writing.
//
// If the started process reads from stdin, it may not exit until
// CloseInput is called.
func (p *Process) Start() {
	if !atomic.CompareAndSwapInt64(&p.running, 0, 1) {
		return
	}
	select {
	case <-p.finished:
		return

	default:
		// not ran yet
	}
	if err := p.cmd.Start(); err != nil {
		p.error = err
		close(p.finished)
		return
	}
	go func() {
		atomic.AddInt64(&p.running, 1)
		p.error = p.cmd.Wait()
		close(p.finished)
	}()
}

// Run starts the process, closing standard input immediately.
//
// Note that whatever input is already buffered (ie. through calls to Input)
// will still be written to the process's stdin.
func (p *Process) Run() {
	if err := p.CloseInput(); err != nil {
		logrus.Warnf("error closing stdin for process: %s", err)
	}
	p.Start()
}

func (p *Process) Done() bool {
	select {
	case <-p.finished:
		return true
	default:
		return false
	}
}

func (p *Process) PID() (int, error) {
	if atomic.LoadInt64(&p.running) < 1 {
		return -1, errors.New("not running")
	}
	select {
	case <-p.finished:
		return -1, errors.New("already finished")
	default:
		return p.cmd.Process.Pid, nil
	}
}

func (p *Process) Result() (*os.ProcessState, error) {
	<-p.finished
	return p.cmd.ProcessState, p.error
}

func (p *Process) ExitCode() int {
	<-p.finished
	return p.cmd.ProcessState.ExitCode()
}

func (p *Process) Kill() error {
	s := atomic.LoadInt64(&p.running)
	if s == 0 {
		return errors.New("not started")
	} else if s == 1 {
		for {
			<-time.After(100 * time.Microsecond)
			if atomic.LoadInt64(&p.running) == 2 {
				break
			}
		}
	}
	select {
	case <-p.finished:
		return errors.New("already finished")
	default:
		return p.cmd.Process.Kill()
	}
}

func (p *Process) Input(input string) error {
	_, err := fmt.Fprint(p.stdio, input)
	return err
}

func (p *Process) CloseInput() error {
	return p.stdio.Close()
}

func (p *Process) Error() string {
	return p.stderr.String()
}

func (p *Process) Output() string {
	return p.stdout.String()
}
