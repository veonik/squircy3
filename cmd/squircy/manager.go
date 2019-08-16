package main

import (
	"code.dopame.me/veonik/squircy3/plugins/babel"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/script"
	"code.dopame.me/veonik/squircy3/vm"
)

type Manager struct {
	plugins *plugin.Manager
	rawConf map[string]interface{}

	RootDir      string   `toml:"root_path"`
	ExtraPlugins []string `toml:"extra_plugins"`

	sig chan os.Signal
}

func NewManager(rootDir string, extraPlugins ...string) (*Manager, error) {
	m := plugin.NewManager()
	// initialize only the config plugin so that it can be configured before
	// other plugins are initialized
	m.RegisterFunc(config.Initialize)
	if err := configure(m); err != nil {
		return nil, err
	}
	// configure the config plugin!
	cf := filepath.Join(rootDir, "config.toml")
	err := config.ConfigurePlugin(m,
		config.WithValuesFromTOMLFile(cf))
	if err != nil {
		return nil, err
	}
	return &Manager{
		plugins: m,
		rawConf: make(map[string]interface{}),
		sig: make(chan os.Signal),
		RootDir: rootDir,
		ExtraPlugins: extraPlugins,
	}, nil
}

func (manager *Manager) Loop() error {
	m := manager.plugins

	// init the remaining built-in plugins
	m.RegisterFunc(event.Initialize)
	m.RegisterFunc(vm.Initialize)
	m.RegisterFunc(irc.Initialize)
	m.RegisterFunc(babel.Initialize)
	m.RegisterFunc(script.Initialize)
	m.Register(plugin.InitializeFromFile(filepath.Join(manager.RootDir, "plugins/squircy2_compat.so")))
	if err := configure(m); err != nil {
		return errors.Wrap(err, "unable to init built-in plugins")
	}

	// start the event dispatcher
	d, err := event.FromPlugins(m)
	if err != nil {
		return errors.Wrap(err, "expected event plugin to exist")
	}
	go d.Loop()

	// start the js runtime
	myVM, err := vm.FromPlugins(m)
	if err != nil {
		return errors.Wrap(err, "expected vm plugin to exist")
	}
	err = myVM.Start()
	if err != nil {
		return errors.Wrap(err, "unable to start vm")
	}

	// load remaining extra plugins
	for _, pl := range manager.ExtraPlugins {
		m.Register(plugin.InitializeFromFile(pl))
	}
	if err := configure(m); err != nil {
		return errors.Wrap(err, "unable to init extra plugins")
	}

	signal.Notify(manager.sig, os.Interrupt, syscall.SIGTERM)
	<-manager.sig
	return nil
}

func configure(m *plugin.Manager) error {
	errs := m.Configure()
	if errs != nil && len(errs) > 0 {
		if len(errs) > 1 {
			return errors.WithMessage(errs[0], fmt.Sprintf("(and %d more...)", len(errs)-1))
		}
		return errs[0]
	}
	return nil
}
