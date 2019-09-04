package script

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const pluginName = "script"

// Initialize is a valid plugin.Initializer
func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	vp, err := vm.FromPlugins(m)
	if err != nil {
		return nil, err
	}
	p := &scriptPlugin{vm: vp}
	return p, nil
}

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

func pluginFromPlugins(m *plugin.Manager) (*scriptPlugin, error) {
	p, err := m.Lookup(pluginName)
	if err != nil {
		return nil, err
	}
	mp, ok := p.(*scriptPlugin)
	if !ok {
		return nil, errors.Errorf("%s: received unexpected plugin type", pluginName)
	}
	return mp, nil
}

type scriptPlugin struct {
	vm      *vm.VM
	manager *Manager
}

func (p *scriptPlugin) HandleRuntimeInit(r *goja.Runtime) {
	logrus.Infoln("loading scripts from", p.manager.rootDir)
	ss, err := p.manager.LoadAll()
	if err != nil {
		logrus.Warnln("error loading scripts at runtime init:", err)
		return
	}
	for _, s := range ss {
		logrus.Infoln("running script", s.Name)
		pr, err := p.vm.Compile(s.Name, s.Body)
		if err != nil {
			logrus.Warnln("error compiling script", s.Name, err)
			return
		}
		_, err = r.RunProgram(pr)
		if err != nil {
			logrus.Warnln("error running script", s.Name, err)
		}
	}
}

func (p *scriptPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithRequiredOption("scripts_path")}
}

func (p *scriptPlugin) Configure(conf config.Config) error {
	r, ok := conf.String("scripts_path")
	if !ok {
		return errors.Errorf("%s: scripts_path cannot be empty", pluginName)
	}
	p.manager = &Manager{rootDir: r}
	return nil
}

func (p *scriptPlugin) Name() string {
	return pluginName
}
