// Package cli makes reusable the core parts of the squircy command.
package cli // import "code.dopame.me/veonik/squircy3/cli"

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tilde "gopkg.in/mattes/go-expand-tilde.v1"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"
)

type Config struct {
	RootDir      string   `toml:"root_path" flag:"root"`
	PluginDir    string   `toml:"plugin_path"`
	ExtraPlugins []string `toml:"extra_plugins" flag:"plugin"`

	PluginOptions map[string]interface{} `flag:"plugin_option"`

	LogLevel logrus.Level `toml:"log_level"`

	// Specify additional plugins that are a part of the main executable.
	LinkedPlugins []plugin.Initializer
}

type Manager struct {
	plugins *plugin.Manager

	Config

	stop chan os.Signal
}

func NewManager() (*Manager, error) {
	m := plugin.NewManager()
	// initialize only the config plugin so that it can be configured before
	// other plugins are initialized
	m.RegisterFunc(config.Initialize)
	if err := configure(m); err != nil {
		return nil, err
	}
	conf := Config{}
	// configure the config plugin!
	err := config.ConfigurePlugin(m,
		config.WithRequiredOptions("root_path", "log_level"),
		config.WithFilteredOption("root_path", func(s string, val config.Value) (config.Value, error) {
			vs, ok := val.(string)
			if !ok {
				return nil, errors.Errorf("expected root_path to be string but got %T", vs)
			}
			vs, err := tilde.Expand(vs)
			if err != nil {
				return nil, errors.Errorf("failed to expand root directory: %s", err)
			}
			return vs, nil
		}),
		config.WithFilteredOption("log_level", func(s string, val config.Value) (config.Value, error) {
			if v, ok := val.(logrus.Level); ok {
				return v, nil
			}
			vs, ok := val.(string)
			if !ok {
				return nil, errors.Errorf("expected log_level to be string but got %T", vs)
			}
			lvl, err := logrus.ParseLevel(vs)
			if err != nil {
				lvl = logrus.InfoLevel
				logrus.Warnf("config: defaulting to info log level: failed to parse %s as log level: %s", lvl, err)
			}
			return lvl, nil
		}),
		config.WithInitValue(&conf),
		config.WithValuesFromFlagSet(flag.CommandLine),
		config.WithValuesFromMap(&conf.PluginOptions))
	if err != nil {
		return nil, err
	}
	cf := filepath.Join(conf.RootDir, "config.toml")
	// Now that we have determined the root directory
	if err := config.ConfigurePlugin(m, config.WithValuesFromTOMLFile(cf)); err != nil {
		return nil, err
	}
	return &Manager{
		plugins: m,
		stop:    make(chan os.Signal, 10),
		Config:  conf,
	}, nil
}

func (manager *Manager) Stop() {
	select {
	case <-manager.stop:
		// already stopped
	default:
		close(manager.stop)
	}
}

func (manager *Manager) Plugins() *plugin.Manager {
	return manager.plugins
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
	pluginDir := manager.PluginDir
	if !filepath.IsAbs(pluginDir) {
		pluginDir = filepath.Join(manager.RootDir, pluginDir)
	}
	for _, pl := range manager.ExtraPlugins {
		logrus.Tracef("core: loading extra plugin: %s", pl)
		if !filepath.IsAbs(pl) {
			pl = filepath.Join(pluginDir, pl)
		}
		m.Register(plugin.InitializeFromFile(pl))
	}
	for _, pl := range manager.LinkedPlugins {
		m.Register(pl)
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
	signal.Notify(st, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-manager.stop:
			logrus.Infoln("Shutting down...")
			if err := manager.Shutdown(); err != nil {
				logrus.Warnln("core: error shutting down:", err)
			}
			return nil
		case s := <-st:
			switch s {
			case syscall.SIGHUP:
				myVM, err := vm.FromPlugins(manager.plugins)
				if err != nil {
					logrus.Warnln("core: unable to reload js vm:", err)
					continue
				}
				logrus.Infoln("Reloading javascript vm")
				if err := myVM.Shutdown(); err != nil {
					logrus.Warnln("core: unable to reload js vm:", err)
					continue
				}
				if err := myVM.Start(); err != nil {
					logrus.Warnln("core: unable to restart js vm:", err)
					continue
				}
			case os.Interrupt:
				fallthrough
			case syscall.SIGTERM:
				manager.Stop()
			default:
				logrus.Debugln("core: received signal", s, "but not doing anything with it")
			}
		}
	}
}

func (manager *Manager) Shutdown() error {
	m := manager.plugins
	m.Shutdown()
	return nil
}

func configure(m *plugin.Manager) error {
	errs := m.Configure()
	if errs != nil && len(errs) > 0 {
		if len(errs) > 1 {
			return errors.Wrapf(errs[0], "(and %d more...)", len(errs)-1)
		}
		return errs[0]
	}
	return nil
}
