package plugin // import "code.dopame.me/veonik/squircy3/plugin"

import (
	"plugin"
	"sync"

	"github.com/pkg/errors"
)

type Plugin interface {
	Name() string
}

type InitHandler interface {
	HandlePluginInit(Plugin)
}

type Initializer interface {
	Initialize(*Manager) (Plugin, error)
}

type InitializerFunc func(*Manager) (Plugin, error)

func (f InitializerFunc) Initialize(m *Manager) (Plugin, error) {
	return f(m)
}

func InitializeFromFile(p string) Initializer {
	return InitializerFunc(func(m *Manager) (Plugin, error) {
		pl, err := plugin.Open(p)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to open plugin (%s)", p)
		}
		in, err := pl.Lookup("Initialize")
		if err != nil {
			return nil, errors.Wrapf(err, "plugin does not export Initialize (%s)", p)
		}
		fn, ok := in.(func(*Manager) (Plugin, error))
		if !ok {
			err := errors.Errorf("plugin has invalid type for Initialize (%s): expected func(*plugin.Manager) (plugin.Plugin, error)", p)
			return nil, err
		}
		plg, err := fn(m)
		if err != nil {
			return nil, errors.Wrapf(err, "plugin init failed (%s)", p)
		}
		return plg, nil
	})
}

type Manager struct {
	plugins []Initializer

	loaded map[string]Plugin

	onInit []InitHandler

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

func (m *Manager) OnPluginInit(h InitHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onInit = append(m.onInit, h)
}

func (m *Manager) Lookup(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if plg, ok := m.loaded[name]; ok {
		return plg, nil
	}
	return nil, errors.Errorf("no plugin named %s", name)
}

func (m *Manager) Register(initfn Initializer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins = append(m.plugins, initfn)
}

func (m *Manager) RegisterFunc(initfn func(m *Manager) (Plugin, error)) {
	m.Register(InitializerFunc(initfn))
}

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
		inits := append([]InitHandler{}, m.onInit...)
		m.mu.RUnlock()
		// Manager should be unlocked while the plugin initializes; the plugin
		// is free to use the Manager itself during init.
		plg, err := p.Initialize(m)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "plugin init failed"))
			continue
		}
		pn := plg.Name()
		m.mu.Lock()
		_, ok := m.loaded[pn]
		if !ok {
			// not already loaded, add it
			m.loaded[pn] = plg
		}
		// unlock outside of any conditional
		m.mu.Unlock()
		if ok {
			// plugin was already loaded
			errs = append(errs, errors.Errorf("plugin already loaded %s", pn))
			continue
		}
		// run other plugin init handlers
		for _, h := range inits {
			h.HandlePluginInit(plg)
		}
	}
	return errs
}
