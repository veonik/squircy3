package config

import (
	"code.dopame.me/veonik/squircy3/plugin"

	"github.com/pkg/errors"
)

func pluginFromPlugins(m *plugin.Manager) (*configPlugin, error) {
	p, err := m.Lookup("config")
	if err != nil {
		return nil, err
	}
	mp, ok := p.(*configPlugin)
	if !ok {
		return nil, errors.Errorf("invalid config: unexpected value type")
	}
	return mp, nil
}

func ConfigurePlugin(m *plugin.Manager, opts ...SetupOption) error {
	mp, err := pluginFromPlugins(m)
	if err != nil {
		return err
	}
	return mp.Configure(opts...)
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	p := &configPlugin{}
	return p, nil
}

type configPlugin struct {
	baseOptions []SetupOption
	current     Config
}

// A configurablePlugin is a plugin that can be configured using this package.
type configurablePlugin interface {
	plugin.Plugin

	Options() []SetupOption
	Configure(config Config) error
}

func (p *configPlugin) HandlePluginInit(op plugin.Plugin) {
	cp, ok := op.(configurablePlugin)
	if !ok {
		return
	}
	err := p.Configure(WithGenericSection(cp.Name(), cp.Options()...))
	if err != nil {
		panic(err)
	}
	v, err := p.current.Section(cp.Name())
	if err != nil {
		panic(err)
	}
	err = cp.Configure(v)
	if err != nil {
		panic(err)
	}
}

func (p *configPlugin) Name() string {
	return "config"
}

func (p *configPlugin) Configure(opts ...SetupOption) error {
	p.baseOptions = append(p.baseOptions, opts...)
	nc, err := Wrap(p.current, p.baseOptions...)
	if err != nil {
		return err
	}
	p.current = nc
	return nil
}
