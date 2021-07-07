package plugin // import "code.dopame.me/veonik/squircy3/plugin"

import (
	"fmt"
	"plugin"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// A Plugin is the basic interface that all squircy3 plugins must implement.
type Plugin interface {
	// Name returns the unique name for the Plugin.
	Name() string
}

// PluginInitHandler is implemented by types that want to be notified when
// another plugin is initialized.
type PluginInitHandler interface {
	// HandlePluginInit is called after each other Plugin is initialized.
	// Plugin initialization occurs when the Configure method is called on
	// a plugin.Manager.
	HandlePluginInit(Plugin)
}

// ShutdownHandler is implemented by types that want to perform some action
// when the application is shutting down.
type ShutdownHandler interface {
	// HandleShutdown is called when the application begins shutting down.
	// All ShutdownHandlers are invoked concurrently, but each has a limited
	// amount of time to gracefully clean up before the application forcefully
	// exits.
	HandleShutdown()
	Name() string
}

// An Initializer is a type that can create a Plugin implementation.
type Initializer interface {
	// Initialize creates the corresponding Plugin for this Initializer.
	Initialize(*Manager) (Plugin, error)
}

// An InitializerFunc is a function that implements Initializer.
type InitializerFunc func(*Manager) (Plugin, error)

func (f InitializerFunc) Initialize(m *Manager) (Plugin, error) {
	return f(m)
}

// InitializeFromFile returns an Initializer that will attempt to load the
// specified plugin shared library from the filesystem.
// Shared library plugins must be built as a `main` package and must have
// an "Initialize" function defined at the package level. The "Initialize"
// function must be compatible with the Initialize method on the Initializer
// interface. That is, it must have the signature:
//   func Initialize(*Manager) (Plugin, error)
func InitializeFromFile(p string) Initializer {
	return InitializerFunc(func(m *Manager) (Plugin, error) {
		pl, err := plugin.Open(p)
		if err != nil {
			return nil, errors.Wrapf(err, "unable (%s) to open plugin", p)
		}
		in, err := pl.Lookup("Initialize")
		if err != nil {
			return nil, errors.Wrapf(err, "plugin (%s) does not export Initialize", p)
		}
		fn, ok := in.(func(*Manager) (Plugin, error))
		if !ok {
			return nil, errors.Errorf("plugin (%s) has invalid type for Initialize: expected func(*plugin.Manager) (plugin.Plugin, error), got %T", p, in)
		}
		plg, err := fn(m)
		if err != nil {
			return nil, errors.Wrapf(err, "plugin (%s) init failed", p)
		}
		return plg, nil
	})
}

// A Manager controls the loading and configuration of plugins.
type Manager struct {
	plugins []Initializer

	loaded map[string]Plugin

	onInit     []PluginInitHandler
	onShutdown []ShutdownHandler

	mu sync.RWMutex
}

func NewManager(plugins ...string) *Manager {
	plgs := make([]Initializer, len(plugins))
	for i, n := range plugins {
		plgs[i] = InitializeFromFile(n)
	}
	return &Manager{
		plugins: plgs,
		loaded:  make(map[string]Plugin),
	}
}

// OnPluginInit adds the given PluginInitHandler to be called when a plugin
// is initialized.
func (m *Manager) OnPluginInit(h PluginInitHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onInit = append(m.onInit, h)
}

// OnShutdown adds the given ShutdownHandler to be called when the appliation
// is shutting down.
func (m *Manager) OnShutdown(h ShutdownHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onShutdown = append(m.onShutdown, h)
}

// Shutdown begins the shut down process.
// Each ShutdownHandler is called concurrently and this method returns after
// all handlers have completed.
func (m *Manager) Shutdown() {
	m.mu.RLock()
	hs := make([]ShutdownHandler, len(m.onShutdown))
	copy(hs, m.onShutdown)
	m.mu.RUnlock()
	wg := &sync.WaitGroup{}
	for _, h := range m.onShutdown {
		wg.Add(1)
		wait := make(chan struct{})
		go func(sh ShutdownHandler) {
			sh.HandleShutdown()
			close(wait)
		}(h)
		go func(sh ShutdownHandler) {
			select {
			case <-wait:
				break
			case <-time.After(2 * time.Second):
				logrus.Warnln("shutdown of", sh.Name(), "timed out after 2 seconds")
				break
			}
			wg.Done()
		}(h)
	}
	wg.Wait()
}

// Loaded returns a list of plugins currently loaded.
func (m *Manager) Loaded() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ns []string
	for n, _ := range m.loaded {
		ns = append(ns, n)
	}
	return ns
}

// Lookup returns the given plugin by name, or an error if it isn't loaded.
func (m *Manager) Lookup(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if plg, ok := m.loaded[name]; ok {
		return plg, nil
	}
	return nil, errors.Errorf("no plugin named %s", name)
}

// Register adds a plugin Initializer to the Manager.
// Invoke all the registered Initializers by calling Configure on the Manager.
func (m *Manager) Register(initfn Initializer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins = append(m.plugins, initfn)
}

// RegisterFunc is a shorthand method to register an InitializerFunc without
// extra type assertions.
func (m *Manager) RegisterFunc(initfn func(m *Manager) (Plugin, error)) {
	m.Register(InitializerFunc(initfn))
}

// Configure attempts to load and configure all registered plugins.
// An error will be returned for each failed initialization.
func (m *Manager) Configure() []error {
	var errs []error
	m.mu.Lock()
	// copy the pointer to the current plugins slice
	plugins := m.plugins
	// and reset the list of pending plugin inits on the Manager.
	m.plugins = nil
	m.mu.Unlock()
	if len(plugins) == 0 {
		return errs
	}
	for _, p := range plugins {
		m.mu.RLock()
		// get a fresh copy of init handlers before each init;
		// plugins may add handlers in this loop and those should be accounted
		// for on subsequent inits.
		inits := append([]PluginInitHandler{}, m.onInit...)
		m.mu.RUnlock()
		// Manager should be unlocked while the plugin initializes; the plugin
		// is free to use the Manager itself during init.
		plg, err := p.Initialize(m)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "plugin (unknown) init failed"))
			continue
		}
		pn := plg.Name()
		m.mu.Lock()
		_, ok := m.loaded[pn]
		if !ok {
			// not already loaded, add it
			m.loaded[pn] = plg
			if ih, ok := plg.(PluginInitHandler); ok {
				m.onInit = append(m.onInit, ih)
			}
			if sh, ok := plg.(ShutdownHandler); ok {
				m.onShutdown = append(m.onShutdown, sh)
			}
		}
		// unlock outside of any conditional
		m.mu.Unlock()
		if ok {
			// plugin was already loaded
			errs = append(errs, errors.Errorf("plugin (%s) already loaded", pn))
			continue
		}
		// run other plugin init handlers
		for _, h := range inits {
			h.HandlePluginInit(plg)
		}
	}
	return errs
}

// Main is a helper function that can be used to provide a consistent main
// func. Plugins aren't normally executed anyway, but Go requires that every
// "main" package have a "main" func.
func Main(pluginName string) {
	fmt.Println(pluginName, "- a plugin for squircy3")
}
