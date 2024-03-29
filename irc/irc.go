// Package irc is used to manage a client IRC connection as a squircy3 plugin.
package irc // import "code.dopame.me/veonik/squircy3/irc"

import (
	"crypto/tls"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	irc "github.com/thoj/go-ircevent"

	"code.dopame.me/veonik/squircy3/event"
)

var ErrNotConnected = errors.New("not connected")

type Config struct {
	Nick     string `toml:"nick"`
	Username string `toml:"user"`

	Network     string `toml:"network"`
	TLS         bool   `toml:"tls"`
	AutoConnect bool   `toml:"auto"`

	SASL         bool   `toml:"sasl"`
	SASLUsername string `toml:"sasl_username"`
	SASLPassword string `toml:"sasl_password"`

	ServerPassword string `toml:"server_password"`

	Version string
}

type Manager struct {
	config *Config
	events *event.Dispatcher
	conn   *Connection

	mu sync.RWMutex
}

func (m *Manager) SetVersionString(v string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Version = v
	if m.conn != nil {
		m.conn.Lock()
		defer m.conn.Unlock()
		m.conn.current.Version = v
		m.conn.Version = v
	}
}

type Connection struct {
	*irc.Connection

	current  Config
	quitting chan struct{}
	done     chan struct{}
}

func (conn *Connection) Connect() error {
	conn.Connection.Lock()
	defer conn.Connection.Unlock()
	return conn.Connection.Connect(conn.current.Network)
}

func (conn *Connection) Quit() (err error) {
	select {
	case <-conn.done:
		// already done, nothing to do

	case <-conn.quitting:
		// already quitting, nothing to do

	default:
		logrus.Debugln("quitting")
		conn.Connection.Quit()
		close(conn.quitting)
	}
	// block until done
	select {
	case <-conn.done:
		break

	case <-time.After(1 * time.Second):
		go func() {
			// go-ircevent may close irc.pwrite multiple times, so catch
			// the ensuing panic when it happens.
			// todo: try to fix this upstream
			defer func() {
				if e := recover(); err != nil {
					if er, ok := err.(error); ok {
						err = errors.Wrapf(er, "error during disconnect")
						return
					}
					err = errors.Errorf("error during disconnect: %v", e)
				}
			}()
			conn.Connection.Disconnect()
		}()
		return errors.Errorf("timed out waiting for quit")
	}
	return nil
}

func (conn *Connection) controlLoop() {
	errC := conn.ErrorChan()
	for {
		select {
		case <-conn.quitting:
			select {
			case <-conn.done:
				// already done
			default:
				close(conn.done)
			}
			return

		case err, ok := <-errC:
			logrus.Debugln("read from errC in controlLoop")
			if !ok {
				// channel was closed
				logrus.Debugln("conn errs already closed")
				return
			}
			logrus.Warnln("Received irc connection error:", err)
			if err != irc.ErrDisconnected {
				if err = conn.Quit(); err != nil {
					logrus.Warnln("Failed to quit irc cleanly:", err)
				}
			}
		}
	}
}

func NewManager(c *Config, ev *event.Dispatcher) *Manager {
	m := &Manager{config: c, events: ev}
	if c.AutoConnect {
		go func() {
			<-time.After(1 * time.Second)
			logrus.Infof("Auto-connecting...")
			if err := m.Connect(); err != nil {
				logrus.Errorln("failed to autoconnect:", err)
			}
		}()
	}
	return m
}

func (m *Manager) Do(fn func(*Connection) error) error {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()
	if conn == nil {
		return errors.New("not connected")
	}
	conn.Lock()
	defer conn.Unlock()
	return fn(conn)
}

func newConnection(c Config) *Connection {
	conn := &Connection{
		current:  c,
		quitting: make(chan struct{}),
		done:     make(chan struct{}),
	}
	conn.Connection = irc.IRC(c.Nick, c.Username)
	conn.Log = log.New(logrus.StandardLogger().WriterLevel(logrus.InfoLevel), "", 0)
	if c.TLS {
		conn.UseTLS = true
		conn.TLSConfig = &tls.Config{}
	}
	if c.SASL {
		conn.UseSASL = true
		conn.SASLLogin = c.SASLUsername
		conn.SASLPassword = c.SASLPassword
	}
	conn.Password = c.ServerPassword
	conn.QuitMessage = "farewell"
	conn.Version = c.Version
	return conn
}

func (m *Manager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn != nil {
		return errors.New("already connected")
	}
	m.conn = newConnection(*m.config)
	m.conn.AddCallback("*", func(ev *irc.Event) {
		m.events.Emit("irc."+ev.Code, map[string]interface{}{
			"User":    ev.User,
			"Host":    ev.Host,
			"Source":  ev.Source,
			"Code":    ev.Code,
			"Message": ev.Message(),
			"Nick":    ev.Nick,
			"Target":  ev.Arguments[0],
			"Raw":     ev.Raw,
			"Args":    append([]string{}, ev.Arguments...),
		})
	})
	err := m.conn.Connect()
	if err == nil {
		go m.conn.controlLoop()
		go func() {
			m.events.Emit("irc.CONNECT", nil)
			<-m.conn.done
			m.events.Emit("irc.DISCONNECT", nil)
			m.mu.Lock()
			defer m.mu.Unlock()
			m.conn = nil
		}()
	} else {
		m.conn = nil
	}
	return err
}

func (m *Manager) Disconnect() error {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()
	if conn == nil {
		return ErrNotConnected
	}
	return conn.Quit()
}
