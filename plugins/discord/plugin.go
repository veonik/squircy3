package discord

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/plugin"
	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const PluginName = "discord"

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	ev, err := event.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (event)", PluginName)
	}
	return &discordPlugin{NewManager(ev)}, nil
}

type discordPlugin struct {
	manager *Manager
}

func (p *discordPlugin) Configure(c config.Config) error {
	if gcv, ok := c.Self().(*Config); ok {
		return p.manager.Configure(*gcv)
	}
	cf := Config{}
	cf.Token, _ = c.String("token")
	cf.OwnerID, _ = c.String("owner")
	cf.ActivityName, _ = c.String("activity")
	return p.manager.Configure(cf)
}

func (p *discordPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithInitValue(&Config{})}
}

func (p *discordPlugin) Name() string {
	return PluginName
}

// must logs the given error as a warning
func must(what string, err error) {
	if err != nil {
		logrus.Warnf("%s: error %s: %s", PluginName, what, err)
	}
}

func (p *discordPlugin) HandleRuntimeInit(gr *goja.Runtime) {
	v := gr.NewObject()
	must("setting connect", v.Set("connect", p.manager.Connect))
	must("setting messageChannel", v.Set("messageChannel", p.manager.MessageChannel))
	must("setting messageChannelTTS", v.Set("messageChannelTTS", p.manager.MessageChannelTTS))
	must("setting getCurrentUsername", v.Set("getCurrentUsername", p.manager.CurrentUsername))
	must("setting getOwnerID", v.Set("getOwnerID", p.manager.OwnerID))
	if err := gr.Set("discord", v); err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", PluginName, err)
	}
}

func (p *discordPlugin) HandleShutdown() {
	if p.manager == nil {
		logrus.Warnf("%s: shutting down uninitialized plugin", PluginName)
		return
	}
	if err := p.manager.Disconnect(); err != nil {
		if err != ErrNotConnected {
			logrus.Warnf("%s: failed to disconnect before shutting down: %s", PluginName, err)
		}
	}
}
