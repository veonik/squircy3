package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"
)

type Config struct {
	RootDir      string   `toml:"root_path"`
	PluginDir    string   `toml:"plugin_path"`
	ExtraPlugins []string `toml:"extra_plugins"`
}

type Manager struct {
	plugins *plugin.Manager

	Config

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
	conf := Config{
		RootDir:      rootDir,
		PluginDir:    filepath.Join(rootDir, "plugins"),
		ExtraPlugins: extraPlugins,
	}
	// configure the config plugin!
	cf := filepath.Join(rootDir, "config.toml")
	err := config.ConfigurePlugin(m,
		config.WithInitValue(&conf),
		config.WithValuesFromTOMLFile(cf))
	if err != nil {
		return nil, err
	}
	return &Manager{
		plugins: m,
		sig:     make(chan os.Signal, 10),
		Config:  conf,
	}, nil
}

func (manager *Manager) Start() error {
	m := manager.plugins

	// init the remaining built-in plugins
	m.RegisterFunc(event.Initialize)
	m.RegisterFunc(vm.Initialize)
	m.RegisterFunc(irc.Initialize)
	if err := configure(m); err != nil {
		return errors.Wrap(err, "unable to init built-in plugins")
	}

	// load remaining extra plugins
	for _, pl := range manager.ExtraPlugins {
		if !filepath.IsAbs(pl) {
			pl = filepath.Join(manager.PluginDir, pl)
		}
		m.Register(plugin.InitializeFromFile(pl))
	}
	if err := configure(m); err != nil {
		return errors.Wrap(err, "unable to init extra plugins")
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
	return nil
}

func (manager *Manager) Loop() error {
	st := make(chan os.Signal, 10)
	signal.Notify(st, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
	signal.Notify(manager.sig, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case s := <-st:
			switch s {
			case syscall.SIGHUP:
				myVM, err := vm.FromPlugins(manager.plugins)
				if err != nil {
					logrus.Warnln("unable to reload js vm:", err)
					continue
				}
				logrus.Infoln("reloading javascript vm")
				if err := myVM.Shutdown(); err != nil {
					logrus.Warnln("unable to reload js vm:", err)
					continue
				}
				if err := myVM.Start(); err != nil {
					logrus.Warnln("unable to restart js vm:", err)
					continue
				}
			default:
				logrus.Infoln("received signal", s, "but not doing anything with it")
			}

		case <-manager.sig:
			logrus.Infoln("shutting down")
			if err := manager.Shutdown(); err != nil {
				logrus.Warnln("error shutting down:", err)
			}
			return nil
		}
	}
}

func (manager *Manager) Shutdown() error {
	m := manager.plugins

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		myVM, err := vm.FromPlugins(m)
		if err != nil {
			logrus.Warnln("error shutting down vm:", err)
		}

		if err := myVM.Shutdown(); err != nil {
			logrus.Warnln("error shutting down vm:", err)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		d, err := event.FromPlugins(m)
		if err != nil {
			logrus.Warnln("error shutting down events:", err)
		}
		d.Stop()
		wg.Done()
	}()
	wg.Wait()
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
