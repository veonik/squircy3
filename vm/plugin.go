package vm

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const pluginName = "vm"

func pluginFromPlugins(m *plugin.Manager) (*vmPlugin, error) {
	p, err := m.Lookup(pluginName)
	if err != nil {
		return nil, err
	}
	mp, ok := p.(*vmPlugin)
	if !ok {
		return nil, errors.Errorf("vm: received unexpected plugin type")
	}
	return mp, nil
}

func FromPlugins(m *plugin.Manager) (*VM, error) {
	mp, err := pluginFromPlugins(m)
	if err != nil {
		return nil, err
	}
	if mp.vm == nil {
		return nil, errors.New("vm: plugin is not configured")
	}
	return mp.vm, nil
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	p := &vmPlugin{}
	return p, nil
}

type vmPlugin struct {
	vm *VM
}

func (p *vmPlugin) Configure(conf config.Config) error {
	r, ok := conf.String("modules_path")
	if !ok {
		return errors.New("vm: modules_path cannot be empty")
	}
	vm, err := New(NewRegistry(r))
	if err != nil {
		return err
	}
	p.vm = vm
	return nil
}

func (p *vmPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithRequiredOption("modules_path")}
}

func (p *vmPlugin) Name() string {
	return pluginName
}

type RuntimeInitHandler interface {
	HandleRuntimeInit(r *goja.Runtime)
}

type PrependRuntimeInitHandler interface {
	PrependRuntimeInitHandler() bool
}

func (p *vmPlugin) HandlePluginInit(o plugin.Plugin) {
	if p.vm == nil {
		logrus.Warnln("vm: handling another plugin init before being configured", o.Name())
		return
	}
	if ih, ok := o.(RuntimeInitHandler); ok {
		if oh, ok := ih.(PrependRuntimeInitHandler); ok && oh.PrependRuntimeInitHandler() {
			p.vm.PrependRuntimeInit(ih.HandleRuntimeInit)
		} else {
			p.vm.OnRuntimeInit(ih.HandleRuntimeInit)
		}
	}
}

func (p *vmPlugin) HandleShutdown() {
	if p.vm == nil {
		logrus.Warnln("vm: shutting down uninitialized plugin")
		return
	}
	if err := p.vm.Shutdown(); err != nil {
		logrus.Warnln("error shutting down vm:", err)
	}
}