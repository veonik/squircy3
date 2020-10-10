package main

import (
	"path/filepath"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const pluginName = "script"

func main() {
	plugin.Main(pluginName)
}

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
	logrus.Infoln("Loading scripts from", p.manager.rootDir)
	ss, err := p.manager.LoadAll()
	if err != nil {
		logrus.Warnf("script: failed to list directory contents of '%s': %s", p.manager.rootDir, err)
		return
	}
	for _, s := range ss {
		logrus.Infoln("Running script", s.Name)
		pr, err := p.vm.Compile(s.Name, s.Body)
		if err != nil {
			logrus.Warnf("script: failed to compile script (%s): %s", s.Name, err)
			return
		}
		_, err = r.RunProgram(pr)
		if err != nil {
			logrus.Warnf("script: error while running script (%s): %s", s.Name, err)
		}
	}
}

func (p *scriptPlugin) Options() []config.SetupOption {
	return []config.SetupOption{
		config.WithRequiredOption("scripts_path"),
		config.WithInheritedOption("root_path")}
}

func (p *scriptPlugin) Configure(conf config.Config) error {
	r, ok := conf.String("scripts_path")
	if !ok {
		r = "scripts"
	}
	if !filepath.IsAbs(r) {
		rr, ok := conf.String("root_path")
		if ok {
			r = filepath.Join(rr, r)
		}
	}
	p.manager = &Manager{rootDir: r}
	return nil
}

func (p *scriptPlugin) Name() string {
	return pluginName
}
