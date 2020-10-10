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
	if _, err := m.Lookup("babel"); err != nil {
		return nil, errors.Wrapf(err, "%s: required dependency missing (babel)", PluginName)
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
	if r.Get("Babel") == nil {
		logrus.Warnf("%s: babel is required but not enabled, disabling node_compat", PluginName)
		return
	}
	if p.EnableExec {
		logrus.Debugf("%s: subprocess execution is enabled", PluginName)
		v := r.NewObject()
		if err := v.Set("Command", NewProcess); err != nil {
			logrus.Warnf("%s: error initializing exec.Command: %s", PluginName, err)
		} else {
			r.Set("exec", v)
		}
	} else {
		logrus.Debugf("%s: subprocess execution is disabled", PluginName)
	}

	v := r.NewObject()
	if err := v.Set("Sum", func(b []byte) (string, error) {
		return fmt.Sprintf("%x", sha1.Sum(b)), nil
	}); err != nil {
		logrus.Warnf("%s: error initializing sha1.Sum: %s", PluginName, err)
	} else {
		r.Set("sha1", v)
	}

	v = r.NewObject()
	if err := v.Set("Dial", native.Dial); err != nil {
		logrus.Warnf("%s: error initializing native.Dial: %s", PluginName, err)
	} else {
		r.Set("native", v)
	}

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
