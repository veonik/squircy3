package main

import (
	"crypto/sha1"
	"fmt"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"code.dopame.me/veonik/squircy3/config"
	"code.dopame.me/veonik/squircy3/plugin"
	"code.dopame.me/veonik/squircy3/plugins/node_compat/native"
	"code.dopame.me/veonik/squircy3/vm"
)

const PluginName = "node_compat"

func main() {
	plugin.Main(PluginName)
}

func Initialize(m *plugin.Manager) (plugin.Plugin, error) {
	vmp, err := vm.FromPlugins(m)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (vm)", PluginName)
	}
	vmp.SetModule(EventEmitter)
	vmp.SetModule(ChildProcess)
	vmp.SetModule(Crypto)
	vmp.SetModule(Stream)
	vmp.SetModule(Net)
	vmp.SetModule(Http)
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
		logrus.Warnf("%s: error initializing runtime: %s", PluginName, err)
	}
	r.Set("exec", v)

	v = r.NewObject()
	if err := v.Set("Sum", func(b []byte) (string, error) {
		return fmt.Sprintf("%x", sha1.Sum(b)), nil
	}); err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", PluginName, err)
	}
	r.Set("sha1", v)

	v = r.NewObject()
	if err := v.Set("Dial", native.Dial); err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", PluginName, err)
	}
	r.Set("native", v)

	_, err := r.RunString(`this.global = this.global || this;
require('core-js-bundle');
this.process = this.process || require('process/browser');
require('regenerator-runtime');`)
	if err != nil {
		logrus.Warnf("%s: error initializing runtime: %s", PluginName, err)
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
	return PluginName
}
