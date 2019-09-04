package babel

import (
	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const pluginName = "babel"

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	v, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (vm)", pluginName)
	}
	return &babelPlugin{vm: v}, nil
}

type babelPlugin struct {
	vm *vm.VM
	enable bool
}

func (p *babelPlugin) Configure(c config.Config) error {
	p.enable, _ = c.Bool("enable")
	return nil
}

func (p *babelPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithOption("enable")}
}

func (p *babelPlugin) Name() string {
	return pluginName
}

func (p *babelPlugin) PrependRuntimeInitHandler() bool {
	return true
}

func (p *babelPlugin) HandleRuntimeInit(gr *goja.Runtime) {
	if !p.enable {
		return
	}
	p.vm.SetTransformer(nil)
	b, err := NewBabel(gr)
	if err != nil {
		logrus.Warnln("unable to run babel init script:", err)
		return
	}
	p.vm.SetTransformer(b.Transform)
}
