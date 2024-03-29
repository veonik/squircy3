package irc

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/plugin"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const pluginName = "irc"

func pluginFromPlugins(m *plugin.Manager) (*ircPlugin, error) {
	p, err := m.Lookup(pluginName)
	if err != nil {
		return nil, err
	}
	mp, ok := p.(*ircPlugin)
	if !ok {
		return nil, errors.Errorf("%s: received unexpected plugin type", pluginName)
	}
	return mp, nil
}

// FromPlugins returns the irc plugin's Manager or an error if it fails.
func FromPlugins(m *plugin.Manager) (*Manager, error) {
	mp, err := pluginFromPlugins(m)
	if err != nil {
		return nil, err
	}
	if mp.manager == nil {
		return nil, errors.Errorf("%s: plugin is not configured", pluginName)
	}
	return mp.manager, nil
}

// Initialize is a plugin.Initializer that initializes an irc plugin.
func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	ev, err := event.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: missing required dependency (event)", pluginName)
	}
	p := &ircPlugin{events: ev}
	return p, nil
}

type ircPlugin struct {
	events *event.Dispatcher

	manager *Manager
}

func (p *ircPlugin) Configure(c config.Config) error {
	co, err := configFromGeneric(c)
	if err != nil {
		return err
	}
	p.manager = NewManager(co, p.events)
	return nil
}

func (p *ircPlugin) HandleShutdown() {
	if p.manager == nil {
		logrus.Warnln("irc: shutting down uninitialized plugin")
		return
	}
	if err := p.manager.Disconnect(); err != nil {
		if err != ErrNotConnected {
			logrus.Warnln("irc: failed to disconnect before shutting down:", err)
		}
	}
}

func configFromGeneric(g config.Config) (c *Config, err error) {
	if gcv, ok := g.Self().(*Config); ok {
		return gcv, nil
	}
	return c, errors.Errorf("%s: value is not a *irc.Config", pluginName)
}

func (p *ircPlugin) Options() []config.SetupOption {
	return []config.SetupOption{
		config.WithInitValue(&Config{}),
		config.WithRequiredOptions("nick", "user", "network")}
}

func (p *ircPlugin) Name() string {
	return pluginName
}
