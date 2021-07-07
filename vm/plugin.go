package vm

import (
	"os"
	"path/filepath"

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

// FromPlugins returns the vm plugin's VM or an error if it fails.
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

// Initialize is a plugin.Initializer that initializes a vm plugin.
func Initialize(*plugin.Manager) (plugin.Plugin, error) {
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
	if !filepath.IsAbs(r) {
		if rr, ok := conf.String("root_path"); ok {
			r = filepath.Join(rr, r)
		}
	}
	logrus.Debugf("vm: configured with modules_path: %s", r)
	if _, err := os.Stat(r); os.IsNotExist(err) {
		logrus.Warnf("vm: modules_path '%s' does not exist, perhaps you need to run `yarn install`?", r)
	}
	vm, err := New(NewRegistry(r))
	if err != nil {
		return err
	}
	p.vm = vm
	return nil
}

func (p *vmPlugin) Options() []config.SetupOption {
	return []config.SetupOption{
		config.WithRequiredOption("modules_path"),
		config.WithInheritedOption("root_path")}
}

func (p *vmPlugin) Name() string {
	return pluginName
}

// A RuntimeInitHandler initializes a newly created goja.Runtime.
type RuntimeInitHandler interface {
	// Initialize and configure the given runtime.
	HandleRuntimeInit(r *goja.Runtime)
}

// A PrependRuntimeInitHandler is a RuntimeInitHandler that may be added at
// the start of the list of handlers.
type PrependRuntimeInitHandler interface {
	RuntimeInitHandler
	// PrependRuntimeInitHandler returns true if the handler should be added
	// to the start of the list of handlers.
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
