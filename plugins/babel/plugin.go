package babel // import "code.dopame.me/veonik/squircy3/plugins/babel"

import (
	"time"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	babel "code.dopame.me/veonik/squircy3/plugins/babel/transformer"
	"code.dopame.me/veonik/squircy3/vm"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const PluginName = "babel"

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	v, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (vm)", PluginName)
	}
	return &babelPlugin{vm: v}, nil
}

type babelPlugin struct {
	vm     *vm.VM
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
	return PluginName
}

func (p *babelPlugin) PrependRuntimeInitHandler() bool {
	return true
}

func (p *babelPlugin) HandleRuntimeInit(gr *goja.Runtime) {
	p.vm.SetTransformer(nil)
	if !p.enable {
		logrus.Debugf("babel: disabled, not initializing")
		return
	}
	logrus.Infoln("Initializing babel.js transformer...")
	st := time.Now()
	b, err := babel.New(gr)
	if err != nil {
		logrus.Warnln("babel: failed to initialize babel.js:", err)
		return
	}
	logrus.Infof("Initialized babel.js transformer (took %s)", time.Now().Sub(st))
	p.vm.SetTransformer(b.Transform)
}
