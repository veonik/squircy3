package squircy2_compat

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/event"
	"code.dopame.me/veonik/squircy3/irc"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
)

const PluginName = "squircy2_compat"

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	im, err := irc.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (irc)", PluginName)
	}
	ev, err := event.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (event)", PluginName)
	}
	v, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (vm)", PluginName)
	}
	return &shimPlugin{NewHelperSet(ev, v, im)}, nil
}

type shimPlugin struct {
	*HelperSet
}

func (p *shimPlugin) Configure(c config.Config) error {
	if gcv, ok := c.Self().(*Config); ok {
		return p.HelperSet.Configure(*gcv)
	}
	cf := Config{}
	cf.EnableFileAPI, _ = c.Bool("enable_file_api")
	cf.FileAPIPath, _ = c.String("file_api_root")
	cf.OwnerNick, _ = c.String("owner_nick")
	cf.OwnerHost, _ = c.String("owner_host")
	cf.DataPath, _ = c.String("data_path")
	cf.RootDir, _ = c.String("root_path")
	return p.HelperSet.Configure(cf)
}

func (p *shimPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithInitValue(&Config{}), config.WithInheritedOption("root_path")}
}

func (p *shimPlugin) Name() string {
	return PluginName
}

func (p *shimPlugin) PrependRuntimeInitHandler() bool {
	return true
}

func (p *shimPlugin) HandleRuntimeInit(gr *goja.Runtime) {
	p.Enable(gr)
}
