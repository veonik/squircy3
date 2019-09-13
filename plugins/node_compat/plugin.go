package node_compat

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/vm"
)

const pluginName = "node_compat"

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	vmp, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (vm)", pluginName)
	}
	vmp.SetModule(eventEmitter)
	vmp.SetModule(childProcess)
	return &nodeCompatPlugin{}, nil
}

type nodeCompatPlugin struct {
	EnableExec bool
}

func (p *nodeCompatPlugin) HandleRuntimeInit(r *goja.Runtime) {
	if !p.EnableExec {
		return
	}
	v := r.NewObject()
	if err := v.Set("Command", NewProcess); err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", pluginName, err)
	}
	r.Set("exec", v)
	_, err := r.RunString(`this.global = this.global || this;
require('core-js-bundle');
this.process = this.process || require('process/browser');
require('regenerator-runtime');`)
	if err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", pluginName, err)
	}
}

func (p *nodeCompatPlugin) Options() []config.SetupOption {
	return []config.SetupOption{config.WithOption("enable_exec")}
}

func (p *nodeCompatPlugin) Configure(conf config.Config) error {
	p.EnableExec, _ = conf.Bool("enable_exec")
	return nil
}

func (p *nodeCompatPlugin) Name() string {
	return pluginName
}
