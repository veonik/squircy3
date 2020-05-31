package internal

import (
	"io"
	"net"
	"sync"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/vm"
)

type NetConn struct {
	conn net.Conn

	buf      []byte
	readable bool
}

type readResult struct {
	ready bool
	value string
	error error

	mu sync.Mutex
}

func (r *readResult) Ready() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ready
}

func (r *readResult) Value() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ready {
		return "", errors.New("not ready")
	}
	return r.value, r.error
}

func (r *readResult) resolve(val string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ready {
		return
	}
	r.value = val
	r.error = err
	r.ready = true
}

func NewNetConn(conn net.Conn) (*NetConn, error) {
	return &NetConn{
		conn:     conn,
		buf:      make([]byte, 1024),
		readable: true,
	}, nil
}

func (c *NetConn) Write(s string) (n int, err error) {
	return c.conn.Write([]byte(s))
}

// Read asynchronously reads from the connection.
// A readResult is returned back to the js vm and once the read completes,
// it can be read from. This allows the js vm to avoid blocking for reads.
func (c *NetConn) Read(_ int) *readResult {
	res := &readResult{}
	if !c.readable {
		logrus.Warnln("reading from unreadable conn")
		return res
	}
	go func() {
		n, err := c.conn.Read(c.buf)
		if err != nil {
			if err != io.EOF {
				res.resolve("", err)
				return
			} else {
				// on the next call to read, we'll return nil to signal done.
				c.readable = false
			}
		}
		logrus.Warnln("read", n, "bytes")
		rb := make([]byte, n)
		copy(rb, c.buf)
		res.resolve(string(rb), nil)
	}()
	return res
}

func (c *NetConn) Close() error {
	return c.conn.Close()
}

func (c *NetConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *NetConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func Dial(kind, addr string) (*NetConn, error) {
	c, err := net.Dial(kind, addr)
	if err != nil {
		return nil, err
	}
	return NewNetConn(c)
}

type Server struct {
	listener net.Listener

	vm        *vm.VM
	onConnect goja.Callable
}

func (s *Server) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logrus.Warnln("failed to accept new connection", err)
			return
		}
		s.vm.Do(func(gr *goja.Runtime) {
			nc, err := NewNetConn(conn)
			if err != nil {
				logrus.Warnln("failed to get NetConn from net.Conn", err)
				return
			}
			if _, err := s.onConnect(nil, gr.ToValue(nc)); err != nil {
				logrus.Warnln("error running on-connect callback", err)
			}
		})
	}
}

func (s *Server) Close() error {
	defer func() {
		s.onConnect = nil
	}()
	return s.listener.Close()
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func Listen(vmp *vm.VM, onConnect goja.Callable, kind, addr string) (*Server, error) {
	l, err := net.Listen(kind, addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener:  l,
		vm:        vmp,
		onConnect: onConnect,
	}
	go s.accept()
	return s, nil
}
